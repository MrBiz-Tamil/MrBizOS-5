package disbursement

import "testing"

func TestStubDisburser(t *testing.T) {
	d := StubDisburser{}

	ok, err := d.Disburse(Request{UserID: 1, AmountPaise: 5000000, BankAccountNumber: "123456789", BankIFSC: "HDFC0000123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok.Status != "processed" || ok.Reference == "" || ok.Provider != "stub" {
		t.Errorf("expected a processed stub disbursement, got %+v", ok)
	}

	if _, err := d.Disburse(Request{UserID: 1, AmountPaise: 0, BankAccountNumber: "1", BankIFSC: "x"}); err == nil {
		t.Error("expected error for non-positive amount")
	}
	if _, err := d.Disburse(Request{UserID: 1, AmountPaise: 100}); err == nil {
		t.Error("expected error for missing bank details")
	}
}

func TestDefaultReturnsStubWithoutCreds(t *testing.T) {
	t.Setenv("DISBURSEMENT_API_KEY", "")
	if _, ok := Default().(StubDisburser); !ok {
		t.Error("Default() should return StubDisburser when no credentials are configured")
	}
}
