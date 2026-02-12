package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates JWT access token from Authorization header
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format. Use: Bearer <token>"})
			c.Abort()
			return
		}

		claims, err := ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("is_admin", claims.IsAdmin)

		c.Next()
	}
}

// AdminMiddleware checks if the authenticated user is an admin.
// Must be used AFTER AuthMiddleware.
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - authentication required"})
			c.Abort()
			return
		}

		isAdmin, exists := c.Get("is_admin")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden - admin access required"})
			c.Abort()
			return
		}

		isAdminBool, ok := isAdmin.(bool)
		if !ok || !isAdminBool {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden - admin access required"})
			c.Abort()
			return
		}

		c.Set("admin_action", true)
		c.Next()
	}
}

// GetUserIDFromContext retrieves user ID from Gin context (set by AuthMiddleware)
func GetUserIDFromContext(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	return userID.(string), true
}
