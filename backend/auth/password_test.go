package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	t.Run("hashes a password successfully", func(t *testing.T) {
		hash, err := HashPassword("mypassword123")
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, "mypassword123", hash)
	})

	t.Run("same password produces different hashes", func(t *testing.T) {
		hash1, err := HashPassword("samepassword")
		require.NoError(t, err)

		hash2, err := HashPassword("samepassword")
		require.NoError(t, err)

		// bcrypt includes a random salt, so hashes should differ
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("hashes empty password", func(t *testing.T) {
		hash, err := HashPassword("")
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})
}

func TestCheckPasswordHash(t *testing.T) {
	t.Run("correct password returns true", func(t *testing.T) {
		password := "secure-password-123"
		hash, err := HashPassword(password)
		require.NoError(t, err)

		assert.True(t, CheckPasswordHash(password, hash))
	})

	t.Run("wrong password returns false", func(t *testing.T) {
		password := "correct-password"
		hash, err := HashPassword(password)
		require.NoError(t, err)

		assert.False(t, CheckPasswordHash("wrong-password", hash))
	})

	t.Run("empty password with valid hash returns false", func(t *testing.T) {
		hash, err := HashPassword("non-empty")
		require.NoError(t, err)

		assert.False(t, CheckPasswordHash("", hash))
	})

	t.Run("invalid hash returns false", func(t *testing.T) {
		assert.False(t, CheckPasswordHash("password", "not-a-valid-hash"))
	})

	t.Run("handles special characters in password", func(t *testing.T) {
		password := "p@$$w0rd!#%^&*()_+-=[]{}|;':\",./<>?"
		hash, err := HashPassword(password)
		require.NoError(t, err)

		assert.True(t, CheckPasswordHash(password, hash))
	})

	t.Run("handles unicode in password", func(t *testing.T) {
		password := "pässwörd_日本語"
		hash, err := HashPassword(password)
		require.NoError(t, err)

		assert.True(t, CheckPasswordHash(password, hash))
	})
}

func TestGetBcryptCost(t *testing.T) {
	// Note: getBcryptCost reads from env at package init.
	// The default cost when BCRYPT_COST is unset should be 12.
	// We test the function behavior indirectly through successful hash/check cycles.
	t.Run("hashing works with default cost", func(t *testing.T) {
		hash, err := HashPassword("test")
		require.NoError(t, err)
		assert.True(t, CheckPasswordHash("test", hash))
	})
}
