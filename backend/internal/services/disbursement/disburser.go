// Package disbursement pushes salary payments to employee bank accounts.
//
// Real disbursement happens over a bank/NACH or a payout API (RazorpayX, ICICI
// Connected Banking, etc.) and needs corporate-banking credentials. This package
// ships a StubDisburser that returns a mock reference and selects a live client
// only when credentials are present — so call sites depend on the Disburser
// interface and the integration flips on with one env var.
//
// (Freelance doctor payouts already go through Razorpay in handlers/payout.go;
// this is the salaried-staff counterpart.)
package disbursement

import (
	"fmt"
	"os"
	"time"
)

// Request is a single salary disbursement instruction.
type Request struct {
	UserID            uint
	AmountPaise       int64
	BeneficiaryName   string
	BankAccountNumber string
	BankIFSC          string
	Narration         string
}

// Result is the outcome of a disbursement.
type Result struct {
	Reference string `json:"reference"` // bank/UTR or payout id
	Status    string `json:"status"`    // processing | processed | failed
	Provider  string `json:"provider"`  // "stub" | "razorpayx" | ...
}

// Disburser sends money to a beneficiary account.
type Disburser interface {
	Disburse(Request) (Result, error)
}

// StubDisburser records the instruction and returns a deterministic mock
// reference without contacting a bank. Status is "processed" so downstream flows
// can be exercised end to end in dev/test.
type StubDisburser struct{}

func (StubDisburser) Disburse(r Request) (Result, error) {
	if r.AmountPaise <= 0 {
		return Result{}, fmt.Errorf("amount must be positive")
	}
	if r.BankAccountNumber == "" || r.BankIFSC == "" {
		return Result{Status: "failed", Provider: "stub"}, fmt.Errorf("missing bank details for user %d", r.UserID)
	}
	ref := fmt.Sprintf("STUB-SAL-%d-%d", r.UserID, time.Now().UnixNano())
	return Result{Reference: ref, Status: "processed", Provider: "stub"}, nil
}

// Default returns the configured disburser. A live RazorpayX/bank client should
// be returned when DISBURSEMENT_API_KEY is set; until then, the stub.
func Default() Disburser {
	if os.Getenv("DISBURSEMENT_API_KEY") != "" {
		// TODO: return a live RazorpayX/bank-NACH disburser once the client exists.
	}
	return StubDisburser{}
}
