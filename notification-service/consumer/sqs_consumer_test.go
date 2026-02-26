package consumer

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// extractSNSPayload
// ---------------------------------------------------------------------------

func TestExtractSNSPayload_ValidSNSEnvelope(t *testing.T) {
	innerMsg := `{"timestamp":1740000000,"source":"polygon_snapshot","symbols":{"AAPL":{"price":152.30}}}`
	envelope := map[string]string{
		"Type":      "Notification",
		"MessageId": "test-id-123",
		"Message":   innerMsg,
	}
	envelopeJSON, _ := json.Marshal(envelope)
	body := string(envelopeJSON)

	payload, err := extractSNSPayload(&body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the extracted payload is the inner message
	if string(payload) != innerMsg {
		t.Errorf("expected inner message, got %s", string(payload))
	}
}

func TestExtractSNSPayload_RawBody_NotSNSEnvelope(t *testing.T) {
	// If the body is not an SNS envelope, return it as-is
	rawMsg := `{"timestamp":1740000000,"source":"test","symbols":{}}`
	body := rawMsg

	payload, err := extractSNSPayload(&body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should get the raw body back since it's not an SNS envelope (no Message field)
	if string(payload) != rawMsg {
		t.Errorf("expected raw body returned, got %s", string(payload))
	}
}

func TestExtractSNSPayload_EmptyMessageField(t *testing.T) {
	// SNS-like envelope but empty Message field — should return raw body
	envelope := map[string]string{
		"Type":    "Notification",
		"Message": "",
	}
	envelopeJSON, _ := json.Marshal(envelope)
	body := string(envelopeJSON)

	payload, err := extractSNSPayload(&body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty Message means fall back to raw body
	if string(payload) != body {
		t.Errorf("expected raw body for empty Message, got %s", string(payload))
	}
}

func TestExtractSNSPayload_NilBody(t *testing.T) {
	_, err := extractSNSPayload(nil)
	if err == nil {
		t.Error("expected error for nil body")
	}
}

func TestExtractSNSPayload_InvalidJSON(t *testing.T) {
	// If the body is not valid JSON, return it as raw bytes
	body := "this is not json"

	payload, err := extractSNSPayload(&body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(payload) != body {
		t.Errorf("expected raw body for invalid JSON, got %s", string(payload))
	}
}

func TestExtractSNSPayload_NestedJSON(t *testing.T) {
	// Real-world SNS message with nested JSON in the Message field
	innerMsg := `{"timestamp":1740000000,"source":"polygon_snapshot","symbols":{"AAPL":{"price":152.30,"volume":45000000,"change_pct":1.25},"TSLA":{"price":245.80,"volume":32000000,"change_pct":-0.50}}}`
	envelope := map[string]interface{}{
		"Type":             "Notification",
		"MessageId":        "abc-123",
		"TopicArn":         "arn:aws:sns:us-east-1:123456789:investorcenter-price-updates",
		"Message":          innerMsg,
		"Timestamp":        "2025-02-20T12:00:00.000Z",
		"SignatureVersion": "1",
	}
	envelopeJSON, _ := json.Marshal(envelope)
	body := string(envelopeJSON)

	payload, err := extractSNSPayload(&body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(payload) != innerMsg {
		t.Errorf("expected inner message extracted from SNS envelope, got %s", string(payload))
	}

	// Verify the extracted message is valid JSON and parseable
	var priceUpdate struct {
		Timestamp int64                          `json:"timestamp"`
		Source    string                         `json:"source"`
		Symbols   map[string]map[string]float64  `json:"symbols"`
	}
	if err := json.Unmarshal(payload, &priceUpdate); err != nil {
		t.Fatalf("extracted payload is not valid JSON: %v", err)
	}
	if priceUpdate.Timestamp != 1740000000 {
		t.Errorf("timestamp = %d, want 1740000000", priceUpdate.Timestamp)
	}
	if priceUpdate.Source != "polygon_snapshot" {
		t.Errorf("source = %s, want polygon_snapshot", priceUpdate.Source)
	}
	if len(priceUpdate.Symbols) != 2 {
		t.Errorf("expected 2 symbols, got %d", len(priceUpdate.Symbols))
	}
}

func TestExtractSNSPayload_EmptyBody(t *testing.T) {
	body := ""
	payload, err := extractSNSPayload(&body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty string is not valid JSON, so returns raw body
	if string(payload) != "" {
		t.Errorf("expected empty string, got %q", string(payload))
	}
}
