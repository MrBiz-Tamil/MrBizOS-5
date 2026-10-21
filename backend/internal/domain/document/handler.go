package document

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"drayya-core/internal/crypto"
	"drayya-core/internal/middleware"
	"drayya-core/internal/services/storage"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Handler struct {
	db    *gorm.DB
	store storage.Store
}

func NewHandler(db *gorm.DB, store storage.Store) *Handler {
	return &Handler{db: db, store: store}
}

// defaultHandler lets other packages (e.g. the legacy health upload) store files
// in the vault without re-wiring the storage backend. Set once at startup.
var defaultHandler *Handler

// SetDefault registers the process-wide vault handler.
func SetDefault(h *Handler) { defaultHandler = h }

// SavePatientFile stores a patient's file in the vault and returns the row. Used
// by other handlers so every upload lands in one private, audited store instead
// of a per-domain mock. category falls back to a generic 10 MB image/PDF rule.
func SavePatientFile(ctx context.Context, fh *multipart.FileHeader, category string, patientID uint) (*Document, error) {
	if defaultHandler == nil {
		return nil, errors.New("document vault not initialized")
	}
	rule, ok := patientCategoryRules[category]
	if !ok {
		category, rule = "other", imgPdf(10)
	}
	doc, _, err := defaultHandler.saveFile(ctx, fh, category, rule, OwnerPatient, patientID, &patientID, false)
	return doc, err
}

// signedURLTTL is how long a presigned read URL stays valid.
const signedURLTTL = 15 * time.Minute

// extMime maps an allowed extension to its canonical MIME type. We derive the
// stored content type from the (validated) extension rather than trusting the
// client-declared type.
var extMime = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
}

// categoryRule constrains what each document category accepts.
type categoryRule struct {
	maxBytes int64
	exts     map[string]bool
}

func imgPdf(maxMB int64) categoryRule {
	return categoryRule{maxBytes: maxMB << 20, exts: map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".pdf": true}}
}

var categoryRules = map[string]categoryRule{
	"passport_photo": {maxBytes: 5 << 20, exts: map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}},
	"resume":         {maxBytes: 10 << 20, exts: map[string]bool{".pdf": true, ".doc": true, ".docx": true}},
	"license_cert":   imgPdf(10),
	"degree":         imgPdf(10),
	"id_proof":       imgPdf(10),
	"agreement":      {maxBytes: 10 << 20, exts: map[string]bool{".pdf": true}},
}

// patientCategoryRules constrain the categories a patient may upload to their own
// medical-record vault.
var patientCategoryRules = map[string]categoryRule{
	"lab_report":   imgPdf(15),
	"scan":         imgPdf(15),
	"prescription": imgPdf(15),
	"insurance":    imgPdf(15),
	"other":        imgPdf(15),
}

var imageExts = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}

// publicCategoryRules are CDN-served assets (no auth on read).
var publicCategoryRules = map[string]categoryRule{
	"product_image": {maxBytes: 5 << 20, exts: imageExts},
	"blog_image":    {maxBytes: 5 << 20, exts: imageExts},
}

// privateExtraRules are private docs uploaded by other domains (insurance,
// prescription PDFs) beyond the applicant/patient sets above.
var privateExtraRules = map[string]categoryRule{
	"prescription_pdf": {maxBytes: 15 << 20, exts: map[string]bool{".pdf": true}},
}

// ruleFor resolves the validation rule for any known category.
func ruleFor(category string) (categoryRule, bool) {
	for _, m := range []map[string]categoryRule{categoryRules, patientCategoryRules, publicCategoryRules, privateExtraRules} {
		if r, ok := m[category]; ok {
			return r, true
		}
	}
	return categoryRule{}, false
}

// SavePrivateFile stores a private file for any owner (insurance docs, Rx PDFs)
// and returns the row. Served later via a signed URL after an access check.
func SavePrivateFile(ctx context.Context, fh *multipart.FileHeader, category, ownerType string, ownerID uint, uploadedBy *uint) (*Document, error) {
	if defaultHandler == nil {
		return nil, errors.New("document vault not initialized")
	}
	rule, ok := ruleFor(category)
	if !ok {
		category, rule = "other", imgPdf(15)
	}
	doc, _, err := defaultHandler.saveFile(ctx, fh, category, rule, ownerType, ownerID, uploadedBy, false)
	return doc, err
}

// SavePrivateReader stores a private file from a reader (e.g. a server-generated
// PDF) rather than a multipart upload, and returns the row.
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

// SignedURLForID returns a short-lived read URL for a document by id, WITHOUT an
// access check — the caller is responsible for authorizing (e.g. the EMR PDF
// download already checks consent before calling this).
func SignedURLForID(ctx context.Context, id uint) (string, error) {
	if defaultHandler == nil {
		return "", errors.New("document vault not initialized")
	}
	var doc Document
	if err := defaultHandler.db.First(&doc, id).Error; err != nil {
		return "", err
	}
	return defaultHandler.store.PresignedGetURL(ctx, doc.StorageKey, signedURLTTL)
}

// SavePublicAsset stores a CDN-served public asset (blog cover, product image)
// and returns the row with its stable PublicURL.
func SavePublicAsset(ctx context.Context, fh *multipart.FileHeader, category, ownerType string, ownerID uint) (*Document, error) {
	if defaultHandler == nil {
		return nil, errors.New("document vault not initialized")
	}
	rule, ok := publicCategoryRules[category]
	if !ok {
		return nil, errors.New("not a public asset category")
	}
	doc, _, err := defaultHandler.saveFile(ctx, fh, category, rule, ownerType, ownerID, nil, true)
	return doc, err
}

// staffRoles may manage applicant/doctor records (credentialing). Patient records
// are NOT covered here — those use the consent path (owner or granted doctor).
var staffRoles = map[string]bool{
	"hr_manager": true, "hr": true, "chief_doctor": true,
	"hospital_head_doctor": true, "manager": true,
}

// canAccess decides whether the caller may read documents for a given owner.
// Applicant/doctor docs → staff (or the doctor themselves). Patient docs → the
// patient, or a doctor holding an active EMR access grant. Admin always.
func (h *Handler) canAccess(c *fiber.Ctx, ownerType string, ownerID uint) bool {
	role, _ := c.Locals("role").(string)
	uid, _ := c.Locals("user_id").(uint)
	if role == "admin" || role == "super_admin" {
		return true
	}
	switch ownerType {
	case OwnerApplication:
		return staffRoles[role]
	case OwnerDoctor:
		return staffRoles[role] || (role == "doctor" && uid == ownerID)
	case OwnerPatient:
		if uid == ownerID {
			return true // the patient themselves
		}
		return role == "doctor" && h.doctorHasActiveGrant(uid, ownerID)
	}
	return false
}

// doctorHasActiveGrant mirrors EMR's consent check: a non-expired emr_access_grant
// from the patient to this doctor. Raw query to avoid importing the emr package.
func (h *Handler) doctorHasActiveGrant(doctorID, patientID uint) bool {
	var exists bool
	h.db.Raw(`SELECT EXISTS (
		SELECT 1 FROM emr_access_grants
		WHERE doctor_id = ? AND patient_id = ? AND expires_at > ?
	)`, doctorID, patientID, time.Now()).Scan(&exists)
	return exists
}

// PublicUpload accepts a single file from the public careers apply form, before
// the applicant has an account. Strictly validated; the file lands in the vault
// owned by "application" with owner_id=0 until the application links it.
//
// POST /careers/documents   multipart: file=<binary>, category=<string>
func (h *Handler) PublicUpload(c *fiber.Ctx) error {
	category := strings.TrimSpace(c.FormValue("category"))
	rule, ok := categoryRules[category]
	if !ok {
		return response.Error(c, fiber.StatusBadRequest, "BAD_CATEGORY", "Unknown document category")
	}
	// The public form only ever uploads these three.
	if category != "passport_photo" && category != "resume" && category != "license_cert" {
		return response.Error(c, fiber.StatusBadRequest, "BAD_CATEGORY", "This category cannot be uploaded here")
	}

	doc, status, err := h.saveUpload(c, category, rule, OwnerApplication, 0, nil)
	if err != nil {
		return response.Error(c, status, "UPLOAD_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, fiber.Map{
		"id":            doc.ID,
		"access_token":  doc.AccessToken,
		"category":      doc.Category,
		"original_name": doc.OriginalName,
		// Stable handle the apply form stores; it 302s to a fresh signed URL.
		"url": h.publicURL(c, doc.AccessToken),
	}, "File uploaded")
}

// PublicAssetUpload stores a CDN-served public asset (blog cover, product image)
// uploaded by a content author, and returns its stable public URL.
//
// POST /documents/public-asset  multipart: file=<binary>, category=blog_image|product_image
func (h *Handler) PublicAssetUpload(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(uint)
	category := strings.TrimSpace(c.FormValue("category"))
	rule, ok := publicCategoryRules[category]
	if !ok {
		return response.Error(c, fiber.StatusBadRequest, "BAD_CATEGORY", "Not a public asset category")
	}
	file, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "FILE_MISSING", "file is required")
	}
	doc, status, err := h.saveFile(c.UserContext(), file, category, rule, "content", uid, &uid, true)
	if err != nil {
		return response.Error(c, status, "UPLOAD_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, fiber.Map{"id": doc.ID, "url": doc.PublicURL}, "Asset uploaded")
}

// PublicFetch resolves an access token to a fresh signed URL and redirects. Used
// to preview an applicant's uploaded file by its unguessable token.
//
// GET /careers/documents/:token
func (h *Handler) PublicFetch(c *fiber.Ctx) error {
	token := c.Params("token")
	if len(token) < 16 {
		return response.Error(c, fiber.StatusBadRequest, "BAD_TOKEN", "Invalid token")
	}
	// Token access is ONLY for pre-hire applicant files. Once a document is
	// re-owned to a doctor or belongs to a patient, the public token route stops
	// working — those are private and go through the authenticated, authorized
	// route. This prevents a leaked token from exposing a private record.
	var doc Document
	if err := h.db.Where("access_token = ? AND owner_type = ?", token, OwnerApplication).First(&doc).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Document not found")
	}
	url, err := h.store.PresignedGetURL(c.UserContext(), doc.StorageKey, signedURLTTL)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "URL_FAILED", "Could not produce a link")
	}
	return c.Redirect(url, fiber.StatusFound)
}

// List returns documents for an owner (HR / head doctor / admin).
//
// GET /documents?owner_type=doctor&owner_id=123
func (h *Handler) List(c *fiber.Ctx) error {
	ownerType := c.Query("owner_type")
	ownerID, _ := strconv.Atoi(c.Query("owner_id"))
	if ownerType == "" || ownerID <= 0 {
		return response.Error(c, fiber.StatusBadRequest, "BAD_QUERY", "owner_type and owner_id are required")
	}
	if !h.canAccess(c, ownerType, uint(ownerID)) {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You don't have access to these documents")
	}
	var docs []Document
	if err := h.db.Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		Order("category ASC, version DESC").Find(&docs).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, docs, "Documents fetched")
}

// SignedURL returns a short-lived read URL for one document (authorized roles).
//
// GET /documents/:id/url
func (h *Handler) SignedURL(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	var doc Document
	if err := h.db.First(&doc, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Document not found")
	}
	if !h.canAccess(c, doc.OwnerType, doc.OwnerID) {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You don't have access to this document")
	}
	url, err := h.store.PresignedGetURL(c.UserContext(), doc.StorageKey, signedURLTTL)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "URL_FAILED", "Could not produce a link")
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"url": url, "expires_in": int(signedURLTTL.Seconds())}, "Signed URL")
}

// Verify records a head-doctor / HR decision on a credential, optionally with an
// expiry date (license_cert). WithContext so the audit log attributes who did it.
//
// POST /documents/:id/verify  { status: verified|rejected, expiry_date?: ISO }
func (h *Handler) Verify(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	reviewerID, _ := c.Locals("user_id").(uint)
	var req struct {
		Status     string `json:"status"`
		ExpiryDate string `json:"expiry_date"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "BAD_REQUEST", "Invalid request")
	}
	if req.Status != StatusVerified && req.Status != StatusRejected {
		return response.Error(c, fiber.StatusBadRequest, "BAD_STATUS", "status must be verified or rejected")
	}
	var doc Document
	if err := h.db.First(&doc, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Document not found")
	}
	now := time.Now()
	updates := map[string]interface{}{
		"status":      req.Status,
		"verified_by": reviewerID,
		"verified_at": now,
	}
	var expiry *time.Time
	if req.ExpiryDate != "" {
		if t, err := time.Parse("2006-01-02", req.ExpiryDate); err == nil {
			updates["expiry_date"] = t
			expiry = &t
		}
	}
	if err := h.db.WithContext(c.UserContext()).Model(&Document{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}

	// Connect a verified license to the compliance monitor: stamp the doctor's
	// profile license_expiry so the existing lapse/renewal checks fire on it.
	if req.Status == StatusVerified && doc.Category == "license_cert" && doc.OwnerType == OwnerDoctor && doc.OwnerID > 0 && expiry != nil {
		h.db.WithContext(c.UserContext()).Table("doctorprofile").
			Where("user_id = ?", doc.OwnerID).
			Update("license_expiry", *expiry)
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"id": id, "status": req.Status}, "Document reviewed")
}

// ── shared upload core ───────────────────────────────────────────────────────

// saveUpload reads the multipart "file" from the request and stores it (private).
func (h *Handler) saveUpload(c *fiber.Ctx, category string, rule categoryRule, ownerType string, ownerID uint, uploadedBy *uint) (*Document, int, error) {
	file, err := c.FormFile("file")
	if err != nil {
		return nil, fiber.StatusBadRequest, fiberErr("file is required")
	}
	return h.saveFile(c.UserContext(), file, category, rule, ownerType, ownerID, uploadedBy, false)
}

// saveFile is the HTTP-independent storage+record core: validate, write to the
// vault, record a Document row. public=true stores a CDN-served asset and fills
// PublicURL; otherwise the object is private (read via signed URL).
func (h *Handler) saveFile(ctx context.Context, file *multipart.FileHeader, category string, rule categoryRule, ownerType string, ownerID uint, uploadedBy *uint, public bool) (*Document, int, error) {
	if file.Size > rule.maxBytes {
		return nil, fiber.StatusRequestEntityTooLarge, fiberErr("file is larger than the allowed size")
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !rule.exts[ext] {
		return nil, fiber.StatusUnsupportedMediaType, fiberErr("file type not allowed for this category")
	}
	contentType := extMime[ext]
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	token := crypto.SecureToken(24)
	// Key groups objects by owner; pre-hire applicant files live under the token.
	key := ownerType + "/"
	if ownerID > 0 {
		key += strconv.FormatUint(uint64(ownerID), 10)
	} else {
		key += "inbox"
	}
	key += "/" + category + "_" + token + ext

	src, err := file.Open()
	if err != nil {
		return nil, fiber.StatusInternalServerError, fiberErr("could not read upload")
	}
	defer src.Close()

	visibility := VisibilityPrivate
	publicURL := ""
	if public {
		url, err := h.store.PutPublic(ctx, key, contentType, src, file.Size)
		if err != nil {
			return nil, fiber.StatusInternalServerError, fiberErr("could not store file")
		}
		visibility, publicURL = VisibilityPublic, url
	} else if err := h.store.Put(ctx, key, contentType, src, file.Size); err != nil {
		return nil, fiber.StatusInternalServerError, fiberErr("could not store file")
	}

	doc := Document{
		OwnerType:    ownerType,
		OwnerID:      ownerID,
		Category:     category,
		StorageKey:   key,
		OriginalName: filepath.Base(file.Filename),
		ContentType:  contentType,
		SizeBytes:    file.Size,
		Status:       StatusPending,
		Visibility:   visibility,
		PublicURL:    publicURL,
		UploadedBy:   uploadedBy,
		Version:      1,
		AccessToken:  token,
	}
	if err := h.db.WithContext(ctx).Create(&doc).Error; err != nil {
		return nil, fiber.StatusInternalServerError, fiberErr("could not record document")
	}
	return &doc, fiber.StatusOK, nil
}

// PatientUpload lets a logged-in patient upload a document to their own medical
// vault (lab report, scan, prescription, …). Owner is always the caller.
//
// POST /documents/me   multipart: file=<binary>, category=<string>
func (h *Handler) PatientUpload(c *fiber.Ctx) error {
	uid, _ := c.Locals("user_id").(uint)
	if uid == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Login required")
	}
	category := strings.TrimSpace(c.FormValue("category"))
	rule, ok := patientCategoryRules[category]
	if !ok {
		return response.Error(c, fiber.StatusBadRequest, "BAD_CATEGORY", "Unknown document category")
	}
	doc, status, err := h.saveUpload(c, category, rule, OwnerPatient, uid, &uid)
	if err != nil {
		return response.Error(c, status, "UPLOAD_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, fiber.Map{
		"id":            doc.ID,
		"category":      doc.Category,
		"original_name": doc.OriginalName,
		"size_bytes":    doc.SizeBytes,
	}, "Document uploaded")
}

func (h *Handler) publicURL(c *fiber.Ctx, token string) string {
	base := strings.TrimRight(c.BaseURL(), "/")
	return base + "/api/v1/careers/documents/" + token
}

// fiberErr is a tiny error type so saveUpload can return a message.
type strErr string

func (e strErr) Error() string { return string(e) }
func fiberErr(s string) error  { return strErr(s) }

// RegisterRoutes wires the vault endpoints.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	// Public applicant flow (no account yet) — strict validation inside.
	router.Post("/careers/documents", h.PublicUpload)
	router.Get("/careers/documents/:token", h.PublicFetch)

	// Authenticated vault access. Authorization is per-document (canAccess):
	// applicant/doctor records → staff or the doctor; patient records → the
	// patient or a doctor with an active EMR consent grant.
	g := router.Group("/documents", middleware.Protected())
	g.Post("/me", h.PatientUpload) // patient uploads to their own vault
	// Public assets (blog covers, product images) — content authors + admins.
	g.Post("/public-asset", middleware.RequireRoles("doctor", "chief_doctor", "hospital_head_doctor", "manager", "admin", "super_admin", "pharmacy_admin", "pharmacist"), h.PublicAssetUpload)
	g.Get("/", h.List)
	g.Get("/:id/url", h.SignedURL)
	// Verify (credentialing) stays staff-only — it's a doctor/applicant action.
	g.Post("/:id/verify", middleware.RequireRoles("hr_manager", "hr", "chief_doctor", "hospital_head_doctor", "manager", "admin"), h.Verify)
}

// ReownToDoctor re-owns an applicant's documents to the doctor account created
// for them at onboard. Called from the career onboarding flow. Best-effort.
func ReownToDoctor(db *gorm.DB, accessTokens []string, doctorUserID uint) {
	tokens := make([]string, 0, len(accessTokens))
	for _, t := range accessTokens {
		if t = strings.TrimSpace(t); t != "" {
			tokens = append(tokens, t)
		}
	}
	if len(tokens) == 0 {
		return
	}
	db.Model(&Document{}).
		Where("access_token IN ? AND owner_type = ?", tokens, OwnerApplication).
		Updates(map[string]interface{}{"owner_type": OwnerDoctor, "owner_id": doctorUserID})
}
