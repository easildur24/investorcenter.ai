package canary

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"notification-service/config"
)

// Handler provides HTTP endpoints for canary health checks.
type Handler struct {
	cfg         *config.Config
	canaryToken string
	sendFn      func(to, name string) error // injectable for testing
}

// NewHandler creates a canary Handler.
// token is the shared secret that callers must provide in the Authorization header.
func NewHandler(cfg *config.Config, token string) *Handler {
	h := &Handler{cfg: cfg, canaryToken: token}
	h.sendFn = h.sendTestEmail
	return h
}

// emailRequest is the JSON body for POST /canary/email.
type emailRequest struct {
	To   string `json:"to"`
	Name string `json:"name"`
}

// emailResponse is the JSON response from POST /canary/email.
type emailResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	SentTo   string `json:"sent_to,omitempty"`
	Duration string `json:"duration,omitempty"`
}

// HandleEmail sends a test email to verify SMTP connectivity and delivery.
// Protected by Bearer token authentication.
//
//	POST /canary/email
//	Authorization: Bearer <CANARY_TOKEN>
//	{"to": "user@example.com", "name": "Test User"}
func (h *Handler) HandleEmail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Only allow POST
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(emailResponse{
			Status:  "error",
			Message: "method not allowed",
		})
		return
	}

	// Authenticate
	if !h.authenticate(r) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(emailResponse{
			Status:  "error",
			Message: "unauthorized — provide Authorization: Bearer <CANARY_TOKEN>",
		})
		return
	}

	// Parse request
	var req emailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(emailResponse{
			Status:  "error",
			Message: "invalid JSON body: " + err.Error(),
		})
		return
	}

	if req.To == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(emailResponse{
			Status:  "error",
			Message: "\"to\" field is required",
		})
		return
	}
	if req.Name == "" {
		req.Name = "Canary Test"
	}

	// Check SMTP configuration
	if h.cfg.SMTPHost == "" || h.cfg.SMTPPassword.Value() == "" {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(emailResponse{
			Status:  "error",
			Message: "SMTP not configured",
		})
		return
	}

	// Send test email
	start := time.Now()
	err := h.sendFn(req.To, req.Name)
	duration := time.Since(start)

	if err != nil {
		log.Printf("Canary email failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(emailResponse{
			Status:   "error",
			Message:  "email delivery failed: " + err.Error(),
			Duration: duration.String(),
		})
		return
	}

	log.Printf("Canary email sent to %s in %s", req.To, duration)
	json.NewEncoder(w).Encode(emailResponse{
		Status:   "ok",
		Message:  "canary email sent successfully",
		SentTo:   req.To,
		Duration: duration.String(),
	})
}

// sendTestEmail delivers a lightweight test email via SMTP.
func (h *Handler) sendTestEmail(to, name string) error {
	auth := smtp.PlainAuth("", h.cfg.SMTPUsername, h.cfg.SMTPPassword.Value(), h.cfg.SMTPHost)

	safeTo := sanitizeHeader(to)
	safeFrom := sanitizeHeader(h.cfg.SMTPFromEmail)
	safeFromName := sanitizeHeader(h.cfg.SMTPFromName)

	subject := "Canary Test — InvestorCenter Notifications"
	body := formatCanaryEmailBody(name, time.Now().UTC())

	msg := fmt.Sprintf(
		"From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		safeFromName, safeFrom, safeTo, subject, body,
	)

	addr := fmt.Sprintf("%s:%s", h.cfg.SMTPHost, h.cfg.SMTPPort)
	return smtp.SendMail(addr, auth, h.cfg.SMTPFromEmail, []string{safeTo}, []byte(msg))
}

// authenticate checks the Authorization header for a valid Bearer token.
func (h *Handler) authenticate(r *http.Request) bool {
	if h.canaryToken == "" {
		return false // No token configured — deny all
	}
	auth := r.Header.Get("Authorization")
	return auth == "Bearer "+h.canaryToken
}

// sanitizeHeader strips CR and LF characters to prevent header injection.
func sanitizeHeader(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}

// formatCanaryEmailBody generates a minimal HTML email for canary tests.
func formatCanaryEmailBody(name string, sentAt time.Time) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="background: #1a1a2e; color: #e0e0e0; padding: 24px; border-radius: 8px;">
    <h2 style="color: #4fc3f7; margin-top: 0;">Canary Test</h2>
    <p>Hi %s,</p>
    <p>This is a canary test email from the InvestorCenter notification service.
       If you received this, email delivery is working correctly.</p>
    <div style="background: #16213e; padding: 16px; border-radius: 6px; margin: 16px 0;">
      <table style="width: 100%%; border-collapse: collapse; color: #e0e0e0;">
        <tr>
          <td style="padding: 8px 0;"><strong>Service</strong></td>
          <td style="padding: 8px 0; text-align: right;">notification-service</td>
        </tr>
        <tr>
          <td style="padding: 8px 0;"><strong>Sent At (UTC)</strong></td>
          <td style="padding: 8px 0; text-align: right;">%s</td>
        </tr>
        <tr>
          <td style="padding: 8px 0;"><strong>Status</strong></td>
          <td style="padding: 8px 0; text-align: right; color: #66bb6a;">PASS</td>
        </tr>
      </table>
    </div>
    <hr style="border: none; border-top: 1px solid #333; margin: 20px 0;">
    <p style="color: #888; font-size: 12px;">
      This is an automated canary test. No action required.
    </p>
  </div>
</body>
</html>`, name, sentAt.Format("2006-01-02 15:04:05 UTC"))
}
