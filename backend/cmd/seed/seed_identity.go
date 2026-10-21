package main

import (
	"fmt"
	"log"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/money"
	"drayya-core/internal/utils/privacy"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func seedIdentity(db *gorm.DB) {
	type userDef struct {
		Email    string
		Password string
		FullName string
		Role     string
		Gender   string
		Phone    string
	}

	users := []userDef{
		{"sithcare@gmail.com", "Test@123", "Admin User", "admin", "male", "9000000001"},
		{"finance@Dr.Ayya.com", "Test@123", "Finance Manager", "finance_manager", "female", "9000000002"},
		{"hr@Dr.Ayya.com", "Test@123", "HR Manager", "hr_manager", "female", "9000000003"},
		{"cpnking3@gmail.com", "Test@123", "Dr. Rajesh Sharma", "doctor", "male", "9100000001"},
		{"dr.patel@Dr.Ayya.com", "Test@123", "Dr. Priya Patel", "doctor", "female", "9100000002"},
		{"dr.gupta@Dr.Ayya.com", "Test@123", "Dr. Amit Gupta", "doctor", "male", "9100000003"},
		{"dr.kumar@Dr.Ayya.com", "Test@123", "Dr. Neha Kumar", "doctor", "female", "9100000004"},
		{"headdoctor@Dr.Ayya.com", "Test@123", "Dr. Ramesh", "hospital_head_doctor", "male", "9100000005"},
		{"nurse1@Dr.Ayya.com", "Test@123", "Anita Singh", "nurse", "female", "9200000001"},
		{"nurse2@Dr.Ayya.com", "Test@123", "Suresh Yadav", "nurse", "male", "9200000002"},
		{"labtech@Dr.Ayya.com", "Test@123", "Vikram Mehta", "lab_technician", "male", "9300000001"},
		{"pharmacist@Dr.Ayya.com", "Test@123", "Sunita Reddy", "pharmacist", "female", "9400000001"},
		{"patient1@drayya.com", "Test@123", "Aarav Kapoor", "patient", "male", "9500000001"},
		{"patient2@Dr.Ayya.com", "Test@123", "Diya Nair", "patient", "female", "9500000002"},
		{"patient3@Dr.Ayya.com", "Test@123", "Rohan Joshi", "patient", "male", "9500000003"},
		{"patient4@Dr.Ayya.com", "Test@123", "Meera Das", "patient", "female", "9500000004"},
		{"patient5@Dr.Ayya.com", "Test@123", "Kabir Malhotra", "patient", "male", "9500000005"},
		{"courier1@Dr.Ayya.com", "Test@123", "Ravi Courier", "bpo", "male", "9600000001"},
		{"courier2@Dr.Ayya.com", "Test@123", "Deepak Courier", "bpo", "male", "9600000002"},
		{"qa@Dr.Ayya.com", "Test@123", "QA Manager", "qa_manager", "female", "9700000001"},
		{"manager@Dr.Ayya.com", "Test@123", "General Manager", "manager", "male", "9700000002"},
		{"store@Dr.Ayya.com", "Test@123", "Store Manager", "store_manager", "male", "9700000003"},
		{"franchise@Dr.Ayya.com", "Test@123", "Franchise Owner", "franchise_owner", "female", "9700000004"},
	}

	for _, u := range users {
		var user identity.User
		hash, _ := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		
		if db.Where("email = ?", u.Email).First(&user).Error == nil {
			// Update existing user
			updates := map[string]interface{}{
				"password_hash": string(hash),
				"role":          u.Role,
				"full_name":     u.FullName,
				"gender":        u.Gender,
				"phone_number":  u.Phone,
				"is_active":     true,
				"is_verified":   true,
			}
			
			// If UHEID is missing, generate one
			if user.UHEID == nil || *user.UHEID == "" {
				date := time.Now().Format("20060102")
				random := uuid.New().String()[:8]
				uheid := fmt.Sprintf("DA-ECON-%s-%s", date, strings.ToUpper(random))
				updates["uheid"] = &uheid
			}
			
			db.Model(&user).Updates(updates)
			log.Printf("      Updated user: %s (%s)", u.Email, u.Role)
		} else {
			// Create new user
			date := time.Now().Format("20060102")
			random := uuid.New().String()[:8]
			uheid := fmt.Sprintf("DA-ECON-%s-%s", date, strings.ToUpper(random))
			
			user = identity.User{
				Email:        u.Email,
				PasswordHash: string(hash),
				FullName:     u.FullName,
				Role:         u.Role,
				Gender:       u.Gender,
				PhoneNumber:  u.Phone,
				IsActive:     true,
				IsVerified:   true,
				UHEID:        &uheid,
			}
			if u.Email == "patient1@drayya.com" {
				user.Tier = "prime"
			}
			if err := db.Create(&user).Error; err != nil {
				log.Printf("      Failed to create %s: %v", u.Email, err)
				continue
			}
			log.Printf("      Created user: %s (%s) ID=%d UHEID=%s", u.Email, u.Role, user.ID, *user.UHEID)
		}
	}

	// Doctor profiles
	type docDef struct {
		Email          string
		Specialization string
		License        string
		Experience     int
		Fee            int64
		Qualification  string
		Bio            string
	}
	doctors := []docDef{
		{"cpnking3@gmail.com", "General Medicine", "MH-GM-001", 15, 50000, "MBBS, MD", "Senior general physician with 15 years of experience"},
		{"dr.patel@Dr.Ayya.com", "Cardiology", "MH-CD-002", 12, 80000, "MBBS, DM Cardiology", "Expert cardiologist specializing in interventional procedures"},
		{"dr.gupta@Dr.Ayya.com", "Pediatrics", "MH-PD-003", 8, 60000, "MBBS, DCH", "Child specialist with focus on preventive care"},
		{"dr.kumar@Dr.Ayya.com", "Dermatology", "MH-DM-004", 10, 70000, "MBBS, MD Dermatology", "Skin and hair specialist with expertise in cosmetic dermatology"},
	}
	for _, d := range doctors {
		var user identity.User
		db.Where("email = ?", d.Email).First(&user)
		var existing identity.DoctorProfile
		if db.Where("user_id = ?", user.ID).First(&existing).Error == nil {
			continue
		}
		now := time.Now()
		dp := identity.DoctorProfile{
			UserID:            user.ID,
			Specialization:    d.Specialization,
			LicenseNumber:     d.License,
			YearsOfExperience: d.Experience,
			Qualification:     d.Qualification,
			ConsultationFees:  money.RegionalPricing{"INR": d.Fee},
			IsVerified:        true,
			IsOnline:          true,
			CurrentStatus:     "online",
			Bio:               d.Bio,
			AverageRating:     4.5,
			LanguagesSpoken:   "English,Hindi",
			LastPing:          &now,
		}
		db.Create(&dp)
	}

	// Doctor availability (Mon-Sat 8am-8pm for all)
	var doctorUsers []identity.User
	db.Where("role = ?", "doctor").Find(&doctorUsers)
	for _, doc := range doctorUsers {
		for day := 1; day <= 6; day++ {
			var existing identity.DoctorAvailability
			if db.Where("doctor_id = ? AND day_of_week = ?", doc.ID, day).First(&existing).Error == nil {
				continue
			}
			db.Create(&identity.DoctorAvailability{
				DoctorID:    doc.ID,
				DayOfWeek:   day,
				StartTime:   "08:00",
				EndTime:     "20:00",
				IsAvailable: true,
			})
		}
	}

	// Patient profiles
	type patDef struct {
		Email      string
		DOB        string
		BloodGroup string
		Height     float64
		Weight     float64
		Allergies  string
		Conditions string
		Contact    string
		Address    string
	}
	patients := []patDef{
		{"patient1@drayya.com", "1990-05-15", "O+", 175, 72, "Penicillin", "None", "9500000010", "123 MG Road, Mumbai"},
		{"patient2@Dr.Ayya.com", "1985-08-22", "A+", 162, 58, "None", "Asthma", "9500000020", "456 Park Street, Chennai"},
		{"patient3@Dr.Ayya.com", "1978-12-01", "B+", 180, 85, "Sulfa drugs", "Diabetes Type 2", "9500000030", "789 Gandhi Nagar, Delhi"},
		{"patient4@Dr.Ayya.com", "1995-03-10", "AB-", 165, 55, "None", "None", "9500000040", "10 Lake View, Bangalore"},
		{"patient5@Dr.Ayya.com", "2000-07-25", "O-", 170, 68, "Dust", "None", "9500000050", "55 Civil Lines, Jaipur"},
	}
	for _, p := range patients {
		var user identity.User
		db.Where("email = ?", p.Email).First(&user)
		var existing identity.PatientProfile
		if db.Where("user_id = ?", user.ID).First(&existing).Error == nil {
			continue
		}
		dob, _ := time.Parse("2006-01-02", p.DOB)
		db.Create(&identity.PatientProfile{
			UserID:            user.ID,
			DateOfBirth:       &dob,
			Gender:            user.Gender,
			BloodGroup:        p.BloodGroup,
			Height:            p.Height,
			Weight:            p.Weight,
			Allergies:         privacy.EncryptedString(p.Allergies),
			ChronicConditions: privacy.EncryptedString(p.Conditions),
			EmergencyContact:  privacy.EncryptedString(p.Contact),
			Address:           privacy.EncryptedString(p.Address),
		})
	}

	// Family members for patient1
	var p1 identity.User
	db.Where("email = ?", "patient1@drayya.com").First(&p1)
	childDOB, _ := time.Parse("2006-01-02", "2018-09-15")
	db.FirstOrCreate(&identity.FamilyMember{
		UserID:       p1.ID,
		FullName:     "Arjun Kapoor",
		Relationship: "son",
		DateOfBirth:  &childDOB,
		Gender:       "male",
		BloodGroup:   "O+",
	}, "user_id = ? AND full_name = ?", p1.ID, "Arjun Kapoor")

	spouseDOB, _ := time.Parse("2006-01-02", "1992-11-20")
	db.FirstOrCreate(&identity.FamilyMember{
		UserID:       p1.ID,
		FullName:     "Sneha Kapoor",
		Relationship: "spouse",
		DateOfBirth:  &spouseDOB,
		Gender:       "female",
		BloodGroup:   "A+",
	}, "user_id = ? AND full_name = ?", p1.ID, "Sneha Kapoor")

	log.Println("       Identity seeding complete")
}

