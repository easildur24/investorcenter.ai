package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"notification-service/canary"
	"notification-service/config"
	"notification-service/consumer"
	"notification-service/database"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// newTestDB creates a *database.DB backed by go-sqlmock.
// The caller should defer mock.ExpectClose().
func newTestDB(t *testing.T) (*database.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	return &database.DB{DB: db}, mock
}

// newTestConsumer creates a *consumer.Consumer suitable for testing.
// consumer.New loads AWS config lazily (no network call), so this
// succeeds even without real AWS credentials.
func newTestConsumer(t *testing.T) *consumer.Consumer {
	t.Helper()
	c, err := consumer.New("https://sqs.us-east-1.amazonaws.com/000000000/test", "us-east-1", 1)
	if err != nil {
		t.Fatalf("failed to create consumer: %v", err)
	}
	return c
}

// TestStartHealthServer_HealthEndpoint_AllHealthy verifies that the /health
// endpoint returns 200 with status "ok" when both DB and SQS consumer are healthy.
func TestStartHealthServer_HealthEndpoint_AllHealthy(t *testing.T) {
	db, mock := newTestDB(t)
	defer db.Close()

	// Expect the Ping from the health check
	mock.ExpectPing()

	sqsConsumer := newTestConsumer(t)
	canaryHandler := canary.NewHandler(&config.Config{}, "test-token")

	port := "19876"
	srv := startHealthServer(port, db, sqsConsumer, canaryHandler)
	defer srv.Shutdown(context.Background())

	// Give the server a moment to start listening
	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/health", port))
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", body["status"])
	}
	if body["db"] != "connected" {
		t.Errorf("expected db=connected, got %q", body["db"])
	}
	if body["sqs"] != "polling" {
		t.Errorf("expected sqs=polling, got %q", body["sqs"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestStartHealthServer_HealthEndpoint_DBUnhealthy verifies that the /health
// endpoint returns 503 with status "degraded" when the DB ping fails.
func TestStartHealthServer_HealthEndpoint_DBUnhealthy(t *testing.T) {
	db, mock := newTestDB(t)
	defer db.Close()

	// Simulate DB ping failure
	mock.ExpectPing().WillReturnError(fmt.Errorf("connection refused"))

	sqsConsumer := newTestConsumer(t)
	canaryHandler := canary.NewHandler(&config.Config{}, "test-token")

	port := "19877"
	srv := startHealthServer(port, db, sqsConsumer, canaryHandler)
	defer srv.Shutdown(context.Background())

	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/health", port))
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["status"] != "degraded" {
		t.Errorf("expected status=degraded, got %q", body["status"])
	}
	if body["db"] != "error" {
		t.Errorf("expected db=error, got %q", body["db"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestStartHealthServer_HealthEndpoint_ContentType verifies the /health
// endpoint returns application/json content type.
func TestStartHealthServer_HealthEndpoint_ContentType(t *testing.T) {
	db, mock := newTestDB(t)
	defer db.Close()

	mock.ExpectPing()

	sqsConsumer := newTestConsumer(t)
	canaryHandler := canary.NewHandler(&config.Config{}, "test-token")

	port := "19878"
	srv := startHealthServer(port, db, sqsConsumer, canaryHandler)
	defer srv.Shutdown(context.Background())

	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/health", port))
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

// TestStartHealthServer_CanaryEndpoint_MethodNotAllowed verifies the
// /canary/email endpoint rejects GET requests.
func TestStartHealthServer_CanaryEndpoint_MethodNotAllowed(t *testing.T) {
	db, mock := newTestDB(t)
	defer db.Close()

	// No DB ping expected for canary endpoint
	_ = mock

	sqsConsumer := newTestConsumer(t)
	canaryHandler := canary.NewHandler(&config.Config{}, "test-token")

	port := "19879"
	srv := startHealthServer(port, db, sqsConsumer, canaryHandler)
	defer srv.Shutdown(context.Background())

	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/canary/email", port))
	if err != nil {
		t.Fatalf("GET /canary/email failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", resp.StatusCode)
	}
}

// TestStartHealthServer_UnknownPath returns 404 for unknown paths.
func TestStartHealthServer_UnknownPath(t *testing.T) {
	db, mock := newTestDB(t)
	defer db.Close()

	_ = mock

	sqsConsumer := newTestConsumer(t)
	canaryHandler := canary.NewHandler(&config.Config{}, "test-token")

	port := "19880"
	srv := startHealthServer(port, db, sqsConsumer, canaryHandler)
	defer srv.Shutdown(context.Background())

	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/nonexistent", port))
	if err != nil {
		t.Fatalf("GET /nonexistent failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}

// TestStartHealthServer_Shutdown verifies the server shuts down cleanly.
func TestStartHealthServer_Shutdown(t *testing.T) {
	db, mock := newTestDB(t)
	defer db.Close()

	mock.ExpectPing()

	sqsConsumer := newTestConsumer(t)
	canaryHandler := canary.NewHandler(&config.Config{}, "test-token")

	port := "19881"
	srv := startHealthServer(port, db, sqsConsumer, canaryHandler)

	time.Sleep(50 * time.Millisecond)

	// Verify server is running
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/health", port))
	if err != nil {
		t.Fatalf("GET /health failed before shutdown: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 before shutdown, got %d", resp.StatusCode)
	}

	// Shut down
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	// Verify server is no longer accepting connections
	_, err = http.Get(fmt.Sprintf("http://127.0.0.1:%s/health", port))
	if err == nil {
		t.Error("expected connection error after shutdown, but request succeeded")
	}
}

// TestStartHealthServer_HealthEndpoint_BothUnhealthy verifies that the /health
// endpoint returns 503 when both DB and SQS consumer are unhealthy.
func TestStartHealthServer_HealthEndpoint_BothUnhealthy(t *testing.T) {
	db, mock := newTestDB(t)
	defer db.Close()

	// Simulate DB ping failure
	mock.ExpectPing().WillReturnError(fmt.Errorf("db down"))

	// Create consumer and start+stop it to mark unhealthy.
	// Consumer.Start marks healthy=false on context cancel.
	sqsConsumer := newTestConsumer(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	// Start will see the cancelled context and set healthy=false
	sqsConsumer.Start(ctx, func(msg []byte) error { return nil })

	canaryHandler := canary.NewHandler(&config.Config{}, "test-token")

	port := "19882"
	srv := startHealthServer(port, db, sqsConsumer, canaryHandler)
	defer srv.Shutdown(context.Background())

	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/health", port))
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["status"] != "degraded" {
		t.Errorf("expected status=degraded, got %q", body["status"])
	}
	if body["db"] != "error" {
		t.Errorf("expected db=error, got %q", body["db"])
	}
	if body["sqs"] != "error" {
		t.Errorf("expected sqs=error, got %q", body["sqs"])
	}
}
