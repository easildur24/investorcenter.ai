package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("securePassword123")
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "securePassword123", hash)
}

func TestHashPassword_DifferentHashesForSamePassword(t *testing.T) {
	hash1, err := HashPassword("samePassword")
	assert.NoError(t, err)

	hash2, err := HashPassword("samePassword")
	assert.NoError(t, err)

	// bcrypt produces different hashes each time due to random salt
	assert.NotEqual(t, hash1, hash2)
}

func TestCheckPasswordHash_Valid(t *testing.T) {
	password := "testPassword123"
	hash, err := HashPassword(password)
	assert.NoError(t, err)

	assert.True(t, CheckPasswordHash(password, hash))
}

func TestCheckPasswordHash_Invalid(t *testing.T) {
	password := "testPassword123"
	hash, err := HashPassword(password)
	assert.NoError(t, err)

	assert.False(t, CheckPasswordHash("wrongPassword", hash))
}

func TestCheckPasswordHash_EmptyPassword(t *testing.T) {
	hash, err := HashPassword("somePassword")
	assert.NoError(t, err)

	assert.False(t, CheckPasswordHash("", hash))
}

func TestCheckPasswordHash_InvalidHash(t *testing.T) {
	assert.False(t, CheckPasswordHash("password", "not-a-valid-hash"))
}
