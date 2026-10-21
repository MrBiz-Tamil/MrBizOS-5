package pan

import "testing"

func TestValidFormat(t *testing.T) {
	valid := []string{"ABCDE1234F", "AAAPL1234C"}
	invalid := []string{"", "ABCD1234F", "ABCDE12345", "abcde1234f", "ABCDE1234", "12345ABCDF"}
	for _, p := range valid {
		if !ValidFormat(p) {
			t.Errorf("expected %q to be valid", p)
		}
	}
	for _, p := range invalid {
		if ValidFormat(p) {
			t.Errorf("expected %q to be invalid", p)
		}
	}
}

func TestStubVerifier(t *testing.T) {
	v := StubVerifier{}
	r, err := v.Verify("ABCDE1234F", "Some Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.Valid || !r.NameMatch || r.Provider != "stub" {
		t.Errorf("stub should confirm a well-formed PAN: %+v", r)
	}
	bad, _ := v.Verify("nope", "X")
	if bad.Valid {
		t.Error("stub should reject a malformed PAN")
	}
}

func TestDefaultReturnsStubWithoutCreds(t *testing.T) {
	t.Setenv("NSDL_PAN_API_KEY", "")
	if _, ok := Default().(StubVerifier); !ok {
		t.Error("Default() should return StubVerifier when no credentials are configured")
	}
}
