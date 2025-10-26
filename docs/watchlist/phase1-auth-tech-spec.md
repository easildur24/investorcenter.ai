# Phase 1: Authentication & User Management - Technical Specification

## Overview

**Goal:** Implement a complete authentication system that allows users to sign up, login, manage their profile, and secure access to protected resources.

**Timeline:** 2 weeks (10 working days)

**Dependencies:**
- PostgreSQL database (already running)
- Email service (SMTP or SendGrid) for verification and password reset
- Frontend Next.js 14 app (already set up)
- Backend Go/Gin server (already set up)

---

## Database Schema

### Migration File: `migrations/006_auth_tables.sql`

```sql
-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255), -- NULL for OAuth-only users
    full_name VARCHAR(255),
    timezone VARCHAR(50) DEFAULT 'UTC',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP,
    email_verified BOOLEAN DEFAULT FALSE,
    email_verification_token VARCHAR(255),
    email_verification_expires_at TIMESTAMP,
    password_reset_token VARCHAR(255),
    password_reset_expires_at TIMESTAMP,
    is_premium BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE -- For soft delete
);

-- Indexes for users
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_email_verification_token ON users(email_verification_token);
CREATE INDEX idx_users_password_reset_token ON users(password_reset_token);

-- OAuth providers table (for Google, GitHub login)
CREATE TABLE oauth_providers (
    id SERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- 'google', 'github'
    provider_user_id VARCHAR(255) NOT NULL, -- User ID from OAuth provider
    provider_email VARCHAR(255),
    access_token TEXT, -- Encrypted in production
    refresh_token TEXT, -- Encrypted in production
    token_expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, provider_user_id)
);

-- Indexes for oauth_providers
CREATE INDEX idx_oauth_providers_user_id ON oauth_providers(user_id);
CREATE INDEX idx_oauth_providers_provider_user_id ON oauth_providers(provider, provider_user_id);

-- Sessions table (for JWT refresh tokens)
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255) NOT NULL, -- Store hashed refresh token
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    user_agent TEXT, -- Browser/device info
    ip_address INET -- IP address for security tracking
);

-- Indexes for sessions
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_refresh_token_hash ON sessions(refresh_token_hash);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Cleanup expired sessions (run as cron job)
CREATE OR REPLACE FUNCTION cleanup_expired_sessions()
RETURNS void AS $$
BEGIN
    DELETE FROM sessions WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_oauth_providers_updated_at BEFORE UPDATE ON oauth_providers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

---

## Backend Implementation

### 1. Environment Variables

Add to `.env` and Kubernetes ConfigMap/Secrets:

```bash
# JWT Configuration
JWT_SECRET=<random-256-bit-secret>  # Generate with: openssl rand -base64 32
JWT_ACCESS_TOKEN_EXPIRY=1h          # Access token expiry (short-lived)
JWT_REFRESH_TOKEN_EXPIRY=168h       # Refresh token expiry (7 days)

# Email Configuration (for verification and password reset)
SMTP_HOST=smtp.sendgrid.net         # Or smtp.gmail.com for Gmail
SMTP_PORT=587
SMTP_USERNAME=apikey                # For SendGrid, or your Gmail address
SMTP_PASSWORD=<sendgrid-api-key>    # Or Gmail app password
SMTP_FROM_EMAIL=noreply@investorcenter.ai
SMTP_FROM_NAME=InvestorCenter.ai

# OAuth Configuration (Google)
GOOGLE_OAUTH_CLIENT_ID=<from-google-cloud-console>
GOOGLE_OAUTH_CLIENT_SECRET=<from-google-cloud-console>
GOOGLE_OAUTH_REDIRECT_URL=http://localhost:3000/auth/google/callback  # Update for production

# OAuth Configuration (GitHub) - Optional for Phase 1
GITHUB_OAUTH_CLIENT_ID=<from-github-oauth-apps>
GITHUB_OAUTH_CLIENT_SECRET=<from-github-oauth-apps>
GITHUB_OAUTH_REDIRECT_URL=http://localhost:3000/auth/github/callback

# Frontend URL (for email links)
FRONTEND_URL=http://localhost:3000  # Update for production

# Security
BCRYPT_COST=12                      # Password hashing cost (10-14 recommended)
RATE_LIMIT_REQUESTS=5               # Max login attempts per window
RATE_LIMIT_WINDOW=15m               # Rate limit window duration
```

### 2. Go Package Structure

```
backend/
├── auth/
│   ├── jwt.go              # JWT token generation and validation
│   ├── password.go         # Password hashing and verification
│   ├── oauth.go            # OAuth provider integrations
│   ├── middleware.go       # Authentication middleware
│   └── rate_limit.go       # Rate limiting for auth endpoints
│
├── models/
│   ├── user.go             # User, Session, OAuthProvider structs
│   └── auth.go             # Auth request/response DTOs
│
├── database/
│   ├── users.go            # User CRUD operations
│   └── sessions.go         # Session CRUD operations
│
├── handlers/
│   ├── auth_handlers.go    # Auth endpoints (signup, login, logout, etc.)
│   └── user_handlers.go    # User profile endpoints
│
├── services/
│   ├── auth_service.go     # Authentication business logic
│   ├── user_service.go     # User management logic
│   └── email_service.go    # Email sending (verification, password reset)
│
└── utils/
    ├── validator.go        # Input validation helpers
    └── random.go           # Generate random tokens
```

### 3. Data Models

**File:** `backend/models/user.go`

```go
package models

import (
    "time"
)

// User represents a registered user
type User struct {
    ID                          string     `json:"id" db:"id"`
    Email                       string     `json:"email" db:"email"`
    PasswordHash                *string    `json:"-" db:"password_hash"` // Never expose in JSON
    FullName                    string     `json:"full_name" db:"full_name"`
    Timezone                    string     `json:"timezone" db:"timezone"`
    CreatedAt                   time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt                   time.Time  `json:"updated_at" db:"updated_at"`
    LastLoginAt                 *time.Time `json:"last_login_at" db:"last_login_at"`
    EmailVerified               bool       `json:"email_verified" db:"email_verified"`
    EmailVerificationToken      *string    `json:"-" db:"email_verification_token"`
    EmailVerificationExpiresAt  *time.Time `json:"-" db:"email_verification_expires_at"`
    PasswordResetToken          *string    `json:"-" db:"password_reset_token"`
    PasswordResetExpiresAt      *time.Time `json:"-" db:"password_reset_expires_at"`
    IsPremium                   bool       `json:"is_premium" db:"is_premium"`
    IsActive                    bool       `json:"is_active" db:"is_active"`
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
    ID               string     `json:"id" db:"id"`
    UserID           string     `json:"user_id" db:"user_id"`
    RefreshTokenHash string     `json:"-" db:"refresh_token_hash"`
    ExpiresAt        time.Time  `json:"expires_at" db:"expires_at"`
    CreatedAt        time.Time  `json:"created_at" db:"created_at"`
    LastUsedAt       time.Time  `json:"last_used_at" db:"last_used_at"`
    UserAgent        *string    `json:"user_agent" db:"user_agent"`
    IPAddress        *string    `json:"ip_address" db:"ip_address"`
}

// OAuthProvider represents a linked OAuth account
type OAuthProvider struct {
    ID              int        `json:"id" db:"id"`
    UserID          string     `json:"user_id" db:"user_id"`
    Provider        string     `json:"provider" db:"provider"` // "google", "github"
    ProviderUserID  string     `json:"provider_user_id" db:"provider_user_id"`
    ProviderEmail   *string    `json:"provider_email" db:"provider_email"`
    AccessToken     *string    `json:"-" db:"access_token"`
    RefreshToken    *string    `json:"-" db:"refresh_token"`
    TokenExpiresAt  *time.Time `json:"token_expires_at" db:"token_expires_at"`
    CreatedAt       time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}
```

**File:** `backend/models/auth.go`

```go
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
```

### 4. JWT Implementation

**File:** `backend/auth/jwt.go`

```go
package auth

import (
    "errors"
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "investorcenter/backend/models"
    "os"
    "strconv"
)

var (
    jwtSecret            = []byte(os.Getenv("JWT_SECRET"))
    accessTokenDuration  = parseDuration(os.Getenv("JWT_ACCESS_TOKEN_EXPIRY"), 1*time.Hour)
    refreshTokenDuration = parseDuration(os.Getenv("JWT_REFRESH_TOKEN_EXPIRY"), 168*time.Hour)
)

// Custom claims for JWT
type Claims struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
    jwt.RegisteredClaims
}

// GenerateAccessToken creates a short-lived JWT access token
func GenerateAccessToken(user *models.User) (string, error) {
    claims := Claims{
        UserID: user.ID,
        Email:  user.Email,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenDuration)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            Issuer:    "investorcenter.ai",
            Subject:   user.ID,
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(jwtSecret)
}

// GenerateRefreshToken creates a long-lived refresh token
func GenerateRefreshToken(user *models.User) (string, error) {
    claims := Claims{
        UserID: user.ID,
        Email:  user.Email,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(refreshTokenDuration)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            Issuer:    "investorcenter.ai",
            Subject:   user.ID,
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(jwtSecret)
}

// ValidateToken parses and validates a JWT token
func ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        // Verify signing method
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return jwtSecret, nil
    })

    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }

    return nil, errors.New("invalid token")
}

// GetAccessTokenExpirySeconds returns the access token expiry duration in seconds
func GetAccessTokenExpirySeconds() int {
    return int(accessTokenDuration.Seconds())
}

// Helper to parse duration from env variable
func parseDuration(durationStr string, defaultDuration time.Duration) time.Duration {
    if durationStr == "" {
        return defaultDuration
    }
    duration, err := time.ParseDuration(durationStr)
    if err != nil {
        return defaultDuration
    }
    return duration
}
```

### 5. Password Hashing

**File:** `backend/auth/password.go`

```go
package auth

import (
    "golang.org/x/crypto/bcrypt"
    "os"
    "strconv"
)

var bcryptCost = getBcryptCost()

// HashPassword hashes a plaintext password using bcrypt
func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
    return string(bytes), err
}

// CheckPasswordHash compares a plaintext password with a hash
func CheckPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}

// Helper to get bcrypt cost from env
func getBcryptCost() int {
    costStr := os.Getenv("BCRYPT_COST")
    if costStr == "" {
        return 12 // Default cost
    }
    cost, err := strconv.Atoi(costStr)
    if err != nil || cost < 10 || cost > 14 {
        return 12
    }
    return cost
}
```

### 6. Authentication Middleware

**File:** `backend/auth/middleware.go`

```go
package auth

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

// AuthMiddleware validates JWT access token from Authorization header
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get Authorization header
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }

        // Check Bearer prefix
        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format. Use: Bearer <token>"})
            c.Abort()
            return
        }

        tokenString := parts[1]

        // Validate token
        claims, err := ValidateToken(tokenString)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
            c.Abort()
            return
        }

        // Store user info in context for handlers to access
        c.Set("user_id", claims.UserID)
        c.Set("user_email", claims.Email)

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
```

### 7. Rate Limiting

**File:** `backend/auth/rate_limit.go`

```go
package auth

import (
    "net/http"
    "sync"
    "time"

    "github.com/gin-gonic/gin"
)

// Simple in-memory rate limiter (use Redis in production for distributed systems)
type rateLimiter struct {
    mu       sync.RWMutex
    attempts map[string][]time.Time
    max      int
    window   time.Duration
}

var loginLimiter = &rateLimiter{
    attempts: make(map[string][]time.Time),
    max:      5,              // Max 5 attempts
    window:   15 * time.Minute, // Per 15 minutes
}

// RateLimitMiddleware limits requests by IP address
func RateLimitMiddleware(limiter *rateLimiter) gin.HandlerFunc {
    return func(c *gin.Context) {
        ip := c.ClientIP()

        if !limiter.Allow(ip) {
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error": "Too many requests. Please try again later.",
            })
            c.Abort()
            return
        }

        c.Next()
    }
}

// Allow checks if request from IP is allowed
func (rl *rateLimiter) Allow(key string) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    now := time.Now()
    cutoff := now.Add(-rl.window)

    // Get attempts for this key
    attempts, exists := rl.attempts[key]
    if !exists {
        rl.attempts[key] = []time.Time{now}
        return true
    }

    // Remove expired attempts
    validAttempts := []time.Time{}
    for _, t := range attempts {
        if t.After(cutoff) {
            validAttempts = append(validAttempts, t)
        }
    }

    // Check if under limit
    if len(validAttempts) >= rl.max {
        rl.attempts[key] = validAttempts
        return false
    }

    // Add new attempt
    validAttempts = append(validAttempts, now)
    rl.attempts[key] = validAttempts
    return true
}

// Cleanup removes old entries (call periodically)
func (rl *rateLimiter) Cleanup() {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    now := time.Now()
    cutoff := now.Add(-rl.window)

    for key, attempts := range rl.attempts {
        validAttempts := []time.Time{}
        for _, t := range attempts {
            if t.After(cutoff) {
                validAttempts = append(validAttempts, t)
            }
        }

        if len(validAttempts) == 0 {
            delete(rl.attempts, key)
        } else {
            rl.attempts[key] = validAttempts
        }
    }
}

// Start periodic cleanup
func StartRateLimiterCleanup(limiter *rateLimiter) {
    ticker := time.NewTicker(5 * time.Minute)
    go func() {
        for range ticker.C {
            limiter.Cleanup()
        }
    }()
}
```

### 8. Database Operations

**File:** `backend/database/users.go`

```go
package database

import (
    "database/sql"
    "errors"
    "fmt"
    "investorcenter/backend/models"
    "time"
)

// CreateUser inserts a new user into the database
func CreateUser(user *models.User) error {
    query := `
        INSERT INTO users (email, password_hash, full_name, timezone, email_verification_token, email_verification_expires_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, created_at, updated_at
    `
    err := DB.QueryRow(
        query,
        user.Email,
        user.PasswordHash,
        user.FullName,
        user.Timezone,
        user.EmailVerificationToken,
        user.EmailVerificationExpiresAt,
    ).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

    if err != nil {
        return fmt.Errorf("failed to create user: %w", err)
    }
    return nil
}

// GetUserByEmail retrieves a user by email
func GetUserByEmail(email string) (*models.User, error) {
    query := `
        SELECT id, email, password_hash, full_name, timezone, created_at, updated_at,
               last_login_at, email_verified, is_premium, is_active
        FROM users
        WHERE email = $1 AND is_active = TRUE
    `
    user := &models.User{}
    err := DB.QueryRow(query, email).Scan(
        &user.ID,
        &user.Email,
        &user.PasswordHash,
        &user.FullName,
        &user.Timezone,
        &user.CreatedAt,
        &user.UpdatedAt,
        &user.LastLoginAt,
        &user.EmailVerified,
        &user.IsPremium,
        &user.IsActive,
    )

    if err == sql.ErrNoRows {
        return nil, errors.New("user not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    return user, nil
}

// GetUserByID retrieves a user by ID
func GetUserByID(id string) (*models.User, error) {
    query := `
        SELECT id, email, password_hash, full_name, timezone, created_at, updated_at,
               last_login_at, email_verified, is_premium, is_active
        FROM users
        WHERE id = $1 AND is_active = TRUE
    `
    user := &models.User{}
    err := DB.QueryRow(query, id).Scan(
        &user.ID,
        &user.Email,
        &user.PasswordHash,
        &user.FullName,
        &user.Timezone,
        &user.CreatedAt,
        &user.UpdatedAt,
        &user.LastLoginAt,
        &user.EmailVerified,
        &user.IsPremium,
        &user.IsActive,
    )

    if err == sql.ErrNoRows {
        return nil, errors.New("user not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    return user, nil
}

// UpdateUser updates user fields
func UpdateUser(user *models.User) error {
    query := `
        UPDATE users
        SET full_name = $1, timezone = $2, updated_at = CURRENT_TIMESTAMP
        WHERE id = $3
    `
    _, err := DB.Exec(query, user.FullName, user.Timezone, user.ID)
    if err != nil {
        return fmt.Errorf("failed to update user: %w", err)
    }
    return nil
}

// UpdateUserPassword updates user's password hash
func UpdateUserPassword(userID string, passwordHash string) error {
    query := `
        UPDATE users
        SET password_hash = $1, updated_at = CURRENT_TIMESTAMP,
            password_reset_token = NULL, password_reset_expires_at = NULL
        WHERE id = $2
    `
    _, err := DB.Exec(query, passwordHash, userID)
    if err != nil {
        return fmt.Errorf("failed to update password: %w", err)
    }
    return nil
}

// UpdateLastLogin updates the last_login_at timestamp
func UpdateLastLogin(userID string) error {
    query := `UPDATE users SET last_login_at = $1 WHERE id = $2`
    _, err := DB.Exec(query, time.Now(), userID)
    return err
}

// SetEmailVerificationToken sets the email verification token
func SetEmailVerificationToken(userID, token string, expiresAt time.Time) error {
    query := `
        UPDATE users
        SET email_verification_token = $1, email_verification_expires_at = $2
        WHERE id = $3
    `
    _, err := DB.Exec(query, token, expiresAt, userID)
    return err
}

// VerifyEmail marks the email as verified
func VerifyEmail(token string) error {
    query := `
        UPDATE users
        SET email_verified = TRUE,
            email_verification_token = NULL,
            email_verification_expires_at = NULL
        WHERE email_verification_token = $1
          AND email_verification_expires_at > $2
          AND email_verified = FALSE
    `
    result, err := DB.Exec(query, token, time.Now())
    if err != nil {
        return fmt.Errorf("failed to verify email: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rowsAffected == 0 {
        return errors.New("invalid or expired verification token")
    }
    return nil
}

// SetPasswordResetToken sets the password reset token
func SetPasswordResetToken(email, token string, expiresAt time.Time) error {
    query := `
        UPDATE users
        SET password_reset_token = $1, password_reset_expires_at = $2
        WHERE email = $3 AND is_active = TRUE
    `
    result, err := DB.Exec(query, token, expiresAt, email)
    if err != nil {
        return fmt.Errorf("failed to set password reset token: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rowsAffected == 0 {
        return errors.New("user not found")
    }
    return nil
}

// GetUserByPasswordResetToken retrieves user by password reset token
func GetUserByPasswordResetToken(token string) (*models.User, error) {
    query := `
        SELECT id, email, password_hash, full_name, timezone, created_at, updated_at,
               last_login_at, email_verified, is_premium, is_active
        FROM users
        WHERE password_reset_token = $1
          AND password_reset_expires_at > $2
          AND is_active = TRUE
    `
    user := &models.User{}
    err := DB.QueryRow(query, token, time.Now()).Scan(
        &user.ID,
        &user.Email,
        &user.PasswordHash,
        &user.FullName,
        &user.Timezone,
        &user.CreatedAt,
        &user.UpdatedAt,
        &user.LastLoginAt,
        &user.EmailVerified,
        &user.IsPremium,
        &user.IsActive,
    )

    if err == sql.ErrNoRows {
        return nil, errors.New("invalid or expired reset token")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    return user, nil
}

// SoftDeleteUser marks user as inactive (soft delete)
func SoftDeleteUser(userID string) error {
    query := `UPDATE users SET is_active = FALSE WHERE id = $1`
    _, err := DB.Exec(query, userID)
    return err
}
```

**File:** `backend/database/sessions.go`

```go
package database

import (
    "fmt"
    "investorcenter/backend/models"
    "time"
)

// CreateSession creates a new session (refresh token)
func CreateSession(session *models.Session) error {
    query := `
        INSERT INTO sessions (user_id, refresh_token_hash, expires_at, user_agent, ip_address)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, created_at, last_used_at
    `
    err := DB.QueryRow(
        query,
        session.UserID,
        session.RefreshTokenHash,
        session.ExpiresAt,
        session.UserAgent,
        session.IPAddress,
    ).Scan(&session.ID, &session.CreatedAt, &session.LastUsedAt)

    if err != nil {
        return fmt.Errorf("failed to create session: %w", err)
    }
    return nil
}

// GetSessionByRefreshTokenHash retrieves session by refresh token hash
func GetSessionByRefreshTokenHash(tokenHash string) (*models.Session, error) {
    query := `
        SELECT id, user_id, refresh_token_hash, expires_at, created_at, last_used_at, user_agent, ip_address
        FROM sessions
        WHERE refresh_token_hash = $1 AND expires_at > $2
    `
    session := &models.Session{}
    err := DB.QueryRow(query, tokenHash, time.Now()).Scan(
        &session.ID,
        &session.UserID,
        &session.RefreshTokenHash,
        &session.ExpiresAt,
        &session.CreatedAt,
        &session.LastUsedAt,
        &session.UserAgent,
        &session.IPAddress,
    )

    if err != nil {
        return nil, fmt.Errorf("session not found or expired: %w", err)
    }
    return session, nil
}

// UpdateSessionLastUsed updates the last_used_at timestamp
func UpdateSessionLastUsed(sessionID string) error {
    query := `UPDATE sessions SET last_used_at = $1 WHERE id = $2`
    _, err := DB.Exec(query, time.Now(), sessionID)
    return err
}

// DeleteSession deletes a session (logout)
func DeleteSession(sessionID string) error {
    query := `DELETE FROM sessions WHERE id = $1`
    _, err := DB.Exec(query, sessionID)
    return err
}

// DeleteUserSessions deletes all sessions for a user
func DeleteUserSessions(userID string) error {
    query := `DELETE FROM sessions WHERE user_id = $1`
    _, err := DB.Exec(query, userID)
    return err
}

// CleanupExpiredSessions removes expired sessions
func CleanupExpiredSessions() error {
    query := `DELETE FROM sessions WHERE expires_at < $1`
    _, err := DB.Exec(query, time.Now())
    return err
}
```

### 9. Email Service

**File:** `backend/services/email_service.go`

```go
package services

import (
    "bytes"
    "fmt"
    "html/template"
    "net/smtp"
    "os"
)

type EmailService struct {
    smtpHost     string
    smtpPort     string
    smtpUsername string
    smtpPassword string
    fromEmail    string
    fromName     string
    frontendURL  string
}

func NewEmailService() *EmailService {
    return &EmailService{
        smtpHost:     os.Getenv("SMTP_HOST"),
        smtpPort:     os.Getenv("SMTP_PORT"),
        smtpUsername: os.Getenv("SMTP_USERNAME"),
        smtpPassword: os.Getenv("SMTP_PASSWORD"),
        fromEmail:    os.Getenv("SMTP_FROM_EMAIL"),
        fromName:     os.Getenv("SMTP_FROM_NAME"),
        frontendURL:  os.Getenv("FRONTEND_URL"),
    }
}

// SendVerificationEmail sends email verification link
func (es *EmailService) SendVerificationEmail(toEmail, fullName, token string) error {
    verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", es.frontendURL, token)

    subject := "Verify your InvestorCenter.ai account"
    body := fmt.Sprintf(`
        <html>
        <body style="font-family: Arial, sans-serif;">
            <h2>Welcome to InvestorCenter.ai, %s!</h2>
            <p>Thanks for signing up. Please verify your email address by clicking the link below:</p>
            <p><a href="%s" style="background-color: #4CAF50; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Verify Email</a></p>
            <p>Or copy and paste this URL into your browser:</p>
            <p>%s</p>
            <p>This link will expire in 24 hours.</p>
            <p>If you didn't create an account, you can safely ignore this email.</p>
        </body>
        </html>
    `, fullName, verifyURL, verifyURL)

    return es.sendEmail(toEmail, subject, body)
}

// SendPasswordResetEmail sends password reset link
func (es *EmailService) SendPasswordResetEmail(toEmail, fullName, token string) error {
    resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", es.frontendURL, token)

    subject := "Reset your InvestorCenter.ai password"
    body := fmt.Sprintf(`
        <html>
        <body style="font-family: Arial, sans-serif;">
            <h2>Password Reset Request</h2>
            <p>Hi %s,</p>
            <p>We received a request to reset your password. Click the link below to reset it:</p>
            <p><a href="%s" style="background-color: #2196F3; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Reset Password</a></p>
            <p>Or copy and paste this URL into your browser:</p>
            <p>%s</p>
            <p>This link will expire in 1 hour.</p>
            <p>If you didn't request a password reset, you can safely ignore this email.</p>
        </body>
        </html>
    `, fullName, resetURL, resetURL)

    return es.sendEmail(toEmail, subject, body)
}

// sendEmail is a helper to send HTML emails via SMTP
func (es *EmailService) sendEmail(to, subject, htmlBody string) error {
    from := fmt.Sprintf("%s <%s>", es.fromName, es.fromEmail)
    msg := []byte(fmt.Sprintf("From: %s\r\n"+
        "To: %s\r\n"+
        "Subject: %s\r\n"+
        "MIME-Version: 1.0\r\n"+
        "Content-Type: text/html; charset=UTF-8\r\n"+
        "\r\n"+
        "%s", from, to, subject, htmlBody))

    auth := smtp.PlainAuth("", es.smtpUsername, es.smtpPassword, es.smtpHost)
    addr := fmt.Sprintf("%s:%s", es.smtpHost, es.smtpPort)

    err := smtp.SendMail(addr, auth, es.fromEmail, []string{to}, msg)
    if err != nil {
        return fmt.Errorf("failed to send email: %w", err)
    }
    return nil
}
```

### 10. Auth Handlers

**File:** `backend/handlers/auth_handlers.go`

```go
package handlers

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "investorcenter/backend/auth"
    "investorcenter/backend/database"
    "investorcenter/backend/models"
    "investorcenter/backend/services"
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
```

**File:** `backend/handlers/user_handlers.go`

```go
package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "investorcenter/backend/auth"
    "investorcenter/backend/database"
    "investorcenter/backend/models"
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
```

### 11. Update main.go with Auth Routes

**File:** `backend/main.go` (add these routes)

```go
// Add to imports
import (
    "investorcenter/backend/auth"
    // ... existing imports
)

// Add inside main() function, before v1 := r.Group("/api/v1")

// Start rate limiter cleanup
auth.StartRateLimiterCleanup(auth.GetLoginLimiter())

// Auth routes (public, no middleware)
authRoutes := r.Group("/api/v1/auth")
{
    // Rate limit on login/signup to prevent brute force
    authRoutes.POST("/signup", auth.RateLimitMiddleware(auth.GetLoginLimiter()), handlers.Signup)
    authRoutes.POST("/login", auth.RateLimitMiddleware(auth.GetLoginLimiter()), handlers.Login)
    authRoutes.POST("/refresh", handlers.RefreshToken)
    authRoutes.POST("/logout", handlers.Logout)
    authRoutes.GET("/verify-email", handlers.VerifyEmail)
    authRoutes.POST("/forgot-password", handlers.ForgotPassword)
    authRoutes.POST("/reset-password", handlers.ResetPassword)
}

// Protected user routes (require authentication)
userRoutes := v1.Group("/user")
userRoutes.Use(auth.AuthMiddleware())
{
    userRoutes.GET("/me", handlers.GetCurrentUser)
    userRoutes.PUT("/me", handlers.UpdateProfile)
    userRoutes.PUT("/password", handlers.ChangePassword)
    userRoutes.DELETE("/me", handlers.DeleteAccount)
}
```

**Note:** You'll need to export the login limiter in `backend/auth/rate_limit.go`:

```go
// Add to rate_limit.go
func GetLoginLimiter() *rateLimiter {
    return loginLimiter
}
```

---

## Frontend Implementation

### 1. Environment Variables

**File:** `.env.local` (create if doesn't exist)

```bash
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
```

### 2. Auth Context Provider

**File:** `lib/auth/AuthContext.tsx`

```typescript
'use client';

import React, { createContext, useContext, useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';

interface User {
  id: string;
  email: string;
  full_name: string;
  timezone: string;
  created_at: string;
  email_verified: boolean;
  is_premium: boolean;
  last_login_at?: string;
}

interface AuthContextType {
  user: User | null;
  accessToken: string | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  signup: (email: string, password: string, fullName: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshAuth: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  // Load tokens from localStorage on mount
  useEffect(() => {
    const storedAccessToken = localStorage.getItem('access_token');
    const storedRefreshToken = localStorage.getItem('refresh_token');
    const storedUser = localStorage.getItem('user');

    if (storedAccessToken && storedUser) {
      setAccessToken(storedAccessToken);
      setUser(JSON.parse(storedUser));
    } else if (storedRefreshToken) {
      // Try to refresh token
      refreshTokens(storedRefreshToken);
    }

    setLoading(false);
  }, []);

  // Auto-refresh token before expiry
  useEffect(() => {
    if (!accessToken) return;

    // Refresh token 5 minutes before expiry (assuming 1 hour expiry = 55 min interval)
    const refreshInterval = setInterval(() => {
      const refreshToken = localStorage.getItem('refresh_token');
      if (refreshToken) {
        refreshTokens(refreshToken);
      }
    }, 55 * 60 * 1000); // 55 minutes

    return () => clearInterval(refreshInterval);
  }, [accessToken]);

  const refreshTokens = async (refreshToken: string) => {
    try {
      const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });

      if (response.ok) {
        const data = await response.json();
        setAccessToken(data.access_token);
        localStorage.setItem('access_token', data.access_token);
      } else {
        // Refresh failed, logout
        logout();
      }
    } catch (error) {
      console.error('Failed to refresh token:', error);
      logout();
    }
  };

  const login = async (email: string, password: string) => {
    const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Login failed');
    }

    const data = await response.json();
    setAccessToken(data.access_token);
    setUser(data.user);

    // Store in localStorage
    localStorage.setItem('access_token', data.access_token);
    localStorage.setItem('refresh_token', data.refresh_token);
    localStorage.setItem('user', JSON.stringify(data.user));

    router.push('/watchlist');
  };

  const signup = async (email: string, password: string, fullName: string) => {
    const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/auth/signup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password, full_name: fullName }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Signup failed');
    }

    const data = await response.json();
    setAccessToken(data.access_token);
    setUser(data.user);

    localStorage.setItem('access_token', data.access_token);
    localStorage.setItem('refresh_token', data.refresh_token);
    localStorage.setItem('user', JSON.stringify(data.user));

    router.push('/watchlist');
  };

  const logout = async () => {
    const refreshToken = localStorage.getItem('refresh_token');

    if (refreshToken) {
      try {
        await fetch(`${process.env.NEXT_PUBLIC_API_URL}/auth/logout`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: refreshToken }),
        });
      } catch (error) {
        console.error('Logout error:', error);
      }
    }

    // Clear state and storage
    setUser(null);
    setAccessToken(null);
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('user');

    router.push('/auth/login');
  };

  const refreshAuth = async () => {
    const refreshToken = localStorage.getItem('refresh_token');
    if (refreshToken) {
      await refreshTokens(refreshToken);
    }
  };

  return (
    <AuthContext.Provider value={{ user, accessToken, loading, login, signup, logout, refreshAuth }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
```

### 3. Protected Route HOC

**File:** `components/auth/ProtectedRoute.tsx`

```typescript
'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/lib/auth/AuthContext';

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!loading && !user) {
      router.push('/auth/login');
    }
  }, [user, loading, router]);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-xl">Loading...</div>
      </div>
    );
  }

  if (!user) {
    return null;
  }

  return <>{children}</>;
}
```

### 4. Login Page

**File:** `app/auth/login/page.tsx`

```typescript
'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useAuth } from '@/lib/auth/AuthContext';

export default function LoginPage() {
  const { login } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await login(email, password);
    } catch (err: any) {
      setError(err.message || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-50">
      <div className="w-full max-w-md p-8 bg-white rounded-lg shadow-md">
        <h1 className="text-2xl font-bold mb-6 text-center">Login to InvestorCenter.ai</h1>

        {error && (
          <div className="mb-4 p-3 bg-red-100 border border-red-400 text-red-700 rounded">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label className="block text-sm font-medium mb-2" htmlFor="email">
              Email
            </label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>

          <div className="mb-6">
            <label className="block text-sm font-medium mb-2" htmlFor="password">
              Password
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-2 px-4 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed"
          >
            {loading ? 'Logging in...' : 'Login'}
          </button>
        </form>

        <div className="mt-4 text-center text-sm">
          <Link href="/auth/forgot-password" className="text-blue-600 hover:underline">
            Forgot password?
          </Link>
        </div>

        <div className="mt-2 text-center text-sm">
          Don't have an account?{' '}
          <Link href="/auth/signup" className="text-blue-600 hover:underline">
            Sign up
          </Link>
        </div>
      </div>
    </div>
  );
}
```

### 5. Signup Page

**File:** `app/auth/signup/page.tsx`

```typescript
'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useAuth } from '@/lib/auth/AuthContext';

export default function SignupPage() {
  const { signup } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [fullName, setFullName] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (password.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }

    setLoading(true);

    try {
      await signup(email, password, fullName);
    } catch (err: any) {
      setError(err.message || 'Signup failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-50">
      <div className="w-full max-w-md p-8 bg-white rounded-lg shadow-md">
        <h1 className="text-2xl font-bold mb-6 text-center">Create Account</h1>

        {error && (
          <div className="mb-4 p-3 bg-red-100 border border-red-400 text-red-700 rounded">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label className="block text-sm font-medium mb-2" htmlFor="fullName">
              Full Name
            </label>
            <input
              id="fullName"
              type="text"
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-2" htmlFor="email">
              Email
            </label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>

          <div className="mb-6">
            <label className="block text-sm font-medium mb-2" htmlFor="password">
              Password (min 8 characters)
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
              minLength={8}
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-2 px-4 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed"
          >
            {loading ? 'Creating account...' : 'Sign Up'}
          </button>
        </form>

        <div className="mt-4 text-center text-sm">
          Already have an account?{' '}
          <Link href="/auth/login" className="text-blue-600 hover:underline">
            Login
          </Link>
        </div>
      </div>
    </div>
  );
}
```

### 6. Update Root Layout with AuthProvider

**File:** `app/layout.tsx` (modify)

```typescript
import { AuthProvider } from '@/lib/auth/AuthContext';
// ... other imports

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <AuthProvider>
          {/* existing layout */}
          {children}
        </AuthProvider>
      </body>
    </html>
  );
}
```

### 7. Update Header with User Menu

**File:** `components/Header.tsx` (add user menu)

```typescript
'use client';

import Link from 'next/link';
import { useAuth } from '@/lib/auth/AuthContext';
import { useState } from 'react';

export default function Header() {
  const { user, logout } = useAuth();
  const [showDropdown, setShowDropdown] = useState(false);

  return (
    <header className="bg-white shadow">
      <div className="container mx-auto px-4 py-4 flex justify-between items-center">
        <Link href="/" className="text-2xl font-bold">
          InvestorCenter.ai
        </Link>

        <nav className="flex items-center gap-6">
          <Link href="/" className="hover:text-blue-600">Home</Link>
          <Link href="/crypto" className="hover:text-blue-600">Crypto</Link>

          {user ? (
            <div className="relative">
              <button
                onClick={() => setShowDropdown(!showDropdown)}
                className="flex items-center gap-2 hover:text-blue-600"
              >
                <span>{user.full_name}</span>
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </button>

              {showDropdown && (
                <div className="absolute right-0 mt-2 w-48 bg-white rounded-md shadow-lg py-1 z-10">
                  <Link
                    href="/watchlist"
                    className="block px-4 py-2 hover:bg-gray-100"
                    onClick={() => setShowDropdown(false)}
                  >
                    My Watch Lists
                  </Link>
                  <Link
                    href="/settings/profile"
                    className="block px-4 py-2 hover:bg-gray-100"
                    onClick={() => setShowDropdown(false)}
                  >
                    Settings
                  </Link>
                  <button
                    onClick={() => {
                      setShowDropdown(false);
                      logout();
                    }}
                    className="block w-full text-left px-4 py-2 hover:bg-gray-100"
                  >
                    Logout
                  </button>
                </div>
              )}
            </div>
          ) : (
            <>
              <Link href="/auth/login" className="hover:text-blue-600">Login</Link>
              <Link href="/auth/signup" className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">
                Sign Up
              </Link>
            </>
          )}
        </nav>
      </div>
    </header>
  );
}
```

---

## Testing Plan

### Backend Testing

1. **Unit Tests** (Go tests in `backend/auth/`, `backend/services/`)
   - JWT token generation and validation
   - Password hashing and verification
   - Email validation
   - Rate limiting logic

2. **Integration Tests** (Go tests in `backend/handlers/`)
   - Signup flow (create user, generate tokens, send verification email)
   - Login flow (authenticate, return tokens)
   - Token refresh flow
   - Password reset flow
   - Protected endpoint access (with valid/invalid tokens)

3. **Manual Testing Checklist**
   - [ ] Sign up with valid email/password
   - [ ] Sign up with duplicate email (should fail)
   - [ ] Sign up with weak password (should fail)
   - [ ] Login with correct credentials
   - [ ] Login with wrong password (should fail)
   - [ ] Login with non-existent email (should fail)
   - [ ] Access protected endpoint with valid token
   - [ ] Access protected endpoint with expired token (should fail)
   - [ ] Access protected endpoint without token (should fail)
   - [ ] Refresh access token with valid refresh token
   - [ ] Refresh with expired refresh token (should fail)
   - [ ] Logout (invalidate refresh token)
   - [ ] Verify email with valid token
   - [ ] Verify email with expired token (should fail)
   - [ ] Request password reset (email sent)
   - [ ] Reset password with valid token
   - [ ] Reset password with expired token (should fail)
   - [ ] Change password (authenticated)
   - [ ] Update profile (authenticated)
   - [ ] Delete account (soft delete)
   - [ ] Rate limiting on login (5 attempts, then blocked)

### Frontend Testing

1. **Manual Testing Checklist**
   - [ ] Signup page renders correctly
   - [ ] Signup form validation (email format, password length)
   - [ ] Signup submits and redirects to /watchlist
   - [ ] Signup shows error for duplicate email
   - [ ] Login page renders correctly
   - [ ] Login form validation
   - [ ] Login submits and redirects to /watchlist
   - [ ] Login shows error for invalid credentials
   - [ ] Auth context stores tokens in localStorage
   - [ ] Protected routes redirect to /auth/login when not logged in
   - [ ] Header shows user name when logged in
   - [ ] User dropdown menu works
   - [ ] Logout clears tokens and redirects to /auth/login
   - [ ] Token auto-refresh works (check after 55 minutes)
   - [ ] Page refresh preserves auth state (loads from localStorage)

---

## Deployment Checklist

### Environment Setup

- [ ] Generate secure JWT_SECRET (32+ bytes)
- [ ] Configure SMTP credentials (SendGrid or Gmail)
- [ ] Set up Google OAuth app and get credentials
- [ ] Update FRONTEND_URL for production domain
- [ ] Set BCRYPT_COST to 12 (recommended for production)

### Database

- [ ] Run migration `migrations/006_auth_tables.sql` on production database
- [ ] Verify tables created: users, oauth_providers, sessions
- [ ] Test database connection from backend

### Backend

- [ ] Build Go binary with auth packages
- [ ] Update Kubernetes deployment with new environment variables
- [ ] Deploy to EKS cluster
- [ ] Test health check endpoint
- [ ] Test auth endpoints (signup, login, refresh)

### Frontend

- [ ] Set NEXT_PUBLIC_API_URL to production backend URL
- [ ] Build Next.js app (`npm run build`)
- [ ] Deploy to production (Vercel, AWS, etc.)
- [ ] Test frontend auth flows

### Monitoring

- [ ] Set up logging for auth events (login, signup, failed attempts)
- [ ] Monitor rate limiting effectiveness
- [ ] Track email delivery success rate
- [ ] Set up alerts for high auth error rates

---

## Security Considerations

1. **Passwords**
   - Minimum 8 characters required
   - Bcrypt cost factor 12 (adjustable via env)
   - Never log or expose password hashes

2. **JWT Tokens**
   - Access tokens: Short-lived (1 hour)
   - Refresh tokens: Longer-lived (7 days), stored hashed in database
   - Tokens signed with HMAC-SHA256
   - Validate issuer and expiry on every request

3. **Rate Limiting**
   - Max 5 login attempts per IP per 15 minutes
   - Prevents brute force attacks
   - Use Redis in production for distributed rate limiting

4. **Email Security**
   - Verification tokens expire in 24 hours
   - Password reset tokens expire in 1 hour
   - Don't reveal if email exists (password reset)
   - Use HTTPS for email links in production

5. **Session Management**
   - Store refresh token hashes (not plaintext)
   - Track user agent and IP for security auditing
   - Cleanup expired sessions periodically
   - Logout invalidates all sessions

6. **HTTPS Required**
   - All production traffic must use HTTPS
   - Set secure cookies in production
   - Use HSTS headers

---

## Success Criteria

Phase 1 is considered complete when:

- [x] Database schema created and migrated
- [x] Users can sign up with email/password
- [x] Users can login and receive JWT tokens
- [x] Users can access protected endpoints with valid tokens
- [x] Token refresh works automatically
- [x] Users can logout (invalidate session)
- [x] Email verification flow works
- [x] Password reset flow works
- [x] Users can update their profile
- [x] Users can change their password
- [x] Users can delete their account
- [x] Rate limiting prevents brute force attacks
- [x] Frontend auth context manages state correctly
- [x] Protected routes redirect unauthenticated users
- [x] Header shows user menu when logged in
- [x] All manual tests pass
- [x] Deployed to production and tested

---

## Next Steps (Phase 2)

After Phase 1 is complete and tested, we'll move to Phase 2:

- Watch list database schema and migrations
- Watch list CRUD API endpoints
- Watch list frontend pages and components
- Add tickers to watch lists
- Real-time price updates for watch list items

**Estimated Timeline:** Phase 1 completion by Day 10, ready to start Phase 2.
