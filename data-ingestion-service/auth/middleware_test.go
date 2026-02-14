package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func TestAuthMiddleware(t *testing.T) {
	setupTestSecret(t)

	t.Run("passes with valid bearer token", func(t *testing.T) {
		token := generateValidToken(t, "user-123", "test@example.com", false)

		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

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

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "user-123", capturedUserID)
		assert.Equal(t, "test@example.com", capturedEmail)
		assert.False(t, capturedIsAdmin)
	})

	t.Run("rejects missing auth header", func(t *testing.T) {
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

	t.Run("rejects invalid format", func(t *testing.T) {
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
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.Use(AuthMiddleware())
		r.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-jwt-token")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "Invalid or expired token", resp["error"])
	})
}

func TestAdminMiddleware(t *testing.T) {
	t.Run("passes for admin user", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.Use(func(c *gin.Context) {
			c.Set("user_id", "admin-1")
			c.Set("is_admin", true)
			c.Next()
		})
		r.Use(AdminMiddleware())
		r.GET("/admin", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest("GET", "/admin", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("rejects non-admin user", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.Use(func(c *gin.Context) {
			c.Set("user_id", "user-1")
			c.Set("is_admin", false)
			c.Next()
		})
		r.Use(AdminMiddleware())
		r.GET("/admin", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest("GET", "/admin", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("rejects when user_id not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.Use(AdminMiddleware())
		r.GET("/admin", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest("GET", "/admin", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("rejects when is_admin not in context", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.Use(func(c *gin.Context) {
			c.Set("user_id", "user-1")
			c.Next()
		})
		r.Use(AdminMiddleware())
		r.GET("/admin", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest("GET", "/admin", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestWorkerMiddleware(t *testing.T) {
	t.Run("passes through (no-op)", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.Use(WorkerMiddleware())
		r.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
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

func TestAuthAdminIntegration(t *testing.T) {
	setupTestSecret(t)

	t.Run("full chain passes for admin", func(t *testing.T) {
		token := generateValidToken(t, "admin-1", "admin@example.com", true)

		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.Use(AuthMiddleware())
		r.Use(AdminMiddleware())
		r.GET("/admin", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest("GET", "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("full chain rejects non-admin", func(t *testing.T) {
		token := generateValidToken(t, "user-1", "user@example.com", false)

		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.Use(AuthMiddleware())
		r.Use(AdminMiddleware())
		r.GET("/admin", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest("GET", "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}
