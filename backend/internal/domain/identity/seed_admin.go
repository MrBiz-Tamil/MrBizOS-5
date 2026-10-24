package identity

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedDefaultAdmin ensures an initial enterprise administrator exists in the database.
func SeedDefaultAdmin(db *gorm.DB) {
	if db == nil {
		return
	}
	var count int64
	db.Model(&User{}).Where("email = ?", "admin@mrbiz.local").Count(&count)
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return
		}
		admin := User{
			Email:        "admin@mrbiz.local",
			PasswordHash: string(hash),
			FullName:     "Executive Administrator",
			Role:         "admin",
			IsActive:     true,
			IsVerified:   true,
		}
		_ = db.Create(&admin).Error
	}
}
