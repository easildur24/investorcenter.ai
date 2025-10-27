package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"investorcenter-api/auth"
	"investorcenter-api/database"
	"investorcenter-api/models"
	"investorcenter-api/services"
)

var emailService = services.NewEmailService()

// Signup creates a new user account
func Signup(c *gin.Context) {
	var req models.SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	existingUser, _ := database.GetUserByEmail(req.Email)
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Generate email verification token
	verificationToken, err := generateRandomToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate verification token"})
		return
	}

	// Set default timezone if not provided
	if req.Timezone == "" {
		req.Timezone = "UTC"
	}

	// Create user
	user := &models.User{
		Email:                      req.Email,
		PasswordHash:               &passwordHash,
		FullName:                   req.FullName,
		Timezone:                   req.Timezone,
		EmailVerified:              false,
		EmailVerificationToken:     &verificationToken,
		EmailVerificationExpiresAt: ptrTime(time.Now().Add(24 * time.Hour)),
	}

	if err := database.CreateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Send verification email (non-blocking)
	go emailService.SendVerificationEmail(user.Email, user.FullName, verificationToken)

	// Generate tokens
	accessToken, err := auth.GenerateAccessToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	refreshToken, err := auth.GenerateRefreshToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Store refresh token hash in sessions table
	refreshTokenHash := hashToken(refreshToken)
	session := &models.Session{
		UserID:           user.ID,
		RefreshTokenHash: refreshTokenHash,
		ExpiresAt:        time.Now().Add(168 * time.Hour), // 7 days
		UserAgent:        ptrString(c.Request.UserAgent()),
		IPAddress:        ptrString(c.ClientIP()),
	}
	if err := database.CreateSession(session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	c.JSON(http.StatusCreated, models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    auth.GetAccessTokenExpirySeconds(),
		User:         user.ToPublic(),
	})
}

// Login authenticates a user with email and password
func Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user by email
	user, err := database.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Check password
	if user.PasswordHash == nil || !auth.CheckPasswordHash(req.Password, *user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Update last login
	database.UpdateLastLogin(user.ID)

	// Generate tokens
	accessToken, err := auth.GenerateAccessToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	refreshToken, err := auth.GenerateRefreshToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Store refresh token
	refreshTokenHash := hashToken(refreshToken)
	session := &models.Session{
		UserID:           user.ID,
		RefreshTokenHash: refreshTokenHash,
		ExpiresAt:        time.Now().Add(168 * time.Hour),
		UserAgent:        ptrString(c.Request.UserAgent()),
		IPAddress:        ptrString(c.ClientIP()),
	}
	if err := database.CreateSession(session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	c.JSON(http.StatusOK, models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    auth.GetAccessTokenExpirySeconds(),
		User:         user.ToPublic(),
	})
}

// RefreshToken generates a new access token using refresh token
func RefreshToken(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash the refresh token to look up in database
	tokenHash := hashToken(req.RefreshToken)

	// Get session
	session, err := database.GetSessionByRefreshTokenHash(tokenHash)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
		return
	}

	// Get user
	user, err := database.GetUserByID(session.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Generate new access token
	accessToken, err := auth.GenerateAccessToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	// Update session last used
	database.UpdateSessionLastUsed(session.ID)

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"expires_in":   auth.GetAccessTokenExpirySeconds(),
	})
}

// Logout invalidates the refresh token
func Logout(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenHash := hashToken(req.RefreshToken)

	session, err := database.GetSessionByRefreshTokenHash(tokenHash)
	if err == nil {
		database.DeleteSession(session.ID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// VerifyEmail verifies user's email with token
func VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Verification token required"})
		return
	}

	err := database.VerifyEmail(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully"})
}

// ForgotPassword sends password reset email
func ForgotPassword(c *gin.Context) {
	var req models.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Always return success to prevent email enumeration
	defer c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a password reset link has been sent"})

	user, err := database.GetUserByEmail(req.Email)
	if err != nil {
		return // User doesn't exist, but we still return success
	}

	// Generate password reset token
	resetToken, err := generateRandomToken(32)
	if err != nil {
		return
	}

	// Store token in database
	err = database.SetPasswordResetToken(user.Email, resetToken, time.Now().Add(1*time.Hour))
	if err != nil {
		return
	}

	// Send password reset email (non-blocking)
	go emailService.SendPasswordResetEmail(user.Email, user.FullName, resetToken)
}

// ResetPassword resets password with token
func ResetPassword(c *gin.Context) {
	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user by reset token
	user, err := database.GetUserByPasswordResetToken(req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset token"})
		return
	}

	// Hash new password
	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Update password
	err = database.UpdateUserPassword(user.ID, passwordHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Invalidate all sessions for security
	database.DeleteUserSessions(user.ID)

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// Helper functions

func generateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func ptrString(s string) *string {
	return &s
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
