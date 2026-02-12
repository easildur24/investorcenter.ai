package auth

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

const testSecret = "test-jwt-secret-for-testing"

func setupJWTTest() {
	os.Setenv("JWT_SECRET", testSecret)
	jwtSecret = []byte(testSecret)
}

// generateTestToken creates a JWT token for testing purposes
func generateTestToken(claims Claims, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func TestValidateToken_ValidToken(t *testing.T) {
	setupJWTTest()

	claims := Claims{
		UserID:  "user-123",
		Email:   "test@example.com",
		IsAdmin: false,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	tokenString, err := generateTestToken(claims, testSecret)
	assert.NoError(t, err)

	result, err := ValidateToken(tokenString)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "user-123", result.UserID)
	assert.Equal(t, "test@example.com", result.Email)
	assert.False(t, result.IsAdmin)
}

func TestValidateToken_AdminToken(t *testing.T) {
	setupJWTTest()

	claims := Claims{
		UserID:  "admin-456",
		Email:   "admin@example.com",
		IsAdmin: true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	tokenString, err := generateTestToken(claims, testSecret)
	assert.NoError(t, err)

	result, err := ValidateToken(tokenString)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "admin-456", result.UserID)
	assert.True(t, result.IsAdmin)
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	setupJWTTest()

	claims := Claims{
		UserID: "user-123",
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	tokenString, err := generateTestToken(claims, testSecret)
	assert.NoError(t, err)

	result, err := ValidateToken(tokenString)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	setupJWTTest()

	claims := Claims{
		UserID: "user-123",
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}

	tokenString, err := generateTestToken(claims, "wrong-secret")
	assert.NoError(t, err)

	result, err := ValidateToken(tokenString)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestValidateToken_InvalidSigningMethod(t *testing.T) {
	setupJWTTest()

	// Create a token with RSA signing method (not HMAC)
	token := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{
		UserID: "user-123",
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	})
	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	assert.NoError(t, err)

	result, err := ValidateToken(tokenString)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestValidateToken_MalformedToken(t *testing.T) {
	setupJWTTest()

	result, err := ValidateToken("not-a-valid-jwt-token")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestValidateToken_EmptyToken(t *testing.T) {
	setupJWTTest()

	result, err := ValidateToken("")
	assert.Error(t, err)
	assert.Nil(t, result)
}
