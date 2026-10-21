package shipping

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const delhiveryBaseURL = "https://track.delhivery.com"

// DelhiveryCreateRequest represents the CMU (Client Manifest Upload) payload.
type DelhiveryCreateRequest struct {
	Shipments []DelhiveryShipment `json:"shipments"`
	Pickup    DelhiveryPickup     `json:"pickup_location"`
}

type DelhiveryShipment struct {
	Name            string  `json:"name"`
	Add             string  `json:"add"`
	Pin             string  `json:"pin"`
	Phone           string  `json:"phone"`
	PaymentMode     string  `json:"payment_mode"` // "Prepaid" or "COD"
	Order           string  `json:"order"`
	TotalAmount     float64 `json:"total_amount"`
	Weight          float64 `json:"weight"` // grams
	Quantity        string  `json:"quantity"`
	ShipmentWidth   float64 `json:"shipment_width"`
	ShipmentHeight  float64 `json:"shipment_height"`
}

type DelhiveryPickup struct {
	Name string `json:"name"`
}

// DelhiveryResponse is the response from Delhivery CMU API.
type DelhiveryResponse struct {
	Success     bool   `json:"success"`
	Packages    []struct {
		Waybill string `json:"waybill"`
		Status  string `json:"status"`
	} `json:"packages"`
}

// DelhiveryDo handles the Delhivery API authentication (via Token in header).
func DelhiveryDo(method, path string, requestBody interface{}) ([]byte, int, error) {
	token := os.Getenv("DELHIVERY_TOKEN")
	if token == "" {
		return nil, 0, fmt.Errorf("DELHIVERY_TOKEN not set")
	}

	var bodyReader io.Reader
	if requestBody != nil {
		b, _ := json.Marshal(requestBody)
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, delhiveryBaseURL+path, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+token)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return raw, resp.StatusCode, nil
}

func CreateDelhiveryShipment(req DelhiveryCreateRequest) (*DelhiveryResponse, error) {
	// Delhivery expects form data for CMU sometimes, but let's assume JSON for modern integration
	raw, status, err := DelhiveryDo("POST", "/api/cmu/create.json", req)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("delhivery error %d: %s", status, string(raw))
	}

	var result DelhiveryResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	log.Printf("[Delhivery] Shipment created, packages: %d", len(result.Packages))
	return &result, nil
}
