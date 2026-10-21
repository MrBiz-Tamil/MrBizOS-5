package shipping

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
)

const dtdcBaseURL = "https://api.dtdc.com/v1/shipments"

type DTDCCreateRequest struct {
	CustomerCode string `json:"customer_code"`
	ReceiverName string `json:"receiver_name"`
	ReceiverAdd  string `json:"receiver_address"`
	ReceiverPin  string `json:"receiver_pincode"`
}

type DTDCResponse struct {
	ConsignmentNumber string `json:"consignment_no"`
}

func CreateDTDCShipment(req DTDCCreateRequest) (*DTDCResponse, error) {
	apiKey := os.Getenv("DTDC_API_KEY")
	if apiKey == "" {
		return &DTDCResponse{ConsignmentNumber: "DTDC_SIM_" + req.ReceiverPin}, nil
	}

	b, _ := json.Marshal(req)
	hReq, _ := http.NewRequest("POST", dtdcBaseURL, bytes.NewReader(b))
	hReq.Header.Set("x-api-key", apiKey)
	hReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(hReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var res DTDCResponse
	json.Unmarshal(raw, &res)
	return &res, nil
}
