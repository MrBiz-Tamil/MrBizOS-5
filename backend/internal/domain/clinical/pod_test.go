package clinical

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"drayya-core/internal/domain/identity"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Lightweight test stand-ins for the identity tables SeedAYUSHPods reads/writes,
// so the test stays isolated from the heavy identity models.
type testDoctorType struct {
	ID   uint `gorm:"primaryKey"`
	Name string
	Slug string
}

func (testDoctorType) TableName() string { return "doctor_types" }

type testDoctorProfile struct {
	ID           uint `gorm:"primaryKey"`
	UserID       uint
	DoctorTypeID *uint
	PodID        *uint
	IsVerified   bool
}

func (testDoctorProfile) TableName() string { return "doctorprofile" }

func podTestDB(t *testing.T) *gorm.DB {
	name := strings.ReplaceAll(t.Name(), "/", "_")
	dsn := "file:" + name + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Pod{}, &testDoctorType{}, &testDoctorProfile{}, &identity.User{}, &DoctorOnboarding{}))
	return db
}

func uptr(v uint) *uint { return &v }

func TestSeedAYUSHPods_CreatesOnePodPerType(t *testing.T) {
	db := podTestDB(t)
	require.NoError(t, db.Create(&[]testDoctorType{
		{ID: 1, Name: "Siddha", Slug: "siddha"},
		{ID: 2, Name: "Ayurveda", Slug: "ayurveda"},
		{ID: 3, Name: "Yoga & Naturopathy", Slug: "yoga_naturopathy"},
		{ID: 4, Name: "Psychology", Slug: "psychology"},
	}).Error)

	SeedAYUSHPods(db)

	var count int64
	db.Model(&Pod{}).Count(&count)
	assert.Equal(t, int64(4), count, "one pod per AYUSH type")

	// Each pod links back to its type.
	var siddhaPod Pod
	require.NoError(t, db.Where("doctor_type_id = ?", 1).First(&siddhaPod).Error)
	assert.Equal(t, "siddha", siddhaPod.Specialty)
	assert.True(t, siddhaPod.IsActive)
}

func TestSeedAYUSHPods_Idempotent(t *testing.T) {
	db := podTestDB(t)
	require.NoError(t, db.Create(&testDoctorType{ID: 1, Name: "Siddha", Slug: "siddha"}).Error)

	SeedAYUSHPods(db)
	SeedAYUSHPods(db) // second run must not duplicate

	var count int64
	db.Model(&Pod{}).Count(&count)
	assert.Equal(t, int64(1), count, "seeding twice must not create duplicate pods")
}

func TestSeedAYUSHPods_BackfillsExistingDoctors(t *testing.T) {
	db := podTestDB(t)
	require.NoError(t, db.Create(&testDoctorType{ID: 1, Name: "Siddha", Slug: "siddha"}).Error)
	// A doctor already typed Siddha but with no pod yet.
	require.NoError(t, db.Create(&testDoctorProfile{ID: 10, UserID: 100, DoctorTypeID: uptr(1)}).Error)
	// A doctor already assigned to some other pod — must NOT be moved.
	require.NoError(t, db.Create(&testDoctorProfile{ID: 11, UserID: 101, DoctorTypeID: uptr(1), PodID: uptr(99)}).Error)

	SeedAYUSHPods(db)

	var siddhaPod Pod
	require.NoError(t, db.Where("doctor_type_id = ?", 1).First(&siddhaPod).Error)

	var unassigned testDoctorProfile
	require.NoError(t, db.First(&unassigned, 10).Error)
	require.NotNil(t, unassigned.PodID)
	assert.Equal(t, siddhaPod.ID, *unassigned.PodID, "unassigned Siddha doctor routed into the Siddha pod")

	var alreadyAssigned testDoctorProfile
	require.NoError(t, db.First(&alreadyAssigned, 11).Error)
	require.NotNil(t, alreadyAssigned.PodID)
	assert.Equal(t, uint(99), *alreadyAssigned.PodID, "doctor already in a pod must not be moved")
}

func TestAssignPodLead_Scenarios(t *testing.T) {
	t.Run("promote doctor to chief_doctor", func(t *testing.T) {
		db := podTestDB(t)
		h := NewHandler(db)

		pod := Pod{ID: 1, Specialty: "siddha", IsActive: true}
		require.NoError(t, db.Create(&pod).Error)

		doc := identity.User{ID: 100, Role: "doctor", Email: "doc@drayya.com"}
		require.NoError(t, db.Create(&doc).Error)

		app := fiber.New()
		app.Post("/clinical/pods/:id/lead", h.AssignPodLead)

		reqBody, _ := json.Marshal(map[string]interface{}{"lead_doctor_id": 100})
		req := httptest.NewRequest("POST", "/clinical/pods/1/lead", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var updatedUser identity.User
		require.NoError(t, db.First(&updatedUser, 100).Error)
		assert.Equal(t, "chief_doctor", updatedUser.Role)
	})

	t.Run("keep chief_doctor role as is", func(t *testing.T) {
		db := podTestDB(t)
		h := NewHandler(db)

		pod := Pod{ID: 1, Specialty: "siddha", IsActive: true}
		require.NoError(t, db.Create(&pod).Error)

		doc := identity.User{ID: 101, Role: "chief_doctor", Email: "chief@drayya.com"}
		require.NoError(t, db.Create(&doc).Error)

		app := fiber.New()
		app.Post("/clinical/pods/:id/lead", h.AssignPodLead)

		reqBody, _ := json.Marshal(map[string]interface{}{"lead_doctor_id": 101})
		req := httptest.NewRequest("POST", "/clinical/pods/1/lead", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var updatedUser identity.User
		require.NoError(t, db.First(&updatedUser, 101).Error)
		assert.Equal(t, "chief_doctor", updatedUser.Role)
	})

	t.Run("keep hospital_head_doctor role as is", func(t *testing.T) {
		db := podTestDB(t)
		h := NewHandler(db)

		pod := Pod{ID: 1, Specialty: "siddha", IsActive: true}
		require.NoError(t, db.Create(&pod).Error)

		doc := identity.User{ID: 102, Role: "hospital_head_doctor", Email: "head@drayya.com"}
		require.NoError(t, db.Create(&doc).Error)

		app := fiber.New()
		app.Post("/clinical/pods/:id/lead", h.AssignPodLead)

		reqBody, _ := json.Marshal(map[string]interface{}{"lead_doctor_id": 102})
		req := httptest.NewRequest("POST", "/clinical/pods/1/lead", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var updatedUser identity.User
		require.NoError(t, db.First(&updatedUser, 102).Error)
		assert.Equal(t, "hospital_head_doctor", updatedUser.Role)
	})

	t.Run("reject non-doctor roles", func(t *testing.T) {
		db := podTestDB(t)
		h := NewHandler(db)

		pod := Pod{ID: 1, Specialty: "siddha", IsActive: true}
		require.NoError(t, db.Create(&pod).Error)

		patient := identity.User{ID: 103, Role: "patient", Email: "patient@drayya.com"}
		require.NoError(t, db.Create(&patient).Error)

		app := fiber.New()
		app.Post("/clinical/pods/:id/lead", h.AssignPodLead)

		reqBody, _ := json.Marshal(map[string]interface{}{"lead_doctor_id": 103})
		req := httptest.NewRequest("POST", "/clinical/pods/1/lead", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
	})
}

func TestOnboardingCandidates_Scenarios(t *testing.T) {
	t.Run("list candidate with role doctor", func(t *testing.T) {
		db := podTestDB(t)
		h := NewHandler(db)

		// Create user and profile
		doc := identity.User{ID: 200, Role: "doctor", Email: "doc200@drayya.com"}
		require.NoError(t, db.Create(&doc).Error)
		profile := testDoctorProfile{UserID: 200}
		require.NoError(t, db.Create(&profile).Error)

		app := fiber.New()
		app.Get("/clinical/onboarding/candidates", h.OnboardingCandidates)

		req := httptest.NewRequest("GET", "/clinical/onboarding/candidates", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		data, ok := body["data"].([]interface{})
		require.True(t, ok)
		assert.Len(t, data, 1)
		candidate := data[0].(map[string]interface{})
		assert.Equal(t, float64(200), candidate["id"])
	})

	t.Run("list candidate with role chief_doctor", func(t *testing.T) {
		db := podTestDB(t)
		h := NewHandler(db)

		doc := identity.User{ID: 201, Role: "chief_doctor", Email: "chief201@drayya.com"}
		require.NoError(t, db.Create(&doc).Error)
		profile := testDoctorProfile{UserID: 201}
		require.NoError(t, db.Create(&profile).Error)

		app := fiber.New()
		app.Get("/clinical/onboarding/candidates", h.OnboardingCandidates)

		req := httptest.NewRequest("GET", "/clinical/onboarding/candidates", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		data, ok := body["data"].([]interface{})
		require.True(t, ok)
		assert.Len(t, data, 1)
		candidate := data[0].(map[string]interface{})
		assert.Equal(t, float64(201), candidate["id"])
	})

	t.Run("scope candidates list to head doctor's pod", func(t *testing.T) {
		db := podTestDB(t)
		h := NewHandler(db)

		// Lead pod
		pod := Pod{ID: 10, Specialty: "siddha", IsActive: true, LeadDoctorID: uptr(5)}
		require.NoError(t, db.Create(&pod).Error)

		// Doctor 1: in led pod
		doc1 := identity.User{ID: 301, Role: "doctor", Email: "doc301@drayya.com"}
		require.NoError(t, db.Create(&doc1).Error)
		profile1 := testDoctorProfile{UserID: 301, PodID: uptr(10)}
		require.NoError(t, db.Create(&profile1).Error)

		// Doctor 2: in another pod (pod 11)
		doc2 := identity.User{ID: 302, Role: "doctor", Email: "doc302@drayya.com"}
		require.NoError(t, db.Create(&doc2).Error)
		profile2 := testDoctorProfile{UserID: 302, PodID: uptr(11)}
		require.NoError(t, db.Create(&profile2).Error)

		app := fiber.New()
		app.Get("/clinical/onboarding/candidates", func(c *fiber.Ctx) error {
			c.Locals("role", "chief_doctor")
			c.Locals("user_id", uint(5))
			return h.OnboardingCandidates(c)
		})

		req := httptest.NewRequest("GET", "/clinical/onboarding/candidates", nil)
		resp, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		data, ok := body["data"].([]interface{})
		require.True(t, ok)
		assert.Len(t, data, 1, "should only see the candidate in their led pod")
		candidate := data[0].(map[string]interface{})
		assert.Equal(t, float64(301), candidate["id"])
	})
}
