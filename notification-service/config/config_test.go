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
