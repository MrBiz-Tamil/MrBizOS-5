package identity

// DoctorSelfUpdateRequest contains only the fields a doctor may update on their own profile.
// The service layer uses this struct — no map[string]interface{} escapes to the DB.
type DoctorSelfUpdateRequest struct {
	Specialization    *string  `json:"specialization"`
	Qualification     *string  `json:"qualification"`
	YearsOfExperience *int     `json:"years_of_experience"`
	ConsultationFee   *int64   `json:"consultation_fee"`
	Bio               *string  `json:"bio"`
	LanguagesSpoken   *string  `json:"languages_spoken"`
	Latitude          *float64 `json:"latitude"`
	Longitude         *float64 `json:"longitude"`
	// Credentials — feed the compliance monitor (Phase 3). LicenseExpiry is an
	// ISO date string (YYYY-MM-DD).
	LicenseNumber     *string  `json:"license_number,omitempty"`
	LicenseExpiry     *string  `json:"license_expiry,omitempty"`
	// Payout fields — separate from clinical profile but doctor-owned
	PayoutUPI         *string `json:"payout_upi,omitempty"`
	BankAccountNumber *string  `json:"bank_account_number,omitempty"`
	BankIFSC          *string  `json:"bank_ifsc,omitempty"`
	DoctorTypeID      *uint    `json:"doctor_type_id"`
}

// DoctorAdminUpdateRequest contains fields only admin/super_admin may change.
type DoctorAdminUpdateRequest struct {
	IsVerified             *bool    `json:"is_verified"`
	AdminBoostScore        *float64 `json:"admin_boost_score"`
	OffersFreeConsultation *bool    `json:"offers_free_consultation"`
	DailyFreeConsultLimit  *int     `json:"daily_free_consult_limit"`
	FreeType               *string  `json:"free_type"`
	DoctorTypeID           *uint    `json:"doctor_type_id"`
	// Credentials — HR/admin may record or correct these on a doctor's behalf for
	// the compliance monitor. LicenseExpiry is an ISO date (YYYY-MM-DD); ""=clear.
	LicenseNumber          *string  `json:"license_number,omitempty"`
	LicenseExpiry          *string  `json:"license_expiry,omitempty"`
	PANVerified            *bool    `json:"pan_verified,omitempty"`
	// EmergencyCapable adds/removes the doctor from the SOS responder pool.
	EmergencyCapable       *bool    `json:"emergency_capable,omitempty"`
}

// DoctorAvailabilitySlot is the validated slot type used throughout the availability flow.
type DoctorAvailabilitySlot struct {
	DayOfWeek   int    `json:"day_of_week"`  // 0 = Sunday … 6 = Saturday
	StartTime   string `json:"start_time"`   // HH:MM (24-hour)
	EndTime     string `json:"end_time"`     // HH:MM (24-hour)
	IsAvailable bool   `json:"is_available"`
}

// UserSelfUpdateRequest contains the fields a user can update on their own User row.
type UserSelfUpdateRequest struct {
	FullName           *string `json:"full_name"`
	PhoneNumber        *string `json:"phone_number"`
	ProfilePicture     *string `json:"profile_picture"`
	PreferredLanguage  *string `json:"preferred_language"`
	SecondaryLanguages *string `json:"secondary_languages"`
	Gender             *string `json:"gender"`
	FCMToken           *string `json:"fcm_token"`
}
