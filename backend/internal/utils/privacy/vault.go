package privacy

import (
	"regexp"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	phoneRegex = regexp.MustCompile(`(\+?\d{1,4}[-.\s]?)?(\(?\d{3}\)?[-.\s]?)?\d{3}[-.\s]?\d{4}`)
	heidRegex  = regexp.MustCompile(`HEID-[A-Z0-9]{8,12}`)
)

// MaskPII replaces sensitive information with generic tokens
func MaskPII(text string) string {
	masked := text

	// Replace Emails
	masked = emailRegex.ReplaceAllString(masked, "[EMAIL_REDACTED]")

	// Replace HEIDs (Apply before phones to avoid partial matches)
	masked = heidRegex.ReplaceAllString(masked, "[HEID_REDACTED]")

	// Replace Phones
	masked = phoneRegex.ReplaceAllString(masked, "[PHONE_REDACTED]")

	return masked
}

// UnmaskResponse re-inserts known user details into the bot's response
// This is done locally to ensure Devi sounds personal without sending PII to the AI
func UnmaskResponse(text, userName string) string {
	res := text
	// Devi often uses placeholders if prompted in the system instruction
	res = strings.ReplaceAll(res, "{{USER_NAME}}", userName)
	res = strings.ReplaceAll(res, "[PATIENT_NAME]", userName)
	return res
}
