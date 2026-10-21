package money

import "testing"

func TestFromRupees_RoundsCorrectly(t *testing.T) {
	cases := []struct {
		in       float64
		expected Paise
	}{
		{0, 0},
		{1, 100},
		{499.99, 49999},
		{500, 50000},
		{500.001, 50000},  // rounds down
		{500.005, 50001},  // rounds up (banker's-ish — math.Round rounds half away from zero)
		{1234.56, 123456},
	}
	for _, c := range cases {
		got := FromRupees(c.in)
		if got != c.expected {
			t.Errorf("FromRupees(%f) = %d, want %d", c.in, got, c.expected)
		}
	}
}

func TestRupees_RoundTrip(t *testing.T) {
	for _, p := range []Paise{0, 100, 49999, 50000, 100_000, 1_00_00_000} {
		round := FromRupees(p.Rupees())
		if round != p {
			t.Errorf("round-trip failed: %d → %f → %d", p, p.Rupees(), round)
		}
	}
}

func TestString_Format(t *testing.T) {
	cases := []struct {
		in       Paise
		expected string
	}{
		{0, "₹0.00"},
		{1, "₹0.01"},
		{99, "₹0.99"},
		{100, "₹1.00"},
		{50000, "₹500.00"},
		{49999, "₹499.99"},
		{1_00_00_000, "₹100000.00"}, // 1 lakh rupees
		{-50000, "-₹500.00"},
	}
	for _, c := range cases {
		got := c.in.String()
		if got != c.expected {
			t.Errorf("Paise(%d).String() = %q, want %q", c.in, got, c.expected)
		}
	}
}

func TestIsValid_RejectsOutOfRange(t *testing.T) {
	cases := []struct {
		in    Paise
		valid bool
	}{
		{0, true},
		{50000, true},
		{1_00_00_000, true},  // 1 lakh rupees
		{100_00_00_000, true}, // 1 crore rupees — boundary
		{100_00_00_001, false}, // > 1 crore — likely a unit bug
		{-1, false},
		{-50000, false},
	}
	for _, c := range cases {
		got := c.in.IsValid()
		if got != c.valid {
			t.Errorf("Paise(%d).IsValid() = %v, want %v", c.in, got, c.valid)
		}
	}
}

func TestArithmetic(t *testing.T) {
	a := Paise(50000) // ₹500
	b := Paise(15000) // ₹150
	if a.Add(b) != 65000 {
		t.Errorf("Add: %d + %d = %d, want 65000", a, b, a.Add(b))
	}
	if a.Sub(b) != 35000 {
		t.Errorf("Sub: %d - %d = %d, want 35000", a, b, a.Sub(b))
	}
	if a.Mul(3) != 150000 {
		t.Errorf("Mul: %d * 3 = %d, want 150000", a, a.Mul(3))
	}
	// 15% of ₹500 = ₹75 = 7500 paise
	if a.PercentOf(15) != 7500 {
		t.Errorf("PercentOf 15: got %d, want 7500", a.PercentOf(15))
	}
}
