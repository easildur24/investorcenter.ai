package delivery

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Router construction
// ---------------------------------------------------------------------------

func TestNewRouter(t *testing.T) {
	// Verify router can be constructed with nil dependencies (for unit testing)
	// In production these would be real EmailDelivery and InAppDelivery instances.
	router := NewRouter(nil, nil)
	if router == nil {
		t.Fatal("expected non-nil router")
	}
	if router.email != nil {
		t.Error("expected nil email field")
	}
	if router.inApp != nil {
		t.Error("expected nil inApp field")
	}
}

func TestNewRouter_WithDeliveries(t *testing.T) {
	email := &EmailDelivery{}
	inApp := &InAppDelivery{}
	router := NewRouter(email, inApp)
	if router.email != email {
		t.Error("email delivery not set correctly")
	}
	if router.inApp != inApp {
		t.Error("inApp delivery not set correctly")
	}
}
