package abdm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// Client manages ABDM (Ayushman Bharat Digital Mission) API interactions.
// Implements the NHA sandbox/production API for ABHA (Ayushman Bharat Health Account) operations.
type Client struct {
	baseURL    string
	clientID   string
	clientSecret string
	token      string
	tokenExpiry time.Time
	mu         sync.Mutex
	httpClient *http.Client
}

// TokenResponse is returned by the ABDM auth endpoint
type TokenResponse struct {
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
	ExpiresIn   int    `json:"expiresIn"` // seconds
}

// AadhaarOTPRequest initiates ABHA creation via Aadhaar OTP
type AadhaarOTPRequest struct {
	Aadhaar string `json:"aadhaar"`
}

// AadhaarOTPResponse contains the transaction ID for OTP verification
type AadhaarOTPResponse struct {
	TxnID   string `json:"txnId"`
	Message string `json:"message"`
}

// ABHAProfile represents a user's ABHA health account
type ABHAProfile struct {
	HealthID    string `json:"healthId"`    // e.g. user@abdm
	HealthIDNumber string `json:"healthIdNumber"` // 14-digit ABHA number
	Name        string `json:"name"`
	DOB         string `json:"dayOfBirth"`
	Gender      string `json:"gender"`
	Mobile      string `json:"mobile"`
	Email       string `json:"email"`
	StateCode   string `json:"stateCode"`
	DistrictCode string `json:"districtCode"`
}

// NewClient creates a new ABDM API client
func NewClient() *Client {
	baseURL := os.Getenv("ABDM_BASE_URL")
	if baseURL == "" {
		baseURL = "https://healthidsbx.abdm.gov.in/api" // sandbox
	}
	return &Client{
		baseURL:      baseURL,
		clientID:     os.Getenv("ABDM_CLIENT_ID"),
		clientSecret: os.Getenv("ABDM_CLIENT_SECRET"),
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// getToken returns a valid access token, refreshing if expired
func (c *Client) getToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return c.token, nil
	}

	payload := map[string]string{
		"clientId":     c.clientID,
		"clientSecret": c.clientSecret,
		"grantType":    "client_credentials",
	}

	body, _ := json.Marshal(payload)
	resp, err := c.httpClient.Post(
		c.baseURL+"/v2/auth/fetch/token",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return "", fmt.Errorf("ABDM token fetch failed: %w", err)
	}
	defer resp.Body.Close()

	var tr TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("ABDM token decode failed: %w", err)
	}

	c.token = tr.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tr.ExpiresIn-60) * time.Second)
	return c.token, nil
}

// doRequest performs an authenticated request to ABDM
func (c *Client) doRequest(method, path string, payload interface{}) ([]byte, int, error) {
	token, err := c.getToken()
	if err != nil {
		return nil, 0, err
	}

	var bodyReader io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		bodyReader = bytes.NewBuffer(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CM-ID", "sbx") // sbx for sandbox, abdm for production

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

// SendAadhaarOTP initiates ABHA creation by sending OTP to Aadhaar-linked mobile
func (c *Client) SendAadhaarOTP(aadhaar string) (*AadhaarOTPResponse, error) {
	body, status, err := c.doRequest("POST", "/v2/registration/aadhaar/generateOtp", map[string]string{"aadhaar": aadhaar})
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("ABDM OTP request failed with status %d: %s", status, string(body))
	}
	var resp AadhaarOTPResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// VerifyAadhaarOTP verifies the OTP and creates/retrieves ABHA profile
func (c *Client) VerifyAadhaarOTP(txnID, otp string) (*ABHAProfile, error) {
	payload := map[string]string{"txnId": txnID, "otp": otp}
	body, status, err := c.doRequest("POST", "/v2/registration/aadhaar/verifyOTP", payload)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("ABDM OTP verification failed with status %d: %s", status, string(body))
	}
	var profile ABHAProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

// GetABHAProfile retrieves full ABHA profile for a given health ID
func (c *Client) GetABHAProfile(healthID string) (*ABHAProfile, error) {
	body, status, err := c.doRequest("GET", "/v1/account/profile", nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("ABDM profile fetch failed: status %d", status)
	}
	_ = healthID // profile returned for authenticated session
	var profile ABHAProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

// SendMobileOTP sends an OTP to the given mobile number for ABHA login
func (c *Client) SendMobileOTP(mobile string) (*AadhaarOTPResponse, error) {
	payload := map[string]string{"mobile": mobile}
	body, status, err := c.doRequest("POST", "/v2/registration/mobile/generateOtp", payload)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("ABDM mobile OTP failed: status %d", status)
	}
	var resp AadhaarOTPResponse
	json.Unmarshal(body, &resp)
	return &resp, nil
}

// LinkHealthFacility links a health facility (hospital/clinic) with ABDM
func (c *Client) LinkHealthFacility(facilityID, facilityName string) error {
	payload := map[string]string{
		"facilityId":   facilityID,
		"facilityName": facilityName,
		"hfrId":        facilityID,
	}
	_, status, err := c.doRequest("POST", "/v1/facility/link", payload)
	if err != nil {
		return err
	}
	if status != 200 && status != 201 {
		return fmt.Errorf("facility link failed with status %d", status)
	}
	return nil
}
