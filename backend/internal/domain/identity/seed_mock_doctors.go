package identity

import (
	"gorm.io/gorm"
)

// SeedMockDoctors deletes all seeded mock doctors from the database.
func SeedMockDoctors(db *gorm.DB) {
	mockEmails := []string{
		"arulmani@medihub.com",
		"selvam@medihub.com",
		"meera@medihub.com",
		"murugan@medihub.com",
		"krishnamurthy@medihub.com",
		"priya.ayur@medihub.com",
		"suresh.ayur@medihub.com",
		"lakshmi.yoga@medihub.com",
		"anand.naturo@medihub.com",
		"radha.yoga@medihub.com",
		"vikram.psych@medihub.com",
		"divya.counselor@medihub.com",
		"arun.trauma@medihub.com",
	}

	// Find user IDs of the mock doctors
	var userIDs []uint
	db.Model(&User{}).Where("email IN ?", mockEmails).Pluck("id", &userIDs)

	if len(userIDs) > 0 {
		// Delete doctor profiles
		db.Unscoped().Where("user_id IN ?", userIDs).Delete(&DoctorProfile{})
		// Delete doctor subspecialty assignments
		db.Unscoped().Where("doctor_user_id IN ?", userIDs).Delete(&DoctorSubSpecialtyAssignment{})
		// Delete users
		db.Unscoped().Where("id IN ?", userIDs).Delete(&User{})
	}
}
