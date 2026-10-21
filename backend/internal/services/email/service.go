// Package email provides a Resend-backed transactional email service.
//
// Required env vars:
//
//	RESEND_API_KEY    — Resend secret key (starts with "re_")
//	                    Get one at: https://resend.com/api-keys
//	RESEND_FROM_EMAIL — verified sender address (e.g. noreply@mrbiz.in)
//	                    The domain MUST be verified at: https://resend.com/domains
//	RESEND_FROM_NAME  — display name shown in the inbox (default: Dr.Ayya)
//
// Quick-start:
//  1. Create a free account at resend.com
//  2. Add your domain (Settings → Domains → Add Domain)
//  3. Add the DNS records shown — takes 1–10 min to verify
//  4. Create an API key (API Keys → Create API Key)
//  5. Set RESEND_API_KEY + RESEND_FROM_EMAIL in .env and restart
package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	resendAPI        = "https://api.resend.com/emails"
	resendDomainsAPI = "https://api.resend.com/domains"
	httpTimeout      = 10 * time.Second
)

// ─── Config helpers ───────────────────────────────────────────────────────────

// Config holds the resolved email configuration.
type Config struct {
	APIKey    string
	FromEmail string
	FromName  string
	FromAddr  string // "Name <email>" combined
}

// LoadConfig reads env vars and returns a Config.
// Defaults RESEND_FROM_EMAIL → noreply@mrbiz.in
// Defaults RESEND_FROM_NAME  → Dr.Ayya
func LoadConfig() Config {
	key := os.Getenv("RESEND_API_KEY")
	addr := os.Getenv("RESEND_FROM_EMAIL")
	if addr == "" {
		addr = "noreply@mrbiz.in"
	}
	name := os.Getenv("RESEND_FROM_NAME")
	if name == "" {
		name = "Dr.Ayya"
	}
	return Config{
		APIKey:    key,
		FromEmail: addr,
		FromName:  name,
		FromAddr:  fmt.Sprintf("%s <%s>", name, addr),
	}
}

// ValidateConfig checks env vars and logs actionable hints.
// Returns an error if RESEND_API_KEY is absent or malformed.
// Does NOT make a live network call — use VerifyLive for that.
func ValidateConfig() error {
	cfg := LoadConfig()

	if cfg.APIKey == "" {
		return fmt.Errorf(
			"RESEND_API_KEY is not set.\n\n" +
				"  Fix:\n" +
				"  1. Go to https://resend.com/api-keys and create a key\n" +
				"  2. Add to .env:  RESEND_API_KEY=re_xxxxxxxxxxxx\n" +
				"  3. Restart the server\n",
		)
	}
	if !strings.HasPrefix(cfg.APIKey, "re_") || len(cfg.APIKey) < 10 {
		return fmt.Errorf(
			"RESEND_API_KEY looks invalid (got %q — should start with 're_').\n" +
				"  Regenerate at https://resend.com/api-keys",
			MaskKey(cfg.APIKey),
		)
	}

	log.Printf("[email] Sender: %s  |  key: %s", cfg.FromAddr, MaskKey(cfg.APIKey))

	// Warn if using the default domain — it may not be verified yet.
	if os.Getenv("RESEND_FROM_EMAIL") == "" {
		log.Printf("[email] WARNING: RESEND_FROM_EMAIL not set — using default %q", cfg.FromEmail)
		log.Printf("[email]   Ensure %q is verified at https://resend.com/domains", cfg.FromEmail)
	}

	return nil
}

// VerifyLive performs a live check against the Resend API:
//  1. Validates the API key by calling GET /domains
//  2. Checks whether the configured sender domain is in the verified list
//
// Returns an error with an actionable message if anything is wrong.
// Call this once at startup after ValidateConfig().
func VerifyLive() error {
	cfg := LoadConfig()
	if cfg.APIKey == "" {
		return fmt.Errorf("RESEND_API_KEY not set — skipping live verification")
	}

	req, err := http.NewRequest("GET", resendDomainsAPI, nil)
	if err != nil {
		return fmt.Errorf("failed to build Resend domains request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Resend API unreachable: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case 200:
		// Key is valid — now check domain verification status
		return checkDomainVerified(cfg.FromEmail, raw)
	case 401:
		return fmt.Errorf(
			"Resend API key is INVALID or REVOKED (HTTP 401).\n" +
				"  Fix:\n" +
				"  1. Go to https://resend.com/api-keys\n" +
				"  2. Regenerate (or create) an API key\n" +
				"  3. Update RESEND_API_KEY in .env and restart",
		)
	case 403:
		return fmt.Errorf("Resend API key does not have permission to list domains (HTTP 403). " +
			"Recreate the key with 'Full Access' permissions.")
	default:
		return fmt.Errorf("Resend domains API returned HTTP %d: %s", resp.StatusCode, string(raw))
	}
}

// checkDomainVerified inspects the Resend domains list response and returns
// an error if the sender domain is missing or not yet verified.
func checkDomainVerified(fromEmail string, body []byte) error {
	// Extract the domain part from the from address
	senderDomain := ""
	if parts := strings.Split(fromEmail, "@"); len(parts) == 2 {
		senderDomain = strings.ToLower(parts[1])
	}
	if senderDomain == "" {
		return fmt.Errorf("cannot parse domain from RESEND_FROM_EMAIL=%q", fromEmail)
	}

	// Resend returns { "data": [ { "name": "...", "status": "verified"|"pending"|"failed" } ] }
	var apiResp struct {
		Data []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		// Cannot parse response — key works, domain check inconclusive
		log.Printf("[email] WARNING: could not parse Resend domains response: %v", err)
		return nil
	}

	for _, d := range apiResp.Data {
		if strings.EqualFold(d.Name, senderDomain) {
			switch d.Status {
			case "verified":
				log.Printf("[email] ✅ Domain %q is verified in Resend — emails will send.", senderDomain)
				return nil
			case "pending":
				return fmt.Errorf(
					"domain %q is PENDING verification in Resend (status: pending).\n"+
						"  Fix:\n"+
						"  1. Go to https://resend.com/domains → click on %q\n"+
						"  2. Add the DNS records shown (TXT + MX or CNAME depending on your plan)\n"+
						"  3. Wait 1–10 minutes for DNS propagation, then click 'Verify'\n"+
						"  Until verified, ALL emails will fail with HTTP 422.",
					senderDomain, senderDomain,
				)
			default:
				return fmt.Errorf(
					"domain %q has status %q in Resend.\n"+
						"  Remove and re-add the domain at https://resend.com/domains",
					senderDomain, d.Status,
				)
			}
		}
	}

	// Domain not found in Resend at all
	return fmt.Errorf(
		"domain %q is NOT added to your Resend account.\n"+
			"  Fix:\n"+
			"  1. Go to https://resend.com/domains → 'Add Domain'\n"+
			"  2. Enter: %s\n"+
			"  3. Add the DNS records shown (takes 1–10 min to verify)\n"+
			"  4. Set RESEND_FROM_EMAIL=noreply@%s in .env\n"+
			"  Until verified, ALL emails will fail with HTTP 422.",
		senderDomain, senderDomain, senderDomain,
	)
}

// ─── Core types ───────────────────────────────────────────────────────────────

// Message is a single outbound email.
type Message struct {
	From        string       `json:"from"`
	To          []string     `json:"to"`
	Subject     string       `json:"subject"`
	HTML        string       `json:"html"`
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment is an inline file attachment.
type Attachment struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"`      // base64-encoded bytes
	ContentType string `json:"content_type"` // e.g. "application/pdf"
}

// Result is the Resend API success response.
type Result struct {
	ID string `json:"id"`
}

// ─── Send ─────────────────────────────────────────────────────────────────────

// Send delivers a single email via the Resend API.
// The From address defaults to the configured sender if not explicitly set.
func Send(msg Message) (*Result, error) {
	cfg := LoadConfig()
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("email: RESEND_API_KEY not set — configure it in .env")
	}
	if msg.From == "" {
		msg.From = cfg.FromAddr
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("email: marshal failed: %w", err)
	}

	req, err := http.NewRequest("POST", resendAPI, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("email: build request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("email: network error: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, resendError(resp.StatusCode, raw, cfg)
	}

	var result Result
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("email: unexpected Resend response: %s", string(raw))
	}
	return &result, nil
}

// SendSimple sends a plain HTML email. The most common call path.
func SendSimple(to, subject, html string) error {
	_, err := Send(Message{
		To:      []string{to},
		Subject: subject,
		HTML:    html,
	})
	return err
}

// SendWithAttachment sends an HTML email with one or more file attachments.
func SendWithAttachment(to, subject, html string, attachments []Attachment) error {
	_, err := Send(Message{
		To:          []string{to},
		Subject:     subject,
		HTML:        html,
		Attachments: attachments,
	})
	return err
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// resendError converts an HTTP error response into an actionable Go error.
func resendError(statusCode int, body []byte, cfg Config) error {
	// Try to extract Resend's own error message
	var resendErr struct {
		Message string `json:"message"`
		Name    string `json:"name"`
	}
	_ = json.Unmarshal(body, &resendErr)

	detail := resendErr.Message
	if detail == "" {
		detail = string(body)
	}

	switch statusCode {
	case 401:
		return fmt.Errorf(
			"Resend rejected API key (HTTP 401): %s\n"+
				"  Fix: regenerate RESEND_API_KEY at https://resend.com/api-keys",
			detail,
		)
	case 403:
		return fmt.Errorf(
			"Resend access denied (HTTP 403): %s\n"+
				"  Fix: verify domain %q at https://resend.com/domains",
			detail, DomainOf(cfg.FromEmail),
		)
	case 422:
		return fmt.Errorf(
			"Resend rejected the email request (HTTP 422): %s\n"+
				"  Most common cause: domain %q is not verified in Resend.\n"+
				"  Fix:\n"+
				"  1. Go to https://resend.com/domains → Add Domain → %s\n"+
				"  2. Add the DNS records shown and wait for verification\n"+
				"  3. Ensure RESEND_FROM_EMAIL=%s matches a verified domain",
			detail, DomainOf(cfg.FromEmail), DomainOf(cfg.FromEmail), cfg.FromEmail,
		)
	case 429:
		return fmt.Errorf("Resend rate limit hit (HTTP 429) — retry in a few seconds: %s", detail)
	default:
		return fmt.Errorf("Resend HTTP %d: %s", statusCode, detail)
	}
}

// MaskKey returns the first 8 characters of an API key followed by "…" for safe logging.
func MaskKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:8] + "…"
}

// DomainOf extracts the domain part from an email address.
func DomainOf(email string) string {
	if parts := strings.Split(email, "@"); len(parts) == 2 {
		return parts[1]
	}
	return email
}
