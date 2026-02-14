package auth

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key-that-is-at-least-32-characters-long"

func setupTestSecret(t *testing.T) {
	t.Helper()
	os.Setenv("JWT_SECRET", testSecret)
	jwtSecret = []byte(testSecret)
	t.Cleanup(func() {
		os.Unsetenv("JWT_SECRET")
	})
}

// generateValidToken creates a signed JWT token for testing
func generateValidToken(t *testing.T, userID, email string, isAdmin bool) string {
	t.Helper()
	claims := Claims{
		UserID:  userID,
		Email:   email,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "investorcenter.ai",
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	require.NoError(t, err)
	return tokenString
}

func TestValidateToken(t *testing.T) {
	setupTestSecret(t)

	t.Run("validates a valid token", func(t *testing.T) {
		token := generateValidToken(t, "user-123", "test@example.com", false)

		claims, err := ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, "user-123", claims.UserID)
		assert.Equal(t, "test@example.com", claims.Email)
		assert.False(t, claims.IsAdmin)
	})

	t.Run("validates admin token", func(t *testing.T) {
		token := generateValidToken(t, "admin-1", "admin@example.com", true)

		claims, err := ValidateToken(token)
		require.NoError(t, err)
		assert.True(t, claims.IsAdmin)
	})

	t.Run("rejects expired token", func(t *testing.T) {
		claims := Claims{
			UserID:  "user-123",
			Email:   "test@example.com",
			IsAdmin: false,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
				Issuer:    "investorcenter.ai",
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(jwtSecret)
		require.NoError(t, err)

		result, err := ValidateToken(tokenString)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("rejects tampered token", func(t *testing.T) {
		token := generateValidToken(t, "user-123", "test@example.com", false)
		tampered := token[:len(token)-1] + "X"

		result, err := ValidateToken(tampered)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("rejects token signed with wrong key", func(t *testing.T) {
		claims := Claims{
			UserID: "user-123",
			Email:  "test@example.com",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte("wrong-secret-key-different"))
		require.NoError(t, err)

		result, err := ValidateToken(tokenString)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("rejects empty token", func(t *testing.T) {
		_, err := ValidateToken("")
		assert.Error(t, err)
	})

	t.Run("rejects garbage token", func(t *testing.T) {
		result, err := ValidateToken("not-a-jwt")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestValidateJWTSecret(t *testing.T) {
	originalSecret := os.Getenv("JWT_SECRET")
	defer func() {
		if originalSecret != "" {
			os.Setenv("JWT_SECRET", originalSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
	}()

	t.Run("panics when JWT_SECRET is empty", func(t *testing.T) {
		os.Unsetenv("JWT_SECRET")
		assert.Panics(t, func() {
			ValidateJWTSecret()
		})
	})

	t.Run("panics when JWT_SECRET is too short", func(t *testing.T) {
		os.Setenv("JWT_SECRET", "short")
		assert.Panics(t, func() {
			ValidateJWTSecret()
		})
	})

	t.Run("succeeds with valid secret", func(t *testing.T) {
		os.Setenv("JWT_SECRET", testSecret)
		assert.NotPanics(t, func() {
			ValidateJWTSecret()
		})
	})

	t.Run("updates package-level jwtSecret", func(t *testing.T) {
		newSecret := "updated-secret-key-that-is-definitely-long-enough"
		os.Setenv("JWT_SECRET", newSecret)
		ValidateJWTSecret()
		assert.Equal(t, []byte(newSecret), jwtSecret)
	})
}
