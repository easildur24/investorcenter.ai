package database

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// PasswordResetToken represents a password reset token
type PasswordResetToken struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
	Used      bool
	CreatedAt time.Time
}

// GenerateResetToken creates a secure random token
func GenerateResetToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreatePasswordResetToken creates a new password reset token for a user
func CreatePasswordResetToken(userID string) (*PasswordResetToken, error) {
	token, err := GenerateResetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Token expires in 1 hour
	expiresAt := time.Now().Add(1 * time.Hour)

	query := `
		INSERT INTO password_reset_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	resetToken := &PasswordResetToken{
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
	}

	err = DB.QueryRow(query, userID, token, expiresAt).Scan(&resetToken.ID, &resetToken.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create reset token: %w", err)
	}

	return resetToken, nil
}

// GetPasswordResetToken retrieves a reset token by token string
func GetPasswordResetToken(token string) (*PasswordResetToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, used, created_at
		FROM password_reset_tokens
		WHERE token = $1
	`

	resetToken := &PasswordResetToken{}
	err := DB.QueryRow(query, token).Scan(
		&resetToken.ID,
		&resetToken.UserID,
		&resetToken.Token,
		&resetToken.ExpiresAt,
		&resetToken.Used,
		&resetToken.CreatedAt,
	)

	if err != nil {
		return nil, errors.New("invalid or expired reset token")
	}

	// Check if token is expired
	if time.Now().After(resetToken.ExpiresAt) {
		return nil, errors.New("reset token has expired")
	}

	// Check if token has been used
	if resetToken.Used {
		return nil, errors.New("reset token has already been used")
	}

	return resetToken, nil
}

// MarkResetTokenAsUsed marks a reset token as used
func MarkResetTokenAsUsed(tokenID string) error {
	query := `UPDATE password_reset_tokens SET used = TRUE WHERE id = $1`
	_, err := DB.Exec(query, tokenID)
	return err
}

// DeleteExpiredResetTokens removes expired or used tokens (cleanup)
func DeleteExpiredResetTokens() error {
	query := `
		DELETE FROM password_reset_tokens
		WHERE expires_at < NOW() OR used = TRUE
	`
	_, err := DB.Exec(query)
	return err
}
