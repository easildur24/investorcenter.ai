package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupMiddlewareTest() {
	os.Setenv("JWT_SECRET", testSecret)
	jwtSecret = []byte(testSecret)
}

func makeValidToken(userID, email string, isAdmin bool) string {
	claims := Claims{
		UserID:  userID,
		Email:   email,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token, _ := generateTestToken(claims, testSecret)
	return token
}

func TestAuthMiddleware_NoAuthorizationHeader(t *testing.T) {
	setupMiddlewareTest()

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/test", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Authorization header required", resp["error"])
}

func TestAuthMiddleware_InvalidFormat_NoBearerPrefix(t *testing.T) {
	setupMiddlewareTest()

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/test", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Token abc123")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "Invalid authorization header format")
}

func TestAuthMiddleware_InvalidFormat_NoSpace(t *testing.T) {
	setupMiddlewareTest()

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/test", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "BearerNoSpace")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	setupMiddlewareTest()

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/test", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Invalid or expired token", resp["error"])
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	setupMiddlewareTest()

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	var capturedUserID, capturedEmail string
	var capturedIsAdmin bool

	r.GET("/test", AuthMiddleware(), func(c *gin.Context) {
		uid, _ := c.Get("user_id")
		capturedUserID = uid.(string)
		em, _ := c.Get("user_email")
		capturedEmail = em.(string)
		isAdmin, _ := c.Get("is_admin")
		capturedIsAdmin = isAdmin.(bool)
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	token := makeValidToken("user-123", "test@example.com", false)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user-123", capturedUserID)
	assert.Equal(t, "test@example.com", capturedEmail)
	assert.False(t, capturedIsAdmin)
}

func TestAuthMiddleware_ValidAdminToken(t *testing.T) {
	setupMiddlewareTest()

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	var capturedIsAdmin bool

	r.GET("/test", AuthMiddleware(), func(c *gin.Context) {
		isAdmin, _ := c.Get("is_admin")
		capturedIsAdmin = isAdmin.(bool)
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	token := makeValidToken("admin-456", "admin@example.com", true)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, capturedIsAdmin)
}

// ==================== AdminMiddleware Tests ====================

func TestAdminMiddleware_NoUserID(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	// No auth middleware — user_id not set
	r.GET("/test", AdminMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminMiddleware_NotAdmin(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	// Simulate AuthMiddleware setting user as non-admin
	r.GET("/test", func(c *gin.Context) {
		c.Set("user_id", "user-123")
		c.Set("is_admin", false)
		c.Next()
	}, AdminMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAdminMiddleware_IsAdmin(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/test", func(c *gin.Context) {
		c.Set("user_id", "admin-456")
		c.Set("is_admin", true)
		c.Next()
	}, AdminMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminMiddleware_IsAdminNotBool(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	// is_admin set to a non-bool value
	r.GET("/test", func(c *gin.Context) {
		c.Set("user_id", "user-123")
		c.Set("is_admin", "yes")
		c.Next()
	}, AdminMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ==================== GetUserIDFromContext Tests ====================

func TestGetUserIDFromContext_Exists(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", "user-123")

	userID, ok := GetUserIDFromContext(c)
	assert.True(t, ok)
	assert.Equal(t, "user-123", userID)
}

func TestGetUserIDFromContext_NotExists(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	userID, ok := GetUserIDFromContext(c)
	assert.False(t, ok)
	assert.Equal(t, "", userID)
}
