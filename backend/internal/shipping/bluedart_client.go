package shipping

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
)

// BlueDartAPI implements the BlueDart/DHL REST API (Enterprise).
const blueDartBaseURL = "https://api.bluedart.com/servlet/RoutingServlet" // Mock/Example URL

type BlueDartCreateRequest struct {
	AwbDetails struct {
		ProductCode string `json:"ProductCode"` // e.g. "A" for Apex
		Pincode     string `json:"Pincode"`
	} `json:"AwbDetails"`
	CustomerDetail struct {
		Name    string `json:"CustomerName"`
		Address string `json:"CustomerAddress1"`
	} `json:"CustomerDetail"`
}

type BlueDartResponse struct {
	AWBNo  string `json:"AWBNo"`
	Status string `json:"Status"`
}

func CreateBlueDartShipment(req BlueDartCreateRequest) (*BlueDartResponse, error) {
	key := os.Getenv("BLUEDART_LICENSE_KEY")
	if key == "" {
		// Simulation for testing without actual license
		return &BlueDartResponse{AWBNo: "BD_SIM_" + req.AwbDetails.Pincode, Status: "Success"}, nil
	}

	b, _ := json.Marshal(req)
	resp, err := http.Post(blueDartBaseURL, "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var res BlueDartResponse
	json.Unmarshal(raw, &res)
	return &res, nil
}
