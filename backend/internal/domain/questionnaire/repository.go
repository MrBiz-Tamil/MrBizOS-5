package questionnaire

import "drayya-core/internal/database"

// ── Templates ─────────────────────────────────────────────────────────────────

func CreateTemplate(t *Template) error {
	return database.DB.Create(t).Error
}

func GetAllTemplates(activeOnly bool) ([]Template, error) {
	var templates []Template
	q := database.DB
	if activeOnly {
		q = q.Where("is_active = true")
	}
	return templates, q.Order("name ASC").Find(&templates).Error
}

func GetTemplateByID(id uint) (*Template, error) {
	var t Template
	err := database.DB.Preload("Questions").First(&t, id).Error
	return &t, err
}

func GetTemplateBySpecialty(specialty string) (*Template, error) {
	var t Template
	err := database.DB.Where("specialty = ? AND is_active = true", specialty).
		Preload("Questions").First(&t).Error
	return &t, err
}

func UpdateTemplate(t *Template) error {
	return database.DB.Save(t).Error
}

// ── Questions ─────────────────────────────────────────────────────────────────

func AddQuestion(q *Question) error {
	return database.DB.Create(q).Error
}

func DeleteQuestion(id uint) error {
	return database.DB.Delete(&Question{}, id).Error
}

// ── Responses ─────────────────────────────────────────────────────────────────

func CreateResponse(r *Response) error {
	return database.DB.Create(r).Error
}

func GetResponseByID(id uint) (*Response, error) {
	var r Response
	err := database.DB.Preload("Answers").First(&r, id).Error
	return &r, err
}

func GetResponsesByPatient(patientUserID uint) ([]Response, error) {
	var responses []Response
	err := database.DB.Where("patient_user_id = ?", patientUserID).
		Order("created_at DESC").Find(&responses).Error
	return responses, err
}

func GetResponseByConsultation(consultationID uint) (*Response, error) {
	var r Response
	err := database.DB.Where("consultation_id = ?", consultationID).
		Preload("Answers").First(&r).Error
	return &r, err
}

func UpdateResponse(r *Response) error {
	return database.DB.Save(r).Error
}

func GetPendingReviews() ([]Response, error) {
	var responses []Response
	err := database.DB.Where("status = 'submitted'").
		Order("created_at ASC").Find(&responses).Error
	return responses, err
}
