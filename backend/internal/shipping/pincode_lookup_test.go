package shipping

import "testing"

func TestLookupPincode(t *testing.T) {
	cases := []struct {
		pincode   string
		wantCity  string
		wantState string
	}{
		// Major metros — exact 3-digit prefix matches.
		{"600028", "Chennai", "Tamil Nadu"},
		{"560001", "Bengaluru", "Karnataka"},
		{"400001", "Mumbai", "Maharashtra"},
		{"110001", "New Delhi", "Delhi"},
		{"700016", "Kolkata", "West Bengal"},
		{"500032", "Hyderabad", "Telangana"},
		{"380001", "Ahmedabad", "Gujarat"},
		{"411001", "Pune", "Maharashtra"},
		{"682001", "Kochi", "Kerala"},
		{"800001", "Patna", "Bihar"},
		// Unmatched 3-digit but known 1-digit zonal fallback — state only.
		{"299999", "", "Uttar Pradesh"},
		{"599999", "", "Karnataka"},
		// Invalid input.
		{"", "", ""},
		{"12345", "", ""},     // too short
		{"abcdef", "", ""},    // non-numeric
		{"6000281", "", ""},   // too long
	}
	for _, c := range cases {
		got := LookupPincode(c.pincode)
		if got.City != c.wantCity || got.State != c.wantState {
			t.Errorf("LookupPincode(%q) = (%q,%q), want (%q,%q)",
				c.pincode, got.City, got.State, c.wantCity, c.wantState)
		}
	}
}
