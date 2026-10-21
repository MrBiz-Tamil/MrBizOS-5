package notification

// FCM HTTP v1 API — uses a Google Service Account JSON to obtain an OAuth2
// access token, then calls the FCM messages:send endpoint.
// No external SDK required — only golang-jwt/jwt/v5 (already in go.mod).

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// serviceAccountJSON is the minimal subset we need from the SA JSON file.
type serviceAccountJSON struct {
	ProjectID    string `json:"project_id"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ClientEmail  string `json:"client_email"`
	TokenURI     string `json:"token_uri"`
}

// cachedToken caches the FCM OAuth2 access token so we don't re-fetch on
// every notification. Token is valid for ~1h; we refresh 5 min early.
var (
	tokenMu      sync.Mutex
	cachedAccess string
	tokenExpiry  time.Time
)

func loadServiceAccount() (*serviceAccountJSON, error) {
	raw := os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON")
	if raw == "" {
		path := os.Getenv("FIREBASE_SERVICE_ACCOUNT_PATH")
		if path == "" {
			return nil, fmt.Errorf("FCM: FIREBASE_SERVICE_ACCOUNT_JSON or FIREBASE_SERVICE_ACCOUNT_PATH not set")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("FCM: read service account file: %w", err)
		}
		raw = string(data)
	}

	// Remove escaped newlines that sometimes appear when pasting JSON into env
	raw = strings.ReplaceAll(raw, `\n`, "\n")

	var sa serviceAccountJSON
	if err := json.Unmarshal([]byte(raw), &sa); err != nil {
		return nil, fmt.Errorf("FCM: parse service account: %w", err)
	}
	return &sa, nil
}

func getAccessToken() (string, error) {
	tokenMu.Lock()
	defer tokenMu.Unlock()

	if cachedAccess != "" && time.Now().Before(tokenExpiry) {
		return cachedAccess, nil
	}

	sa, err := loadServiceAccount()
	if err != nil {
		return "", err
	}

	// Decode PEM private key
	block, _ := pem.Decode([]byte(sa.PrivateKey))
	if block == nil {
		return "", fmt.Errorf("FCM: failed to decode PEM private key")
	}
	var privateKey *rsa.PrivateKey
	switch block.Type {
	case "RSA PRIVATE KEY":
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, e := x509.ParsePKCS8PrivateKey(block.Bytes)
		if e != nil {
			return "", fmt.Errorf("FCM: parse PKCS8: %w", e)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return "", fmt.Errorf("FCM: not an RSA key")
		}
	default:
		return "", fmt.Errorf("FCM: unsupported key type: %s", block.Type)
	}
	if err != nil {
		return "", fmt.Errorf("FCM: parse private key: %w", err)
	}

	tokenURI := sa.TokenURI
	if tokenURI == "" {
		tokenURI = "https://oauth2.googleapis.com/token"
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   sa.ClientEmail,
		"sub":   sa.ClientEmail,
		"aud":   tokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
		"scope": "https://www.googleapis.com/auth/firebase.messaging",
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	if sa.PrivateKeyID != "" {
		tok.Header["kid"] = sa.PrivateKeyID
	}
	signed, err := tok.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("FCM: sign JWT: %w", err)
	}

	// Exchange JWT for access token
	resp, err := http.PostForm(tokenURI, url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {signed},
	})
	if err != nil {
		return "", fmt.Errorf("FCM: fetch access token: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil || tokenResp.AccessToken == "" {
		return "", fmt.Errorf("FCM: bad token response: %s", string(body))
	}

	cachedAccess = tokenResp.AccessToken
	tokenExpiry = now.Add(time.Duration(tokenResp.ExpiresIn-300) * time.Second)
	return cachedAccess, nil
}

// -------------------------------------------------------------------
// Public API
// -------------------------------------------------------------------

type FCMMessage struct {
	Title string
	Body  string
	Data  map[string]string
	// ImageURL is optional (shown in notification tray on Android)
	ImageURL string
}

// SendToToken sends a push notification to a single FCM device token.
// Returns nil if FCM is not configured (non-fatal — token might not be set).
func SendToToken(deviceToken string, msg FCMMessage) error {
	if deviceToken == "" {
		return nil
	}

	sa, err := loadServiceAccount()
	if err != nil {
		log.Printf("[FCM] Not configured, skipping: %v", err)
		return nil // non-fatal if FCM not set up
	}

	accessToken, err := getAccessToken()
	if err != nil {
		return fmt.Errorf("FCM: get access token: %w", err)
	}

	notification := map[string]interface{}{
		"title": msg.Title,
		"body":  msg.Body,
	}
	if msg.ImageURL != "" {
		notification["image"] = msg.ImageURL
	}

	payload := map[string]interface{}{
		"message": map[string]interface{}{
			"token":        deviceToken,
			"notification": notification,
			"data":         msg.Data,
			"android": map[string]interface{}{
				"priority": "high",
				"notification": map[string]interface{}{
					"sound":        "default",
					"click_action": "FLUTTER_NOTIFICATION_CLICK",
				},
			},
			"apns": map[string]interface{}{
				"payload": map[string]interface{}{
					"aps": map[string]interface{}{
						"sound": "default",
					},
				},
			},
		},
	}

	b, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", sa.ProjectID)

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("FCM: send request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("FCM: send error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// -------------------------------------------------------------------
// Domain-specific helpers — call these from handlers
// -------------------------------------------------------------------

// NotifyAppointmentBooked sends a confirmation push to the patient.
func NotifyAppointmentBooked(deviceToken string, doctorName string, startTime time.Time) {
	go func() {
		if err := SendToToken(deviceToken, FCMMessage{
			Title: "Appointment Confirmed",
			Body:  fmt.Sprintf("Your consultation with Dr. %s is booked for %s", doctorName, startTime.Format("Jan 2, 3:04 PM")),
			Data:  map[string]string{"type": "appointment_booked"},
		}); err != nil {
			log.Printf("[FCM] NotifyAppointmentBooked: %v", err)
		}
	}()
}

// NotifyCallStarted sends a push to the patient when doctor starts the LiveKit session.
func NotifyCallStarted(deviceToken string, doctorName string, appointmentID uint) {
	go func() {
		if err := SendToToken(deviceToken, FCMMessage{
			Title: "Doctor is ready!",
			Body:  fmt.Sprintf("Dr. %s has started your consultation. Join now.", doctorName),
			Data: map[string]string{
				"type":           "call_started",
				"appointment_id": fmt.Sprintf("%d", appointmentID),
			},
		}); err != nil {
			log.Printf("[FCM] NotifyCallStarted: %v", err)
		}
	}()
}

// NotifyPaymentSuccess sends a push to patient after wallet payment.
func NotifyPaymentSuccess(deviceToken string, amount float64, appointmentID uint) {
	go func() {
		if err := SendToToken(deviceToken, FCMMessage{
			Title: "Payment Successful",
			Body:  fmt.Sprintf("₹%.0f paid. Your appointment is confirmed.", amount),
			Data: map[string]string{
				"type":           "payment_success",
				"appointment_id": fmt.Sprintf("%d", appointmentID),
			},
		}); err != nil {
			log.Printf("[FCM] NotifyPaymentSuccess: %v", err)
		}
	}()
}

// NotifyAppointmentReminder sends a reminder push 1hr before.
func NotifyAppointmentReminder(deviceToken string, doctorName string, startTime time.Time, appointmentID uint) {
	go func() {
		if err := SendToToken(deviceToken, FCMMessage{
			Title: "Appointment in 1 Hour",
			Body:  fmt.Sprintf("Your consultation with Dr. %s starts at %s", doctorName, startTime.Format("3:04 PM")),
			Data: map[string]string{
				"type":           "appointment_reminder",
				"appointment_id": fmt.Sprintf("%d", appointmentID),
			},
		}); err != nil {
			log.Printf("[FCM] NotifyAppointmentReminder: %v", err)
		}
	}()
}
