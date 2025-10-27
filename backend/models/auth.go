package models

// Auth request/response DTOs

// SignupRequest represents signup form data
type SignupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"required"`
	Timezone string `json:"timezone"` // Optional, defaults to UTC
}

// LoginRequest represents login form data
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
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
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ForgotPasswordRequest for password reset email
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest for resetting password with token
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ChangePasswordRequest for changing password (authenticated)
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// UpdateProfileRequest for updating user profile
type UpdateProfileRequest struct {
	FullName string `json:"full_name"`
	Timezone string `json:"timezone"`
}

// OAuthCallbackRequest (query params from OAuth redirect)
type OAuthCallbackRequest struct {
	Code  string `form:"code" binding:"required"`
	State string `form:"state" binding:"required"`
}
