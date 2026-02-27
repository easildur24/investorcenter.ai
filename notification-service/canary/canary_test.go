package canary

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"notification-service/config"
)

func TestHandleEmail_MethodNotAllowed(t *testing.T) {
	h := NewHandler(&config.Config{}, "test-token")

	req := httptest.NewRequest(http.MethodGet, "/canary/email", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	h.HandleEmail(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleEmail_Unauthorized_NoToken(t *testing.T) {
	h := NewHandler(&config.Config{}, "test-token")

	body := bytes.NewBufferString(`{"to":"user@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/canary/email", body)
	w := httptest.NewRecorder()

	h.HandleEmail(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleEmail_Unauthorized_WrongToken(t *testing.T) {
	h := NewHandler(&config.Config{}, "test-token")

	body := bytes.NewBufferString(`{"to":"user@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/canary/email", body)
	req.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()

	h.HandleEmail(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleEmail_Unauthorized_EmptyCanaryToken(t *testing.T) {
	// If CANARY_TOKEN is not configured, all requests should be denied
	h := NewHandler(&config.Config{}, "")

	body := bytes.NewBufferString(`{"to":"user@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/canary/email", body)
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()

	h.HandleEmail(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleEmail_BadJSON(t *testing.T) {
	h := NewHandler(&config.Config{}, "test-token")

	body := bytes.NewBufferString(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/canary/email", body)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	h.HandleEmail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleEmail_MissingTo(t *testing.T) {
	h := NewHandler(&config.Config{}, "test-token")

	body := bytes.NewBufferString(`{"name":"Test"}`)
	req := httptest.NewRequest(http.MethodPost, "/canary/email", body)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	h.HandleEmail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp emailResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Message != "\"to\" field is required" {
		t.Errorf("unexpected message: %s", resp.Message)
	}
}

func TestHandleEmail_SMTPNotConfigured(t *testing.T) {
	cfg := &config.Config{
		SMTPHost: "", // Not configured
	}
	h := NewHandler(cfg, "test-token")

	body := bytes.NewBufferString(`{"to":"user@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/canary/email", body)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	h.HandleEmail(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}

	var resp emailResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Message != "SMTP not configured" {
		t.Errorf("unexpected message: %s", resp.Message)
	}
}

func TestSanitizeHeader(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal string", "normal string"},
		{"has\rnewline", "hasnewline"},
		{"has\nnewline", "hasnewline"},
		{"has\r\nboth", "hasboth"},
		{"clean@email.com", "clean@email.com"},
	}

	for _, tc := range tests {
		result := sanitizeHeader(tc.input)
		if result != tc.expected {
			t.Errorf("sanitizeHeader(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestFormatCanaryEmailBody(t *testing.T) {
	ts := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	body := formatCanaryEmailBody("Test User", ts)
	if body == "" {
		t.Error("expected non-empty email body")
	}
	if !bytes.Contains([]byte(body), []byte("Canary Test")) {
		t.Error("expected body to contain 'Canary Test'")
	}
	if !bytes.Contains([]byte(body), []byte("Test User")) {
		t.Error("expected body to contain recipient name")
	}
	if !bytes.Contains([]byte(body), []byte("PASS")) {
		t.Error("expected body to contain PASS status")
	}
}

func TestAuthenticate(t *testing.T) {
	h := NewHandler(&config.Config{}, "secret123")

	tests := []struct {
		name   string
		header string
		want   bool
	}{
		{"valid token", "Bearer secret123", true},
		{"wrong token", "Bearer wrong", false},
		{"no header", "", false},
		{"no bearer prefix", "secret123", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			if got := h.authenticate(req); got != tc.want {
				t.Errorf("authenticate() = %v, want %v", got, tc.want)
			}
		})
	}
}
