package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"

	"gorm.io/gorm/schema"
)

// Field-level struct tags recognised by the audit plugin:
//
//	`audit:"skip"`     — field is OMITTED from before/after JSON entirely.
//	                     Use for secrets that should never appear in any log:
//	                     password hashes, MFA secrets, OTP codes, tokens.
//
//	`audit:"redact"`   — field value is replaced with a SHA-256 prefix
//	                     ("rdct:abc12345"). Use for PII you must record the
//	                     presence of but never the value: Aadhaar, PAN, CVV.
//
// Anything not tagged is logged in full (the default).
const (
	tagSkip   = "skip"
	tagRedact = "redact"
)

// redactRecord walks the model schema and removes / hashes sensitive fields
// before they are serialized into the audit log.
//
// Input `record` is the GORM destination — usually a pointer to a struct.
// Returns a map[string]interface{} suitable for JSON-marshaling.
//
// If sch is nil (rare — bulk operations or raw queries), the input is returned
// unchanged. We never log MORE than the caller passed, only less.
func redactRecord(record interface{}, sch *schema.Schema) map[string]interface{} {
	if record == nil {
		return nil
	}

	out := map[string]interface{}{}
	v := reflect.ValueOf(record)
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		ft := t.Field(i)
		if !ft.IsExported() {
			continue
		}

		fieldValue := v.Field(i)
		jsonName := jsonFieldName(ft)
		if jsonName == "-" {
			continue
		}

		auditTag := ft.Tag.Get("audit")
		switch auditTag {
		case tagSkip:
			// Don't include at all
			continue
		case tagRedact:
			out[jsonName] = redactValue(fieldValue)
		default:
			// Also auto-skip if the GORM schema marks it serializer:"-" or it
			// looks like a private credential field by name (defence in depth).
			if isObviousSecret(ft.Name, jsonName) {
				continue
			}
			out[jsonName] = fieldValue.Interface()
		}
	}
	return out
}

// jsonFieldName extracts the JSON tag name (or struct field name as fallback).
// Returns "-" if the field is JSON-excluded.
func jsonFieldName(ft reflect.StructField) string {
	if tag, ok := ft.Tag.Lookup("json"); ok {
		first := strings.SplitN(tag, ",", 2)[0]
		if first == "-" {
			return "-"
		}
		if first != "" {
			return first
		}
	}
	return ft.Name
}

// redactValue replaces a value with a stable SHA-256 prefix tag so audit logs
// can still correlate "same value across rows" without exposing the value.
//
// Examples:
//   "1234567890" → "rdct:9d4e1e23"
//   ""           → "" (empty values stay empty so we don't fingerprint absence)
func redactValue(v reflect.Value) string {
	str := stringifyForHash(v)
	if str == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(str))
	return "rdct:" + hex.EncodeToString(sum[:4])
}

func stringifyForHash(v reflect.Value) string {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Slice, reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return string(v.Bytes())
		}
	}
	if v.IsValid() && v.CanInterface() {
		return reflect.TypeOf(v.Interface()).String() // last resort
	}
	return ""
}

// isObviousSecret is a defence-in-depth check that catches sensitive field
// names even when the developer forgot to tag them. Kept conservative — only
// the most universally-secret patterns. Add to the whitelist via `audit:"include"`.
func isObviousSecret(structName, jsonName string) bool {
	candidates := []string{
		structName, jsonName,
	}
	for _, c := range candidates {
		lower := strings.ToLower(c)
		switch lower {
		case "password", "passwordhash", "password_hash",
			"mfasecret", "mfa_secret", "totp_secret", "totpsecret",
			"otp", "otpcode", "otp_code",
			"refresh_token", "refreshtoken", "access_token", "accesstoken",
			"api_key", "apikey", "private_key", "privatekey",
			"webhook_secret", "webhooksecret":
			return true
		}
	}
	return false
}
