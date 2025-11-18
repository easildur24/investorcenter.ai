package database

import (
	"database/sql"
	"errors"
	"fmt"
	"investorcenter-api/models"
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
		       last_login_at, email_verified, is_premium, is_active, is_admin
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
		&user.IsAdmin,
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
		       last_login_at, email_verified, is_premium, is_active, is_admin
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
		&user.IsAdmin,
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
		       last_login_at, email_verified, is_premium, is_active, is_admin
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
		&user.IsAdmin,
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
