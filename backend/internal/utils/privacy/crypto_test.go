package privacy

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	originalText := "Sensitive patient diagnosis: Type 2 Diabetes"

	cipherText, err := Encrypt(originalText)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	if cipherText == originalText {
		t.Fatalf("Encryption did not modify the text")
	}

	decryptedText, err := Decrypt(cipherText)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if decryptedText != originalText {
		t.Errorf("Decrypted text %q does not match original %q", decryptedText, originalText)
	}
}

func TestEncryptedStringScannerValuer(t *testing.T) {
	// 1. Test Valuer (writes encrypted data to DB)
	plainStr := EncryptedString("Patient allergies: Penicillin")
	val, err := plainStr.Value()
	if err != nil {
		t.Fatalf("Failed to get Value from EncryptedString: %v", err)
	}

	dbVal, ok := val.(string)
	if !ok {
		t.Fatalf("Valuer did not return string, got %T", val)
	}

	if dbVal == string(plainStr) {
		t.Fatalf("Valuer did not encrypt the string")
	}

	// 2. Test Scanner (reads and decrypts from DB)
	var scannedStr EncryptedString
	err = scannedStr.Scan(dbVal)
	if err != nil {
		t.Fatalf("Failed to Scan EncryptedString: %v", err)
	}

	if string(scannedStr) != string(plainStr) {
		t.Errorf("Scanned/decrypted value %q does not match original %q", scannedStr, plainStr)
	}

	// 3. Test Scanner Fallback (reads plain text from existing/legacy DB data)
	legacyText := "legacy unencrypted note"
	var legacyScan EncryptedString
	err = legacyScan.Scan(legacyText)
	if err != nil {
		t.Fatalf("Failed to Scan legacy plain text: %v", err)
	}

	if string(legacyScan) != legacyText {
		t.Errorf("Scanner did not fallback correctly, got %q, wanted %q", legacyScan, legacyText)
	}
}
