package pharmacy

import "testing"

// The mock OCR text in services/ocr.go has three Rx lines. The parser must
// pull all three with their dosage + days, and skip the doctor/clinic/Rx
// header/signature lines.
func TestParseRxText_ExtractsThreeMedicines(t *testing.T) {
	const mock = `Dr. R. Krishnan, MBBS
Apollo Clinic, Chennai

Rx
1. Metformin 500mg — 1 tab twice daily — 30 days
2. Atorvastatin 10mg — 1 tab at bedtime — 30 days
3. Amlodipine 5mg — 1 tab once daily — 30 days

Signature: Dr. R. Krishnan`

	got := ParseRxText(mock)
	if len(got) != 3 {
		t.Fatalf("expected 3 parsed medicines, got %d: %+v", len(got), got)
	}
	if got[0].Name != "Metformin 500mg" {
		t.Errorf("first med name wrong: %q", got[0].Name)
	}
	if got[0].QtyDays != 30 {
		t.Errorf("first med days wrong: %d", got[0].QtyDays)
	}
	for _, m := range got {
		if m.Dosage == "" {
			t.Errorf("dosage missing for %q", m.Name)
		}
	}
}

// Lines that look like a prescription line (header / signature) must not
// be returned — they'd confuse the cart auto-fill.
func TestParseRxText_SkipsNonMedicineLines(t *testing.T) {
	for _, line := range []string{
		"Dr. R. Krishnan — MBBS",
		"Signature: Dr. R. Krishnan",
		"Apollo Clinic, Chennai",
	} {
		if got := ParseRxText(line); len(got) != 0 {
			t.Errorf("%q wrongly parsed as medicine: %+v", line, got)
		}
	}
}
