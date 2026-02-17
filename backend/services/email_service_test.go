package services

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewEmailService — env-based configuration
// ---------------------------------------------------------------------------

func TestNewEmailService_DefaultsFromEnv(t *testing.T) {
	// Clear all SMTP env vars
	os.Unsetenv("SMTP_HOST")
	os.Unsetenv("SMTP_PORT")
	os.Unsetenv("SMTP_USERNAME")
	os.Unsetenv("SMTP_PASSWORD")
	os.Unsetenv("SMTP_FROM_EMAIL")
	os.Unsetenv("SMTP_FROM_NAME")
	os.Unsetenv("FRONTEND_URL")

	es := NewEmailService()
	require.NotNil(t, es)
	assert.Empty(t, es.smtpHost)
	assert.Empty(t, es.smtpPort)
	assert.Empty(t, es.smtpPassword)
	assert.Empty(t, es.frontendURL)
}

func TestNewEmailService_ReadsEnvVars(t *testing.T) {
	os.Setenv("SMTP_HOST", "smtp.example.com")
	os.Setenv("SMTP_PORT", "587")
	os.Setenv("SMTP_USERNAME", "user")
	os.Setenv("SMTP_PASSWORD", "secret")
	os.Setenv("SMTP_FROM_EMAIL", "noreply@example.com")
	os.Setenv("SMTP_FROM_NAME", "Test App")
	os.Setenv("FRONTEND_URL", "https://app.example.com")
	defer func() {
		os.Unsetenv("SMTP_HOST")
		os.Unsetenv("SMTP_PORT")
		os.Unsetenv("SMTP_USERNAME")
		os.Unsetenv("SMTP_PASSWORD")
		os.Unsetenv("SMTP_FROM_EMAIL")
		os.Unsetenv("SMTP_FROM_NAME")
		os.Unsetenv("FRONTEND_URL")
	}()

	es := NewEmailService()
	require.NotNil(t, es)
	assert.Equal(t, "smtp.example.com", es.smtpHost)
	assert.Equal(t, "587", es.smtpPort)
	assert.Equal(t, "user", es.smtpUsername)
	assert.Equal(t, "secret", es.smtpPassword)
	assert.Equal(t, "noreply@example.com", es.fromEmail)
	assert.Equal(t, "Test App", es.fromName)
	assert.Equal(t, "https://app.example.com", es.frontendURL)
}

// ---------------------------------------------------------------------------
// sendEmail — skips when SMTP not configured
// ---------------------------------------------------------------------------

func TestSendEmail_SkipsWhenNotConfigured(t *testing.T) {
	es := &EmailService{
		smtpHost:     "",
		smtpPassword: "",
	}

	// Should not error — just skips
	err := es.sendEmail("test@example.com", "Test Subject", "<h1>Hello</h1>")
	assert.NoError(t, err)
}

func TestSendEmail_SkipsWhenHostEmpty(t *testing.T) {
	es := &EmailService{
		smtpHost:     "",
		smtpPassword: "some-password",
	}

	err := es.sendEmail("test@example.com", "Test", "<p>body</p>")
	assert.NoError(t, err)
}

func TestSendEmail_SkipsWhenPasswordEmpty(t *testing.T) {
	es := &EmailService{
		smtpHost:     "smtp.example.com",
		smtpPassword: "",
	}

	err := es.sendEmail("test@example.com", "Test", "<p>body</p>")
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// SendVerificationEmail — template generation
// ---------------------------------------------------------------------------

func TestSendVerificationEmail_SkipsWhenNotConfigured(t *testing.T) {
	es := &EmailService{
		smtpHost:    "",
		frontendURL: "https://app.example.com",
	}

	err := es.SendVerificationEmail("user@example.com", "John Doe", "token123")
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// SendPasswordResetEmail — template generation
// ---------------------------------------------------------------------------

func TestSendPasswordResetEmail_SkipsWhenNotConfigured(t *testing.T) {
	es := &EmailService{
		smtpHost:    "",
		frontendURL: "https://app.example.com",
	}

	err := es.SendPasswordResetEmail("user@example.com", "Jane Doe", "reset-token")
	assert.NoError(t, err)
}
