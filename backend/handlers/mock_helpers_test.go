package handlers

import (
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"investorcenter-api/database"
)

// setupMockDB creates a sqlmock database and assigns it to database.DB.
// Returns the mock for setting expectations and a cleanup function.
// This mirrors the pattern used in task-service/handlers/test_helpers_test.go.
func setupMockDB(t *testing.T) (sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}

	origDB := database.DB
	database.DB = sqlx.NewDb(db, "sqlmock")

	cleanup := func() {
		database.DB = origDB
		db.Close()
	}

	return mock, cleanup
}

// getDatabaseDB returns the current database.DB for later restoration.
func getDatabaseDB() *sqlx.DB {
	return database.DB
}

// setDatabaseDBNil sets database.DB to nil (for testing nil-guard branches).
func setDatabaseDBNil() {
	database.DB = nil
}

// restoreDatabaseDB restores database.DB to a previously saved value.
func restoreDatabaseDB(db *sqlx.DB) {
	database.DB = db
}

// setupMockRouter creates a gin router with a mock auth middleware
// that sets user_id in context (to simulate AuthMiddleware).
func setupMockRouter(userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	return r
}

// setupMockRouterNoAuth creates a gin router without auth middleware.
func setupMockRouterNoAuth() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// ensureJWTSecret sets JWT_SECRET for tests that need token generation.
func ensureJWTSecret(t *testing.T) {
	t.Helper()
	if os.Getenv("JWT_SECRET") == "" {
		os.Setenv("JWT_SECRET", "test-jwt-secret-for-handler-testing-min-32-chars")
		t.Cleanup(func() {
			os.Unsetenv("JWT_SECRET")
		})
	}
}

