package devi

// services.go — ServiceContainer holds all domain service dependencies
// needed by the Devi engine.  Injected at startup; no flow file imports
// database.DB directly.

import (
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/lab"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/domain/store"
	"drayya-core/internal/domain/support"
	notifSvc "drayya-core/internal/services/notification"
)

// ServiceContainer bundles every external dependency the Devi engine needs.
// Construct it in server/app.go and pass it to NewEngine.
type ServiceContainer struct {
	// Infrastructure
	DB    *gorm.DB
	Redis *redis.Client

	// Domain services
	AppointmentSvc appointment.Service
	FinanceSvc     finance.Service
	StoreSvc       store.Service
	PharmacySvc    pharmacy.Service
	LabSvc         lab.Service
	SupportSvc     support.Service

	// Repositories (for lookups not covered by the service interface)
	IdentityRepo identity.Repository

	// Notification outbox
	NotifSvc *notifSvc.Service
}
