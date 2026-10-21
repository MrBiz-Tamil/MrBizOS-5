// Package ocr extracts prescription text from an image URL. The provider is
// pluggable: a real Google Cloud Vision call when GOOGLE_VISION_API_KEY is set,
// or a deterministic mock for dev/tests when unconfigured (same pattern as the
// Stripe service). Downstream code never branches on the mode — it just calls
// ExtractText.
package ocr

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const visionEndpoint = "https://vision.googleapis.com/v1/images:annotate"

func apiKey() string { return os.Getenv("GOOGLE_VISION_API_KEY") }

func mockMode() bool { return apiKey() == "" }

// mockText is what dev returns. Three realistic prescription lines so the
// downstream parser has something to match against without hitting a network.
const mockText = `Dr. R. Krishnan, MBBS
Apollo Clinic, Chennai

Rx
1. Metformin 500mg — 1 tab twice daily — 30 days
2. Atorvastatin 10mg — 1 tab at bedtime — 30 days
3. Amlodipine 5mg — 1 tab once daily — 30 days

Signature: Dr. R. Krishnan`

// ExtractText returns the raw text on a prescription image. In mock mode it
// returns the canned mockText. Otherwise it calls Google Cloud Vision's
// DOCUMENT_TEXT_DETECTION feature with the image bytes fetched from imageURL.
func ExtractText(imageURL string) (string, error) {
	if mockMode() {
		return mockText, nil
	}

	// Fetch the image bytes (the Vision API accepts base64-encoded content).
	imgResp, err := (&http.Client{Timeout: 10 * time.Second}).Get(imageURL)
	if err != nil {
		return "", fmt.Errorf("ocr: fetch image: %w", err)
	}
	defer imgResp.Body.Close()
	raw, err := io.ReadAll(imgResp.Body)
	if err != nil {
		return "", fmt.Errorf("ocr: read image: %w", err)
	}

	body := map[string]interface{}{
		"requests": []map[string]interface{}{{
			"image":    map[string]string{"content": base64.StdEncoding.EncodeToString(raw)},
			"features": []map[string]interface{}{{"type": "DOCUMENT_TEXT_DETECTION"}},
		}},
	}
	bs, _ := json.Marshal(body)

	url := visionEndpoint + "?key=" + apiKey()
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Post(url, "application/json", bytes.NewReader(bs))
	if err != nil {
		return "", fmt.Errorf("ocr: vision call: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("ocr: vision API %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed struct {
		Responses []struct {
			FullTextAnnotation struct {
				Text string `json:"text"`
			} `json:"fullTextAnnotation"`
		} `json:"responses"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("ocr: bad response: %w", err)
	}
	if len(parsed.Responses) == 0 {
		return "", nil
	}
	return strings.TrimSpace(parsed.Responses[0].FullTextAnnotation.Text), nil
}
