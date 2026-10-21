package database

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
	dsn := os.Getenv("DB_URL")
	isProd := os.Getenv("APP_ENV") == "production" || os.Getenv("APP_ENV") == "prod"
	useSQLite := os.Getenv("USE_SQLITE") == "true"

	if dsn == "" {
		sslmode := "disable"
		if isProd {
			sslmode = "require"
		}
		// Fallback to constructing from individual vars if full URL not set
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
			os.Getenv("POSTGRES_SERVER"),
			os.Getenv("POSTGRES_USER"),
			os.Getenv("POSTGRES_PASSWORD"),
			os.Getenv("POSTGRES_DB"),
			os.Getenv("POSTGRES_PORT"),
			sslmode,
		)
	} else if isProd && !strings.Contains(dsn, "sslmode=") && !useSQLite {
		if strings.Contains(dsn, "?") {
			dsn = dsn + "&sslmode=require"
		} else {
			dsn = dsn + "?sslmode=require"
		}
	}

	var err error
	isSQLiteDSN := (len(dsn) > 7 && dsn[:7] == "sqlite:") || (len(dsn) > 5 && dsn[:5] == "file:")

	if useSQLite || isSQLiteDSN {
		// Handle sqlite/file prefix for GORM
		var sqliteDB string
		if len(dsn) > 10 && dsn[:10] == "sqlite:///" {
			sqliteDB = dsn[10:]
		} else if len(dsn) > 7 && dsn[:7] == "sqlite:" {
			sqliteDB = dsn[7:]
		} else if len(dsn) > 5 && dsn[:5] == "file:" {
			sqliteDB = dsn // Keep file: prefix for :memory: support
		} else if useSQLite {
			sqliteDB = "Dr.Ayya.db"
		}

		logger.Default.LogMode(logger.Info)
		DB, err = gorm.Open(sqlite.Open(sqliteDB), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})

		if err == nil {
			sqlDB, _ := DB.DB()
			sqlDB.SetMaxOpenConns(10)
			sqlDB.SetMaxIdleConns(5)
			// Enable WAL mode and set busy timeout
			DB.Exec("PRAGMA journal_mode=WAL;")
			DB.Exec("PRAGMA synchronous=NORMAL;")
			DB.Exec("PRAGMA busy_timeout=5000;")
		}
	} else {
		log.Printf("Connecting to Postgres...")
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			sqlDB, errDb := DB.DB()
			if errDb == nil {
				// Pool sized for a single app instance serving the launch-time scale
				// (10k+ doctors + their patients). Tune via env if needed.
				//   MaxOpen 100   keeps PG connection count bounded across instances
				//   MaxIdle  25   warm connections for steady traffic
				//   Lifetime 30m  forces periodic reconnect (TLS cert rotation, etc.)
				//   IdleTime  5m  reclaims connections after burst traffic dies down
				sqlDB.SetMaxIdleConns(25)
				sqlDB.SetMaxOpenConns(100)
				sqlDB.SetConnMaxLifetime(30 * time.Minute)
				sqlDB.SetConnMaxIdleTime(5 * time.Minute)
				log.Println("Postgres connection pool configured (MaxOpen: 100, MaxIdle: 25, Lifetime: 30m, IdleTime: 5m)")
			} else {
				log.Printf("Warning: failed to get sqlDB instance for Postgres connection pooling: %v", errDb)
			}
		}
	}

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Database connection established")

	// Auto-migrate the User struct to ensure schema compatibility
	// Note: In production, use a proper migration tool like golang-migrate
	log.Println("Running AutoMigrate for User & Ledger models...")
	if err := DB.AutoMigrate(
	// &models.User{},
	// &models.LedgerAccount{},
	// &models.LedgerTransaction{},
	// &models.LedgerEntry{},
	// &models.Prescription{},
	// &models.MedicalRecord{},
	// &models.PrescriptionTemplate{},
	// &models.ActivityLog{},
	// &models.VideoConsultation{},
	// &models.PatientVitals{},
	// &models.HealthTimelineEvent{},
	// &models.Blog{},
	// &models.Franchise{},
	// &models.FranchiseRoyaltyLog{},
	// &models.Ticket{},
	// &models.TicketMessage{},
	); err != nil {
		log.Printf("                         AutoMigrate warning: %v", err)
	}
}

