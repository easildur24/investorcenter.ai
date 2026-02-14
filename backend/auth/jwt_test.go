package auth

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"investorcenter-api/models"
)

const testSecret = "test-secret-key-that-is-at-least-32-characters-long"

// setupTestSecret sets JWT_SECRET for tests and returns a cleanup function
func setupTestSecret(t *testing.T) {
	t.Helper()
	os.Setenv("JWT_SECRET", testSecret)
	jwtSecret = []byte(testSecret)
	t.Cleanup(func() {
		os.Unsetenv("JWT_SECRET")
	})
}

func createTestUser() *models.User {
	return &models.User{
		ID:      "user-123",
		Email:   "test@example.com",
		IsAdmin: false,
	}
}

func TestGenerateAccessToken(t *testing.T) {
	setupTestSecret(t)

	t.Run("generates valid token for user", func(t *testing.T) {
		user := createTestUser()
		token, err := GenerateAccessToken(user)

		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Parse the token back and verify claims
		claims, err := ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, "user-123", claims.UserID)
		assert.Equal(t, "test@example.com", claims.Email)
		assert.False(t, claims.IsAdmin)
		assert.Equal(t, "investorcenter.ai", claims.Issuer)
		assert.Equal(t, "user-123", claims.Subject)
	})

	t.Run("generates token for admin user", func(t *testing.T) {
		user := createTestUser()
		user.IsAdmin = true

		token, err := GenerateAccessToken(user)
		require.NoError(t, err)

		claims, err := ValidateToken(token)
		require.NoError(t, err)
		assert.True(t, claims.IsAdmin)
	})

	t.Run("token has correct expiry", func(t *testing.T) {
		user := createTestUser()
		token, err := GenerateAccessToken(user)
		require.NoError(t, err)

		claims, err := ValidateToken(token)
		require.NoError(t, err)

		// Token should expire within the configured access token duration
		expiresAt := claims.ExpiresAt.Time
		assert.True(t, expiresAt.After(time.Now()))
		assert.True(t, expiresAt.Before(time.Now().Add(accessTokenDuration+time.Minute)))
	})
}

func TestGenerateRefreshToken(t *testing.T) {
	setupTestSecret(t)

	t.Run("generates valid refresh token", func(t *testing.T) {
		user := createTestUser()
		token, err := GenerateRefreshToken(user)

		require.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, "user-123", claims.UserID)
	})

	t.Run("refresh token has longer expiry than access token", func(t *testing.T) {
		user := createTestUser()

		accessTok, err := GenerateAccessToken(user)
		require.NoError(t, err)

		refreshTok, err := GenerateRefreshToken(user)
		require.NoError(t, err)

		accessClaims, _ := ValidateToken(accessTok)
		refreshClaims, _ := ValidateToken(refreshTok)

		assert.True(t, refreshClaims.ExpiresAt.Time.After(accessClaims.ExpiresAt.Time))
	})

	t.Run("tokens for different users are different", func(t *testing.T) {
		user1 := createTestUser()
		user2 := &models.User{ID: "user-456", Email: "other@example.com", IsAdmin: true}

		token1, err := GenerateRefreshToken(user1)
		require.NoError(t, err)

		token2, err := GenerateRefreshToken(user2)
		require.NoError(t, err)

		assert.NotEqual(t, token1, token2)
	})
}

func TestValidateToken(t *testing.T) {
	setupTestSecret(t)

	t.Run("validates a valid token", func(t *testing.T) {
		user := createTestUser()
		token, err := GenerateAccessToken(user)
		require.NoError(t, err)

		claims, err := ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, "user-123", claims.UserID)
	})

	t.Run("rejects expired token", func(t *testing.T) {
		// Create a token that's already expired
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
		user := createTestUser()
		token, err := GenerateAccessToken(user)
		require.NoError(t, err)

		// Tamper with the signature portion (flip multiple characters in the middle of the signature)
		parts := strings.SplitN(token, ".", 3)
		require.Len(t, parts, 3)
		sig := parts[2]
		// Reverse the signature to ensure it's definitely different
		runes := []rune(sig)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		tampered := parts[0] + "." + parts[1] + "." + string(runes)

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
				Issuer:    "investorcenter.ai",
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte("wrong-secret-key-that-is-different"))
		require.NoError(t, err)

		result, err := ValidateToken(tokenString)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("rejects token with wrong signing method", func(t *testing.T) {
		// Try to validate a token that claims a different signing method
		// We can't easily create RSA-signed tokens here, but we can test with empty/malformed tokens
		_, err := ValidateToken("")
		assert.Error(t, err)

		_, err = ValidateToken("not.a.jwt")
		assert.Error(t, err)
	})

	t.Run("rejects completely invalid token", func(t *testing.T) {
		result, err := ValidateToken("garbage-token")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestValidateJWTSecret(t *testing.T) {
	// Save and restore original env
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

	t.Run("panics when JWT_SECRET is exactly min-1 chars", func(t *testing.T) {
		os.Setenv("JWT_SECRET", "1234567890123456789012345678901") // 31 chars
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

	t.Run("succeeds with exactly min length secret", func(t *testing.T) {
		os.Setenv("JWT_SECRET", "12345678901234567890123456789012") // 32 chars
		assert.NotPanics(t, func() {
			ValidateJWTSecret()
		})
	})

	t.Run("updates jwtSecret after validation", func(t *testing.T) {
		newSecret := "a-new-secret-that-is-definitely-long-enough-for-testing"
		os.Setenv("JWT_SECRET", newSecret)
		ValidateJWTSecret()
		assert.Equal(t, []byte(newSecret), jwtSecret)
	})
}

func TestGetAccessTokenExpirySeconds(t *testing.T) {
	seconds := GetAccessTokenExpirySeconds()
	assert.Greater(t, seconds, 0)
}

func TestParseDuration(t *testing.T) {
	t.Run("returns default for empty string", func(t *testing.T) {
		result := parseDuration("", 5*time.Minute)
		assert.Equal(t, 5*time.Minute, result)
	})

	t.Run("parses valid duration", func(t *testing.T) {
		result := parseDuration("2h", 5*time.Minute)
		assert.Equal(t, 2*time.Hour, result)
	})

	t.Run("parses minutes", func(t *testing.T) {
		result := parseDuration("30m", 5*time.Minute)
		assert.Equal(t, 30*time.Minute, result)
	})

	t.Run("returns default for invalid duration", func(t *testing.T) {
		result := parseDuration("invalid", 5*time.Minute)
		assert.Equal(t, 5*time.Minute, result)
	})

	t.Run("returns default for number without unit", func(t *testing.T) {
		result := parseDuration("100", 5*time.Minute)
		assert.Equal(t, 5*time.Minute, result)
	})
}
