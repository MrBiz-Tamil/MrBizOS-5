package payments

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/lab"
	"drayya-core/internal/domain/store"
)

// End-to-end: a signed-off (mock) payment_intent.succeeded webhook for a STORE
// order drives the order to paid and posts revenue to the ledger.
func TestStripeWebhook_ConfirmsStoreOrder(t *testing.T) {
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	database.DB = db
	require.NoError(t, finance.AutoMigrate(db))

	storeRepo := store.NewRepository(db)
	require.NoError(t, storeRepo.AutoMigrate())
	labRepo := lab.NewRepository(db)
	require.NoError(t, labRepo.AutoMigrate())

	finSvc := finance.NewService(finance.NewRepository(db))
	storeSvc := store.NewService(storeRepo, finSvc)
	labSvc := lab.NewService(labRepo, finSvc)

	// Seed a product, stock, and an order.
	require.NoError(t, db.Create(&store.WellnessProduct{ID: 1, Name: "Vit C", SellingPrice: 50000, IsActive: true}).Error)
	require.NoError(t, db.Create(&store.WellnessInventory{ProductID: 1, PharmacyID: 1, StockQuantity: 5}).Error)
	order, err := storeSvc.CreateOrder(context.Background(), 7, []store.OrderItemRequest{{ProductID: 1, Quantity: 1}}, "addr", 1, "1234567890", "test@test.com")
	require.NoError(t, err)

	app := fiber.New()
	NewStripeWebhookHandler(labSvc, storeSvc).RegisterRoutes(app)

	body := `{"type":"payment_intent.succeeded","data":{"object":{"id":"pi_abc","metadata":{"ref":"STORE-` +
		itoa(order.ID) + `"}}}}`
	req := httptest.NewRequest("POST", "/webhooks/stripe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Order is now paid and revenue is booked.
	var got store.WellnessOrder
	require.NoError(t, db.First(&got, order.ID).Error)
	assert.Equal(t, "paid", got.PaymentStatus)

	var stripeAcct finance.LedgerAccount
	require.NoError(t, db.Where("name = ?", finance.AcctStripe).First(&stripeAcct).Error)
	assert.Equal(t, int64(50000), stripeAcct.Balance, "INR amount booked into Asset:Stripe")
}

func itoa(u uint) string {
	if u == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for u > 0 {
		i--
		b[i] = byte('0' + u%10)
		u /= 10
	}
	return string(b[i:])
}
