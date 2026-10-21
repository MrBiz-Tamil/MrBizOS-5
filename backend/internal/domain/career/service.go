package career

import (
	"fmt"
	"drayya-core/internal/services/email"
	"drayya-core/internal/services/notification"
	"os"
	"strings"
)

type Service interface {
	ListJobs() ([]JobOpening, error)
	Apply(app *JobApplication) error
	MyApplications(userID uint) ([]JobApplication, error)
	ListAllApplications() ([]JobApplication, error)
	UpdateApplicationStatus(id uint, status string) error
	UpdateApplicationStatusWithNotes(id uint, status, interviewDate, notes string) error
	GetApplicationByID(id uint) (*JobApplication, error)
	GetJobByID(id uint) (*JobOpening, error)
	CreateJob(job *JobOpening) error
	UpdateJob(job *JobOpening) error
	GetAllJobs() ([]JobOpening, error)

	ListTemplates() ([]RecruitmentTemplate, error)
	GetTemplateByID(id uint) (*RecruitmentTemplate, error)
	CreateTemplate(t *RecruitmentTemplate) error
	UpdateTemplate(t *RecruitmentTemplate) error
	DeleteTemplate(id uint) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) ListJobs() ([]JobOpening, error) {
	return s.repo.GetJobs()
}

func (s *service) Apply(app *JobApplication) error {
	// Validate required fields
	if app.FullName == "" || app.Email == "" {
		return fmt.Errorf("full_name and email are required")
	}

	// Prevent duplicate applications
	if app.JobID != nil {
		// Specific job: check same job+email
		if dup, _ := s.repo.GetApplicationByJobAndEmail(*app.JobID, app.Email); dup != nil {
			return fmt.Errorf("you have already applied for this position")
		}
	} else {
		// General application (no specific job): check any recent application by same email
		if dup, _ := s.repo.GetApplicationByEmail(app.Email); dup != nil {
			return fmt.Errorf("we already have an application on file for %s — our team will be in touch soon", app.Email)
		}
	}

	if err := s.repo.CreateApplication(app); err != nil {
		return err
	}

	// Send confirmation email to applicant (non-blocking)
	go func() {
		emailSubject := "Application Received — Dr.Ayya / Dr.Ayya"
		emailBody := email.ApplicationReceivedEmail(email.JobApplicationEmailData{
			FullName:        app.FullName,
			Position:        app.Position,
			ExperienceYears: app.ExperienceYears,
			LicenseNumber:   app.LicenseNumber,
			WhatsappNumber:  app.WhatsappNumber,
		})
		_ = email.SendSimple(app.Email, emailSubject, emailBody)
	}()

	// Send confirmation WhatsApp to applicant (non-blocking)
	whatsappPhone := app.WhatsappNumber
	if whatsappPhone == "" {
		whatsappPhone = app.PhoneNumber
	}
	if whatsappPhone != "" {
		go func() {
			msg := fmt.Sprintf("Dear %s, thank you for applying for the %s position at Dr.Ayya. Your application has been received and is under review.", app.FullName, app.Position)
			if tmpl, err := s.repo.GetTemplateByTypeAndTitle("WhatsApp", "Application Received Confirmation"); err == nil && tmpl != nil {
				msg = replaceVariables(tmpl.Content, map[string]string{
					"name":     app.FullName,
					"position": app.Position,
				})
			}
			_ = notification.SendText(whatsappPhone, msg)
		}()
	}

	// Notify HR inbox (non-blocking)
	hrEmail := os.Getenv("HR_EMAIL")
	if hrEmail == "" {
		hrEmail = os.Getenv("SMTP_FROM")
	}
	if hrEmail != "" {
		go func() {
			emailBody := email.HRApplicationNotificationEmail(email.JobApplicationEmailData{
				FullName:          app.FullName,
				Email:             app.Email,
				PhoneNumber:       app.PhoneNumber,
				WhatsappNumber:    app.WhatsappNumber,
				Position:          app.Position,
				ExperienceYears:   app.ExperienceYears,
				LicenseNumber:     app.LicenseNumber,
				DateOfBirth:       app.DateOfBirth,
				Gender:            app.Gender,
				ResumeURL:         app.ResumeURL,
				TreatmentApproach: app.TreatmentApproach,
				EducationDetails:  app.EducationDetails,
			})
			_ = email.SendSimple(
				hrEmail,
				fmt.Sprintf("New Application: %s — %s", app.Position, app.FullName),
				emailBody,
			)
		}()
	}

	return nil
}

func (s *service) MyApplications(userID uint) ([]JobApplication, error) {
	return s.repo.GetApplicationsByUserID(userID)
}

func (s *service) ListAllApplications() ([]JobApplication, error) {
	return s.repo.GetAllApplications()
}

func (s *service) UpdateApplicationStatusWithNotes(id uint, status, interviewDate, notes string) error {
	if err := s.repo.UpdateApplicationStatusWithNotes(id, status, interviewDate, notes); err != nil {
		return err
	}
	go func() {
		app, err := s.repo.GetApplicationByID(id)
		if err != nil || app.Email == "" {
			return
		}

		emailSubject := "Your Application Status Has Been Updated — Dr.Ayya"
		var statusLine, statusColor, friendlyMsg string
		switch status {
		case "reviewing":
			statusLine = "Under Review"
			statusColor = "#d97706"
			friendlyMsg = "Your profile is being reviewed by our recruitment team."
		case "interviewing":
			statusLine = "Interview Stage"
			statusColor = "#7c3aed"
			friendlyMsg = "You've been shortlisted for an interview! Our team will contact you shortly with the details."
			if interviewDate != "" {
				friendlyMsg = fmt.Sprintf("You've been shortlisted for an interview scheduled on %s. Notes: %s", interviewDate, notes)
			}
		case "offered":
			statusLine = "Offer Extended"
			statusColor = "#059669"
			friendlyMsg = "Congratulations! You've received an offer from Dr.Ayya. Our HR team will reach out with the next steps."
		case "rejected":
			statusLine = "Application Closed"
			statusColor = "#dc2626"
			friendlyMsg = "Thank you for applying to Dr.Ayya. We've decided to move forward with other candidates at this time; we will keep your profile on file for future openings."
		default:
			statusLine = status
			statusColor = "#374151"
			friendlyMsg = "Your application status has been updated."
		}

		emailBody := email.ApplicationStatusUpdateEmail(email.JobApplicationEmailData{
			FullName: app.FullName,
			Position: app.Position,
		}, statusLine, statusColor, friendlyMsg)
		_ = email.SendSimple(app.Email, emailSubject, emailBody)

		whatsappPhone := app.WhatsappNumber
		if whatsappPhone == "" {
			whatsappPhone = app.PhoneNumber
		}
		if whatsappPhone != "" {
			msg := fmt.Sprintf("Application Update — Dr.Ayya\nDear %s,\n\n%s", app.FullName, friendlyMsg)
			if tmpl, err := s.repo.GetTemplateByTypeAndTitle("WhatsApp", "Application Status Update"); err == nil && tmpl != nil {
				msg = replaceVariables(tmpl.Content, map[string]string{
					"name":    app.FullName,
					"message": friendlyMsg,
				})
			}
			_ = notification.SendText(whatsappPhone, msg)
		}
	}()
	return nil
}

func (s *service) UpdateApplicationStatus(id uint, status string) error {
	if err := s.repo.UpdateApplicationStatus(id, status); err != nil {
		return err
	}
	// Non-blocking status notification email to applicant
	go func() {
		app, err := s.repo.GetApplicationByID(id)
		if err != nil || app.Email == "" {
			return
		}

		emailSubject := "Your Application Status Has Been Updated — Dr.Ayya / Dr.Ayya"
		var statusLine, statusColor, friendlyMsg string
		switch status {
		case "reviewing":
			statusLine = "Under Review"
			statusColor = "#d97706"
			friendlyMsg = "Your profile is being reviewed by our recruitment team."
		case "interviewing":
			statusLine = "Interview Stage"
			statusColor = "#7c3aed"
			friendlyMsg = "You've been shortlisted for an interview! Our team will contact you shortly with the details."
		case "offered":
			statusLine = "Offer Extended"
			statusColor = "#059669"
			friendlyMsg = "Congratulations! You've received an offer from Dr.Ayya. Our HR team will reach out with the next steps."
		case "rejected":
			statusLine = "Application Closed"
			statusColor = "#dc2626"
			friendlyMsg = "Thank you for applying to Dr.Ayya. We've decided to move forward with other candidates at this time; we will keep your profile on file for future openings."
		default:
			statusLine = status
			statusColor = "#374151"
			friendlyMsg = "Your application status has been updated."
		}

		emailBody := email.ApplicationStatusUpdateEmail(email.JobApplicationEmailData{
			FullName: app.FullName,
			Position: app.Position,
		}, statusLine, statusColor, friendlyMsg)
		_ = email.SendSimple(app.Email, emailSubject, emailBody)

		whatsappPhone := app.WhatsappNumber
		if whatsappPhone == "" {
			whatsappPhone = app.PhoneNumber
		}
		if whatsappPhone != "" {
			msg := fmt.Sprintf("Application Update — Dr.Ayya\nDear %s,\n\n%s", app.FullName, friendlyMsg)
			if tmpl, err := s.repo.GetTemplateByTypeAndTitle("WhatsApp", "Application Status Update"); err == nil && tmpl != nil {
				msg = replaceVariables(tmpl.Content, map[string]string{
					"name":    app.FullName,
					"message": friendlyMsg,
				})
			}
			_ = notification.SendText(whatsappPhone, msg)
		}
	}()
	return nil
}

func replaceVariables(content string, vars map[string]string) string {
	for k, v := range vars {
		content = strings.ReplaceAll(content, "{{"+k+"}}", v)
	}
	return content
}

func (s *service) GetApplicationByID(id uint) (*JobApplication, error) {
	return s.repo.GetApplicationByID(id)
}

func (s *service) GetJobByID(id uint) (*JobOpening, error) {
	return s.repo.GetJobByID(id)
}

func (s *service) CreateJob(job *JobOpening) error {
	if job.Title == "" {
		return fmt.Errorf("title is required")
	}
	return s.repo.CreateJob(job)
}

func (s *service) UpdateJob(job *JobOpening) error {
	return s.repo.UpdateJob(job)
}

func (s *service) GetAllJobs() ([]JobOpening, error) {
	return s.repo.GetAllJobs()
}

func (s *service) ListTemplates() ([]RecruitmentTemplate, error) {
	return s.repo.GetTemplates()
}

func (s *service) GetTemplateByID(id uint) (*RecruitmentTemplate, error) {
	return s.repo.GetTemplateByID(id)
}

func (s *service) CreateTemplate(t *RecruitmentTemplate) error {
	if t.Title == "" || t.Content == "" {
		return fmt.Errorf("title and content are required")
	}
	return s.repo.CreateTemplate(t)
}

func (s *service) UpdateTemplate(t *RecruitmentTemplate) error {
	if t.Title == "" || t.Content == "" {
		return fmt.Errorf("title and content are required")
	}
	return s.repo.UpdateTemplate(t)
}

func (s *service) DeleteTemplate(id uint) error {
	return s.repo.DeleteTemplate(id)
}


