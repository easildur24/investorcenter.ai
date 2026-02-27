package database

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfigFromEnv_Defaults(t *testing.T) {
	// Clear env vars to test defaults
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
	os.Setenv("DB_HOST", "custom-host")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_USER", "custom_user")
	os.Setenv("DB_PASSWORD", "secret123")
	os.Setenv("DB_NAME", "custom_db")
	os.Setenv("DB_SSLMODE", "disable")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_SSLMODE")
	}()

	config := LoadConfigFromEnv()

	assert.Equal(t, "custom-host", config.Host)
	assert.Equal(t, "5433", config.Port)
	assert.Equal(t, "custom_user", config.User)
	assert.Equal(t, "secret123", config.Password)
	assert.Equal(t, "custom_db", config.DBName)
	assert.Equal(t, "disable", config.SSLMode)
}

func TestGetEnvWithDefault_ReturnsEnvValue(t *testing.T) {
	os.Setenv("TEST_KEY_123", "custom_value")
	defer os.Unsetenv("TEST_KEY_123")

	result := getEnvWithDefault("TEST_KEY_123", "default")
	assert.Equal(t, "custom_value", result)
}

func TestGetEnvWithDefault_ReturnsDefault(t *testing.T) {
	os.Unsetenv("TEST_KEY_MISSING")

	result := getEnvWithDefault("TEST_KEY_MISSING", "default_value")
	assert.Equal(t, "default_value", result)
}

func TestGetEnvWithDefault_EmptyEnvReturnsDefault(t *testing.T) {
	os.Setenv("TEST_KEY_EMPTY", "")
	defer os.Unsetenv("TEST_KEY_EMPTY")

	result := getEnvWithDefault("TEST_KEY_EMPTY", "fallback")
	assert.Equal(t, "fallback", result)
}

func TestClose_NilDB(t *testing.T) {
	DB = nil
	err := Close()
	assert.NoError(t, err)
}

func TestHealthCheck_NilDB(t *testing.T) {
	DB = nil
	err := HealthCheck()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}
