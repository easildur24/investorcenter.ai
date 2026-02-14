package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"investorcenter-api/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// generateTestToken creates a valid JWT for testing middleware
func generateTestToken(t *testing.T, user *models.User) string {
	t.Helper()
	setupTestSecret(t)
	token, err := GenerateAccessToken(user)
	require.NoError(t, err)
	return token
}

func TestAuthMiddleware(t *testing.T) {
	t.Run("passes with valid bearer token", func(t *testing.T) {
		setupTestSecret(t)
		user := createTestUser()
		token := generateTestToken(t, user)

		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		var capturedUserID, capturedEmail string
		var capturedIsAdmin bool

		r.Use(AuthMiddleware())
		r.GET("/test", func(c *gin.Context) {
			uid, _ := c.Get("user_id")
			capturedUserID = uid.(string)
			em, _ := c.Get("user_email")
			capturedEmail = em.(string)
			isAdmin, _ := c.Get("is_admin")
			capturedIsAdmin = isAdmin.(bool)
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, c.Request)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "user-123", capturedUserID)
		assert.Equal(t, "test@example.com", capturedEmail)
		assert.False(t, capturedIsAdmin)
	})

	t.Run("rejects missing authorization header", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.Use(AuthMiddleware())
		r.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "Authorization header required", resp["error"])
	})

	t.Run("rejects invalid format - no Bearer prefix", func(t *testing.T) {
		setupTestSecret(t)
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.Use(AuthMiddleware())
		r.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Token some-token")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp["error"], "Invalid authorization header format")
	})

	t.Run("rejects invalid format - just Bearer", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.Use(AuthMiddleware())
		r.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("rejects expired token", func(t *testing.T) {
		setupTestSecret(t)
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.Use(AuthMiddleware())
		r.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "Invalid or expired token", resp["error"])
	})

	t.Run("sets admin context for admin user", func(t *testing.T) {
		setupTestSecret(t)
		user := createTestUser()
		user.IsAdmin = true
		token := generateTestToken(t, user)

		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		var capturedIsAdmin bool
		r.Use(AuthMiddleware())
		r.GET("/test", func(c *gin.Context) {
			isAdmin, _ := c.Get("is_admin")
			capturedIsAdmin = isAdmin.(bool)
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, capturedIsAdmin)
	})
}

func TestGetUserIDFromContext(t *testing.T) {
	t.Run("returns user ID when set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", "user-456")

		userID, exists := GetUserIDFromContext(c)
		assert.True(t, exists)
		assert.Equal(t, "user-456", userID)
	})

	t.Run("returns empty when not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		userID, exists := GetUserIDFromContext(c)
		assert.False(t, exists)
		assert.Empty(t, userID)
	})
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}
