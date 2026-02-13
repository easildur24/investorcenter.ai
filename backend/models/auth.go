package models

// Auth request/response DTOs

// SignupRequest represents signup form data
type SignupRequest struct {
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=8,max=128"`
	FullName string `json:"full_name" binding:"required,max=255"`
	Timezone string `json:"timezone" binding:"max=100"` // Optional, defaults to UTC
}

// LoginRequest represents login form data
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,max=128"`
}

// AuthResponse is returned after successful login/signup
type AuthResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	ExpiresIn    int        `json:"expires_in"` // Seconds until access token expires
	User         UserPublic `json:"user"`
}

// RefreshTokenRequest for refreshing access token
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required,max=1024"`
}

// ForgotPasswordRequest for password reset email
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email,max=254"`
}

// ResetPasswordRequest for resetting password with token
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required,max=512"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=128"`
}

// ChangePasswordRequest for changing password (authenticated)
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required,max=128"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=128"`
}

// UpdateProfileRequest for updating user profile
type UpdateProfileRequest struct {
	FullName string `json:"full_name" binding:"max=255"`
	Timezone string `json:"timezone" binding:"max=100"`
}

// OAuthCallbackRequest (query params from OAuth redirect)
type OAuthCallbackRequest struct {
	Code  string `form:"code" binding:"required,max=2048"`
	State string `form:"state" binding:"required,max=512"`
}
