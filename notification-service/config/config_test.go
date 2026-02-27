package config

import (
	"encoding/json"
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------
// RedactedString
// ---------------------------------------------------------------------------

func TestRedactedString_String(t *testing.T) {
	r := RedactedString{val: "super-secret-password"}
	got := r.String()
	if got != "[REDACTED]" {
		t.Errorf("String() = %q, want [REDACTED]", got)
	}
}

func TestRedactedString_GoString(t *testing.T) {
	r := RedactedString{val: "super-secret-password"}
	got := r.GoString()
	if got != "[REDACTED]" {
		t.Errorf("GoString() = %q, want [REDACTED]", got)
	}
}

func TestRedactedString_Value(t *testing.T) {
	r := RedactedString{val: "super-secret-password"}
	got := r.Value()
	if got != "super-secret-password" {
		t.Errorf("Value() = %q, want super-secret-password", got)
	}
}

func TestRedactedString_Sprintf(t *testing.T) {
	r := RedactedString{val: "password123"}
	got := fmt.Sprintf("%v", r)
	if got != "[REDACTED]" {
		t.Errorf("Sprintf(%%v) = %q, want [REDACTED]", got)
	}
}

func TestRedactedString_SprintfPlus(t *testing.T) {
	r := RedactedString{val: "password123"}
	got := fmt.Sprintf("%+v", r)
	if got != "[REDACTED]" {
		t.Errorf("Sprintf(%%+v) = %q, want [REDACTED]", got)
	}
}

func TestRedactedString_MarshalJSON(t *testing.T) {
	r := RedactedString{val: "password123"}
	got, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != `"[REDACTED]"` {
		t.Errorf("MarshalJSON() = %s, want \"[REDACTED]\"", string(got))
	}
}

func TestRedactedString_EmptyValue(t *testing.T) {
	r := RedactedString{val: ""}
	if r.Value() != "" {
		t.Errorf("Value() should return empty string")
	}
	if r.String() != "[REDACTED]" {
		t.Errorf("String() should still return [REDACTED] even for empty values")
	}
}

// ---------------------------------------------------------------------------
// Load
// ---------------------------------------------------------------------------

func TestLoad_Defaults(t *testing.T) {
	cfg := Load()

	if cfg.Port != "8003" {
		t.Errorf("Port = %s, want 8003", cfg.Port)
	}
	if cfg.AWSRegion != "us-east-1" {
		t.Errorf("AWSRegion = %s, want us-east-1", cfg.AWSRegion)
	}
	if cfg.DBSSLMode != "require" {
		t.Errorf("DBSSLMode = %s, want require", cfg.DBSSLMode)
	}
	if cfg.SQSMaxMessages != 1 {
		t.Errorf("SQSMaxMessages = %d, want 1", cfg.SQSMaxMessages)
	}
	if cfg.SMTPPassword.Value() != "" {
		t.Errorf("SMTPPassword should default to empty")
	}
	if cfg.SMTPPassword.String() != "[REDACTED]" {
		t.Errorf("SMTPPassword.String() = %s, want [REDACTED]", cfg.SMTPPassword.String())
	}
}

func TestLoad_WithEnvVars(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("AWS_REGION", "eu-west-1")
	t.Setenv("SQS_QUEUE_URL", "https://sqs.eu-west-1.amazonaws.com/123456789/my-queue")
	t.Setenv("DB_HOST", "db.example.com")
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PASSWORD", "s3cret!")
	t.Setenv("CANARY_TOKEN", "canary-tok-42")
	t.Setenv("FRONTEND_URL", "https://app.example.com")

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("Port = %s, want 9090", cfg.Port)
	}
	if cfg.AWSRegion != "eu-west-1" {
		t.Errorf("AWSRegion = %s, want eu-west-1", cfg.AWSRegion)
	}
	if cfg.SQSQueueURL != "https://sqs.eu-west-1.amazonaws.com/123456789/my-queue" {
		t.Errorf("SQSQueueURL = %s, want https://sqs.eu-west-1.amazonaws.com/123456789/my-queue", cfg.SQSQueueURL)
	}
	if cfg.DBHost != "db.example.com" {
		t.Errorf("DBHost = %s, want db.example.com", cfg.DBHost)
	}
	if cfg.SMTPHost != "smtp.example.com" {
		t.Errorf("SMTPHost = %s, want smtp.example.com", cfg.SMTPHost)
	}
	if cfg.SMTPPassword.Value() != "s3cret!" {
		t.Errorf("SMTPPassword.Value() = %s, want s3cret!", cfg.SMTPPassword.Value())
	}
	if cfg.CanaryToken != "canary-tok-42" {
		t.Errorf("CanaryToken = %s, want canary-tok-42", cfg.CanaryToken)
	}
	if cfg.FrontendURL != "https://app.example.com" {
		t.Errorf("FrontendURL = %s, want https://app.example.com", cfg.FrontendURL)
	}
}

func TestLoad_SQSMaxMessages_Valid(t *testing.T) {
	t.Setenv("SQS_MAX_MESSAGES", "5")

	cfg := Load()

	if cfg.SQSMaxMessages != 5 {
		t.Errorf("SQSMaxMessages = %d, want 5", cfg.SQSMaxMessages)
	}
}

func TestLoad_SQSMaxMessages_Invalid(t *testing.T) {
	t.Setenv("SQS_MAX_MESSAGES", "abc")

	cfg := Load()

	if cfg.SQSMaxMessages != 1 {
		t.Errorf("SQSMaxMessages = %d, want 1 (fallback for non-numeric)", cfg.SQSMaxMessages)
	}
}

func TestLoad_SQSMaxMessages_OutOfRange(t *testing.T) {
	t.Setenv("SQS_MAX_MESSAGES", "20")

	cfg := Load()

	if cfg.SQSMaxMessages != 1 {
		t.Errorf("SQSMaxMessages = %d, want 1 (fallback for out-of-range)", cfg.SQSMaxMessages)
	}
}

func TestLoad_SQSMaxMessages_Zero(t *testing.T) {
	t.Setenv("SQS_MAX_MESSAGES", "0")

	cfg := Load()

	if cfg.SQSMaxMessages != 1 {
		t.Errorf("SQSMaxMessages = %d, want 1 (fallback for zero)", cfg.SQSMaxMessages)
	}
}
