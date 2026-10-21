package tokens

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func getSecretKey() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("JWT_SECRET environment variable is required but not set")
	}
	return []byte(secret)
}

type Claims struct {
	UserID     string `json:"sub"`
	Role       string `json:"role"`
	Type       string `json:"type"`
	HospitalID *uint  `json:"hospital_id,omitempty"`
	PharmacyID *uint  `json:"pharmacy_id,omitempty"`
	jwt.RegisteredClaims
}

func CreateAccessToken(userID string, role string, hospitalID *uint, pharmacyID *uint) (string, error) {
	expirationTime := time.Now().Add(15 * time.Minute)
	claims := &Claims{
		UserID:     userID,
		Role:       role,
		Type:       "access",
		HospitalID: hospitalID,
		PharmacyID: pharmacyID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getSecretKey())
}

// CreateMFAToken creates a short-lived (10-minute) JWT used exclusively during
// the MFA verification step. The role is "mfa_pending" so that it cannot be
// used to access any protected resource other than /auth/mfa/verify and
// /auth/mfa/resend.
func CreateMFAToken(userID string, hospitalID *uint) (string, error) {
	expirationTime := time.Now().Add(10 * time.Minute)
	claims := &Claims{
		UserID:     userID,
		Role:       "mfa_pending",
		Type:       "mfa",
		HospitalID: hospitalID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getSecretKey())
}



func CreateRefreshToken(userID string, role string, hospitalID *uint, pharmacyID *uint) (string, error) {
	expirationTime := time.Now().Add(7 * 24 * time.Hour)
	claims := &Claims{
		UserID:     userID,
		Role:       role,
		Type:       "refresh",
		HospitalID: hospitalID,
		PharmacyID: pharmacyID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getSecretKey())
}

func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return getSecretKey(), nil
	}, jwt.WithLeeway(5*time.Second))

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
