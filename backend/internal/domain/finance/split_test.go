package finance_test

import (
	"testing"

	"drayya-core/internal/domain/finance"
)

// TestComputeConsultationSplit_Canonical pins the exact paise breakdown the rest
// of the system depends on (and the E2E integration test asserts): a ₹1000 fee
// splits into ₹300 commission, ₹54 GST, ₹64.60 TDS, ₹581.40 doctor net.
func TestComputeConsultationSplit_Canonical(t *testing.T) {
	s := finance.ComputeConsultationSplit(100000) // ₹1000 in paise

	cases := []struct {
		name string
		got  int64
		want int64
	}{
		{"Total", s.Total, 100000},
		{"Commission", s.Commission, 30000},
		{"GST", s.GST, 5400},
		{"DoctorGross", s.DoctorGross, 64600},
		{"TDS", s.TDS, 6460},
		{"DoctorNet", s.DoctorNet, 58140},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
}

// TestComputeConsultationSplit_Invariant guarantees the parts always sum back to
// the whole across a range of fees — the property that lets every ledger entry
// built from a split balance exactly (debits == credits).
func TestComputeConsultationSplit_Invariant(t *testing.T) {
	for _, total := range []int64{0, 1, 99, 100, 4999, 50000, 100000, 123457, 999999, 100000000} {
		s := finance.ComputeConsultationSplit(total)
		if s.Commission+s.GST+s.TDS+s.DoctorNet != total {
			t.Errorf("invariant broken for total=%d: %d+%d+%d+%d != %d",
				total, s.Commission, s.GST, s.TDS, s.DoctorNet, total)
		}
		if s.DoctorGross != s.TDS+s.DoctorNet {
			t.Errorf("DoctorGross mismatch for total=%d: %d != %d+%d",
				total, s.DoctorGross, s.TDS, s.DoctorNet)
		}
		if s.Commission < 0 || s.GST < 0 || s.TDS < 0 || s.DoctorNet < 0 {
			t.Errorf("negative component for total=%d: %+v", total, s)
		}
	}
}
