package automigrate

import (
	"log"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/analytics"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/billing"
	"drayya-core/internal/domain/budget"
	"drayya-core/internal/domain/career"
	"drayya-core/internal/domain/clinical"
	"drayya-core/internal/domain/cms"
	"drayya-core/internal/domain/communication"
	"drayya-core/internal/domain/consulting"
	"drayya-core/internal/domain/content"
	"drayya-core/internal/domain/corporate"
	"drayya-core/internal/domain/customer"
	devdomain "drayya-core/internal/domain/devi"
	"drayya-core/internal/domain/document"
	"drayya-core/internal/domain/ecommerce"
	"drayya-core/internal/domain/emr"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/franchise"
	"drayya-core/internal/domain/hospital"
	"drayya-core/internal/domain/hr"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/insurance"
	"drayya-core/internal/domain/journey"
	"drayya-core/internal/domain/lab"
	"drayya-core/internal/domain/loyalty"
	payoutdomain "drayya-core/internal/domain/payout"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/domain/questionnaire"
	"drayya-core/internal/domain/rating"
	"drayya-core/internal/domain/realestate"
	"drayya-core/internal/domain/referral"
	"drayya-core/internal/domain/refill"
	"drayya-core/internal/domain/store"
	domainSub "drayya-core/internal/domain/subscription"
	"drayya-core/internal/domain/support"
	"drayya-core/internal/domain/task"
	"drayya-core/internal/domain/travel"
	"drayya-core/internal/domain/zone"

	"gorm.io/gorm"
)

// Run runs all domain GORM AutoMigrate calls and baseline seeds.
// This is completely decoupled from any legacy REST server.
func Run(db *gorm.DB) {
	if db == nil {
		log.Println("[automigrate] WARNING: Database is nil, skipping migrations")
		return
	}

	log.Println("[automigrate] Starting automated schema migrations for MrBiz-OS domain models...")

	// Pre-migration fixes for Postgres schema drift
	if db.Dialector.Name() == "postgres" {
		db.Exec(`CREATE EXTENSION IF NOT EXISTS vector;`)
		db.Exec(`
			DO $$ 
			BEGIN 
			  IF EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_name='prescriptions' AND column_name='lab_tests' AND data_type='ARRAY'
			  ) THEN 
				ALTER TABLE prescriptions ALTER COLUMN lab_tests TYPE jsonb USING to_jsonb(lab_tests); 
			  END IF; 
			  IF EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_name='prescriptions' AND column_name='medicines' AND data_type='ARRAY'
			  ) THEN 
				ALTER TABLE prescriptions ALTER COLUMN medicines TYPE jsonb USING to_jsonb(medicines); 
			  END IF; 
			END $$;
		`)
		_ = db.Exec(`ALTER TABLE IF EXISTS activity_logs DROP CONSTRAINT IF EXISTS fk_activity_logs_user`).Error
		_ = db.Exec(`
			DELETE FROM activity_logs
			WHERE user_id <> 0
			  AND user_id NOT IN (SELECT id FROM "user")
		`).Error
	}

	migrations := []func() error{
		func() error { return identity.AutoMigrate(db) },
		func() error { return appointment.AutoMigrate(db) },
		func() error { return hospital.AutoMigrate(db) },
		func() error { return domainSub.AutoMigrate(db) },
		func() error { return finance.AutoMigrate(db) },
		func() error { return insurance.AutoMigrate(db) },
		func() error { return support.AutoMigrate(db) },
		func() error { return career.AutoMigrate(db) },
		func() error { return referral.AutoMigrate(db) },
		func() error { return rating.AutoMigrate(db) },
		func() error { return clinical.AutoMigrate(db) },
		func() error { return zone.AutoMigrate(db) },
		func() error { return hr.AutoMigrate(db) },
		func() error { return emr.AutoMigrate(db) },
		func() error { return communication.AutoMigrate(db) },
		func() error { return store.AutoMigrate(db) },
		func() error { return billing.AutoMigrate(db) },
		func() error { return corporate.AutoMigrate(db) },
		func() error { return loyalty.AutoMigrate(db) },
		func() error { return questionnaire.AutoMigrate(db) },
		func() error { return analytics.AutoMigrate(db) },
		func() error { return budget.AutoMigrate(db) },
		func() error { return pharmacy.AutoMigrate(db) },
		func() error { return payoutdomain.AutoMigrate(db) },
		func() error { return devdomain.AutoMigrate(db) },
		func() error { return lab.NewRepository(db).AutoMigrate() },
		func() error { return content.NewRepository(db).AutoMigrate() },
		func() error { return franchise.NewRepository(db).AutoMigrate() },
		func() error { return db.AutoMigrate(&refill.RefillSchedule{}, &refill.RefillOrder{}) },
		func() error { return document.AutoMigrate(db) },
		func() error { return task.AutoMigrate(db) },
		func() error { return journey.AutoMigrate(db) },
		func() error { return customer.AutoMigrate(db) },
		func() error { return travel.AutoMigrate(db) },
		func() error { return realestate.AutoMigrate(db) },
		func() error { return consulting.AutoMigrate(db) },
		func() error { return ecommerce.AutoMigrate(db) },
		func() error { return cms.AutoMigrate(db) },
	}

	for _, m := range migrations {
		if err := m(); err != nil {
			log.Printf("[automigrate] Error during domain migration: %v", err)
		}
	}

	// Idempotent baseline seeds
	identity.SeedDoctorTypes(db)
	clinical.SeedAYUSHPods(db)
	identity.SeedMockDoctors(db)
	identity.SeedDefaultAdmin(db)
	hospital.SeedIfEmpty(db)
	finance.SeedTaxRules(db)
	hr.BackfillEmploymentTypes(db)
	clinical.BackfillOnboarding(db)
	pharmacy.SeedAYUSHCatalog(db)
	lab.SeedIfEmpty(db)
	career.SeedTemplates(db)
	database.ApplyConstraints(db)

	log.Println("[automigrate] Schema migrations and seed checks completed successfully.")
}
