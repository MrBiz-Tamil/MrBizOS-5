package shipping

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
)

const shadowfaxBaseURL = "https://api.shadowfax.in/v1/orders"

type ShadowfaxCreateRequest struct {
	OrderID    string  `json:"order_number"`
	DestName   string  `json:"customer_name"`
	DestAdd    string  `json:"customer_address"`
	DestPin    string  `json:"customer_pincode"`
	OrderValue float64 `json:"order_value"`
}

type ShadowfaxResponse struct {
	AWB string `json:"awb_number"`
}

func CreateShadowfaxShipment(req ShadowfaxCreateRequest) (*ShadowfaxResponse, error) {
	token := os.Getenv("SHADOWFAX_TOKEN")
	if token == "" {
		return &ShadowfaxResponse{AWB: "SFX_SIM_" + req.OrderID}, nil
	}

	b, _ := json.Marshal(req)
	hReq, _ := http.NewRequest("POST", shadowfaxBaseURL, bytes.NewReader(b))
	hReq.Header.Set("Authorization", "Token "+token)
	hReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(hReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var res ShadowfaxResponse
	json.Unmarshal(raw, &res)
	return &res, nil
}
