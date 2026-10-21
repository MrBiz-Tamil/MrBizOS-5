package server

import (
	"context"
	"flag"
	"fmt"
	"log"
	"drayya-core/internal/audit"
	"drayya-core/internal/audit/anomaly"
	"drayya-core/internal/cron"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/analytics"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/billing"
	"drayya-core/internal/domain/budget"
	"drayya-core/internal/domain/career"
	"drayya-core/internal/domain/ceo_ai"
	"drayya-core/internal/domain/communication"
	"drayya-core/internal/domain/content"
	"drayya-core/internal/domain/corporate"
	devdomain "drayya-core/internal/domain/devi"
	deviv2 "drayya-core/internal/domain/devi/v2"
	"drayya-core/internal/domain/emr"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/franchise"
	"drayya-core/internal/domain/hospital"
	"drayya-core/internal/domain/hr"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/insurance"
	"drayya-core/internal/domain/lab"
	"drayya-core/internal/domain/loyalty"
	payoutdomain "drayya-core/internal/domain/payout"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/domain/questionnaire"
	"drayya-core/internal/domain/clinical"
	"drayya-core/internal/domain/cms"
	"drayya-core/internal/domain/consulting"
	"drayya-core/internal/domain/customer"
	"drayya-core/internal/domain/document"
	"drayya-core/internal/domain/ecommerce"
	"drayya-core/internal/domain/journey"
	"drayya-core/internal/domain/rating"
	"drayya-core/internal/domain/realestate"
	"drayya-core/internal/domain/task"
	"drayya-core/internal/domain/travel"
	"drayya-core/internal/domain/zone"
	"drayya-core/internal/domain/referral"
	"drayya-core/internal/domain/refill"
	"drayya-core/internal/domain/store"
	domainSub "drayya-core/internal/domain/subscription"
	"drayya-core/internal/domain/support"
	"drayya-core/internal/events"
	"drayya-core/internal/handlers"
	abdmHandler "drayya-core/internal/handlers/abdm"
	"drayya-core/internal/handlers/admin"
	"drayya-core/internal/handlers/payments"
	chatHandler "drayya-core/internal/handlers/chat"
	dispatchHandler "drayya-core/internal/handlers/dispatch"
	emrRefHandler "drayya-core/internal/handlers/emr"
	patientHandler "drayya-core/internal/handlers/patient"
	"drayya-core/internal/handlers/sos"
	wellnessHandler "drayya-core/internal/handlers/wellness"
	"drayya-core/internal/services/automation"
	"drayya-core/internal/services/storage"
	deviB2B "drayya-core/internal/services/devi"
	emailSvc "drayya-core/internal/services/email"
	notifSvc "drayya-core/internal/services/notification"
	"drayya-core/internal/services/embedding"
	bankreconcile "drayya-core/internal/services/finance"
	financialreports "drayya-core/internal/services/financials"
	franchiseSvc "drayya-core/internal/services/franchise"
	"drayya-core/internal/services/gst"
	livekitQuality "drayya-core/internal/services/livekit"
	"drayya-core/internal/services/tds"
	"drayya-core/internal/shipping"
	"os"
	"strings"
	"time"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/getsentry/sentry-go/fiber"

	"drayya-core/internal/middleware"
	"drayya-core/internal/ratelimit"
	"drayya-core/internal/roleconfig"
	"drayya-core/internal/telemetry"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/contrib/websocket"
	fiberSwagger "github.com/swaggo/fiber-swagger"
	"drayya-core/internal/presence"
)

// EventBus is the global domain event bus backed by Redis Streams.
// Handlers publish fire-and-forget; the worker goroutine consumes reliably.
var EventBus *events.Bus

// RunMigrations runs all GORM AutoMigrate calls and verifies schema safety.
// It fails loudly (panics) if any model-to-schema mismatch or migration error occurs.
func RunMigrations() {
	db := database.DB

	// Pre-migration fixes for Postgres schema drift that GORM cannot auto-resolve
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
	}

	// ── Pre-migration cleanup ──────────────────────────────────────────────
	// Earlier deploys created activity_logs.fk_activity_logs_user as a strict
	// FK to user(id). We intentionally support user_id=0 for failed-login /
	// anonymous events, which makes that FK incompatible. Two things to do:
	//   1. Drop the FK if it exists (so AutoMigrate doesn't try to validate it)
	//   2. Clean up any orphan rows (user_id references that don't exist)
	// The model itself is now annotated with constraint:- so future AutoMigrate
	// runs won't re-add it. This block makes the upgrade safe on hot DBs.
	if db.Dialector.Name() == "postgres" {
		_ = db.Exec(`ALTER TABLE IF EXISTS activity_logs DROP CONSTRAINT IF EXISTS fk_activity_logs_user`).Error
		// Delete rows whose user_id doesn't exist in user table — these would
		// re-trigger a constraint violation if anyone re-adds the FK.
		// user_id=0 stays (it's our sentinel) since 0 won't match any real id.
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
		func() error { patientHandler.MigrateWellness(db); return nil },
		func() error { return livekitQuality.NewQualityIngestService(db).AutoMigrate() },
		func() error { return bankreconcile.NewReconciliationService(db).AutoMigrate() },
		func() error { return deviB2B.NewLicensingService(db).AutoMigrate() },
		func() error { return notifSvc.AutoMigrateOutbox(db) },
		func() error { return document.AutoMigrate(db) },
		func() error { return task.AutoMigrate(db) },
		func() error { return journey.AutoMigrate(db) },
		func() error { return customer.AutoMigrate(db) },
		func() error { return travel.AutoMigrate(db) },
		func() error { return realestate.AutoMigrate(db) },
		func() error { return consulting.AutoMigrate(db) },
		func() error { return ecommerce.AutoMigrate(db) },
		func() error { return cms.AutoMigrate(db) },
		func() error { return finance.AutoMigrate(db) },
	}

	for _, m := range migrations {
		if err := m(); err != nil {
			panic(fmt.Sprintf("[startup] FATAL: Schema mismatch or migration failure: %v\nFix the DB schema or code models and restart.", err))
		}
	}

	// Seed data (idempotent)
	identity.SeedDoctorTypes(db)
	clinical.SeedAYUSHPods(db) // one pod per AYUSH type → per-system head doctors
	identity.SeedMockDoctors(db)
	hospital.SeedIfEmpty(db)
	finance.SeedTaxRules(db)
	hr.BackfillEmploymentTypes(db)  // one-time: classify existing doctor profiles as freelancers
	clinical.BackfillOnboarding(db) // sweep existing unverified doctors into the verification queue
	emr.BackfillEncryptedFields(db) // encrypt any pre-encryption-migration plaintext PHI
	pharmacy.SeedAYUSHCatalog(db)   // starter Siddha/Ayurveda/Wellness SKUs (idempotent)
	lab.SeedIfEmpty(db)
	embedding.SetupPgVector(db)
	career.SeedTemplates(db)
	database.ApplyConstraints(db)
}

func New() *fiber.App {
	// ── Email startup validation ──────────────────────────────────────────────
	// Phase 1: check env vars are present and well-formed (no network call).
	if err := emailSvc.ValidateConfig(); err != nil {
		fmt.Printf("\n[startup] ❌ EMAIL CONFIG ERROR:\n%v\n\n", err)
	} else {
		// Phase 2: live check — validate API key against Resend + confirm domain
		// is verified.  Runs in a goroutine so a slow DNS lookup doesn't block
		// the server from accepting requests.
		go func() {
			if err := emailSvc.VerifyLive(); err != nil {
				fmt.Printf("\n[startup] ⚠️  RESEND LIVE CHECK FAILED:\n%v\n\n", err)
			}
		}()
	}
	notifSvc.ValidateWAConfig()

	app := fiber.New(fiber.Config{
		AppName:      "Dr.Ayya Core (Go)",
		ErrorHandler: middleware.ErrorHandler,
		BodyLimit:    50 * 1024 * 1024, // 50MB
	})

	// Initialize Devi engine and service
	deviSvc := devdomain.NewService(devdomain.ServiceContainer{
		DB:    database.DB,
		Redis: database.Rdb,
	})
	handlers.InitWebSocketHub(automation.NewService(deviSvc))
	handlers.SetDeviMemoryEngine(deviSvc)

	// Start background presence cleanup (syncs DB with Redis heartbeats).
	// Scheduler → worker-only; on api replicas the periodic sweep stays silent.
	if flag.Lookup("test.v") == nil && roleconfig.RunsSchedulers() {
		startPresenceCleanup()
	}

	// Initialize the domain event bus (Redis Streams) and start the worker.
	// Falls back gracefully if Redis is unavailable.
	if database.Rdb != nil {
		EventBus = events.New(database.Rdb)
		go EventBus.Subscribe(context.Background(), "Dr.Ayya-workers", "worker-1", events.DefaultHandler)
		// Inject the bus into the handlers package so doctor status endpoints can publish events.
		handlers.SetEventBus(EventBus)
	}

	// Add recover middleware to catch panics and prevent server crashes
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	// Sentry middleware to capture exceptions and route tracking
	app.Use(sentryfiber.New(sentryfiber.Options{
		Repanic:         true,
		WaitForDelivery: false,
	}))

	// CORS must be first (after recover) — ensures every response (including 429s) carries
	// Access-Control-Allow-Origin. Without this, rate-limited responses lack
	// CORS headers and the browser reports a misleading "CORS policy" error
	// instead of the real 429 Too Many Requests.
	//
	// AllowOriginFunc is used instead of AllowOrigins so we can:
	//   1. Accept any *.mrbiz.in subdomain in production without hardcoding.
	//   2. Accept localhost on any port for local dev.
	//   3. Still fall back to the CORS_ALLOWED_ORIGINS env-var for strict
	//      deployments that need an explicit allowlist.
	explicitOrigins := strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",")
	hasExplicit := len(explicitOrigins) > 0 && explicitOrigins[0] != ""
	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			// Always allow mrbiz.in domains and localhost for development
			if strings.HasSuffix(origin, ".mrbiz.in") ||
				origin == "https://mrbiz.in" ||
				strings.HasPrefix(origin, "http://localhost") ||
				strings.HasPrefix(origin, "http://127.0.0.1") {
				return true
			}
			if hasExplicit {
				for _, o := range explicitOrigins {
					if strings.TrimSpace(o) == origin {
						return true
					}
				}
			}
			return false
		},
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, Cache-Control, Pragma, Expires, X-Requested-With",
		AllowCredentials: true,
	}))

	// Global rate limiter — raised to 300/min to accommodate dashboard polling
	// (doctor dashboard polls 4+ endpoints every few seconds simultaneously)
	if os.Getenv("SKIP_LIMITER") != "true" {
		// Redis-backed storage so the per-IP counter is GLOBAL across api
		// replicas. Falls back to in-memory (nil Storage) when Redis isn't up,
		// which preserves single-instance dev behaviour.
		app.Use(limiter.New(limiter.Config{
			Next: func(c *fiber.Ctx) bool {
				return c.Method() == "OPTIONS"
			},
			Max:        300,
			Expiration: 1 * time.Minute,
			Storage:    ratelimit.New(),
		}))
	}

	// Prometheus Metrics
	prometheus := fiberprometheus.New("Dr.Ayya")
	prometheus.RegisterAt(app, "/metrics")
	app.Use(prometheus.Middleware)

	// Structured Logging
	app.Use(telemetry.SlogMiddleware())

	// Audit Context — captures actor/role/request_id/ip onto the Go context
	// for the GORM audit plugin to consume. Must be AFTER any auth middleware
	// that sets c.Locals("user_id"), but for unprotected endpoints it still
	// records the IP/UA/request_id of the caller.
	app.Use(middleware.AuditCtx())

	app.Get("/health", func(c *fiber.Ctx) error {
		return response.JSON(c, fiber.StatusOK, fiber.Map{
			"status":  "healthy",
			"service": "core-go",
			"version": "1.0.0",
		}, "Service is healthy")
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return response.JSON(c, fiber.StatusOK, fiber.Map{
			"service": "Dr.Ayya API",
			"status":  "online",
		}, "Welcome to Dr.Ayya API")
	})

	// Serve uploaded assets (medicine photos, etc.) from local disk. In
	// production this directory would be backed by a CDN/object store; the
	// route stays the same so handlers can keep building /uploads/... URLs.
	app.Static("/uploads", "./storage/uploads", fiber.Static{
		ByteRange: true,
		MaxAge:    3600,
	})

	app.Use("/ws", handlers.WebSocketUpgrade)
	app.Get("/ws", websocket.New(handlers.WebSocketHandler))

	// Live audit-event stream (admin-only) for the 🔴 Real-time tab
	audit.MountWebSocket(app)

	// Live finance/ledger stream (admin-only) — pushes every committed ledger
	// transaction to the real-time finance dashboard.
	finance.MountWebSocket(app)

	// api Group (Legacy / API Gateway entry point)
	api := app.Group("/api", middleware.APILimiter())

	// Rewrite /api/... to /api/v1/... internally if version is missing.
	// Done as a middleware on the /api group so it applies to all subgroups.
	api.Use(func(c *fiber.Ctx) error {
		path := c.Path()
		if strings.HasPrefix(path, "/api") && !strings.HasPrefix(path, "/api/v1") {
			suffix := strings.TrimPrefix(path, "/api")
			c.Path("/api/v1" + suffix)
		}
		return c.Next()
	})

	// v1 Group
	v1 := api.Group("/v1")
	v1.Use(middleware.BusinessContext())

	// Group Business OS Shared Core Services Registration
	taskRepo := task.NewRepository(database.DB)
	taskSvc := task.NewService(taskRepo)
	taskHandler := task.NewHandler(taskSvc)
	taskHandler.RegisterRoutes(v1)

	journeySvc := journey.NewService(database.DB)
	journeyHandler := journey.NewHandler(journeySvc)
	journeyHandler.RegisterRoutes(v1)

	customerSvc := customer.NewService(database.DB)
	customerHandler := customer.NewHandler(customerSvc)
	customerHandler.RegisterRoutes(v1)

	travelRepo := travel.NewRepository(database.DB)
	travelSvc := travel.NewService(travelRepo, journeySvc, taskSvc)
	travelHandler := travel.NewHandler(travelSvc)
	travelHandler.RegisterRoutes(v1)

	reRepo := realestate.NewRepository(database.DB)
	reSvc := realestate.NewService(reRepo, journeySvc, taskSvc)
	reHandler := realestate.NewHandler(reSvc)
	reHandler.RegisterRoutes(v1)

	consultingRepo := consulting.NewRepository(database.DB)
	consultingSvc := consulting.NewService(consultingRepo, journeySvc, taskSvc)
	consultingHandler := consulting.NewHandler(consultingSvc)
	consultingHandler.RegisterRoutes(v1)

	ecommerceRepo := ecommerce.NewRepository(database.DB)
	ecommerceSvc := ecommerce.NewService(ecommerceRepo, taskSvc)
	ecommerceHandler := ecommerce.NewHandler(ecommerceSvc)
	ecommerceHandler.RegisterRoutes(v1)

	cmsRepo := cms.NewRepository(database.DB)
	cmsSvc := cms.NewService(cmsRepo)
	cmsHandler := cms.NewHandler(cmsSvc)
	cmsHandler.RegisterRoutes(v1)

	financeRepo := finance.NewRepository(database.DB)
	financeSvc := finance.NewService(financeRepo)
	financeHandler := finance.NewHandler(financeSvc)
	financeHandler.RegisterRoutes(v1)

	ceoAIRepo := ceo_ai.NewRepository(database.DB)
	ceoAISvc := ceo_ai.NewService(ceoAIRepo, financeSvc)
	ceoAIHandler := ceo_ai.NewHandler(ceoAISvc)
	ceoAIHandler.RegisterRoutes(v1)
	
	v1.Get("/core/verify", func(c *fiber.Ctx) error {
		return response.JSON(c, fiber.StatusOK, fiber.Map{"status": "verified"}, "Hello from Go Core Service!")
	})

	// Move services needed by multiple domains up
	financeRepo = finance.NewRepository(database.DB)
	financeSvc = finance.NewService(financeRepo)

	// Doctor Management Routes (PUBLIC)
	handlers.RegisterDoctorRoutes(v1, financeSvc)
	admin.RegisterSpecializationRoutes(v1)
	handlers.RegisterFranchiseRoutes(v1)

	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// Modular Route Registration
	identityRepo := identity.NewRepository(database.DB)
	identitySvc := identity.NewService(identityRepo)

	// Notification outbox: create service, wire into identity, start worker,
	// and register the package-level singleton used by WhatsApp* helpers and
	// the EnqueueEmail wrapper throughout the codebase.
	notificationSvc := notifSvc.NewService(database.DB)
	notifSvc.Init(notificationSvc)
	identitySvc.SetNotifier(notificationSvc)
	// The notification outbox already uses SKIP LOCKED atomic claim so it would
	// be safe horizontally — but it's a scheduler, so for clarity we still gate
	// it via ROLE alongside the other workers.
	if flag.Lookup("test.v") == nil && roleconfig.RunsSchedulers() {
		notifSvc.StartWorker(context.Background(), database.DB, notificationSvc.Senders(), 0)
	}

	identityHandler := identity.NewHandler(identitySvc)
	if EventBus != nil {
		identityHandler.SetEventBus(EventBus)
	}
	identityHandler.RegisterRoutes(v1)

	appointmentRepo := appointment.NewDefaultRepository()

	sos.RegisterSOSRoutes(v1)
	dispatchHandler.RegisterDispatchRoutes(v1)
	hospitalRepo := hospital.NewDefaultRepository(database.DB)
	hospitalSvc := hospital.NewService(hospitalRepo)
	hospitalHandler := hospital.NewHandler(hospitalSvc)


	// --- Public Catalog & Store Modules (Registered early for clarity) ---
	storeRepo := store.NewRepository(database.DB)
	storeSvc := store.NewService(storeRepo, financeSvc)
	storeHandler := store.NewHandler(storeSvc)
	storeHandler.RegisterRoutes(v1)
	wellnessHandler.RegisterWellnessRoutes(v1, storeSvc)

	pharmacyRepo := pharmacy.NewDefaultRepository(database.DB)
	pharmacySvc := pharmacy.NewService(pharmacyRepo, financeSvc)
	pharmacyHandler := pharmacy.NewHandler(pharmacySvc, database.DB)
	pharmacyHandler.RegisterRoutes(v1)
	pharmacy.RegisterRecallRoutes(v1)

	// ADR-0001 pilot: mount the pharmacy hexagon (Dispense use case via
	// port/adapter + the Generic Resource engine for Coupon CRUD).
	// mountPharmacyPilot(app, database.DB, financeSvc, EventBus)

	// --- Manufacturing & Raw Material Workflow (Multi-City) ---
	pharmacy.AutoMigrateManufacturing(database.DB)
	pharmacy.AutoMigrateRawMaterial(database.DB)
	mfgRepo := pharmacy.NewManufacturingRepository(database.DB)
	rmRepo := pharmacy.NewRawMaterialRepository(database.DB)
	rmSvc := pharmacy.NewRawMaterialService(rmRepo)
	mfgSvc := pharmacy.NewManufacturingService(mfgRepo, rmRepo, pharmacyRepo)
	mfgHandler := pharmacy.NewManufacturingHandler(mfgSvc, rmSvc)
	mfgHandler.RegisterRoutes(v1)
	rmHandler := pharmacy.NewRawMaterialHandler(rmSvc)
	rmHandler.RegisterRoutes(v1)

	// --- Pharmacy Admin Audit System ---
	auditSvc := pharmacy.NewAuditService(database.DB)
	auditHandler := pharmacy.NewAuditHandler(auditSvc)
	auditHandler.RegisterRoutes(v1)

	labRepo := lab.NewRepository(database.DB)
	labSvc := lab.NewService(labRepo, financeSvc)
	labHandler := lab.NewHandler(labSvc)
	labHandler.RegisterRoutes(v1)

	// Stripe webhook — unauthenticated (Stripe calls it) but signature-verified.
	payments.NewStripeWebhookHandler(labSvc, storeSvc).RegisterRoutes(app)

	hospitalHandler.RegisterRoutes(v1)
	// Legacy registrations (to be migrated)
	domainSub.RegisterRoutes(v1)

	financeHandler = finance.NewHandler(financeSvc)
	financeHandler.RegisterRoutes(v1)

	// Insurance
	insuranceRepo := insurance.NewRepository(database.DB)
	insuranceSvc := insurance.NewService(insuranceRepo, financeSvc)
	insuranceHandler := insurance.NewHandler(insuranceSvc)
	insuranceHandler.RegisterRoutes(v1, middleware.Protected())
	handlers.RegisterProductivityRoutes(v1)
	handlers.RegisterSOPRoutes(v1)
	audit.RegisterRoutes(v1)
	audit.RegisterAnalyticsRoutes(v1)
	anomaly.RegisterRoutes(v1)
	handlers.AutoMigrateSOPs(database.DB)
	supportRepo := support.NewRepository(database.DB)
	supportSvc := support.NewService(supportRepo)
	supportHandler := support.NewHandler(supportSvc)
	supportHandler.RegisterRoutes(v1, middleware.Protected())
	handlers.RegisterLiveKitRoutes(v1)
	chatHandler.RegisterChatRoutes(v1)
	handlers.RegisterWalletRoutes(v1, financeSvc)

	patientProfileHandler := patientHandler.NewHandler(database.DB)
	patientProfileHandler.RegisterRoutes(v1)

	// Shipping
	shipping.ShippingService = shipping.NewService(financeSvc)
	shipping.InitDispatchDock() // must be called after ShippingService is set (fixes nil pointer)
	shippingHandler := handlers.NewShippingHandler(shipping.ShippingService)
	shippingHandler.RegisterRoutes(v1)
	shipping.RegisterCourierRoutes(v1)  // courier app APIs: location ping, status update, POD
	shipping.RegisterWebhookRoutes(app) // 3PL inbound webhooks (public, HMAC-verified)

	// Career
	careerRepo := career.NewRepository(database.DB)
	careerSvc := career.NewService(careerRepo)
	careerHandler := handlers.NewCareerHandler(careerSvc)
	careerHandler.RegisterRoutes(v1)

	// Document vault — applicant/doctor records (photo, CV, license, etc.).
	if docStore, err := storage.New(); err != nil {
		log.Printf("[storage] vault disabled: %v", err)
	} else {
		log.Printf("[storage] document vault backend: %s", docStore.Backend())
		docHandler := document.NewHandler(database.DB, docStore)
		docHandler.RegisterRoutes(v1)
		document.SetDefault(docHandler) // let other handlers (health upload) use the vault
	}

	// Referral
	referralRepo := referral.NewRepository(database.DB)
	referralSvc := referral.NewService(referralRepo)
	referralHandler := handlers.NewReferralHandler(referralSvc)
	referralHandler.RegisterRoutes(v1)

	// Rating
	ratingRepo := rating.NewRepository(database.DB)
	ratingSvc := rating.NewService(ratingRepo)
	ratingHandler := handlers.NewRatingHandler(ratingSvc)
	ratingHandler.RegisterRoutes(v1)

	// Clinical oversight — chief-doctor consultation QA, pods, scorecard, compliance.
	clinical.NewHandler(database.DB).RegisterRoutes(v1)
	if roleconfig.RunsSchedulers() {
		clinical.StartComplianceScanner(database.DB)
	}

	// Geographic service zones — zone-first in-person matching.
	zone.NewHandler(database.DB).RegisterRoutes(v1)

	handlers.RegisterHealthRoutes(v1)
	// handlers.RegisterLabRoutes(v1) - Deprecated, using lab domain
	hrRepo := hr.NewRepository(database.DB)
	hrSvc := hr.NewService(hrRepo)
	hrOnboardingRepo := hr.NewOnboardingRepository(database.DB)
	hrOnboardingSvc := hr.NewOnboardingService(hrOnboardingRepo)
	hrOnboardingHandler := hr.NewOnboardingHandler(hrOnboardingSvc, hrRepo)
	hrHandler := hr.NewHandler(hrSvc, hrOnboardingHandler)
	hrHandler.RegisterRoutes(v1)

	// --- Domain-Driven Modules ---
	emrRepo := emr.NewRepository(database.DB)
	emrSvc := emr.NewService(emrRepo, identityRepo, appointmentRepo)
	emrHandler := emr.NewHandler(emrSvc, database.DB)
	emrHandler.RegisterRoutes(v1)

	commRepo := communication.NewRepository(database.DB)
	commSvc := communication.NewService(commRepo, identityRepo)
	commHandler := communication.NewHandler(commSvc)
	commHandler.RegisterRoutes(v1)
	// Inject communication service into the WebSocket hub so messages are
	// persisted via the domain layer and room membership is validated.
	handlers.SetHubCommSvc(commSvc)

	// Migration-only domains (breaking legacy cycles)
	contentRepo := content.NewRepository(database.DB)
	contentSvc := content.NewService(contentRepo, commSvc)
	contentHandler := content.NewHandler(contentSvc)
	contentHandler.RegisterRoutes(v1)

	// Direct blog routes on the /api group (no /v1 prefix) — required for the
	// frontend axios interceptor which rewrites /blog/posts → /blogs → /api/blogs.
	apiBlogs := api.Group("/blogs")
	apiBlogs.Get("/", contentHandler.GetPublicPosts)
	apiBlogs.Get("/:id", contentHandler.GetPost)
	apiBlogs.Post("/subscribe", contentHandler.Subscribe)

	// Update appointment service with finance service
	appointmentSvc := appointment.NewService(appointmentRepo, financeSvc)
	appointmentHandler := appointment.NewHandler(appointmentSvc)
	if EventBus != nil {
		appointmentHandler.SetEventBus(EventBus)
	}
	appointmentHandler.RegisterRoutes(v1)

	billingRepo := billing.NewDefaultRepository()
	billingSvc := billing.NewService(billingRepo, financeSvc)
	billingHandler := billing.NewHandler(billingSvc)
	billingHandler.RegisterRoutes(v1)

	// Corporate Domain
	corporate.RegisterRoutes(v1)

	// Loyalty / Rewards
	loyalty.SeedDefaultRules()
	loyaltyHandler := loyalty.NewHandler()
	loyaltyHandler.RegisterRoutes(v1)

	// Pre-consultation Questionnaire
	questionnaireHandler := questionnaire.NewHandler()
	questionnaireHandler.RegisterRoutes(v1)

	// Analytics Events Layer
	analyticsHandler := analytics.NewHandler()
	analyticsHandler.RegisterRoutes(v1)

	// Budget Management
	budgetHandler := budget.NewHandler()
	budgetHandler.RegisterRoutes(v1)

	// HR Leave Management + Payroll Compliance
	hr.SeedDefaultLeaveTypes()
	hr.RegisterLeaveRoutes(v1)
	hr.RegisterPayrollRoutes(v1)

	// Financial Reports
	financialreports.RegisterReportRoutes(v1)

	// GST Service
	gst.RegisterGSTRoutes(v1)

	// TDS Service
	tds.RegisterTDSRoutes(v1)

	// Devi V2 — LLM-first agent runtime. Mounted at /devi/v2/* so we can
	// run V1 and V2 side by side during the A/B rollout. The rollout shim
	// MUST be registered BEFORE V1's /devi/chat handler so Fiber's
	// middleware-applies-to-subsequently-mounted-routes rule activates it.
	_ = deviv2.AutoMigrate(database.DB)
	v2agent := deviv2.NewAgent(
		deviv2.DefaultRegistry(),
		deviv2.NewLLMClient(),
		deviv2.NewMemory(0, 0),
		deviv2.NewLimiter(),
	)
	v2handler := deviv2.NewHandler(v2agent)
	v2handler.RegisterRoutes(v1)
	v2handler.RegisterAdminRoutes(v1)

	// A/B rollout shim. Mounted BEFORE V1 below so it intercepts.
	// Only intercept if the Gemini key is present so we don't break chat.
	if os.Getenv("GEMINI_API_KEY") != "" {
		v1.Use("/devi/chat", func(c *fiber.Ctx) error {
			if c.Method() != "POST" {
				return c.Next()
			}
			userID, _ := c.Locals("user_id").(uint)
			role, _ := c.Locals("role").(string)
			browser := c.Query("guest_browser_id")
			var uidPtr *uint
			if userID > 0 {
				uidPtr = &userID
			}
			if deviv2.ShouldRouteToV2(role, uidPtr, browser) {
				return v2handler.HandleChat(c)
			}
			return c.Next()
		})
	}

	// Devi HTTP chat + WhatsApp webhook + admin endpoints (V1).
	deviHandler := devdomain.NewHandler(deviSvc, handlers.Hub)
	deviHandler.SetInfra(database.DB, database.Rdb)
	deviHandler.RegisterRoutes(v1)

	// Wire Devi engine to the WhatsApp inbound webhook so replies are sent automatically.
	// Done here (not inside automation package) to avoid an import cycle:
	//   automation → handlers → ws → automation
	handlers.RegisterWADeviProcessor(func(msg *communication.ChatMessage, phone, sessionID string) string {
		return deviSvc.ProcessWhatsApp(context.Background(), phone, sessionID, msg.Message)
	})

	// Background cron jobs: appointment reminders (every 1min) + weekly doctor
	// payouts. Worker-role only — at 2x replicas the *same* payout cron would
	// fire twice (double ledger postings) without this gate.
	if os.Getenv("JWT_SECRET") != "test-secret" && os.Getenv("JWT_SECRET") != "test_secret_key" && os.Getenv("USE_SQLITE") != "true" && roleconfig.RunsSchedulers() {
		cron.Start(financeSvc)
	}

	// Notification routes (FCM token registration)
	handlers.RegisterNotificationRoutes(v1)


	// Payout management (admin trigger + doctor payout-details)
	handlers.RegisterPayoutRoutes(v1, financeSvc)

	// Devi symptom memory routes
	// Memory routes now handled by devdomain handler above

	admin.RegisterAdminRoutes(v1)
	admin.RegisterRevenueRoutes(v1.Group("/admin"))

	// Notification admin — list, stats, retry dead jobs + test-email endpoint
	notifAdmin := notifSvc.NewAdminHandler(database.DB)
	adminGroup := v1.Group("/admin", middleware.Protected(), middleware.AdminOnly())
	notifAdmin.RegisterAdminRoutes(adminGroup)
	registerEmailAdminRoutes(adminGroup)

	// WhatsApp Cloud API webhook (public — Meta sends unauthenticated requests)
	handlers.RegisterWhatsAppWebhookRoutes(app)

	// AI Engine Routes (Local Rule-based)
	aiGroup := v1.Group("/ai")
	aiGroup.All("/*", handlers.AIBridge)

	// ── NEW DOMAINS ──────────────────────────────────────────────────────────

	// ABDM / ABHA integration (Ayushman Bharat Digital Mission)
	abdmHandler.RegisterABHARoutes(v1)

	// EMR Clinical Referrals (doctor-to-specialist referral network)
	// (models migrated via emr.AutoMigrate above which now includes ClinicalReferral + ReferralNote)
	emrRefHandler.RegisterReferralRoutes(v1)

	// Medicine Auto-Refill
	refill.RegisterRefillRoutes(v1)

	// LiveKit Call Quality Ingest — webhook endpoint from LiveKit server
	qualitySvc := livekitQuality.NewQualityIngestService(database.DB)
	v1.Post("/livekit/quality", middleware.Protected(), func(c *fiber.Ctx) error {
		var stats livekitQuality.RoomStats
		if err := c.BodyParser(&stats); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_PAYLOAD", "Invalid quality stats payload")
		}
		if err := qualitySvc.IngestRoomStats(&stats); err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "INGEST_FAILED", err.Error())
		}
		return response.JSON(c, fiber.StatusOK, nil, "Quality stats ingested")
	})
	v1.Get("/livekit/quality/:appointment_id", middleware.Protected(), func(c *fiber.Ctx) error {
		id := c.Params("appointment_id")
		var apptID uint
		if _, err := fmt.Sscanf(id, "%d", &apptID); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid appointment ID")
		}
		reports, err := qualitySvc.GetRoomQuality(apptID)
		if err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
		}
		return response.JSON(c, fiber.StatusOK, reports, "Quality reports fetched")
	})

	// Bank Reconciliation (admin only)
	reconSvc := bankreconcile.NewReconciliationService(database.DB)
	reconGroup := v1.Group("/finance/reconciliation", middleware.Protected(), middleware.AdminOnly())
	reconGroup.Post("/import", func(c *fiber.Ctx) error {
		accountID := c.Query("account_id", "default")
		// Expects JSON array of CSV rows
		var rows [][]string
		if err := c.BodyParser(&rows); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Expected JSON array of CSV rows")
		}
		n, err := reconSvc.ImportCSV(accountID, rows)
		if err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "IMPORT_FAILED", err.Error())
		}
		return response.JSON(c, fiber.StatusOK, fiber.Map{"imported": n}, "Bank statement imported")
	})
	reconGroup.Post("/run", func(c *fiber.Ctx) error {
		var req struct {
			AccountID string `json:"account_id"`
			StartDate string `json:"start_date"`
			EndDate   string `json:"end_date"`
		}
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
		}
		start, _ := time.Parse("2006-01-02", req.StartDate)
		end, _ := time.Parse("2006-01-02", req.EndDate)
		result, err := reconSvc.Reconcile(req.AccountID, start, end)
		if err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "RECONCILE_FAILED", err.Error())
		}
		return response.JSON(c, fiber.StatusOK, result, "Reconciliation complete")
	})

	// Devi B2B API Licensing Portal (admin)
	b2bSvc := deviB2B.NewLicensingService(database.DB)
	b2bAdmin := v1.Group("/admin/devi/licenses", middleware.Protected(), middleware.AdminOnly())
	b2bAdmin.Post("/", func(c *fiber.Ctx) error {
		var req struct {
			OrgName      string `json:"org_name"`
			OrgType      string `json:"org_type"`
			ContactName  string `json:"contact_name"`
			ContactEmail string `json:"contact_email"`
			Plan         string `json:"plan"`
			WebhookURL   string `json:"webhook_url"`
		}
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		}
		lic, err := b2bSvc.CreateLicense(req.OrgName, req.OrgType, req.ContactName, req.ContactEmail, req.Plan, req.WebhookURL)
		if err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
		}
		return response.JSON(c, fiber.StatusCreated, lic, "B2B license created")
	})
	b2bAdmin.Get("/", func(c *fiber.Ctx) error {
		licenses, err := b2bSvc.ListLicenses()
		if err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
		}
		return response.JSON(c, fiber.StatusOK, licenses, "Licenses fetched")
	})
	b2bAdmin.Get("/:id/usage", func(c *fiber.Ctx) error {
		var id uint
		fmt.Sscanf(c.Params("id"), "%d", &id)
		summary, err := b2bSvc.GetUsageSummary(id)
		if err != nil {
			return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", err.Error())
		}
		return response.JSON(c, fiber.StatusOK, summary, "Usage summary fetched")
	})
	b2bAdmin.Patch("/:id", func(c *fiber.Ctx) error {
		var id uint
		fmt.Sscanf(c.Params("id"), "%d", &id)
		var req struct {
			IsActive *bool  `json:"is_active"`
			Plan     string `json:"plan"`
		}
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
		}
		if req.IsActive != nil && !*req.IsActive {
			if err := b2bSvc.RevokeLicense(id); err != nil {
				return response.Error(c, fiber.StatusInternalServerError, "REVOKE_FAILED", err.Error())
			}
		} else if req.Plan != "" {
			if err := b2bSvc.UpgradePlan(id, req.Plan); err != nil {
				return response.Error(c, fiber.StatusInternalServerError, "UPGRADE_FAILED", err.Error())
			}
		}
		return response.JSON(c, fiber.StatusOK, nil, "License updated")
	})
	b2bAdmin.Patch("/:id/revoke", func(c *fiber.Ctx) error {
		var id uint
		fmt.Sscanf(c.Params("id"), "%d", &id)
		if err := b2bSvc.RevokeLicense(id); err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "REVOKE_FAILED", err.Error())
		}
		return response.JSON(c, fiber.StatusOK, nil, "License revoked")
	})
	b2bAdmin.Delete("/:id", func(c *fiber.Ctx) error {
		var id uint
		fmt.Sscanf(c.Params("id"), "%d", &id)
		if err := b2bSvc.RevokeLicense(id); err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", err.Error())
		}
		return response.JSON(c, fiber.StatusOK, nil, "License deleted")
	})

	// Devi admin stats/logs/config are registered by deviHandler.RegisterRoutes above.

	// Enterprise managed units (for franchise/unit manager dashboards)
	v1.Get("/enterprise/managed-units", middleware.Protected(), func(c *fiber.Ctx) error {
		userID, _ := c.Locals("user_id").(uint)
		units, err := franchiseSvc.NewService().GetManagedUnits(userID)
		if err != nil {
			return response.JSON(c, fiber.StatusOK, []interface{}{}, "No managed units")
		}
		return response.JSON(c, fiber.StatusOK, units, "Managed units fetched")
	})
	v1.Post("/enterprise/clients", middleware.Protected(), middleware.AdminOnly(), func(c *fiber.Ctx) error {
		var req struct {
			Name   string `json:"name"`
			Domain string `json:"domain"`
			Type   string `json:"type"`
		}
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
		}
		return response.JSON(c, fiber.StatusCreated, fiber.Map{"message": "Enterprise client created", "name": req.Name}, "")
	})
	v1.Get("/enterprise/clients", middleware.Protected(), middleware.AdminOnly(), func(c *fiber.Ctx) error {
		clients, err := franchiseSvc.NewService().GetEnterpriseClients()
		if err != nil {
			return response.JSON(c, fiber.StatusOK, []interface{}{}, "")
		}
		return response.JSON(c, fiber.StatusOK, clients, "Enterprise clients fetched")
	})

	return app
}

// ─── Email admin endpoints ────────────────────────────────────────────────────

// registerEmailAdminRoutes wires up two admin-only email diagnostic routes:
//
//	GET  /admin/email/status   — live Resend key + domain verification check
//	POST /admin/email/test     — send a test email to a target address
func registerEmailAdminRoutes(adminGroup fiber.Router) {
	emailGroup := adminGroup.Group("/email")

	// GET /admin/email/status
	// Returns the current email configuration health.
	emailGroup.Get("/status", func(c *fiber.Ctx) error {
		cfg := emailSvc.LoadConfig()

		status := fiber.Map{
			"api_key_set":    cfg.APIKey != "",
			"api_key_masked": emailSvc.MaskKey(cfg.APIKey),
			"from_email":     cfg.FromEmail,
			"from_name":      cfg.FromName,
			"domain":         emailSvc.DomainOf(cfg.FromEmail),
		}

		if err := emailSvc.ValidateConfig(); err != nil {
			status["config_ok"] = false
			status["config_error"] = err.Error()
			return response.JSON(c, fiber.StatusOK, status, "Email config has errors")
		}
		status["config_ok"] = true

		// Live check
		if err := emailSvc.VerifyLive(); err != nil {
			status["domain_verified"] = false
			status["domain_error"] = err.Error()
			return response.JSON(c, fiber.StatusOK, status, "Domain not verified")
		}
		status["domain_verified"] = true
		return response.JSON(c, fiber.StatusOK, status, "Email config is healthy")
	})

	// POST /admin/email/test
	// Body: { "to": "admin@example.com" }
	// Sends a test email so you can confirm delivery end-to-end.
	emailGroup.Post("/test", func(c *fiber.Ctx) error {
		var req struct {
			To string `json:"to"`
		}
		if err := c.BodyParser(&req); err != nil || req.To == "" {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST",
				`{"to":"admin@example.com"} is required`)
		}

		cfg := emailSvc.LoadConfig()
		html := fmt.Sprintf(`
<h2>✅ Dr.Ayya Email Test</h2>
<p>This is a test email from <strong>%s</strong>.</p>
<p>If you received this, your Resend configuration is working correctly.</p>
<hr>
<small>Sent via Resend API · from: %s</small>`, cfg.FromName, cfg.FromAddr)

		if err := emailSvc.SendSimple(req.To, "Dr.Ayya — Email Test", html); err != nil {
			return response.Error(c, fiber.StatusBadGateway, "SEND_FAILED", err.Error())
		}
		return response.JSON(c, fiber.StatusOK,
			fiber.Map{"sent_to": req.To, "from": cfg.FromAddr},
			"Test email sent — check inbox (and spam folder)")
	})
}

func startPresenceCleanup() {
	go func() {
		// Wait a bit for startup to settle
		time.Sleep(30 * time.Second)
		// Run every 2 min (Redis TTL is 5 min) so stale "online" rows are
		// cleared within at most 7 min of a doctor's heartbeat dying.
		ticker := time.NewTicker(2 * time.Minute)
		for range ticker.C {
			if database.DB == nil || database.Rdb == nil {
				continue
			}
			ctx := context.Background()
			// log.Println("    [Cleanup] Syncing doctor online status...")

			// Get all doctors currently marked online in DB
			var profiles []identity.DoctorProfile
			database.DB.Where("is_online = ?", true).Find(&profiles)

			for _, p := range profiles {
				if !presence.IsDoctorOnline(ctx, fmt.Sprintf("%d", p.UserID)) {
					// Redis heartbeat expired, mark offline in DB

					database.DB.Model(&identity.DoctorProfile{}).
						Where("id = ?", p.ID).
						Updates(map[string]interface{}{
							"is_online":      false,
							"current_status": "offline",
						})
					// log.Printf("    [Cleanup] Doctor %d marked OFFLINE (heartbeat expired)", p.UserID)
				}
			}
		}
	}()
}
