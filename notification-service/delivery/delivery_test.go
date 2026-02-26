package delivery

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Router construction
// ---------------------------------------------------------------------------

func TestNewRouter(t *testing.T) {
	// Verify router can be constructed with nil dependencies (for unit testing).
	// In production this would be a real EmailDelivery instance.
	router := NewRouter(nil)
	if router == nil {
		t.Fatal("expected non-nil router")
	}
	if router.email != nil {
		t.Error("expected nil email field")
	}
}

func TestNewRouter_WithEmail(t *testing.T) {
	email := &EmailDelivery{}
	router := NewRouter(email)
	if router.email != email {
		t.Error("email delivery not set correctly")
	}
}
