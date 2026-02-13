package auth

import (
	"errors"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

const minJWTSecretLength = 32

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

// ValidateJWTSecret checks that JWT_SECRET is set and meets minimum length requirements.
// Must be called from main() after loading environment variables.
// Panics if the secret is missing or too short to prevent the service from running insecurely.
func ValidateJWTSecret() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("FATAL: JWT_SECRET environment variable is not set. The service cannot start without a signing key.")
	}
	if len(secret) < minJWTSecretLength {
		panic(fmt.Sprintf("FATAL: JWT_SECRET is too short (%d chars). Minimum %d characters required for security.", len(secret), minJWTSecretLength))
	}
	// Refresh the package-level variable in case env was loaded after package init
	jwtSecret = []byte(secret)
}

// Custom claims for JWT
type Claims struct {
	UserID  string `json:"user_id"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

// ValidateToken parses and validates a JWT token
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
