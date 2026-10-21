package patient

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/communication"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/models"
	"drayya-core/internal/services/visibility"
	"drayya-core/internal/shipping"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)


type PatientDashboardSummary struct {
	User          identity.User                `json:"user"`
	Profile       identity.PatientProfile      `json:"profile"`
	Appointments  []appointment.Appointment    `json:"appointments"`
	OnlineDoctors []visibility.DoctorView      `json:"online_doctors"`
	FreeDoctors   []visibility.DoctorView      `json:"free_doctors"`
	Notifications []communication.Notification `json:"notifications"`
	Vitals        []models.PatientVitals       `json:"vitals"`
	Deliveries    interface{}                  `json:"deliveries"`
}

// GetDashboardSummary fetches all metrics needed by the patient landing page in one parallel request
func GetDashboardSummary(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	ctx := c.Context()
	db := database.DB.WithContext(ctx)

	var summary PatientDashboardSummary
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []string

	addErr := func(msg string, err error) {
		mu.Lock()
		defer mu.Unlock()
		errs = append(errs, fmt.Sprintf("%s: %v", msg, err))
	}

	// 1. Fetch User & Profile (Synchronous prerequisite for other filters if needed, but we can query them concurrently)
	wg.Add(2)
	go func() {
		defer wg.Done()
		var u identity.User
		if err := db.First(&u, userID).Error; err != nil {
			addErr("Failed to fetch user", err)
		} else {
			u.PasswordHash = ""
			mu.Lock()
			summary.User = u
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		var prof identity.PatientProfile
		if err := db.Where("user_id = ?", userID).First(&prof).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			addErr("Failed to fetch patient profile", err)
		} else {
			mu.Lock()
			summary.Profile = prof
			mu.Unlock()
		}
	}()

	// 2. Appointments (Next 3 upcoming visits)
	wg.Add(1)
	go func() {
		defer wg.Done()
		var appts []appointment.Appointment
		now := time.Now()
		err := db.Where("patient_id = ? AND status NOT IN ? AND start_time >= ?",
			userID, []string{"cancelled", "completed", "no_show", "declined"}, now).
			Order("start_time ASC").
			Limit(3).
			Preload("Doctor").
			Find(&appts).Error
		if err != nil {
			addErr("Failed to fetch appointments", err)
		} else {
			if appts == nil {
				appts = []appointment.Appointment{}
			}
			mu.Lock()
			summary.Appointments = appts
			mu.Unlock()
		}
	}()

	// 3. Online Doctors (top 5)
	wg.Add(1)
	go func() {
		defer wg.Done()
		docs, err := visibility.GetOnlineDoctors()
		if err != nil {
			addErr("Failed to fetch online doctors", err)
		} else {
			if len(docs) > 5 {
				docs = docs[:5]
			}
			if docs == nil {
				docs = []visibility.DoctorView{}
			}
			mu.Lock()
			summary.OnlineDoctors = docs
			mu.Unlock()
		}
	}()

	// 4. Free Doctors (top 5)
	wg.Add(1)
	go func() {
		defer wg.Done()
		docs, err := visibility.GetFreeDoctors()
		if err != nil {
			addErr("Failed to fetch free doctors", err)
		} else {
			if len(docs) > 5 {
				docs = docs[:5]
			}
			if docs == nil {
				docs = []visibility.DoctorView{}
			}
			mu.Lock()
			summary.FreeDoctors = docs
			mu.Unlock()
		}
	}()

	// 5. Notifications (last 50)
	wg.Add(1)
	go func() {
		defer wg.Done()
		var notifs []communication.Notification
		err := db.Where("user_id = ?", userID).
			Order("created_at DESC").
			Limit(50).
			Find(&notifs).Error
		if err != nil {
			addErr("Failed to fetch notifications", err)
		} else {
			if notifs == nil {
				notifs = []communication.Notification{}
			}
			mu.Lock()
			summary.Notifications = notifs
			mu.Unlock()
		}
	}()

	// 6. Vitals (last 10 history entries)
	wg.Add(1)
	go func() {
		defer wg.Done()
		var vitals []models.PatientVitals
		err := db.Where("patient_id = ?", userID).
			Order("recorded_at DESC").
			Limit(10).
			Find(&vitals).Error
		if err != nil {
			addErr("Failed to fetch vitals", err)
		} else {
			if vitals == nil {
				vitals = []models.PatientVitals{}
			}
			mu.Lock()
			summary.Vitals = vitals
			mu.Unlock()
		}
	}()

	// 7. Active courier tracking deliveries
	wg.Add(1)
	go func() {
		defer wg.Done()
		if shipping.ShippingService != nil {
			deliveries, err := shipping.ShippingService.GetPatientDeliveries(userID)
			if err != nil {
				addErr("Failed to fetch deliveries", err)
			} else {
				mu.Lock()
				if deliveries == nil {
					summary.Deliveries = []interface{}{}
				} else {
					summary.Deliveries = deliveries
				}
				mu.Unlock()
			}
		} else {
			mu.Lock()
			summary.Deliveries = []interface{}{}
			mu.Unlock()
		}
	}()

	wg.Wait()

	if len(errs) > 0 {
		logErrors(errs)
	}

	return response.JSON(c, fiber.StatusOK, summary, "Dashboard summary loaded successfully")
}

func logErrors(errs []string) {
	for _, e := range errs {
		println("[GetDashboardSummary error] " + e)
	}
}
