package emr

import (
	"drayya-core/internal/utils/privacy"
	"fmt"

	"gorm.io/gorm"
)

// BackfillEncryptedFields encrypts any plaintext rows in `prescriptions` whose
// diagnosis / symptoms / instructions columns are still cleartext from before
// the field-level encryption migration.
//
// Strategy: read each row's raw text via the DB driver (bypassing the GORM
// EncryptedString Scan); if Decrypt fails, the value is still plaintext, so
// encrypt and rewrite it. Rows that decrypt cleanly are skipped.
//
// Idempotent: a second run finds no plaintext rows and does nothing.
// Safe to run on every boot.
func BackfillEncryptedFields(db *gorm.DB) {
	if !db.Migrator().HasTable("prescriptions") {
		return
	}

	type row struct {
		ID           uint
		Diagnosis    string
		Symptoms     string
		Instructions string
	}

	var rows []row
	if err := db.Table("prescriptions").
		Select("id, diagnosis, symptoms, instructions").
		Find(&rows).Error; err != nil {
		fmt.Printf("[emr] encryption backfill: skipped — read failed: %v\n", err)
		return
	}

	var migrated int
	for _, r := range rows {
		updates := map[string]interface{}{}
		if enc, ok := encryptIfPlaintext(r.Diagnosis); ok {
			updates["diagnosis"] = enc
		}
		if enc, ok := encryptIfPlaintext(r.Symptoms); ok {
			updates["symptoms"] = enc
		}
		if enc, ok := encryptIfPlaintext(r.Instructions); ok {
			updates["instructions"] = enc
		}
		if len(updates) == 0 {
			continue
		}
		if err := db.Table("prescriptions").Where("id = ?", r.ID).Updates(updates).Error; err != nil {
			fmt.Printf("[emr] encryption backfill: prescription %d update failed: %v\n", r.ID, err)
			continue
		}
		migrated++
	}
	if migrated > 0 {
		fmt.Printf("[emr] encryption backfill: encrypted %d plaintext prescription field(s)\n", migrated)
	}
}

// encryptIfPlaintext returns (ciphertext, true) when the value looks like
// plaintext (decryption fails) and could be encrypted, or ("", false) when it is
// already a valid AES-GCM ciphertext or empty.
func encryptIfPlaintext(value string) (string, bool) {
	if value == "" {
		return "", false
	}
	if _, err := privacy.Decrypt(value); err == nil {
		// Already encrypted — leave it alone.
		return "", false
	}
	enc, err := privacy.Encrypt(value)
	if err != nil {
		return "", false
	}
	return enc, true
}
