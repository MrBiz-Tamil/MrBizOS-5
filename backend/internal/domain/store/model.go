package store

import (
	"drayya-core/internal/utils/privacy"
	"time"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&WellnessProduct{},
		&WellnessInventory{},
		&WellnessOrder{},
		&WellnessOrderItem{},
		&City{},
		&ServiceableArea{},
	)
}

type City struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex;not null" json:"name"` // e.g. Chennai, Bangalore
	State     string    `json:"state"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type ServiceableArea struct {
	ID         uint   `gorm:"primaryKey"`
	Pincode    string `gorm:"index;uniqueIndex:idx_pincode_pharmacy" json:"pincode"`
	CityID     uint   `gorm:"index" json:"city_id"`
	PharmacyID uint   `gorm:"index;uniqueIndex:idx_pincode_pharmacy" json:"pharmacy_id"`
}

type WellnessProduct struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Name          string    `gorm:"not null" json:"name"`
	Description   string    `json:"description"`
	Category      string    `json:"category"`
	Brand         string    `json:"brand"`
	SellingPrice  int64     `gorm:"not null" json:"selling_price"` // In paise
	DiscountPrice int64     `json:"discount_price"`
	ImageURL      string    `json:"image_url"`
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type WellnessInventory struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ProductID     uint      `gorm:"index;not null" json:"product_id"`
	PharmacyID    uint      `gorm:"index;not null" json:"pharmacy_id"` // Fulfilling Hub
	StockQuantity int       `gorm:"default:0" json:"stock_quantity"`
	ReorderLevel  int       `gorm:"default:10" json:"reorder_level"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type WellnessOrder struct {
	ID              uint                    `gorm:"primaryKey" json:"id"`
	OrderNumber     string                  `gorm:"uniqueIndex" json:"order_number"`
	UserID          uint                    `gorm:"index;not null" json:"user_id"`
	PharmacyID      uint                    `gorm:"index" json:"pharmacy_id"` // Fulfilling Hub
	TotalAmount     int64                   `gorm:"not null" json:"total_amount"`
	Status          string                  `gorm:"default:'pending'" json:"status"`         // pending, processing, shipped, delivered, cancelled
	PaymentStatus   string                  `gorm:"default:'pending'" json:"payment_status"` // pending, paid, refunded
	PaymentRef      string                  `json:"payment_ref"`
	DeliveryAddress privacy.EncryptedString `json:"delivery_address"`
	ContactPhone    privacy.EncryptedString `json:"contact_phone"`
	ContactEmail    privacy.EncryptedString `json:"contact_email"`
	Items           []WellnessOrderItem     `gorm:"foreignKey:OrderID" json:"items"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

type WellnessOrderItem struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	OrderID   uint            `gorm:"index;not null" json:"order_id"`
	ProductID uint            `gorm:"index;not null" json:"product_id"`
	Product   WellnessProduct `gorm:"foreignKey:ProductID" json:"product"`
	Quantity  int             `gorm:"not null" json:"quantity"`
	UnitPrice int64           `gorm:"not null" json:"unit_price"`
}
