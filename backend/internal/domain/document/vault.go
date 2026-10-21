package document

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"drayya-core/internal/crypto"
	"drayya-core/internal/services/storage"

	"gorm.io/gorm"
)

// Handler owns the vault's DB + object-storage backend. The Fiber HTTP upload
// handlers that lived alongside it were removed with the REST server; this
// retains the process-wide vault so server-side code (e.g. EMR prescription
// PDFs) can still store private files through one audited path.
type Handler struct {
	db    *gorm.DB
	store storage.Store
}

// NewHandler builds a vault handler bound to a DB and storage backend.
func NewHandler(db *gorm.DB, store storage.Store) *Handler {
	return &Handler{db: db, store: store}
}

// defaultHandler lets other packages store files in the vault without re-wiring
// the storage backend. Set once at startup via SetDefault.
var defaultHandler *Handler

// SetDefault registers the process-wide vault handler.
func SetDefault(h *Handler) { defaultHandler = h }

// SavePrivateReader streams a reader into the private vault and records the
// Document row. Returns an error (handled gracefully by callers) when the vault
// has not been initialized.
func SavePrivateReader(ctx context.Context, r io.Reader, size int64, originalName, contentType, category, ownerType string, ownerID uint) (*Document, error) {
	if defaultHandler == nil {
		return nil, errors.New("document vault not initialized")
	}
	h := defaultHandler
	token := crypto.SecureToken(24)
	ext := strings.ToLower(filepath.Ext(originalName))
	if ext == "" {
		ext = ".bin"
	}
	key := ownerType + "/"
	if ownerID > 0 {
		key += strconv.FormatUint(uint64(ownerID), 10)
	} else {
		key += "inbox"
	}
	key += "/" + category + "_" + token + ext
	if err := h.store.Put(ctx, key, contentType, r, size); err != nil {
		return nil, err
	}
	doc := Document{
		OwnerType: ownerType, OwnerID: ownerID, Category: category,
		StorageKey: key, OriginalName: filepath.Base(originalName), ContentType: contentType,
		SizeBytes: size, Status: StatusPending, Visibility: VisibilityPrivate,
		Version: 1, AccessToken: token,
	}
	if err := h.db.WithContext(ctx).Create(&doc).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}
