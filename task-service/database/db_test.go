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
	os.Setenv("DB_HOST", "task-db.example.com")
	os.Setenv("DB_PORT", "5434")
	os.Setenv("DB_USER", "task_user")
	os.Setenv("DB_PASSWORD", "task_secret")
	os.Setenv("DB_NAME", "task_db")
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

	assert.Equal(t, "task-db.example.com", config.Host)
	assert.Equal(t, "5434", config.Port)
	assert.Equal(t, "task_user", config.User)
	assert.Equal(t, "task_secret", config.Password)
	assert.Equal(t, "task_db", config.DBName)
	assert.Equal(t, "disable", config.SSLMode)
}

func TestGetEnvWithDefault_ReturnsEnvValue(t *testing.T) {
	os.Setenv("TEST_TASK_KEY", "custom_value")
	defer os.Unsetenv("TEST_TASK_KEY")

	assert.Equal(t, "custom_value", getEnvWithDefault("TEST_TASK_KEY", "default"))
}

func TestGetEnvWithDefault_ReturnsDefault(t *testing.T) {
	os.Unsetenv("TEST_TASK_KEY_MISS")
	assert.Equal(t, "fallback", getEnvWithDefault("TEST_TASK_KEY_MISS", "fallback"))
}

func TestGetEnvWithDefault_EmptyEnvReturnsDefault(t *testing.T) {
	os.Setenv("TEST_TASK_KEY_EMPTY", "")
	defer os.Unsetenv("TEST_TASK_KEY_EMPTY")

	assert.Equal(t, "fallback", getEnvWithDefault("TEST_TASK_KEY_EMPTY", "fallback"))
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
