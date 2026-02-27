package database

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfigFromEnv_Defaults(t *testing.T) {
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_PORT")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("DB_NAME")
	os.Unsetenv("DB_SSLMODE")

	config := LoadConfigFromEnv()

	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, "5432", config.Port)
	assert.Equal(t, "investorcenter", config.User)
	assert.Equal(t, "", config.Password)
	assert.Equal(t, "investorcenter_db", config.DBName)
	assert.Equal(t, "require", config.SSLMode)
}

func TestLoadConfigFromEnv_CustomValues(t *testing.T) {
	os.Setenv("DB_HOST", "prod-db.example.com")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_USER", "prod_user")
	os.Setenv("DB_PASSWORD", "secret123")
	os.Setenv("DB_NAME", "prod_db")
	os.Setenv("DB_SSLMODE", "verify-full")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_SSLMODE")
	}()

	config := LoadConfigFromEnv()

	assert.Equal(t, "prod-db.example.com", config.Host)
	assert.Equal(t, "5433", config.Port)
	assert.Equal(t, "prod_user", config.User)
	assert.Equal(t, "secret123", config.Password)
	assert.Equal(t, "prod_db", config.DBName)
	assert.Equal(t, "verify-full", config.SSLMode)
}

func TestGetEnvWithDefault_ReturnsEnvValue(t *testing.T) {
	os.Setenv("TEST_DB_KEY_1", "custom")
	defer os.Unsetenv("TEST_DB_KEY_1")

	assert.Equal(t, "custom", getEnvWithDefault("TEST_DB_KEY_1", "default"))
}

func TestGetEnvWithDefault_ReturnsDefault(t *testing.T) {
	os.Unsetenv("TEST_DB_KEY_MISS")
	assert.Equal(t, "fallback", getEnvWithDefault("TEST_DB_KEY_MISS", "fallback"))
}

func TestGetEnvWithDefault_EmptyEnvReturnsDefault(t *testing.T) {
	os.Setenv("TEST_DB_KEY_EMPTY", "")
	defer os.Unsetenv("TEST_DB_KEY_EMPTY")

	assert.Equal(t, "fallback", getEnvWithDefault("TEST_DB_KEY_EMPTY", "fallback"))
}

func TestClose_NilDB(t *testing.T) {
	origDB := DB
	DB = nil
	defer func() { DB = origDB }()

	err := Close()
	assert.NoError(t, err)
}

func TestHealthCheck_NilDB(t *testing.T) {
	origDB := DB
	DB = nil
	defer func() { DB = origDB }()

	err := HealthCheck()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}
