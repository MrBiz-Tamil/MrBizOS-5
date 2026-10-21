package shipping

// shiprocket_client.go — Shiprocket API v1 HTTP client.
//
// Required environment variables:
//   SHIPROCKET_EMAIL    — account email
//   SHIPROCKET_PASSWORD — account password
//
// Token is cached for 24 h (Shiprocket tokens are valid for 24 h).
// All calls are synchronous; callers should wrap in goroutines if needed.
//
// Shiprocket API base: https://apiv2.shiprocket.in/v1/external

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const shiprocketBaseURL = "https://apiv2.shiprocket.in/v1/external"

// ─── Token cache ─────────────────────────────────────────────────────────────

var (
	srTokenMu     sync.Mutex
	srCachedToken string
	srTokenExpiry time.Time
)

// shiprocketToken returns a valid Bearer token, refreshing if needed.
func shiprocketToken() (string, error) {
	srTokenMu.Lock()
	defer srTokenMu.Unlock()

	if srCachedToken != "" && time.Now().Before(srTokenExpiry) {
		return srCachedToken, nil
	}

	email := os.Getenv("SHIPROCKET_EMAIL")
	password := os.Getenv("SHIPROCKET_PASSWORD")
	if email == "" || password == "" {
		return "", fmt.Errorf("SHIPROCKET_EMAIL / SHIPROCKET_PASSWORD not set")
	}

	payload := map[string]string{"email": email, "password": password}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(shiprocketBaseURL+"/auth/login",
		"application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("shiprocket login: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || result.Token == "" {
		return "", fmt.Errorf("shiprocket login bad response: %s", string(raw))
	}

	srCachedToken = result.Token
	srTokenExpiry = time.Now().Add(23 * time.Hour) // tokens valid 24 h; refresh 1 h early
	return srCachedToken, nil
}

// shiprocketDo is a generic authenticated HTTP helper.
func shiprocketDo(method, path string, requestBody interface{}) ([]byte, int, error) {
	token, err := shiprocketToken()
	if err != nil {
		return nil, 0, err
	}

	var bodyReader io.Reader
	if requestBody != nil {
		b, _ := json.Marshal(requestBody)
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, shiprocketBaseURL+path, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("shiprocket %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return raw, resp.StatusCode, nil
}

// ─── Data types ──────────────────────────────────────────────────────────────

// ShiprocketOrderItem is one line in a Shiprocket order.
type ShiprocketOrderItem struct {
	Name         string  `json:"name"`
	SKU          string  `json:"sku"`
	Units        int     `json:"units"`
	SellingPrice float64 `json:"selling_price"`
	HSN          string  `json:"hsn,omitempty"`
}

// ShiprocketCreateOrderRequest mirrors the Shiprocket adhoc create-order body.
type ShiprocketCreateOrderRequest struct {
	OrderID              string                `json:"order_id"`
	OrderDate            string                `json:"order_date"` // "YYYY-MM-DD HH:MM"
	PickupLocation       string                `json:"pickup_location"` // warehouse name in SR panel
	ChannelID            string                `json:"channel_id,omitempty"`
	BillingCustomerName  string                `json:"billing_customer_name"`
	BillingAddress       string                `json:"billing_address"`
	BillingCity          string                `json:"billing_city"`
	BillingPincode       string                `json:"billing_pincode"`
	BillingState         string                `json:"billing_state"`
	BillingCountry       string                `json:"billing_country"`
	BillingEmail         string                `json:"billing_email"`
	BillingPhone         string                `json:"billing_phone"`
	ShippingIsBilling    bool                  `json:"shipping_is_billing"`
	OrderItems           []ShiprocketOrderItem `json:"order_items"`
	PaymentMethod        string                `json:"payment_method"` // "Prepaid" | "COD"
	SubTotal             float64               `json:"sub_total"`
	Length               float64               `json:"length"`  // cm
	Breadth              float64               `json:"breadth"` // cm
	Height               float64               `json:"height"`  // cm
	Weight               float64               `json:"weight"`  // kg
}

// ShiprocketCreateOrderResponse holds the fields we need from the create response.
type ShiprocketCreateOrderResponse struct {
	OrderID    int    `json:"order_id"`
	ShipmentID int    `json:"shipment_id"`
	Status     string `json:"status"`
	AWBCode    string `json:"awb_code"`
}

// ShiprocketTrackResponse holds tracking status for an AWB.
type ShiprocketTrackResponse struct {
	TrackingData struct {
		ShipmentTrackActivities []struct {
			Date     string `json:"date"`
			Activity string `json:"activity"`
			Location string `json:"location"`
		} `json:"shipment_track_activities"`
		ShipmentTrack []struct {
			CurrentStatus string `json:"current_status"`
			AWBCode       string `json:"awb_code"`
			EDD           string `json:"edd"` // estimated delivery date
		} `json:"shipment_track"`
	} `json:"tracking_data"`
}

// ─── Public API ──────────────────────────────────────────────────────────────

// SRCreateOrder creates a shipment order on Shiprocket and returns the
// ShiprocketCreateOrderResponse (which includes the AWB code).
func SRCreateOrder(req ShiprocketCreateOrderRequest) (*ShiprocketCreateOrderResponse, error) {
	raw, status, err := shiprocketDo("POST", "/orders/create/adhoc", req)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("shiprocket create order HTTP %d: %s", status, string(raw))
	}

	var result ShiprocketCreateOrderResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("shiprocket create order parse: %w — raw: %s", err, string(raw))
	}
	log.Printf("[Shiprocket] Order created: SR order_id=%d shipment_id=%d awb=%s",
		result.OrderID, result.ShipmentID, result.AWBCode)
	return &result, nil
}

// SRTrackAWB returns the latest tracking status for the given AWB code.
func SRTrackAWB(awb string) (*ShiprocketTrackResponse, error) {
	raw, status, err := shiprocketDo("GET", "/courier/track/awb/"+awb, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("shiprocket track HTTP %d: %s", status, string(raw))
	}
	var result ShiprocketTrackResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("shiprocket track parse: %w", err)
	}
	return &result, nil
}

// SRCancelOrder cancels an order on Shiprocket by its SR order IDs.
func SRCancelOrder(orderIDs []int) error {
	payload := map[string]interface{}{"ids": orderIDs}
	raw, status, err := shiprocketDo("POST", "/orders/cancel", payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("shiprocket cancel HTTP %d: %s", status, string(raw))
	}
	return nil
}
