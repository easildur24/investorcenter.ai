package database

import (
	"fmt"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// skipIfNoTestDB skips the test if INTEGRATION_TEST_DB is not set.
// This allows integration tests to run in CI (with PostgreSQL service container)
// while skipping gracefully in local dev without a database.
func skipIfNoTestDB(t *testing.T) {
	t.Helper()
	if os.Getenv("INTEGRATION_TEST_DB") != "true" {
		t.Skip("Skipping integration test: INTEGRATION_TEST_DB not set")
	}
}

// setupTestDB connects to the test database, runs the schema, swaps
// database.DB to point at the test DB, and registers cleanup to restore
// the original DB and drop tables.
func setupTestDB(t *testing.T) {
	t.Helper()
	skipIfNoTestDB(t)

	host := getEnvOrDefault("DB_HOST", "localhost")
	port := getEnvOrDefault("DB_PORT", "5432")
	user := getEnvOrDefault("DB_USER", "testuser")
	pass := getEnvOrDefault("DB_PASSWORD", "testpass")
	name := getEnvOrDefault("DB_NAME", "investorcenter_test")
	sslmode := getEnvOrDefault("DB_SSLMODE", "disable")

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pass, name, sslmode,
	)

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		t.Fatalf("failed to connect to test DB: %v", err)
	}

	// Read and execute schema
	schema, err := os.ReadFile("testdata/schema_test.sql")
	if err != nil {
		db.Close()
		t.Fatalf("failed to read test schema: %v", err)
	}
	_, err = db.Exec(string(schema))
	if err != nil {
		db.Close()
		t.Fatalf("failed to execute test schema: %v", err)
	}

	// Save original DB and swap
	origDB := DB
	DB = db

	t.Cleanup(func() {
		// Truncate all tables instead of dropping — other test
		// packages (e.g. handlers) may run concurrently against
		// the same database, and DROP TABLE would cause races.
		// Schema uses CREATE TABLE IF NOT EXISTS, so tables
		// persist across tests without issue.
		db.Exec(`TRUNCATE
			tickers, users, watch_lists, watch_list_items, screener_data,
			financial_statements, eps_estimates, valuation_ratios, fundamental_metrics_extended,
			mv_latest_sector_percentiles, alert_rules, alert_logs, sessions, password_reset_tokens,
			notification_preferences, notification_queue, sentiment_lexicon, social_posts,
			reddit_heatmap_daily, heatmap_configs, subscription_plans, user_subscriptions
			CASCADE`)
		db.Close()
		DB = origDB
	})
}

// cleanTables truncates all test tables for isolation between tests.
func cleanTables(t *testing.T) {
	t.Helper()
	DB.MustExec(`TRUNCATE
		tickers, users, watch_lists, watch_list_items, screener_data,
		financial_statements, eps_estimates, valuation_ratios, fundamental_metrics_extended,
		mv_latest_sector_percentiles, alert_rules, alert_logs, sessions, password_reset_tokens,
		notification_preferences, notification_queue, sentiment_lexicon, social_posts,
		reddit_heatmap_daily, heatmap_configs, subscription_plans, user_subscriptions
		CASCADE`)
}

// getEnvOrDefault returns environment variable value or a default.
func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
