package models

import (
	"time"
)

// User represents a registered user
type User struct {
	ID                         string     `json:"id" db:"id"`
	Email                      string     `json:"email" db:"email"`
	PasswordHash               *string    `json:"-" db:"password_hash"` // Never expose in JSON
	FullName                   string     `json:"full_name" db:"full_name"`
	Timezone                   string     `json:"timezone" db:"timezone"`
	CreatedAt                  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at" db:"updated_at"`
	LastLoginAt                *time.Time `json:"last_login_at" db:"last_login_at"`
	EmailVerified              bool       `json:"email_verified" db:"email_verified"`
	EmailVerificationToken     *string    `json:"-" db:"email_verification_token"`
	EmailVerificationExpiresAt *time.Time `json:"-" db:"email_verification_expires_at"`
	PasswordResetToken         *string    `json:"-" db:"password_reset_token"`
	PasswordResetExpiresAt     *time.Time `json:"-" db:"password_reset_expires_at"`
	IsPremium                  bool       `json:"is_premium" db:"is_premium"`
	IsActive                   bool       `json:"is_active" db:"is_active"`
}

// UserPublic is the public-facing user data (safe to expose in API)
type UserPublic struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	FullName      string     `json:"full_name"`
	Timezone      string     `json:"timezone"`
	CreatedAt     time.Time  `json:"created_at"`
	EmailVerified bool       `json:"email_verified"`
	IsPremium     bool       `json:"is_premium"`
	LastLoginAt   *time.Time `json:"last_login_at"`
}

// ToPublic converts User to UserPublic (safe for API responses)
func (u *User) ToPublic() UserPublic {
	return UserPublic{
		ID:            u.ID,
		Email:         u.Email,
		FullName:      u.FullName,
		Timezone:      u.Timezone,
		CreatedAt:     u.CreatedAt,
		EmailVerified: u.EmailVerified,
		IsPremium:     u.IsPremium,
		LastLoginAt:   u.LastLoginAt,
	}
}

// Session represents a user session (refresh token)
type Session struct {
	ID               string    `json:"id" db:"id"`
	UserID           string    `json:"user_id" db:"user_id"`
	RefreshTokenHash string    `json:"-" db:"refresh_token_hash"`
	ExpiresAt        time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	LastUsedAt       time.Time `json:"last_used_at" db:"last_used_at"`
	UserAgent        *string   `json:"user_agent" db:"user_agent"`
	IPAddress        *string   `json:"ip_address" db:"ip_address"`
}

// OAuthProvider represents a linked OAuth account
type OAuthProvider struct {
	ID             int        `json:"id" db:"id"`
	UserID         string     `json:"user_id" db:"user_id"`
	Provider       string     `json:"provider" db:"provider"` // "google", "github"
	ProviderUserID string     `json:"provider_user_id" db:"provider_user_id"`
	ProviderEmail  *string    `json:"provider_email" db:"provider_email"`
	AccessToken    *string    `json:"-" db:"access_token"`
	RefreshToken   *string    `json:"-" db:"refresh_token"`
	TokenExpiresAt *time.Time `json:"token_expires_at" db:"token_expires_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}
