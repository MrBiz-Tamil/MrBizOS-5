package hr

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"drayya-core/internal/services/email"
)

type OnboardingHandler struct {
	onboardingService OnboardingService
	staffRepo         Repository
}

func NewOnboardingHandler(os OnboardingService, staffRepo Repository) *OnboardingHandler {
	return &OnboardingHandler{
		onboardingService: os,
		staffRepo:         staffRepo,
	}
}

func (h *OnboardingHandler) GetAll(c *fiber.Ctx) error {
	records, err := h.onboardingService.GetAllOnboarding()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch onboarding records"})
	}
	return c.Status(fiber.StatusOK).JSON(records)
}

func (h *OnboardingHandler) GetByID(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	record, err := h.onboardingService.GetOnboarding(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Onboarding record not found"})
	}
	return c.Status(fiber.StatusOK).JSON(record)
}

func (h *OnboardingHandler) UpdateDocument(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))

	var req struct {
		DocType string `json:"doc_type"`
		Status  string `json:"status"` // pending, submitted, verified
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.onboardingService.UpdateDocumentStatus(uint(id), req.DocType, req.Status); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update document"})
	}

	// Trigger email if marked as verified
	if req.Status == "verified" {
		record, err := h.onboardingService.GetOnboarding(uint(id))
		if err == nil && record != nil {
			profile, errProfile := h.staffRepo.GetStaffProfile(record.StaffID)
			if errProfile == nil && profile != nil {
				emailAddr, fullName, errEmail := h.staffRepo.GetUserEmailByID(profile.UserID)
				if errEmail == nil && emailAddr != "" {
					go func() {
						docName := req.DocType
						switch req.DocType {
						case "certificate":
							docName = "Education Certificate"
						case "aadhar_id":
							docName = "Aadhar ID"
						case "pan_card":
							docName = "PAN Card"
						case "bank_account":
							docName = "Bank Account"
						case "passport_photo":
							docName = "Passport Photo"
						}
						subject := fmt.Sprintf("Document Verified: %s", docName)
						content := email.DocumentVerifiedEmail(
							fullName,
							docName,
							"HR Department",
							time.Now().Format("02 Jan 2006"),
						)
						_ = email.SendSimple(emailAddr, subject, content)
					}()
				}
			}
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Document updated"})
}

func (h *OnboardingHandler) UpdateTask(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))

	var req struct {
		TaskType   string `json:"task_type"`
		IsComplete bool   `json:"is_complete"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.onboardingService.UpdateTaskStatus(uint(id), req.TaskType, req.IsComplete); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update task"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Task updated"})
}

func (h *OnboardingHandler) UpdateStatus(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))

	var req struct {
		Status string `json:"status"` // pending_docs, in_training, completed
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	record, err := h.onboardingService.GetOnboarding(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Record not found"})
	}

	oldStatus := record.Status

	if err := h.onboardingService.UpdateOnboardingStatus(uint(id), req.Status); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update status"})
	}

	// Trigger emails on status changes
	if oldStatus != "in_training" && req.Status == "in_training" {
		profile, err := h.staffRepo.GetStaffProfile(record.StaffID)
		if err == nil && profile != nil {
			// IT provisioning email
			go func() {
				subject := "New Employee Onboarding: IT Provisioning Required"
				content := email.ITProvisioningEmail(
					profile.EmployeeID,
					profile.Department,
					profile.Designation,
					"Automated system request for initial provisioning.",
				)
				_ = email.SendSimple("it-support@mrbiz.in", subject, content)
			}()

			// Training started email to employee
			emailAddr, fullName, errEmail := h.staffRepo.GetUserEmailByID(profile.UserID)
			if errEmail == nil && emailAddr != "" {
				go func() {
					subject := "Your Orientation & Training Begins! 🎓"
					content := email.TrainingStartedEmail(
						fullName,
						profile.EmployeeID,
						profile.Department,
						profile.Designation,
						time.Now().Format("02 Jan 2006"),
					)
					_ = email.SendSimple(emailAddr, subject, content)
				}()
			}
		}
	}

	if oldStatus != "completed" && req.Status == "completed" {
		profile, err := h.staffRepo.GetStaffProfile(record.StaffID)
		if err == nil && profile != nil {
			emailAddr, fullName, errEmail := h.staffRepo.GetUserEmailByID(profile.UserID)
			if errEmail == nil && emailAddr != "" {
				// Onboarding complete email to employee
				go func() {
					subject := "Onboarding Complete — Welcome to the Team!"
					content := email.OnboardingCompletedEmail(
						fullName,
						profile.EmployeeID,
						profile.Department,
						profile.Designation,
						profile.JoiningDate.Format("Mon, 02 Jan 2006"),
					)
					_ = email.SendSimple(emailAddr, subject, content)
				}()
			}

			// Notify HR manager or department head
			var managerEmail string
			if profile.ManagerID != nil {
				mgrProfile, errMgr := h.staffRepo.GetStaffProfile(*profile.ManagerID)
				if errMgr == nil && mgrProfile != nil {
					managerEmail, _, _ = h.staffRepo.GetUserEmailByID(mgrProfile.UserID)
				}
			}
			if managerEmail == "" {
				managerEmail = "hr@mrbiz.in"
			}

			go func() {
				subject := fmt.Sprintf("Onboarding Completed - %s (%s)", fullName, profile.EmployeeID)
				body := fmt.Sprintf("<h2>Onboarding Complete</h2><p>The onboarding process for <strong>%s</strong> (%s) has been marked as completed. They are now an active employee in the <strong>%s</strong> department.</p>", fullName, profile.EmployeeID, profile.Department)
				content := email.WrapInLayout(subject, "", body)
				_ = email.SendSimple(managerEmail, subject, content)
			}()
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Status updated successfully"})
}

func (h *OnboardingHandler) InitiateOnboarding(c *fiber.Ctx) error {
	var req struct {
		StaffID uint `json:"staff_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	record, err := h.onboardingService.InitiateOnboarding(req.StaffID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to initiate onboarding"})
	}

	// Trigger welcome email
	profile, err := h.staffRepo.GetStaffProfile(record.StaffID)
	if err == nil && profile != nil {
		emailAddr, fullName, errEmail := h.staffRepo.GetUserEmailByID(profile.UserID)
		if errEmail == nil && emailAddr != "" {
			go func() {
				subject := "Welcome to Dr.Ayya!"
				content := email.OnboardingWelcomeEmail(
					fullName,
					profile.EmployeeID,
					profile.Department,
					profile.Designation,
					profile.JoiningDate.Format("Mon, 02 Jan 2006"),
				)
				_ = email.SendSimple(emailAddr, subject, content)
			}()
		}
	}

	return c.Status(fiber.StatusOK).JSON(record)
}

func (h *OnboardingHandler) SendDocumentReminder(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))

	record, err := h.onboardingService.GetOnboarding(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Record not found"})
	}

	profile, err := h.staffRepo.GetStaffProfile(record.StaffID)
	if err != nil || profile == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch staff details"})
	}

	emailAddr, fullName, errEmail := h.staffRepo.GetUserEmailByID(profile.UserID)
	if errEmail != nil || emailAddr == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch employee email address"})
	}

	var pending []string
	if record.Certificate != "verified" && record.Certificate != "submitted" {
		pending = append(pending, "Education Certificate")
	}
	if record.AadharID != "verified" && record.AadharID != "submitted" {
		pending = append(pending, "Aadhar ID")
	}
	if record.PanCard != "verified" && record.PanCard != "submitted" {
		pending = append(pending, "PAN Card")
	}
	if record.BankAccount != "verified" && record.BankAccount != "submitted" {
		pending = append(pending, "Bank Account Details")
	}
	if record.PassportPhoto != "verified" && record.PassportPhoto != "submitted" {
		pending = append(pending, "Passport Size Photo")
	}

	if len(pending) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "All documents are already submitted or verified"})
	}

	go func() {
		subject := "Document Submission Reminder - Dr.Ayya"
		deadline := profile.JoiningDate.Format("Mon, 02 Jan 2006")
		content := email.DocumentReminderEmail(
			fullName,
			profile.EmployeeID,
			pending,
			deadline,
		)
		_ = email.SendSimple(emailAddr, subject, content)
	}()

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Document reminder email sent successfully"})
}

func (h *OnboardingHandler) SendITRequest(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))

	var req struct {
		Message string `json:"message"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	record, err := h.onboardingService.GetOnboarding(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Record not found"})
	}

	profile, err := h.staffRepo.GetStaffProfile(record.StaffID)
	if err != nil || profile == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not fetch staff details"})
	}

	// Send manual IT email in background
	go func() {
		subject := fmt.Sprintf("IT Request - Onboarding [%s]", profile.EmployeeID)
		content := email.ITProvisioningEmail(
			profile.EmployeeID,
			profile.Department,
			profile.Designation,
			req.Message,
		)
		_ = email.SendSimple("it-support@mrbiz.in", subject, content)
	}()

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "IT Request sent"})
}

