package handlers

import (
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"task-service/database"
)

func init() {
	gin.SetMode(gin.TestMode)
	os.Setenv("JWT_SECRET", "test-jwt-secret-for-testing")
}

// setupMockDB creates a sqlmock database and assigns it to database.DB
// Returns the mock for setting expectations and a cleanup function
func setupMockDB(t *testing.T) (sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}

	database.DB = sqlx.NewDb(db, "sqlmock")

	cleanup := func() {
		database.DB = nil
		db.Close()
	}

	return mock, cleanup
}

// setupRouterWithMockAuth creates a gin router with a mock auth middleware
// that sets user_id, user_email, and is_admin in context
func setupRouterWithMockAuth(userID, email string, isAdmin bool) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("user_email", email)
		c.Set("is_admin", isAdmin)
		c.Next()
	})
	return r
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// intPtr returns a pointer to an int
func intPtr(i int) *int {
	return &i
}
