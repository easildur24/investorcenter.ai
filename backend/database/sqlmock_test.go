package database

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"investorcenter-api/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// setupMock creates a sqlmock DB, wraps it in sqlx, and assigns it to the
// global database.DB. It returns the mock for setting expectations and
// registers cleanup to restore the original DB pointer.
func setupMock(t *testing.T) sqlmock.Sqlmock {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	origDB := DB
	DB = sqlx.NewDb(db, "sqlmock")
	t.Cleanup(func() {
		DB = origDB
		db.Close()
	})
	return mock
}

// ---------------------------------------------------------------------------
// users.go
// ---------------------------------------------------------------------------

func TestCreateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		now := time.Now()

		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs("test@example.com", sqlmock.AnyArg(), "Test User", "UTC", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow("user-123", now, now))

		hash := "hash123"
		user := &models.User{
			Email:        "test@example.com",
			PasswordHash: &hash,
			FullName:     "Test User",
			Timezone:     "UTC",
		}
		err := CreateUser(user)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if user.ID != "user-123" {
			t.Fatalf("expected user ID user-123, got %s", user.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("db_error", func(t *testing.T) {
		mock := setupMock(t)

		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("connection refused"))

		hash := "hash"
		user := &models.User{
			Email:        "test@example.com",
			PasswordHash: &hash,
			FullName:     "Test",
			Timezone:     "UTC",
		}
		err := CreateUser(user)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

// userColumns returns the column names used in GetUserByEmail / GetUserByID / GetUserByPasswordResetToken queries.
func userColumns() []string {
	return []string{
		"id", "email", "password_hash", "full_name", "timezone",
		"created_at", "updated_at", "last_login_at", "email_verified",
		"is_premium", "is_active", "is_admin", "is_worker", "last_activity_at",
	}
}

// userRow returns a representative set of column values for a test user.
func userRow() []driver.Value {
	now := time.Now()
	hash := "hashed"
	return []driver.Value{
		"user-1", "user@example.com", &hash, "Full Name", "UTC",
		now, now, nil, true,
		false, true, false, false, nil,
	}
}

func TestGetUserByEmail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM users WHERE email = \$1`).
			WithArgs("user@example.com").
			WillReturnRows(sqlmock.NewRows(userColumns()).AddRow(userRow()...))

		user, err := GetUserByEmail("user@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.ID != "user-1" {
			t.Fatalf("expected user-1, got %s", user.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM users WHERE email = \$1`).
			WithArgs("nobody@example.com").
			WillReturnError(sql.ErrNoRows)

		_, err := GetUserByEmail("nobody@example.com")
		if err == nil || err.Error() != "user not found" {
			t.Fatalf("expected 'user not found', got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("db_error", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM users WHERE email = \$1`).
			WithArgs("user@example.com").
			WillReturnError(errors.New("db down"))

		_, err := GetUserByEmail("user@example.com")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestGetUserByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM users WHERE id = \$1`).
			WithArgs("user-1").
			WillReturnRows(sqlmock.NewRows(userColumns()).AddRow(userRow()...))

		user, err := GetUserByID("user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.Email != "user@example.com" {
			t.Fatalf("expected user@example.com, got %s", user.Email)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM users WHERE id = \$1`).
			WithArgs("nonexistent").
			WillReturnError(sql.ErrNoRows)

		_, err := GetUserByID("nonexistent")
		if err == nil || err.Error() != "user not found" {
			t.Fatalf("expected 'user not found', got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestUpdateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`UPDATE users`).
			WithArgs("New Name", "America/New_York", "user-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		user := &models.User{ID: "user-1", FullName: "New Name", Timezone: "America/New_York"}
		err := UpdateUser(user)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("db_error", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`UPDATE users`).
			WithArgs("Name", "UTC", "user-1").
			WillReturnError(errors.New("write error"))

		user := &models.User{ID: "user-1", FullName: "Name", Timezone: "UTC"}
		err := UpdateUser(user)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestUpdateUserPassword(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`UPDATE users`).
			WithArgs("newhash", "user-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := UpdateUserPassword("user-1", "newhash")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestUpdateLastLogin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`UPDATE users SET last_login_at`).
			WithArgs(sqlmock.AnyArg(), "user-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := UpdateLastLogin("user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestVerifyEmail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`UPDATE users`).
			WithArgs("token123", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := VerifyEmail("token123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("expired_token", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`UPDATE users`).
			WithArgs("expired-token", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := VerifyEmail("expired-token")
		if err == nil || err.Error() != "invalid or expired verification token" {
			t.Fatalf("expected expired token error, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("db_error", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`UPDATE users`).
			WithArgs("token", sqlmock.AnyArg()).
			WillReturnError(errors.New("db error"))

		err := VerifyEmail("token")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestSetPasswordResetToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		expires := time.Now().Add(time.Hour)
		mock.ExpectExec(`UPDATE users`).
			WithArgs("reset-token", expires, "user@example.com").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := SetPasswordResetToken("user@example.com", "reset-token", expires)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("user_not_found", func(t *testing.T) {
		mock := setupMock(t)
		expires := time.Now().Add(time.Hour)
		mock.ExpectExec(`UPDATE users`).
			WithArgs("reset-token", expires, "nobody@example.com").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := SetPasswordResetToken("nobody@example.com", "reset-token", expires)
		if err == nil || err.Error() != "user not found" {
			t.Fatalf("expected 'user not found', got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestGetUserByPasswordResetToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM users WHERE password_reset_token = \$1`).
			WithArgs("valid-token", sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows(userColumns()).AddRow(userRow()...))

		user, err := GetUserByPasswordResetToken("valid-token")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.ID != "user-1" {
			t.Fatalf("expected user-1, got %s", user.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("expired", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM users WHERE password_reset_token = \$1`).
			WithArgs("expired-token", sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)

		_, err := GetUserByPasswordResetToken("expired-token")
		if err == nil || err.Error() != "invalid or expired reset token" {
			t.Fatalf("expected 'invalid or expired reset token', got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestSoftDeleteUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`UPDATE users SET is_active = FALSE`).
			WithArgs("user-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := SoftDeleteUser("user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// sessions.go
// ---------------------------------------------------------------------------

func TestCreateSession(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		now := time.Now()
		expires := now.Add(24 * time.Hour)

		mock.ExpectQuery(`INSERT INTO sessions`).
			WithArgs("user-1", "tokenhash", expires, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "last_used_at"}).
				AddRow("sess-1", now, now))

		session := &models.Session{
			UserID:           "user-1",
			RefreshTokenHash: "tokenhash",
			ExpiresAt:        expires,
		}
		err := CreateSession(session)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session.ID != "sess-1" {
			t.Fatalf("expected sess-1, got %s", session.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("db_error", func(t *testing.T) {
		mock := setupMock(t)
		expires := time.Now().Add(24 * time.Hour)

		mock.ExpectQuery(`INSERT INTO sessions`).
			WithArgs("user-1", "tokenhash", expires, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("insert failed"))

		session := &models.Session{
			UserID:           "user-1",
			RefreshTokenHash: "tokenhash",
			ExpiresAt:        expires,
		}
		err := CreateSession(session)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestGetSessionByRefreshTokenHash(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		now := time.Now()
		expires := now.Add(24 * time.Hour)
		agent := "Mozilla/5.0"
		ip := "127.0.0.1"

		cols := []string{"id", "user_id", "refresh_token_hash", "expires_at", "created_at", "last_used_at", "user_agent", "ip_address"}
		mock.ExpectQuery(`SELECT .+ FROM sessions WHERE refresh_token_hash = \$1`).
			WithArgs("tokenhash", sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows(cols).
				AddRow("sess-1", "user-1", "tokenhash", expires, now, now, &agent, &ip))

		session, err := GetSessionByRefreshTokenHash("tokenhash")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session.ID != "sess-1" {
			t.Fatalf("expected sess-1, got %s", session.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM sessions WHERE refresh_token_hash = \$1`).
			WithArgs("nonexistent", sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)

		_, err := GetSessionByRefreshTokenHash("nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestDeleteSession(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`DELETE FROM sessions WHERE id = \$1`).
			WithArgs("sess-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := DeleteSession("sess-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestDeleteUserSessions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`DELETE FROM sessions WHERE user_id = \$1`).
			WithArgs("user-1").
			WillReturnResult(sqlmock.NewResult(0, 3))

		err := DeleteUserSessions("user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestCleanupExpiredSessions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`DELETE FROM sessions WHERE expires_at`).
			WithArgs(sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 5))

		err := CleanupExpiredSessions()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// stocks.go
// ---------------------------------------------------------------------------

// stockColumns returns the columns selected in GetStockBySymbol / SearchStocks.
func stockColumns() []string {
	return []string{
		"id", "symbol", "name", "exchange", "sector", "industry",
		"country", "currency", "market_cap", "description", "website",
		"asset_type", "logo_url", "created_at", "updated_at",
	}
}

func stockRow() []driver.Value {
	now := time.Now()
	return []driver.Value{
		1, "AAPL", "Apple Inc.", "NASDAQ", "Technology", "Consumer Electronics",
		"US", "USD", nil, "Apple Inc. designs and manufactures consumer electronics.", "https://apple.com",
		"stock", "https://logo.url/aapl.png", now, now,
	}
}

func TestGetStockBySymbol(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM tickers WHERE`).
			WithArgs("AAPL").
			WillReturnRows(sqlmock.NewRows(stockColumns()).AddRow(stockRow()...))

		stock, err := GetStockBySymbol("AAPL")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stock.Symbol != "AAPL" {
			t.Fatalf("expected AAPL, got %s", stock.Symbol)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM tickers WHERE`).
			WithArgs("ZZZZZ").
			WillReturnError(sql.ErrNoRows)

		_, err := GetStockBySymbol("ZZZZZ")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestSearchStocks(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM tickers WHERE`).
			WithArgs("%AAPL%", "%AAPL%", "AAPL", "AAPL%", "%AAPL%", 10).
			WillReturnRows(sqlmock.NewRows(stockColumns()).AddRow(stockRow()...))

		stocks, err := SearchStocks("AAPL", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(stocks) != 1 {
			t.Fatalf("expected 1 stock, got %d", len(stocks))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("empty_results", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM tickers WHERE`).
			WithArgs("%ZZZZZ%", "%ZZZZZ%", "ZZZZZ", "ZZZZZ%", "%ZZZZZ%", 10).
			WillReturnRows(sqlmock.NewRows(stockColumns()))

		stocks, err := SearchStocks("ZZZZZ", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(stocks) != 0 {
			t.Fatalf("expected 0 stocks, got %d", len(stocks))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

// popularStockColumns returns columns from GetPopularStocks (no asset_type, no logo_url).
func popularStockColumns() []string {
	return []string{
		"id", "symbol", "name", "exchange", "sector", "industry",
		"country", "currency", "market_cap", "description", "website",
		"created_at", "updated_at",
	}
}

func popularStockRow() []driver.Value {
	now := time.Now()
	return []driver.Value{
		1, "AAPL", "Apple Inc.", "NASDAQ", "Technology", "Consumer Electronics",
		"US", "USD", nil, "Apple Inc. designs...", "https://apple.com",
		now, now,
	}
}

func TestGetPopularStocks(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM tickers WHERE symbol IN`).
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows(popularStockColumns()).
				AddRow(popularStockRow()...))

		stocks, err := GetPopularStocks(10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(stocks) != 1 {
			t.Fatalf("expected 1 stock, got %d", len(stocks))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestGetStockCount(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM tickers`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1500))

		count, err := GetStockCount()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 1500 {
			t.Fatalf("expected 1500, got %d", count)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// notifications.go
// ---------------------------------------------------------------------------

func notifPrefColumns() []string {
	return []string{
		"id", "user_id", "email_enabled", "email_address", "email_verified",
		"price_alerts_enabled", "volume_alerts_enabled", "news_alerts_enabled",
		"earnings_alerts_enabled", "sec_filing_alerts_enabled",
		"daily_digest_enabled", "daily_digest_time", "weekly_digest_enabled",
		"weekly_digest_day", "weekly_digest_time",
		"digest_include_portfolio_summary", "digest_include_top_movers",
		"digest_include_recent_alerts", "digest_include_news_highlights",
		"quiet_hours_enabled", "quiet_hours_start", "quiet_hours_end",
		"quiet_hours_timezone", "max_alerts_per_day", "max_emails_per_day",
		"created_at", "updated_at",
	}
}

func notifPrefRow() []driver.Value {
	now := time.Now()
	email := "user@example.com"
	return []driver.Value{
		"pref-1", "user-1", true, &email, true,
		true, false, true,
		true, false,
		true, "08:00", false,
		1, "08:00",
		true, true,
		true, true,
		false, "22:00", "06:00",
		"UTC", 100, 50,
		now, now,
	}
}

func TestGetNotificationPreferences(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM notification_preferences WHERE user_id = \$1`).
			WithArgs("user-1").
			WillReturnRows(sqlmock.NewRows(notifPrefColumns()).AddRow(notifPrefRow()...))

		prefs, err := GetNotificationPreferences("user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if prefs.ID != "pref-1" {
			t.Fatalf("expected pref-1, got %s", prefs.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM notification_preferences WHERE user_id = \$1`).
			WithArgs("user-999").
			WillReturnError(sql.ErrNoRows)

		_, err := GetNotificationPreferences("user-999")
		if err == nil || err.Error() != "notification preferences not found" {
			t.Fatalf("expected 'notification preferences not found', got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestUpdateNotificationPreferences(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		// With a single field, the query is deterministic.
		mock.ExpectExec(`UPDATE notification_preferences SET`).
			WithArgs(true, "user-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		updates := map[string]interface{}{"email_enabled": true}
		err := UpdateNotificationPreferences("user-1", updates)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`UPDATE notification_preferences SET`).
			WithArgs(false, "user-999").
			WillReturnResult(sqlmock.NewResult(0, 0))

		updates := map[string]interface{}{"email_enabled": false}
		err := UpdateNotificationPreferences("user-999", updates)
		if err == nil || err.Error() != "notification preferences not found" {
			t.Fatalf("expected 'notification preferences not found', got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("empty_updates", func(t *testing.T) {
		_ = setupMock(t)
		updates := map[string]interface{}{}
		err := UpdateNotificationPreferences("user-1", updates)
		if err == nil || err.Error() != "no fields to update" {
			t.Fatalf("expected 'no fields to update', got %v", err)
		}
	})
}

func TestCreateInAppNotification(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		now := time.Now()
		expires := now.Add(7 * 24 * time.Hour)

		mock.ExpectQuery(`INSERT INTO notification_queue`).
			WithArgs("user-1", sqlmock.AnyArg(), "alert", "Price Alert", "AAPL is up", sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "expires_at", "is_read", "is_dismissed"}).
				AddRow("notif-1", now, expires, false, false))

		notif := &models.InAppNotification{
			UserID:  "user-1",
			Type:    "alert",
			Title:   "Price Alert",
			Message: "AAPL is up",
		}
		err := CreateInAppNotification(notif)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if notif.ID != "notif-1" {
			t.Fatalf("expected notif-1, got %s", notif.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func inAppNotifColumns() []string {
	return []string{
		"id", "user_id", "alert_log_id", "type", "title", "message", "metadata",
		"is_read", "read_at", "is_dismissed", "dismissed_at", "created_at", "expires_at",
	}
}

func inAppNotifRow() []driver.Value {
	now := time.Now()
	meta, _ := json.Marshal(map[string]string{"symbol": "AAPL"})
	return []driver.Value{
		"notif-1", "user-1", nil, "alert", "Price Alert", "AAPL up 5%", meta,
		false, nil, false, nil, now, now.Add(7 * 24 * time.Hour),
	}
}

func TestGetInAppNotifications(t *testing.T) {
	t.Run("success_with_unread_filter", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM notification_queue WHERE user_id = \$1`).
			WithArgs("user-1", 10).
			WillReturnRows(sqlmock.NewRows(inAppNotifColumns()).AddRow(inAppNotifRow()...))

		notifications, err := GetInAppNotifications("user-1", true, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notifications) != 1 {
			t.Fatalf("expected 1 notification, got %d", len(notifications))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("success_without_unread_filter", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM notification_queue WHERE user_id = \$1`).
			WithArgs("user-1", 20).
			WillReturnRows(sqlmock.NewRows(inAppNotifColumns()).AddRow(inAppNotifRow()...))

		notifications, err := GetInAppNotifications("user-1", false, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notifications) != 1 {
			t.Fatalf("expected 1 notification, got %d", len(notifications))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestMarkNotificationAsRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`UPDATE notification_queue`).
			WithArgs("notif-1", "user-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := MarkNotificationAsRead("notif-1", "user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`UPDATE notification_queue`).
			WithArgs("notif-999", "user-1").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := MarkNotificationAsRead("notif-999", "user-1")
		if err == nil || err.Error() != "notification not found" {
			t.Fatalf("expected 'notification not found', got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestMarkAllNotificationsAsRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`UPDATE notification_queue`).
			WithArgs("user-1").
			WillReturnResult(sqlmock.NewResult(0, 5))

		err := MarkAllNotificationsAsRead("user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestDismissNotification(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`UPDATE notification_queue`).
			WithArgs("notif-1", "user-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := DismissNotification("notif-1", "user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`UPDATE notification_queue`).
			WithArgs("notif-999", "user-1").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := DismissNotification("notif-999", "user-1")
		if err == nil || err.Error() != "notification not found" {
			t.Fatalf("expected 'notification not found', got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestGetUnreadNotificationCount(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT COUNT\(\*\)(.|\n)*FROM notification_queue`).
			WithArgs("user-1").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))

		count, err := GetUnreadNotificationCount("user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 7 {
			t.Fatalf("expected 7, got %d", count)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// alerts.go
// ---------------------------------------------------------------------------

func alertReturnColumns() []string {
	return []string{"id", "created_at", "updated_at", "trigger_count"}
}

func TestCreateAlertRule(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		now := time.Now()
		conditions, _ := json.Marshal(map[string]interface{}{"threshold": 150.0})

		mock.ExpectQuery(`INSERT INTO alert_rules`).
			WithArgs(
				"user-1", "wl-1", sqlmock.AnyArg(), "AAPL", "price_above",
				conditions, true, "once", true, true,
				"AAPL above 150", sqlmock.AnyArg(),
			).
			WillReturnRows(sqlmock.NewRows(alertReturnColumns()).AddRow("alert-1", now, now, 0))

		alert := &models.AlertRule{
			UserID:      "user-1",
			WatchListID: "wl-1",
			Symbol:      "AAPL",
			AlertType:   "price_above",
			Conditions:  conditions,
			IsActive:    true,
			Frequency:   "once",
			NotifyEmail: true,
			NotifyInApp: true,
			Name:        "AAPL above 150",
		}
		err := CreateAlertRule(alert)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if alert.ID != "alert-1" {
			t.Fatalf("expected alert-1, got %s", alert.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("duplicate_error", func(t *testing.T) {
		mock := setupMock(t)
		conditions, _ := json.Marshal(map[string]interface{}{"threshold": 150.0})

		mock.ExpectQuery(`INSERT INTO alert_rules`).
			WithArgs(
				"user-1", "wl-1", sqlmock.AnyArg(), "AAPL", "price_above",
				conditions, true, "once", true, true,
				"AAPL above 150", sqlmock.AnyArg(),
			).
			WillReturnError(&pq.Error{Code: "23505", Message: "duplicate key"})

		alert := &models.AlertRule{
			UserID:      "user-1",
			WatchListID: "wl-1",
			Symbol:      "AAPL",
			AlertType:   "price_above",
			Conditions:  conditions,
			IsActive:    true,
			Frequency:   "once",
			NotifyEmail: true,
			NotifyInApp: true,
			Name:        "AAPL above 150",
		}
		err := CreateAlertRule(alert)
		if !errors.Is(err, ErrAlertAlreadyExists) {
			t.Fatalf("expected ErrAlertAlreadyExists, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func alertRuleColumns() []string {
	return []string{
		"id", "user_id", "watch_list_id", "watch_list_item_id", "symbol", "alert_type",
		"conditions", "is_active", "frequency", "notify_email", "notify_in_app",
		"name", "description", "last_triggered_at", "trigger_count", "created_at", "updated_at",
	}
}

func alertRuleRow() []driver.Value {
	now := time.Now()
	conditions, _ := json.Marshal(map[string]interface{}{"threshold": 150.0})
	return []driver.Value{
		"alert-1", "user-1", "wl-1", nil, "AAPL", "price_above",
		conditions, true, "once", true, true,
		"AAPL above 150", nil, nil, 0, now, now,
	}
}

func TestGetAlertRuleByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM alert_rules WHERE id = \$1`).
			WithArgs("alert-1", "user-1").
			WillReturnRows(sqlmock.NewRows(alertRuleColumns()).AddRow(alertRuleRow()...))

		alert, err := GetAlertRuleByID("alert-1", "user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if alert.Symbol != "AAPL" {
			t.Fatalf("expected AAPL, got %s", alert.Symbol)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM alert_rules WHERE id = \$1`).
			WithArgs("alert-999", "user-1").
			WillReturnError(sql.ErrNoRows)

		_, err := GetAlertRuleByID("alert-999", "user-1")
		if err == nil || err.Error() != "alert rule not found" {
			t.Fatalf("expected 'alert rule not found', got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestUpdateAlertRule(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		// Single key map to make args deterministic.
		mock.ExpectExec(`UPDATE alert_rules SET`).
			WithArgs(false, "alert-1", "user-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		updates := map[string]interface{}{"is_active": false}
		err := UpdateAlertRule("alert-1", "user-1", updates)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`UPDATE alert_rules SET`).
			WithArgs(true, "alert-999", "user-1").
			WillReturnResult(sqlmock.NewResult(0, 0))

		updates := map[string]interface{}{"is_active": true}
		err := UpdateAlertRule("alert-999", "user-1", updates)
		if err == nil || err.Error() != "alert rule not found" {
			t.Fatalf("expected 'alert rule not found', got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("empty_updates", func(t *testing.T) {
		_ = setupMock(t)
		updates := map[string]interface{}{}
		err := UpdateAlertRule("alert-1", "user-1", updates)
		if err == nil || err.Error() != "no fields to update" {
			t.Fatalf("expected 'no fields to update', got %v", err)
		}
	})
}

func TestDeleteAlertRule(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`DELETE FROM alert_rules WHERE id = \$1`).
			WithArgs("alert-1", "user-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := DeleteAlertRule("alert-1", "user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`DELETE FROM alert_rules WHERE id = \$1`).
			WithArgs("alert-999", "user-1").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := DeleteAlertRule("alert-999", "user-1")
		if err == nil || err.Error() != "alert rule not found" {
			t.Fatalf("expected 'alert rule not found', got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestUpdateAlertRuleTrigger(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`UPDATE alert_rules`).
			WithArgs("alert-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := UpdateAlertRuleTrigger("alert-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestCountAlertRulesByUserID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM alert_rules`).
			WithArgs("user-1").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		count, err := CountAlertRulesByUserID("user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 5 {
			t.Fatalf("expected 5, got %d", count)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestAlertExistsForSymbol(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs("wl-1", "AAPL").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		exists, err := AlertExistsForSymbol("wl-1", "AAPL")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Fatal("expected true, got false")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_exists", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs("wl-1", "MSFT").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		exists, err := AlertExistsForSymbol("wl-1", "MSFT")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists {
			t.Fatal("expected false, got true")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestCreateAlertRuleIfNotExists(t *testing.T) {
	t.Run("inserted", func(t *testing.T) {
		mock := setupMock(t)
		now := time.Now()
		conditions, _ := json.Marshal(map[string]interface{}{"threshold": 200.0})

		mock.ExpectQuery(`INSERT INTO alert_rules`).
			WithArgs(
				"user-1", "wl-1", sqlmock.AnyArg(), "AAPL", "price_above",
				conditions, true, "once", true, true,
				"AAPL above 200", sqlmock.AnyArg(),
			).
			WillReturnRows(sqlmock.NewRows(alertReturnColumns()).AddRow("alert-2", now, now, 0))

		alert := &models.AlertRule{
			UserID:      "user-1",
			WatchListID: "wl-1",
			Symbol:      "AAPL",
			AlertType:   "price_above",
			Conditions:  conditions,
			IsActive:    true,
			Frequency:   "once",
			NotifyEmail: true,
			NotifyInApp: true,
			Name:        "AAPL above 200",
		}
		inserted, err := CreateAlertRuleIfNotExists(alert)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !inserted {
			t.Fatal("expected inserted=true, got false")
		}
		if alert.ID != "alert-2" {
			t.Fatalf("expected alert-2, got %s", alert.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("conflict_returns_false", func(t *testing.T) {
		mock := setupMock(t)
		conditions, _ := json.Marshal(map[string]interface{}{"threshold": 200.0})

		mock.ExpectQuery(`INSERT INTO alert_rules`).
			WithArgs(
				"user-1", "wl-1", sqlmock.AnyArg(), "AAPL", "price_above",
				conditions, true, "once", true, true,
				"AAPL above 200", sqlmock.AnyArg(),
			).
			WillReturnError(sql.ErrNoRows)

		alert := &models.AlertRule{
			UserID:      "user-1",
			WatchListID: "wl-1",
			Symbol:      "AAPL",
			AlertType:   "price_above",
			Conditions:  conditions,
			IsActive:    true,
			Frequency:   "once",
			NotifyEmail: true,
			NotifyInApp: true,
			Name:        "AAPL above 200",
		}
		inserted, err := CreateAlertRuleIfNotExists(alert)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inserted {
			t.Fatal("expected inserted=false, got true")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// watchlists.go
// ---------------------------------------------------------------------------

func TestCreateWatchList(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		now := time.Now()

		mock.ExpectQuery(`INSERT INTO watch_lists`).
			WithArgs("user-1", "My List", sqlmock.AnyArg(), false).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "display_order"}).
				AddRow("wl-1", now, now, 0))

		wl := &models.WatchList{
			UserID: "user-1",
			Name:   "My List",
		}
		err := CreateWatchList(wl)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if wl.ID != "wl-1" {
			t.Fatalf("expected wl-1, got %s", wl.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("db_error", func(t *testing.T) {
		mock := setupMock(t)

		mock.ExpectQuery(`INSERT INTO watch_lists`).
			WithArgs("user-1", "My List", sqlmock.AnyArg(), false).
			WillReturnError(errors.New("insert failed"))

		wl := &models.WatchList{
			UserID: "user-1",
			Name:   "My List",
		}
		err := CreateWatchList(wl)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func watchListSummaryColumns() []string {
	return []string{"id", "name", "description", "is_default", "created_at", "updated_at", "item_count"}
}

func TestGetWatchListsByUserID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		now := time.Now()

		mock.ExpectQuery(`SELECT .+ FROM watch_lists wl`).
			WithArgs("user-1").
			WillReturnRows(sqlmock.NewRows(watchListSummaryColumns()).
				AddRow("wl-1", "My List", nil, true, now, now, 3))

		lists, err := GetWatchListsByUserID("user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(lists) != 1 {
			t.Fatalf("expected 1 list, got %d", len(lists))
		}
		if lists[0].ItemCount != 3 {
			t.Fatalf("expected item_count=3, got %d", lists[0].ItemCount)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("empty", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM watch_lists wl`).
			WithArgs("user-1").
			WillReturnRows(sqlmock.NewRows(watchListSummaryColumns()))

		lists, err := GetWatchListsByUserID("user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(lists) != 0 {
			t.Fatalf("expected 0 lists, got %d", len(lists))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestGetWatchListByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		now := time.Now()

		cols := []string{"id", "user_id", "name", "description", "is_default", "display_order", "is_public", "public_slug", "created_at", "updated_at"}
		mock.ExpectQuery(`SELECT .+ FROM watch_lists WHERE id = \$1`).
			WithArgs("wl-1", "user-1").
			WillReturnRows(sqlmock.NewRows(cols).
				AddRow("wl-1", "user-1", "My List", nil, true, 0, false, nil, now, now))

		wl, err := GetWatchListByID("wl-1", "user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if wl.Name != "My List" {
			t.Fatalf("expected 'My List', got %s", wl.Name)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM watch_lists WHERE id = \$1`).
			WithArgs("wl-999", "user-1").
			WillReturnError(sql.ErrNoRows)

		_, err := GetWatchListByID("wl-999", "user-1")
		if !errors.Is(err, ErrWatchListNotFound) {
			t.Fatalf("expected ErrWatchListNotFound, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestUpdateWatchList(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		desc := "updated description"
		mock.ExpectExec(`UPDATE watch_lists`).
			WithArgs("Updated Name", &desc, "wl-1", "user-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		wl := &models.WatchList{ID: "wl-1", UserID: "user-1", Name: "Updated Name", Description: &desc}
		err := UpdateWatchList(wl)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`UPDATE watch_lists`).
			WithArgs("Name", sqlmock.AnyArg(), "wl-999", "user-1").
			WillReturnResult(sqlmock.NewResult(0, 0))

		wl := &models.WatchList{ID: "wl-999", UserID: "user-1", Name: "Name"}
		err := UpdateWatchList(wl)
		if !errors.Is(err, ErrWatchListNotFound) {
			t.Fatalf("expected ErrWatchListNotFound, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestDeleteWatchList(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`DELETE FROM watch_lists WHERE id = \$1`).
			WithArgs("wl-1", "user-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := DeleteWatchList("wl-1", "user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`DELETE FROM watch_lists WHERE id = \$1`).
			WithArgs("wl-999", "user-1").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := DeleteWatchList("wl-999", "user-1")
		if !errors.Is(err, ErrWatchListNotFound) {
			t.Fatalf("expected ErrWatchListNotFound, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestAddTickerToWatchList(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		now := time.Now()

		// First query: verify ticker exists
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs("AAPL").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Second query: insert item
		mock.ExpectQuery(`INSERT INTO watch_list_items`).
			WithArgs("wl-1", "AAPL", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "added_at", "display_order"}).
				AddRow("item-1", now, 0))

		item := &models.WatchListItem{
			WatchListID: "wl-1",
			Symbol:      "AAPL",
			Tags:        []string{},
		}
		err := AddTickerToWatchList(item)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.ID != "item-1" {
			t.Fatalf("expected item-1, got %s", item.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("ticker_not_found", func(t *testing.T) {
		mock := setupMock(t)

		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs("INVALID").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		item := &models.WatchListItem{
			WatchListID: "wl-1",
			Symbol:      "INVALID",
			Tags:        []string{},
		}
		err := AddTickerToWatchList(item)
		if !errors.Is(err, ErrTickerNotFound) {
			t.Fatalf("expected ErrTickerNotFound, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		mock := setupMock(t)

		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs("AAPL").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		mock.ExpectQuery(`INSERT INTO watch_list_items`).
			WithArgs("wl-1", "AAPL", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(&pq.Error{Code: "23505", Message: "duplicate key"})

		item := &models.WatchListItem{
			WatchListID: "wl-1",
			Symbol:      "AAPL",
			Tags:        []string{},
		}
		err := AddTickerToWatchList(item)
		if !errors.Is(err, ErrTickerAlreadyExists) {
			t.Fatalf("expected ErrTickerAlreadyExists, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func watchListItemColumns() []string {
	return []string{"id", "watch_list_id", "symbol", "notes", "tags", "target_buy_price", "target_sell_price", "added_at", "display_order"}
}

func TestGetWatchListItems(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		now := time.Now()

		mock.ExpectQuery(`SELECT .+ FROM watch_list_items WHERE watch_list_id = \$1`).
			WithArgs("wl-1").
			WillReturnRows(sqlmock.NewRows(watchListItemColumns()).
				AddRow("item-1", "wl-1", "AAPL", nil, pq.Array([]string{"tech"}), nil, nil, now, 0))

		items, err := GetWatchListItems("wl-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(items))
		}
		if items[0].Symbol != "AAPL" {
			t.Fatalf("expected AAPL, got %s", items[0].Symbol)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestRemoveTickerFromWatchList(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`DELETE FROM watch_list_items WHERE watch_list_id = \$1`).
			WithArgs("wl-1", "AAPL").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := RemoveTickerFromWatchList("wl-1", "AAPL")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`DELETE FROM watch_list_items WHERE watch_list_id = \$1`).
			WithArgs("wl-1", "ZZZZ").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := RemoveTickerFromWatchList("wl-1", "ZZZZ")
		if !errors.Is(err, ErrWatchListItemNotFound) {
			t.Fatalf("expected ErrWatchListItemNotFound, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestGetWatchListItemByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		now := time.Now()

		mock.ExpectQuery(`SELECT .+ FROM watch_list_items WHERE id = \$1`).
			WithArgs("item-1").
			WillReturnRows(sqlmock.NewRows(watchListItemColumns()).
				AddRow("item-1", "wl-1", "AAPL", nil, pq.Array([]string{}), nil, nil, now, 0))

		item, err := GetWatchListItemByID("item-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.Symbol != "AAPL" {
			t.Fatalf("expected AAPL, got %s", item.Symbol)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM watch_list_items WHERE id = \$1`).
			WithArgs("item-999").
			WillReturnError(sql.ErrNoRows)

		_, err := GetWatchListItemByID("item-999")
		if !errors.Is(err, ErrWatchListItemNotFound) {
			t.Fatalf("expected ErrWatchListItemNotFound, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestUpdateItemDisplayOrder(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectExec(`UPDATE watch_list_items SET display_order`).
			WithArgs(3, "item-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := UpdateItemDisplayOrder("item-1", 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestCreateWatchListAtomic(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		now := time.Now()

		mock.ExpectQuery(`INSERT INTO watch_lists`).
			WithArgs("user-1", "New List", sqlmock.AnyArg(), false, 3).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "display_order"}).
				AddRow("wl-2", now, now, 1))

		wl := &models.WatchList{
			UserID: "user-1",
			Name:   "New List",
		}
		err := CreateWatchListAtomic(wl, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if wl.ID != "wl-2" {
			t.Fatalf("expected wl-2, got %s", wl.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("limit_reached", func(t *testing.T) {
		mock := setupMock(t)

		mock.ExpectQuery(`INSERT INTO watch_lists`).
			WithArgs("user-1", "Fourth List", sqlmock.AnyArg(), false, 3).
			WillReturnError(sql.ErrNoRows)

		wl := &models.WatchList{
			UserID: "user-1",
			Name:   "Fourth List",
		}
		err := CreateWatchListAtomic(wl, 3)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !contains(err.Error(), "watch list limit reached") {
			t.Fatalf("expected limit error, got %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestGetUserTags(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM watch_list_items wli`).
			WithArgs("user-1").
			WillReturnRows(sqlmock.NewRows([]string{"name", "count"}).
				AddRow("tech", 5).
				AddRow("growth", 2))

		tags, err := GetUserTags("user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tags) != 2 {
			t.Fatalf("expected 2 tags, got %d", len(tags))
		}
		if tags[0].Name != "tech" || tags[0].Count != 5 {
			t.Fatalf("unexpected first tag: %+v", tags[0])
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("empty", func(t *testing.T) {
		mock := setupMock(t)
		mock.ExpectQuery(`SELECT .+ FROM watch_list_items wli`).
			WithArgs("user-1").
			WillReturnRows(sqlmock.NewRows([]string{"name", "count"}))

		tags, err := GetUserTags("user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tags) != 0 {
			t.Fatalf("expected 0 tags, got %d", len(tags))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

// contains is a helper that checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
