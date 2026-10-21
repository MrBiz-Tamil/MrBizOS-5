package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// When S3 isn't configured, New() returns the local-disk store and round-trips a
// file: Put writes it under storage/uploads/<key>, PresignedGetURL builds the
// public /uploads URL, and Delete removes it.
func TestLocalStore_RoundTrip(t *testing.T) {
	// Isolate the working dir so we don't pollute the repo.
	dir := t.TempDir()
	wd, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer os.Chdir(wd)

	t.Setenv("S3_BUCKET", "")
	t.Setenv("S3_ACCESS_KEY", "")
	t.Setenv("ASSET_BASE_URL", "https://cdn.example.com")

	store, err := New()
	require.NoError(t, err)
	assert.Equal(t, "local", store.Backend())

	ctx := context.Background()
	key := "application/inbox/passport_photo_abc123.png"
	require.NoError(t, store.Put(ctx, key, "image/png", strings.NewReader("fake-bytes"), 9))

	// File landed on disk.
	onDisk := filepath.Join(dir, "storage/uploads", filepath.FromSlash(key))
	_, statErr := os.Stat(onDisk)
	require.NoError(t, statErr, "uploaded file should exist on disk")

	url, err := store.PresignedGetURL(ctx, key, 0)
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.example.com/uploads/"+key, url)

	// PutPublic stores and returns a stable public URL.
	pubKey := "content/1/blog_image_xyz.png"
	pubURL, err := store.PutPublic(ctx, pubKey, "image/png", strings.NewReader("img"), 3)
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.example.com/uploads/"+pubKey, pubURL)
	_, statErr2 := os.Stat(filepath.Join(dir, "storage/uploads", filepath.FromSlash(pubKey)))
	assert.NoError(t, statErr2, "public asset should exist on disk")

	require.NoError(t, store.Delete(ctx, key))
	_, statErr = os.Stat(onDisk)
	assert.True(t, os.IsNotExist(statErr), "file should be gone after Delete")

	// Delete of a missing key is a no-op, not an error.
	assert.NoError(t, store.Delete(ctx, key))
}

// sseFromEnv parses the encryption mode and enforces a KMS key when required.
func TestSSEFromEnv(t *testing.T) {
	cases := []struct {
		mode, key string
		wantSSE   bool
		wantErr   bool
	}{
		{"", "", false, false},
		{"none", "", false, false},
		{"s3", "", true, false},
		{"kms", "key-123", true, false},
		{"kms", "", false, true},   // missing key
		{"bogus", "", false, true}, // invalid mode
	}
	for _, c := range cases {
		t.Setenv("S3_SSE_MODE", c.mode)
		t.Setenv("S3_SSE_KMS_KEY", c.key)
		sse, err := sseFromEnv()
		if c.wantErr {
			assert.Error(t, err, "mode %q", c.mode)
			continue
		}
		require.NoError(t, err, "mode %q", c.mode)
		assert.Equal(t, c.wantSSE, sse != nil, "mode %q sse presence", c.mode)
	}
}
