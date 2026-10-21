package shipping

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

// IndiaPostShipmentRequest represents the data needed to book a Speed Post shipment.
type IndiaPostShipmentRequest struct {
	OrderID         string  `json:"order_id"`
	CustomerName    string  `json:"customer_name"`
	Address         string  `json:"address"`
	Pincode         string  `json:"pincode"`
	WeightGrams     int     `json:"weight_grams"`
	Contents        string  `json:"contents"`
	IsInsured       bool    `json:"is_insured"`
	SenderWarehouse string  `json:"sender_warehouse"`
}

// IndiaPostShipmentResponse represents the response from the India Post booking service.
type IndiaPostShipmentResponse struct {
	AWB          string    `json:"awb"`
	BookingDate  time.Time `json:"booking_date"`
	ExpectedDate time.Time `json:"expected_date"`
	Status       string    `json:"status"`
}

// IndiaPostTrackResponse represents a simplified tracking event.
type IndiaPostTrackResponse struct {
	AWB           string `json:"awb"`
	CurrentStatus string `json:"current_status"`
	LastLocation  string `json:"last_location"`
	LastUpdate    time.Time `json:"last_update"`
}

// IPCreateShipment simulates an India Post Speed Post booking.
// In a production environment, this would call a 3rd party aggregator API
// or a government EDI portal if accessible.
func IPCreateShipment(req IndiaPostShipmentRequest) (*IndiaPostShipmentResponse, error) {
	// Validation
	if req.Pincode == "" || len(req.Pincode) != 6 {
		return nil, fmt.Errorf("invalid India Post pincode: %s", req.Pincode)
	}

	log.Printf("[IndiaPost] Booking Speed Post for Order %s (Weight: %dg, Pincode: %s)", 
		req.OrderID, req.WeightGrams, req.Pincode)

	// Simulate API latency
	time.Sleep(200 * time.Millisecond)

	// Generate a standard India Post AWB format: EE123456789IN
	awb := fmt.Sprintf("EE%09dIN", rand.Intn(1000000000))

	return &IndiaPostShipmentResponse{
		AWB:          awb,
		BookingDate:  time.Now(),
		ExpectedDate: time.Now().Add(72 * time.Hour), // Speed Post target: 2-4 days
		Status:       "Item Booked",
	}, nil
}

// IPTrackAWB simulates fetching tracking status from India Post.
func IPTrackAWB(awb string) (*IndiaPostTrackResponse, error) {
	// Mock statuses
	statuses := []string{"Item Booked", "Item Received", "Out for Delivery", "Item Delivered"}
	
	return &IndiaPostTrackResponse{
		AWB:           awb,
		CurrentStatus: statuses[rand.Intn(len(statuses))],
		LastLocation:  "Hub-Delhi-NCR",
		LastUpdate:    time.Now().Add(-2 * time.Hour),
	}, nil
}
