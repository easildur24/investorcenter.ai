package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"investorcenter-api/auth"
	"investorcenter-api/database"
	"investorcenter-api/models"
)

// GetCurrentUser returns the authenticated user's profile
func GetCurrentUser(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, err := database.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user.ToPublic())
}

// UpdateProfile updates the user's profile information
func UpdateProfile(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := database.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update fields
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Timezone != "" {
		user.Timezone = req.Timezone
	}

	err = database.UpdateUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, user.ToPublic())
}

// ChangePassword changes the user's password
func ChangePassword(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := database.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Verify current password
	if user.PasswordHash == nil || !auth.CheckPasswordHash(req.CurrentPassword, *user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}

	// Hash new password
	newPasswordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Update password
	err = database.UpdateUserPassword(user.ID, newPasswordHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// DeleteAccount soft-deletes the user account
func DeleteAccount(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Soft delete user
	err := database.SoftDeleteUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete account"})
		return
	}

	// Delete all sessions
	database.DeleteUserSessions(userID)

	c.JSON(http.StatusOK, gin.H{"message": "Account deleted successfully"})
}
