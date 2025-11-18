package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminMiddleware checks if the authenticated user is an admin
// Must be used AFTER AuthMiddleware
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context (set by AuthMiddleware)
		_, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized - authentication required",
			})
			c.Abort()
			return
		}

		// Get isAdmin flag from context
		isAdmin, exists := c.Get("is_admin")
		if !exists || !isAdmin.(bool) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Forbidden - admin access required",
			})
			c.Abort()
			return
		}

		// Log admin action
		c.Set("admin_action", true)
		c.Next()
	}
}
