package database

import (
	"fmt"
	"investorcenter-api/models"
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
