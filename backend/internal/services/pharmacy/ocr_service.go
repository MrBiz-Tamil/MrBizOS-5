package pharmacy

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

// PrescriptionOCRResult holds medicines extracted from a scanned prescription
type PrescriptionOCRResult struct {
	RawText     string            `json:"raw_text"`
	Medicines   []ExtractedMed    `json:"medicines"`
	DoctorName  string            `json:"doctor_name"`
	PatientName string            `json:"patient_name"`
	Date        string            `json:"date"`
	Confidence  float64           `json:"confidence"` // 0–1
	Provider    string            `json:"provider"`   // "google_vision" | "aws_textract" | "stub"
}

// ExtractedMed is a medicine parsed from an OCR result
type ExtractedMed struct {
	Name     string  `json:"name"`
	Dosage   string  `json:"dosage"`
	Frequency string `json:"frequency"`
	Duration string  `json:"duration"`
	Quantity int     `json:"quantity"`
	Confidence float64 `json:"confidence"`
}

// OCRService handles prescription image OCR
type OCRService struct {
	provider string
	apiKey   string
	endpoint string
}

// NewOCRService creates a new OCR service instance
// Uses Google Cloud Vision if GOOGLE_VISION_API_KEY is set,
// AWS Textract if AWS credentials present, otherwise stub mode.
func NewOCRService() *OCRService {
	if key := os.Getenv("GOOGLE_VISION_API_KEY"); key != "" {
		return &OCRService{
			provider: "google_vision",
			apiKey:   key,
			endpoint: "https://vision.googleapis.com/v1/images:annotate",
		}
	}
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" {
		return &OCRService{
			provider: "aws_textract",
		}
	}
	return &OCRService{provider: "stub"}
}

// ExtractFromBase64 processes a base64-encoded image and returns extracted prescription data
func (s *OCRService) ExtractFromBase64(imageBase64, mimeType string) (*PrescriptionOCRResult, error) {
	switch s.provider {
	case "google_vision":
		return s.extractViaGoogleVision(imageBase64, mimeType)
	case "aws_textract":
		return s.extractViaAWSTextract(imageBase64)
	default:
		return s.stubExtract(), nil
	}
}

// ExtractFromURL downloads an image from a URL and runs OCR
func (s *OCRService) ExtractFromURL(imageURL string) (*PrescriptionOCRResult, error) {
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	imgBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}

	imgBase64 := base64.StdEncoding.EncodeToString(imgBytes)
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	return s.ExtractFromBase64(imgBase64, contentType)
}

// extractViaGoogleVision calls Google Cloud Vision API
func (s *OCRService) extractViaGoogleVision(imageBase64, mimeType string) (*PrescriptionOCRResult, error) {
	payload := map[string]interface{}{
		"requests": []map[string]interface{}{
			{
				"image": map[string]string{
					"content": imageBase64,
				},
				"features": []map[string]interface{}{
					{"type": "DOCUMENT_TEXT_DETECTION", "maxResults": 1},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)
	url := s.endpoint + "?key=" + s.apiKey

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("Google Vision API error: %w", err)
	}
	defer resp.Body.Close()

	var visionResp struct {
		Responses []struct {
			FullTextAnnotation struct {
				Text string `json:"text"`
			} `json:"fullTextAnnotation"`
		} `json:"responses"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&visionResp); err != nil {
		return nil, fmt.Errorf("Vision API decode error: %w", err)
	}

	var rawText string
	if len(visionResp.Responses) > 0 {
		rawText = visionResp.Responses[0].FullTextAnnotation.Text
	}

	return s.parsePrescriptionText(rawText, "google_vision"), nil
}

// extractViaAWSTextract calls AWS Textract (simplified — production needs proper SDK)
func (s *OCRService) extractViaAWSTextract(imageBase64 string) (*PrescriptionOCRResult, error) {
	// In production: use github.com/aws/aws-sdk-go-v2/service/textract
	// For now, fall through to stub
	return s.stubExtract(), nil
}

// parsePrescriptionText parses raw OCR text to extract medicine names, dosages, etc.
func (s *OCRService) parsePrescriptionText(rawText, provider string) *PrescriptionOCRResult {
	result := &PrescriptionOCRResult{
		RawText:    rawText,
		Provider:   provider,
		Medicines:  []ExtractedMed{},
		Confidence: 0.7,
	}

	// Extract doctor name (typically "Dr. <Name>" pattern)
	lines := strings.Split(rawText, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Dr.") || strings.HasPrefix(trimmed, "DR.") {
			result.DoctorName = trimmed
			break
		}
	}

	// Common medicine patterns: "Tab.", "Cap.", "Syp.", "Inj.", "Oint."
	medicineKeywords := []string{"Tab.", "Cap.", "Syp.", "Inj.", "Oint.", "Drops", "mg", "ml", "mcg"}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		for _, kw := range medicineKeywords {
			if strings.Contains(trimmed, kw) && len(trimmed) > 4 {
				med := ExtractedMed{
					Name:       trimmed,
					Confidence: 0.75,
				}
				// Extract dosage from patterns like "500mg", "10ml"
				if strings.Contains(trimmed, "mg") {
					parts := strings.Fields(trimmed)
					for _, p := range parts {
						if strings.HasSuffix(p, "mg") || strings.HasSuffix(p, "ml") {
							med.Dosage = p
						}
					}
				}
				// Extract frequency patterns: OD, BD, TDS, QID
				freqMap := map[string]string{
					"OD":  "Once daily",
					"BD":  "Twice daily",
					"TDS": "Three times daily",
					"QID": "Four times daily",
					"SOS": "As needed",
					"HS":  "At bedtime",
				}
				for abbr, full := range freqMap {
					if strings.Contains(strings.ToUpper(trimmed), abbr) {
						med.Frequency = full
						break
					}
				}
				result.Medicines = append(result.Medicines, med)
				break
			}
		}
	}

	return result
}

// stubExtract returns a mock OCR result for development/testing
func (s *OCRService) stubExtract() *PrescriptionOCRResult {
	return &PrescriptionOCRResult{
		RawText:     "STUB OCR OUTPUT — Configure GOOGLE_VISION_API_KEY or AWS credentials for real OCR",
		DoctorName:  "Dr. Demo Doctor",
		PatientName: "Demo Patient",
		Date:        time.Now().Format("2006-01-02"),
		Confidence:  0.0,
		Provider:    "stub",
		Medicines: []ExtractedMed{
			{Name: "Tab. Paracetamol 500mg", Dosage: "500mg", Frequency: "Three times daily", Duration: "5 days", Quantity: 15, Confidence: 0.0},
			{Name: "Tab. Amoxicillin 250mg", Dosage: "250mg", Frequency: "Twice daily", Duration: "7 days", Quantity: 14, Confidence: 0.0},
		},
	}
}
