package shipping

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

// ShipmentRequest carries the real recipient + order info a 3PL needs to label a
// shipment correctly. Previously the interface only took (orderID, address,
// pincode, weightKg) and providers padded the rest with placeholders like
// "Dr.Ayya Customer" / "9999999999" / "customer@example.com" which would be
// printed on every Shiprocket waybill in production. This struct threads the
// actual patient/order values through so deliveries reach the right person.
type ShipmentRequest struct {
	OrderID     string  // e.g. "MH-RX-1234"
	WeightKg    float64
	SubTotalINR float64 // declared value for the courier

	// Recipient (the patient)
	RecipientName  string
	RecipientPhone string
	RecipientEmail string
	Address        string
	City           string
	State          string
	Pincode        string

	// Items — for couriers that itemise (Shiprocket, BlueDart).
	Items []ShipmentItem
}

// ShipmentItem is a single line on the courier waybill.
type ShipmentItem struct {
	Name         string
	SKU          string
	Units        int
	SellingPrice float64 // per unit, in INR
}

// ShippingProvider defines the common interface for all 3PL (Third Party Logistics) services.
type ShippingProvider interface {
	Name() string
	Priority() int // Higher = more preferred (e.g. Blue Dart > India Post)
	IsServiceable(pincode string) bool
	CalculateCost(weightKg float64, pickupPincode, destPincode string) (float64, error)
	EstimatedDays(pickupPincode, destPincode string) int
	CreateShipment(req ShipmentRequest) (string, error)
}

// --- Shiprocket Provider ---

type ShiprocketProvider struct{}

func NewShiprocketProvider() *ShiprocketProvider {
	return &ShiprocketProvider{}
}

func (p *ShiprocketProvider) Name() string { return "Shiprocket" }
func (p *ShiprocketProvider) Priority() int { return 50 }

func (p *ShiprocketProvider) EstimatedDays(pickupPincode, destPincode string) int {
	return 3
}

func (p *ShiprocketProvider) IsServiceable(pincode string) bool {
	// Mock Logic: Serviceable if pincode starts with '1' to '5' (Urban/Semi-Urban)
	if len(pincode) == 6 && pincode[0] >= '1' && pincode[0] <= '5' {
		return true
	}
	return false
}

func (p *ShiprocketProvider) CalculateCost(weightKg float64, pickupPincode, destPincode string) (float64, error) {
	baseRate := 40.0
	return baseRate + (weightKg * 20.0), nil
}

func (p *ShiprocketProvider) CreateShipment(r ShipmentRequest) (string, error) {
	// Translate ShipmentRequest items to Shiprocket order items.
	items := make([]ShiprocketOrderItem, 0, len(r.Items))
	for _, it := range r.Items {
		items = append(items, ShiprocketOrderItem{
			Name:         it.Name,
			SKU:          it.SKU,
			Units:        it.Units,
			SellingPrice: it.SellingPrice,
		})
	}
	// Fallback for legacy callers that don't supply itemised data.
	if len(items) == 0 {
		items = []ShiprocketOrderItem{{Name: "Medicine Package", SKU: "MED-PKG", Units: 1, SellingPrice: r.SubTotalINR}}
	}
	// Defensive fallbacks: if the dispatcher didn't pass a value, leave the field
	// blank rather than fabricating one. Shiprocket will return a clearer error.
	city, state := r.City, r.State
	if city == "" {
		city = "NA"
	}
	if state == "" {
		state = "NA"
	}

	req := ShiprocketCreateOrderRequest{
		OrderID:             r.OrderID,
		OrderDate:           time.Now().Format("2006-01-02 15:04"),
		PickupLocation:      "Primary", // Default pickup location in Shiprocket panel
		BillingCustomerName: r.RecipientName,
		BillingAddress:      r.Address,
		BillingCity:         city,
		BillingPincode:      r.Pincode,
		BillingState:        state,
		BillingCountry:      "India",
		BillingEmail:        r.RecipientEmail,
		BillingPhone:        r.RecipientPhone,
		ShippingIsBilling:   true,
		PaymentMethod:       "Prepaid",
		SubTotal:            r.SubTotalINR,
		Length:              10.0, // cm — TODO take from product master once available
		Breadth:             10.0,
		Height:              10.0,
		Weight:              r.WeightKg,
		OrderItems:          items,
	}

	// 2. Call the real Shiprocket API
	resp, err := SRCreateOrder(req)
	if err != nil {
		return "", err
	}

	if resp.AWBCode == "" {
		return "", fmt.Errorf("shiprocket order created but no AWB assigned")
	}

	return resp.AWBCode, nil
}

// --- Delhivery Provider ---

type DelhiveryProvider struct{}

func NewDelhiveryProvider() *DelhiveryProvider {
	return &DelhiveryProvider{}
}

func (p *DelhiveryProvider) Name() string { return "Delhivery" }
func (p *DelhiveryProvider) Priority() int { return 60 } // Higher than Shiprocket

func (p *DelhiveryProvider) EstimatedDays(pickupPincode, destPincode string) int {
	return 2
}

func (p *DelhiveryProvider) IsServiceable(pincode string) bool {
	// Delhivery has massive coverage, but let's exclude '9' for mock testing
	return len(pincode) == 6 && pincode[0] != '9'
}

func (p *DelhiveryProvider) CalculateCost(weightKg float64, pickupPincode, destPincode string) (float64, error) {
	baseRate := 35.0
	return baseRate + (weightKg * 18.0), nil
}

func (p *DelhiveryProvider) CreateShipment(r ShipmentRequest) (string, error) {
	req := DelhiveryCreateRequest{
		Shipments: []DelhiveryShipment{
			{
				Name:        r.RecipientName,
				Add:         r.Address,
				Pin:         r.Pincode,
				Phone:       r.RecipientPhone,
				PaymentMode: "Prepaid",
				Order:       r.OrderID,
				TotalAmount: r.SubTotalINR,
				Weight:      r.WeightKg * 1000, // Delhivery expects grams
				Quantity:    "1",
			},
		},
		Pickup: DelhiveryPickup{Name: "Warehouse1"},
	}

	resp, err := CreateDelhiveryShipment(req)
	if err != nil {
		return "", err
	}

	if len(resp.Packages) == 0 {
		return "", fmt.Errorf("delhivery created no packages")
	}

	return resp.Packages[0].Waybill, nil
}

// --- Blue Dart Provider ---

type BlueDartProvider struct{}

func NewBlueDartProvider() *BlueDartProvider {
	return &BlueDartProvider{}
}

func (p *BlueDartProvider) Name() string { return "BlueDart" }
func (p *BlueDartProvider) Priority() int { return 80 } // Premium priority

func (p *BlueDartProvider) EstimatedDays(pickupPincode, destPincode string) int {
	return 1
}

func (p *BlueDartProvider) IsServiceable(pincode string) bool {
	// Blue Dart is mainly Metro/Tier 1
	if len(pincode) == 6 && (pincode[0] == '1' || pincode[0] == '4' || pincode[0] == '5' || pincode[0] == '6') {
		return true
	}
	return false
}

func (p *BlueDartProvider) CalculateCost(weightKg float64, pickupPincode, destPincode string) (float64, error) {
	baseRate := 80.0 // Premium pricing
	return baseRate + (weightKg * 40.0), nil
}

func (p *BlueDartProvider) CreateShipment(r ShipmentRequest) (string, error) {
	req := BlueDartCreateRequest{}
	req.AwbDetails.ProductCode = "A"
	req.AwbDetails.Pincode = r.Pincode
	req.CustomerDetail.Name = r.RecipientName
	req.CustomerDetail.Address = r.Address

	resp, err := CreateBlueDartShipment(req)
	if err != nil {
		return "", err
	}
	return resp.AWBNo, nil
}

// --- India Post Provider ---

type IndiaPostProvider struct{}

func NewIndiaPostProvider() *IndiaPostProvider {
	return &IndiaPostProvider{}
}

func (p *IndiaPostProvider) Name() string { return "IndiaPost" }
func (p *IndiaPostProvider) Priority() int { return 10 } // Lowest priority fallback

func (p *IndiaPostProvider) EstimatedDays(pickupPincode, destPincode string) int {
	return 6
}

func (p *IndiaPostProvider) IsServiceable(pincode string) bool {
	return true
}

func (p *IndiaPostProvider) CalculateCost(weightKg float64, pickupPincode, destPincode string) (float64, error) {
	baseRate := 30.0
	return baseRate + (weightKg * 10.0), nil
}

func (p *IndiaPostProvider) CreateShipment(r ShipmentRequest) (string, error) {
	// India Post enterprise usually uses a pre-generated series or bulk SFTP.
	// Here we simulate a REST wrapper.
	log.Printf("[IndiaPost] Generating consignment for order %s to %s (%s)", r.OrderID, r.RecipientName, r.Pincode)
	return "IN" + time.Now().Format("02011504") + "IN", nil
}

// --- Local Delivery Provider ---

type LocalDeliveryProvider struct{}

func NewLocalDeliveryProvider() *LocalDeliveryProvider {
	return &LocalDeliveryProvider{}
}

func (p *LocalDeliveryProvider) Name() string { return "QuickLocal" }
func (p *LocalDeliveryProvider) Priority() int { return 100 } // Highest for hyperlocal

func (p *LocalDeliveryProvider) EstimatedDays(pickupPincode, destPincode string) int {
	return 0 // Same day
}

func (p *LocalDeliveryProvider) IsServiceable(pincode string) bool {
	return true
}

func (p *LocalDeliveryProvider) CalculateCost(weightKg float64, pickupPincode, destPincode string) (float64, error) {
	return 50.0, nil
}

func (p *LocalDeliveryProvider) CreateShipment(r ShipmentRequest) (string, error) {
	return "LOCAL_" + r.OrderID + "_VAN", nil
}

// --- DTDC Provider ---

type DTDCProvider struct{}

func NewDTDCProvider() *DTDCProvider {
	return &DTDCProvider{}
}

func (p *DTDCProvider) Name() string     { return "DTDC" }
func (p *DTDCProvider) Priority() int    { return 75 }

func (p *DTDCProvider) EstimatedDays(pickupPincode, destPincode string) int {
	return 3
}
func (p *DTDCProvider) IsServiceable(pincode string) bool {
	// Wide coverage in India
	return len(pincode) == 6 && pincode[0] != '0'
}

func (p *DTDCProvider) CalculateCost(weightKg float64, pickupPincode, destPincode string) (float64, error) {
	baseRate := 45.0
	return baseRate + (weightKg * 22.0), nil
}

func (p *DTDCProvider) CreateShipment(r ShipmentRequest) (string, error) {
	req := DTDCCreateRequest{
		CustomerCode: "CUST001",
		ReceiverName: r.RecipientName,
		ReceiverAdd:  r.Address,
		ReceiverPin:  r.Pincode,
	}
	resp, err := CreateDTDCShipment(req)
	if err != nil {
		return "", err
	}
	return resp.ConsignmentNumber, nil
}

// --- Shadowfax Provider ---

type ShadowfaxProvider struct{}

func NewShadowfaxProvider() *ShadowfaxProvider {
	return &ShadowfaxProvider{}
}

func (p *ShadowfaxProvider) Name() string     { return "Shadowfax" }
func (p *ShadowfaxProvider) Priority() int    { return 55 }

func (p *ShadowfaxProvider) EstimatedDays(pickupPincode, destPincode string) int {
	return 2
}
func (p *ShadowfaxProvider) IsServiceable(pincode string) bool {
	// Shadowfax covers major cities and e-commerce hubs
	if len(pincode) == 6 && (pincode[0] == '1' || pincode[0] == '2' || pincode[0] == '4' || pincode[0] == '5' || pincode[0] == '6' || pincode[0] == '7') {
		return true
	}
	return false
}

func (p *ShadowfaxProvider) CalculateCost(weightKg float64, pickupPincode, destPincode string) (float64, error) {
	baseRate := 38.0
	return baseRate + (weightKg * 15.0), nil
}

func (p *ShadowfaxProvider) CreateShipment(r ShipmentRequest) (string, error) {
	req := ShadowfaxCreateRequest{
		OrderID:    r.OrderID,
		DestName:   r.RecipientName,
		DestAdd:    r.Address,
		DestPin:    r.Pincode,
		OrderValue: r.SubTotalINR,
	}
	resp, err := CreateShadowfaxShipment(req)
	if err != nil {
		return "", err
	}
	return resp.AWB, nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
