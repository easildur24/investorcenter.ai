package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAdminMiddleware(t *testing.T) {
	t.Run("passes for admin user", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.Use(func(c *gin.Context) {
			// Simulate AuthMiddleware setting context
			c.Set("user_id", "admin-user-1")
			c.Set("user_email", "admin@example.com")
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

	t.Run("sets admin_action flag", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		var adminAction bool
		r.Use(func(c *gin.Context) {
			c.Set("user_id", "admin-user-1")
			c.Set("is_admin", true)
			c.Next()
		})
		r.Use(AdminMiddleware())
		r.GET("/admin", func(c *gin.Context) {
			val, exists := c.Get("admin_action")
			adminAction = exists && val.(bool)
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest("GET", "/admin", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, adminAction)
	})

	t.Run("rejects non-admin user", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.Use(func(c *gin.Context) {
			c.Set("user_id", "regular-user")
			c.Set("user_email", "user@example.com")
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

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "Forbidden - admin access required", resp["error"])
	})

	t.Run("rejects when user_id is not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		// No AuthMiddleware — user_id not set
		r.Use(AdminMiddleware())
		r.GET("/admin", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest("GET", "/admin", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "Unauthorized - authentication required", resp["error"])
	})

	t.Run("rejects when is_admin is not in context", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.Use(func(c *gin.Context) {
			c.Set("user_id", "user-with-no-admin-flag")
			// Deliberately not setting is_admin
			c.Next()
		})
		r.Use(AdminMiddleware())
		r.GET("/admin", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		req, _ := http.NewRequest("GET", "/admin", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp["debug"], "is_admin not found in context")
	})

	t.Run("rejects when is_admin is not a bool", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)

		r.Use(func(c *gin.Context) {
			c.Set("user_id", "user-bad-admin")
			c.Set("is_admin", "not-a-bool")
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

	t.Run("integration with AuthMiddleware chain", func(t *testing.T) {
		setupTestSecret(t)

		// Create an admin user token
		user := createTestUser()
		user.IsAdmin = true
		token := generateTestToken(t, user)

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

	t.Run("integration rejects non-admin through full chain", func(t *testing.T) {
		setupTestSecret(t)

		// Create a non-admin user token
		user := createTestUser()
		user.IsAdmin = false
		token := generateTestToken(t, user)

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
