// Package storage is the document vault's storage abstraction. It writes uploaded
// files to S3-compatible object storage (Cloudflare R2 / AWS S3 / MinIO) when
// configured, and falls back to local disk for dev / before a bucket is
// provisioned. The rest of the app only depends on the Store interface, so the
// backend can change without touching callers.
//
// Configuration (object storage active when S3_BUCKET + S3_ACCESS_KEY are set):
//
//	S3_ENDPOINT    host only, no scheme — e.g. <acct>.r2.cloudflarestorage.com
//	S3_BUCKET      bucket name
//	S3_ACCESS_KEY  / S3_SECRET_KEY
//	S3_REGION      default "auto" (R2) — use the AWS region for S3
//	S3_USE_SSL     "false" to disable TLS (local MinIO); default true
//
// Encryption at rest (applied to every uploaded object):
//
//	S3_SSE_MODE    "" / "none" (default), "s3" (SSE-S3, AES256, provider-managed),
//	               or "kms" (SSE-KMS with a customer-managed key)
//	S3_SSE_KMS_KEY KMS key id — required when S3_SSE_MODE=kms
//
// Note: Cloudflare R2 and modern AWS S3 encrypt at rest by default; "kms" lets
// you control the key for the most sensitive (patient) records. SSE-S3/KMS are
// transparent on read — no key is needed to fetch.
//
// Local fallback writes under storage/uploads/<key> and serves via the app's
// public /uploads static route — acceptable for dev only; production must use
// object storage so the private vault and ×N replicas work correctly.
package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/encrypt"
)

// Store is the document vault's storage backend.
type Store interface {
	// Put writes a PRIVATE object at key. contentType is the MIME type. Private
	// objects are read via PresignedGetURL after an access check.
	Put(ctx context.Context, key, contentType string, r io.Reader, size int64) error
	// PutPublic writes an object meant to be served to everyone (blog covers,
	// product images) and returns a stable, cacheable public URL (via the CDN
	// base, or a long-lived signed URL as a fallback). Public objects never use
	// SSE-KMS so anonymous CDN reads can decrypt.
	PutPublic(ctx context.Context, key, contentType string, r io.Reader, size int64) (string, error)
	// PresignedGetURL returns a time-limited URL to read a private object.
	PresignedGetURL(ctx context.Context, key string, ttl time.Duration) (string, error)
	// Delete removes the object (best-effort).
	Delete(ctx context.Context, key string) error
	// Backend reports "s3" or "local" for diagnostics.
	Backend() string
}

// publicFallbackTTL is how long a public object's signed URL lasts when no CDN
// base is configured.
const publicFallbackTTL = 7 * 24 * time.Hour

// New picks the backend from the environment: object storage if configured,
// otherwise local disk. A misconfigured S3 falls back to local (with a log)
// rather than blocking startup.
func New() (Store, error) {
	bucket := strings.TrimSpace(os.Getenv("S3_BUCKET"))
	access := strings.TrimSpace(os.Getenv("S3_ACCESS_KEY"))
	if bucket == "" || access == "" {
		return newLocalStore()
	}
	s, err := newS3Store(bucket, access)
	if err != nil {
		log.Printf("[storage] S3 init failed (%v) — falling back to local disk", err)
		return newLocalStore()
	}
	return s, nil
}

// ── S3-compatible backend ────────────────────────────────────────────────────

type s3Store struct {
	client     *minio.Client
	bucket     string
	sse        encrypt.ServerSide // nil = rely on provider/bucket default
	publicBase string             // CDN/public-bucket base for PutPublic
}

func newS3Store(bucket, access string) (Store, error) {
	endpoint := strings.TrimSpace(os.Getenv("S3_ENDPOINT"))
	if endpoint == "" {
		return nil, fmt.Errorf("storage: S3_ENDPOINT required when S3_BUCKET is set")
	}
	secret := strings.TrimSpace(os.Getenv("S3_SECRET_KEY"))
	region := strings.TrimSpace(os.Getenv("S3_REGION"))
	if region == "" {
		region = "auto" // Cloudflare R2 default
	}
	useSSL := !strings.EqualFold(strings.TrimSpace(os.Getenv("S3_USE_SSL")), "false")

	sse, err := sseFromEnv()
	if err != nil {
		return nil, err
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(access, secret, ""),
		Secure: useSSL,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: init s3 client: %w", err)
	}
	if sse != nil {
		log.Printf("[storage] server-side encryption enabled (%s)", strings.ToLower(os.Getenv("S3_SSE_MODE")))
	}
	return &s3Store{
		client:     client,
		bucket:     bucket,
		sse:        sse,
		publicBase: strings.TrimRight(os.Getenv("S3_PUBLIC_BASE"), "/"),
	}, nil
}

// sseFromEnv builds the server-side-encryption config from S3_SSE_MODE.
func sseFromEnv() (encrypt.ServerSide, error) {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("S3_SSE_MODE"))) {
	case "", "none":
		return nil, nil
	case "s3":
		return encrypt.NewSSE(), nil
	case "kms":
		key := strings.TrimSpace(os.Getenv("S3_SSE_KMS_KEY"))
		if key == "" {
			return nil, fmt.Errorf("storage: S3_SSE_KMS_KEY required when S3_SSE_MODE=kms")
		}
		return encrypt.NewSSEKMS(key, nil)
	default:
		return nil, fmt.Errorf("storage: invalid S3_SSE_MODE (use none, s3, or kms)")
	}
}

func (s *s3Store) Put(ctx context.Context, key, contentType string, r io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{
		ContentType:          contentType,
		ServerSideEncryption: s.sse, // nil is fine — provider/bucket default applies
	})
	return err
}

func (s *s3Store) PutPublic(ctx context.Context, key, contentType string, r io.Reader, size int64) (string, error) {
	// No SSE-KMS on public objects — anonymous CDN reads can't decrypt KMS.
	if _, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType}); err != nil {
		return "", err
	}
	if s.publicBase != "" {
		return s.publicBase + "/" + key, nil
	}
	// No CDN configured — fall back to a long-lived signed URL.
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, publicFallbackTTL, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *s3Store) PresignedGetURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, ttl, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *s3Store) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *s3Store) Backend() string { return "s3" }

// ── Local disk fallback (dev only) ───────────────────────────────────────────

type localStore struct {
	root    string // storage/uploads
	baseURL string // public base for /uploads serving
}

func newLocalStore() (Store, error) {
	root := "storage/uploads"
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("storage: prepare local dir: %w", err)
	}
	base := strings.TrimRight(os.Getenv("ASSET_BASE_URL"), "/")
	return &localStore{root: root, baseURL: base}, nil
}

func (l *localStore) Put(_ context.Context, key, _ string, r io.Reader, _ int64) error {
	dest := filepath.Join(l.root, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (l *localStore) PutPublic(ctx context.Context, key, contentType string, r io.Reader, size int64) (string, error) {
	if err := l.Put(ctx, key, contentType, r, size); err != nil {
		return "", err
	}
	return l.baseURL + "/uploads/" + key, nil
}

func (l *localStore) PresignedGetURL(_ context.Context, key string, _ time.Duration) (string, error) {
	// No real signing locally — return the public static URL. Dev only.
	return l.baseURL + "/uploads/" + key, nil
}

func (l *localStore) Delete(_ context.Context, key string) error {
	dest := filepath.Join(l.root, filepath.FromSlash(key))
	err := os.Remove(dest)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (l *localStore) Backend() string { return "local" }
