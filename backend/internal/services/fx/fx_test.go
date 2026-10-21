package fx

import "testing"

func TestToPresentment_INRPassThrough(t *testing.T) {
	got, rate, err := ToPresentment(100000, "INR")
	if err != nil || got != 100000 || rate != 1 {
		t.Fatalf("INR should pass through unchanged: got=%d rate=%v err=%v", got, rate, err)
	}
}

func TestToPresentment_USD(t *testing.T) {
	// ₹1000 at ₹83/USD ≈ $12.05 = 1205 cents
	got, _, err := ToPresentment(100000, "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 1205 {
		t.Fatalf("expected 1205 cents, got %d", got)
	}
}

func TestToPresentment_UnsupportedErrors(t *testing.T) {
	if _, _, err := ToPresentment(100000, "XYZ"); err == nil {
		t.Fatal("unsupported currency must error, not guess")
	}
}

func TestSupported(t *testing.T) {
	if !Supported("usd") || Supported("xyz") {
		t.Fatal("Supported() wrong")
	}
}
