package money

import (
	"testing"
)

func TestMoney_Creation(t *testing.T) {
	m := NewMoney(1500, "USD")
	if m.Amount != 1500 {
		t.Errorf("Expected amount 1500, got %d", m.Amount)
	}
	if m.Currency != "USD" {
		t.Errorf("Expected currency USD, got %s", m.Currency)
	}
}

func TestMoney_DefaultCurrency(t *testing.T) {
	m := NewMoney(500, "")
	if m.Currency != "USD" {
		t.Errorf("Expected fallback currency USD, got %s", m.Currency)
	}
}

func TestMoney_Add_SameCurrency(t *testing.T) {
	m1 := NewMoney(1500, "AED")
	m2 := NewMoney(500, "AED")
	sum := m1.Add(m2)
	
	if sum.Amount != 2000 {
		t.Errorf("Expected sum 2000, got %d", sum.Amount)
	}
	if sum.Currency != "AED" {
		t.Errorf("Expected AED, got %s", sum.Currency)
	}
}

func TestMoney_Add_CurrencyMismatch_Panics(t *testing.T) {
	m1 := NewMoney(1500, "USD")
	m2 := NewMoney(50000, "INR")
	
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when adding USD to INR, but did not panic")
		}
	}()
	
	m1.Add(m2)
}

func TestMoney_RegionalPricing_JSON(t *testing.T) {
	prices := RegionalPricing{"INR": 50000, "USD": 1500, "AED": 6000}
	
	val, err := prices.Value()
	if err != nil {
		t.Errorf("Failed to marshal RegionalPricing: %v", err)
	}
	
	var newPrices RegionalPricing
	err = newPrices.Scan(val)
	if err != nil {
		t.Errorf("Failed to scan RegionalPricing: %v", err)
	}
	
	if newPrices["AED"] != 6000 {
		t.Errorf("Expected AED 6000, got %d", newPrices["AED"])
	}
}
