package main

import (
	"log"
	"os"
	"time"

	"drayya-core/internal/audit"
	"drayya-core/internal/audit/alerts"
	"drayya-core/internal/audit/anomaly"
	"drayya-core/internal/database"
	"drayya-core/internal/dispatch"
	"drayya-core/internal/livekit"
	"drayya-core/internal/roleconfig"
	"drayya-core/internal/server"
	"drayya-core/internal/shipping"
	"drayya-core/internal/telemetry"
	_ "drayya-core/docs" // Swagger documentation

	"github.com/joho/godotenv"
	"github.com/getsentry/sentry-go"
)

// @title Dr.Ayya Core API
// @version 1.0
// @description This is the API documentation for the Dr.Ayya Core Service.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.Dr.Ayya.com/support
// @contact.email support@Dr.Ayya.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

func main() {
	if err := godotenv.Load(".env", "../../.env"); err != nil {
		log.Printf("Note: .env file not found or error loading: %v, using system environment variables", err)
	}

	telemetry.InitLogger()

	// Initialize Sentry for error tracking
	if dsn := os.Getenv("SENTRY_DSN"); dsn != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              dsn,
			EnableTracing:    true,
			TracesSampleRate: 1.0,
			Environment:      os.Getenv("ENV"),
		})
		if err != nil {
			log.Fatalf("sentry.Init: %s", err)
		}
		defer sentry.Flush(2 * time.Second)
		log.Println("✅ Sentry initialized")
	}

	database.Connect()
	database.InitRedis()
	livekit.InitService()
	roleconfig.LogRole()

	// Schedulers gated by ROLE so they don't double-act across horizontal replicas.
	// ROLE=worker (or unset → legacy "all") owns all background pipelines below.
	if roleconfig.RunsSchedulers() {
		dispatch.StartWorker()
		// Courier dispatch automation — zero-human-intervention pipeline:
		//   • SLA monitor: scan every 30 s, escalate / auto-cancel breached orders
		//   • Batch optimiser: cluster same-pharmacy nearby orders every 3 min
		go shipping.RunSLAMonitor()
		go shipping.RunBatchOptimiser()
	} else {
		log.Println("[role] ROLE=api — skipping dispatch/SLA/batch schedulers")
	}

	// Run all GORM AutoMigrate calls in the background so the HTTP server
	// starts listening immediately. Docker health-checks hit /health which
	// needs no DB, so the container becomes healthy within seconds while
	// migrations complete in the background (typically 5-30s on PG).
	//
	// Concurrent-safe across horizontal replicas via a Postgres advisory lock
	// (SQLite is single-writer so the no-op fallback is fine for dev). Only one
	// process wins the lock and migrates; the others wait and skip.
	go func() {
		runMigrationsOnce()
		// Audit plugin must be installed AFTER migrations so the audit_logs
		// table exists by the time the first DB write fires its callbacks.
		// Install via the package-level helper which keeps a single global
		// plugin instance for the application lifetime.
		audit.InstallGlobal(database.DB)

		// Anomaly detector + daily digest — schedulers, so worker-only.
		if !roleconfig.RunsSchedulers() {
			return
		}
		if err := anomaly.AutoMigrate(database.DB); err != nil {
			log.Printf("[anomaly] migrate failed: %v", err)
		} else {
			// Wire alerts BEFORE scanner starts so the first scan's
			// critical hits don't slip through un-notified.
			anomaly.SetNotifier(alerts.OnCriticalAnomaly)
			// Add financial-integrity rules to the registry before scanning.
			anomaly.RegisterFinanceRules()
			go anomaly.StartScanner(15 * time.Minute)
			go alerts.RunDailyDigest()
		}
	}()

	app := server.New()

	port := os.Getenv("GO_PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("     Starting Go Core Service on port %s", port)

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("    Error starting server: %v", err)
	}
}

// migrationAdvisoryKey is a stable arbitrary 64-bit key for the Postgres
// advisory lock that gates schema migration across horizontal replicas.
// (Stable so concurrent boots contend for the same lock.)
const migrationAdvisoryKey int64 = 7331_4242_4242_7331

// runMigrationsOnce executes RunMigrations under a global Postgres advisory
// lock so only one replica migrates at a time. The other replicas block on
// pg_advisory_lock until the migrator releases, then re-enter and find the
// schema already current — AutoMigrate is idempotent.
//
// On SQLite (dev) the advisory-lock SQL is unsupported and silently fails;
// SQLite is single-writer so the fallback is safe.
func runMigrationsOnce() {
	if database.DB == nil {
		log.Println("[migrate] no DB — skipping migrations")
		return
	}
	dialect := database.DB.Dialector.Name()
	if dialect == "postgres" {
		if err := database.DB.Exec("SELECT pg_advisory_lock(?)", migrationAdvisoryKey).Error; err != nil {
			log.Printf("[migrate] advisory_lock failed (%v) — proceeding without coordination", err)
		} else {
			defer func() {
				if err := database.DB.Exec("SELECT pg_advisory_unlock(?)", migrationAdvisoryKey).Error; err != nil {
					log.Printf("[migrate] advisory_unlock failed: %v", err)
				}
			}()
		}
	}
	server.RunMigrations()
}

