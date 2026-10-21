package privacy

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

var encryptionKey []byte

// devDefaultKey is the deterministic key used in dev/test only so seeded data is
// readable across restarts. It is NEVER used in production: a missing
// DB_ENCRYPTION_KEY in production aborts startup.
const devDefaultKey = "medi_hub_default_secret_key_32bytes"

// minKeyBytes is the minimum custom-key length we accept before padding.
// AES-256 needs 32 bytes; we reject anything weaker than a 24-byte secret to
// prevent accidentally deploying with a short key that gets null-padded.
const minKeyBytes = 24

func init() {
	keyStr := os.Getenv("DB_ENCRYPTION_KEY")
	isProd := os.Getenv("APP_ENV") == "production" || os.Getenv("APP_ENV") == "prod"

	if keyStr == "" {
		if isProd {
			// In production we will not silently fall back to a hardcoded key — that
			// would mean anyone with source can decrypt the database. Refuse to boot.
			panic("[privacy] FATAL: DB_ENCRYPTION_KEY is not set. " +
				"Medical PHI fields (prescription diagnosis/symptoms/instructions) are stored " +
				"encrypted with AES-256-GCM and require a real key in production. " +
				"Generate one with: openssl rand -base64 32")
		}
		fmt.Println("[privacy] WARNING: DB_ENCRYPTION_KEY is not set; using the dev default. " +
			"NEVER deploy with this key — set DB_ENCRYPTION_KEY before going to production.")
		keyStr = devDefaultKey
	} else if len(keyStr) < minKeyBytes {
		if isProd {
			panic(fmt.Sprintf("[privacy] FATAL: DB_ENCRYPTION_KEY is too short (%d bytes); "+
				"need at least %d. Generate one with: openssl rand -base64 32",
				len(keyStr), minKeyBytes))
		}
		fmt.Printf("[privacy] WARNING: DB_ENCRYPTION_KEY is only %d bytes; "+
			"recommended minimum is %d. Continuing because APP_ENV is not production.\n",
			len(keyStr), minKeyBytes)
	}

	// Pad or truncate key to exactly 32 bytes for AES-256
	keyBytes := []byte(keyStr)
	if len(keyBytes) < 32 {
		padded := make([]byte, 32)
		copy(padded, keyBytes)
		encryptionKey = padded
	} else {
		encryptionKey = keyBytes[:32]
	}
}

// Encrypt encrypts plain text using AES-256-GCM.
func Encrypt(plainText string) (string, error) {
	if plainText == "" {
		return "", nil
	}
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt decrypts AES-256-GCM encrypted base64 string.
func Decrypt(cipherTextB64 string) (string, error) {
	if cipherTextB64 == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(cipherTextB64)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, cipherText := data[:nonceSize], data[nonceSize:]
	decrypted, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}

// EncryptedString represents a string that is encrypted in the database but decrypted in memory.
type EncryptedString string

// Value implements driver.Valuer interface, encrypting the value before database write.
func (es EncryptedString) Value() (driver.Value, error) {
	enc, err := Encrypt(string(es))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}
	return enc, nil
}

// Scan implements sql.Scanner interface, decrypting the value after database query.
func (es *EncryptedString) Scan(value interface{}) error {
	if value == nil {
		*es = ""
		return nil
	}
	var cipherText string
	switch v := value.(type) {
	case string:
		cipherText = v
	case []byte:
		cipherText = string(v)
	default:
		return fmt.Errorf("unsupported type for EncryptedString scan: %T", value)
	}
	
	// Fast check: if empty or doesn't look like base64, return raw string to support existing DB values
	if cipherText == "" || len(cipherText) < 16 || strings.Contains(cipherText, " ") {
		*es = EncryptedString(cipherText)
		return nil
	}

	dec, err := Decrypt(cipherText)
	if err != nil {
		// Fallback to raw string if decryption fails (e.g. existing unencrypted data in DB)
		*es = EncryptedString(cipherText)
		return nil
	}
	*es = EncryptedString(dec)
	return nil
}
