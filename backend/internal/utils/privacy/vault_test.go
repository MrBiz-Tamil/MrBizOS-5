package privacy

import (
	"testing"
)

func TestMaskPII(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "My email is john.doe@example.com and phone is +1-202-555-0123. My HEID is HEID-ABC123456789.",
			expected: "My email is [EMAIL_REDACTED] and phone is [PHONE_REDACTED]. My HEID is [HEID_REDACTED].",
		},
		{
			input:    "No PII here.",
			expected: "No PII here.",
		},
	}

	for _, tt := range tests {
		result := MaskPII(tt.input)
		if result != tt.expected {
			t.Errorf("MaskPII(%q) = %q; want %q", tt.input, result, tt.expected)
		}
	}
}

func TestUnmaskResponse(t *testing.T) {
	tests := []struct {
		input    string
		name     string
		expected string
	}{
		{
			input:    "Hello {{USER_NAME}}, how can I help you?",
			name:     "John",
			expected: "Hello John, how can I help you?",
		},
		{
			input:    "Sure [PATIENT_NAME], booking your appointment.",
			name:     "Alice",
			expected: "Sure Alice, booking your appointment.",
		},
	}

	for _, tt := range tests {
		result := UnmaskResponse(tt.input, tt.name)
		if result != tt.expected {
			t.Errorf("UnmaskResponse(%q, %q) = %q; want %q", tt.input, tt.name, result, tt.expected)
		}
	}
}
