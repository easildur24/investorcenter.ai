package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

// newAdminHandler creates an AdminDataHandler backed by sqlmock.
func newAdminHandler(t *testing.T) (*AdminDataHandler, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	sdb := sqlx.NewDb(db, "sqlmock")
	handler := NewAdminDataHandler(sdb)
	cleanup := func() { db.Close() }
	return handler, mock, cleanup
}

// ---------------------------------------------------------------------------
// GetUsers — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetUsers_Mock_DBError(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	// count query
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	// data query fails
	mock.ExpectQuery("SELECT .+ FROM users").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/admin/users", handler.GetUsers)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetUsers_Mock_Success(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"id", "email", "full_name", "timezone", "created_at", "updated_at",
		"last_login_at", "email_verified", "is_premium", "is_active", "is_admin",
	}).AddRow("u1", "test@test.com", "Test User", "UTC", now, now, nil, true, false, true, false)

	mock.ExpectQuery("SELECT .+ FROM users").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/admin/users", handler.GetUsers)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test@test.com")
}

func TestGetUsers_Mock_WithSearch(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT .+ FROM users").WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "email", "full_name", "timezone", "created_at", "updated_at",
			"last_login_at", "email_verified", "is_premium", "is_active", "is_admin",
		}),
	)

	r := setupMockRouterNoAuth()
	r.GET("/admin/users", handler.GetUsers)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users?search=test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// GetNewsArticles — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetNewsArticles_Mock_DBError(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT .+ FROM news_articles").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/admin/news", handler.GetNewsArticles)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/news", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetNewsArticles_Mock_Success(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"id", "tickers", "title", "summary", "source", "url", "sentiment_label",
		"author", "published_at", "created_at",
	}).AddRow(1, "{AAPL}", "Test Article", "Summary", "Reuters", "http://test.com", "bullish",
		"Author", now, now)

	mock.ExpectQuery("SELECT .+ FROM news_articles").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/admin/news", handler.GetNewsArticles)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/news", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Test Article")
}

// ---------------------------------------------------------------------------
// GetFundamentals — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetFundamentals_Mock_DBError(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/admin/fundamentals", handler.GetFundamentals)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/fundamentals", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetFundamentals_Mock_Success(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"ticker", "calculation_date", "ttm_period_start", "ttm_period_end",
		"ttm_pe_ratio", "ttm_pb_ratio", "ttm_ps_ratio",
		"revenue", "eps_diluted", "ttm_market_cap", "created_at",
	}).AddRow("AAPL", now, now, now, 25.5, 10.2, 5.3, int64(394000000000), 6.57, int64(3000000000000), now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/admin/fundamentals", handler.GetFundamentals)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/fundamentals", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// GetAlerts — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestAdminGetAlerts_Mock_DBError(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT .+ FROM alert_rules").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/admin/alerts", handler.GetAlerts)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/alerts", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAdminGetAlerts_Mock_Success(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "watch_list_id", "symbol", "alert_type",
		"frequency", "notify_email", "notify_in_app", "is_active", "created_at",
	}).AddRow("a1", "u1", "wl1", "AAPL", "price_above", "once", true, true, true, now)

	mock.ExpectQuery("SELECT .+ FROM alert_rules").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/admin/alerts", handler.GetAlerts)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/alerts", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "AAPL")
}

// ---------------------------------------------------------------------------
// GetWatchLists — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestAdminGetWatchLists_Mock_DBError(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT .+ FROM watch_lists").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/admin/watchlists", handler.GetWatchLists)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/watchlists", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAdminGetWatchLists_Mock_Success(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "name", "description", "is_default",
		"is_public", "created_at", "updated_at",
	}).AddRow("wl1", "u1", "My List", "desc", true, false, now, now)

	mock.ExpectQuery("SELECT .+ FROM watch_lists").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/admin/watchlists", handler.GetWatchLists)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/watchlists", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "My List")
}

// ---------------------------------------------------------------------------
// GetSECFinancials — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetSECFinancials_Mock_DBError(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/admin/sec-financials", handler.GetSECFinancials)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/sec-financials", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetSECFinancials_Mock_Success(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"id", "ticker", "period_end_date", "fiscal_year", "fiscal_quarter",
		"revenue", "cost_of_revenue", "gross_profit", "operating_expenses",
		"operating_income", "net_income", "eps_basic", "eps_diluted",
		"shares_outstanding", "total_assets", "total_liabilities",
		"shareholders_equity", "cash_and_equivalents", "short_term_debt",
		"long_term_debt", "roa", "roe", "roic", "gross_margin",
		"operating_margin", "net_margin", "created_at",
	}).AddRow(1, "AAPL", now, int64(2024), int64(4),
		int64(100000), int64(50000), int64(50000), int64(20000),
		int64(30000), int64(25000), 6.0, 5.8,
		int64(15000000), int64(300000), int64(200000),
		int64(100000), int64(50000), int64(10000),
		int64(90000), 0.08, 0.25, 0.15, 0.50,
		0.30, 0.25, now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/admin/sec-financials", handler.GetSECFinancials)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/sec-financials", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "AAPL")
}

// ---------------------------------------------------------------------------
// GetTTMFinancials — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetTTMFinancials_Mock_DBError(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/admin/ttm-financials", handler.GetTTMFinancials)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/ttm-financials", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetTTMFinancials_Mock_Success(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"id", "ticker", "calculation_date", "ttm_period_start", "ttm_period_end",
		"revenue", "cost_of_revenue", "gross_profit", "operating_expenses",
		"operating_income", "net_income", "eps_basic", "eps_diluted",
		"shares_outstanding", "total_assets", "total_liabilities",
		"shareholders_equity", "cash_and_equivalents", "short_term_debt",
		"long_term_debt", "operating_cash_flow", "investing_cash_flow",
		"financing_cash_flow", "free_cash_flow", "capex", "created_at",
	}).AddRow(1, "AAPL", now, now, now,
		int64(394000000000), int64(200000000000), int64(194000000000), int64(60000000000),
		int64(134000000000), int64(97000000000), 6.57, 6.42,
		int64(15000000000), int64(352000000000), int64(290000000000),
		int64(62000000000), int64(50000000000), int64(10000000000),
		int64(109000000000), int64(110000000000), int64(-10000000000),
		int64(-85000000000), int64(100000000000), int64(-10000000000), now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/admin/ttm-financials", handler.GetTTMFinancials)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/ttm-financials", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "AAPL")
}

// ---------------------------------------------------------------------------
// GetValuationRatios — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetValuationRatios_Mock_DBError(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/admin/valuation-ratios", handler.GetValuationRatios)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/valuation-ratios", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetValuationRatios_Mock_Success(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"id", "ticker", "ttm_financial_id", "calculation_date",
		"stock_price", "ttm_market_cap", "ttm_pe_ratio", "ttm_pb_ratio", "ttm_ps_ratio",
		"ttm_period_start", "ttm_period_end", "created_at",
	}).AddRow(int64(1), "AAPL", int64(10), now,
		150.5, int64(3000000000000), 25.5, 10.2, 5.3,
		now, now, now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/admin/valuation-ratios", handler.GetValuationRatios)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/valuation-ratios", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "AAPL")
}

// ---------------------------------------------------------------------------
// GetAnalystRatings — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetAnalystRatings_Mock_DBError(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/admin/analyst-ratings", handler.GetAnalystRatings)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/analyst-ratings", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetAnalystRatings_Mock_Success(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"id", "ticker", "rating_date", "analyst_name", "analyst_firm",
		"rating", "rating_numeric", "price_target", "prior_rating",
		"prior_price_target", "action", "notes", "source", "created_at",
	}).AddRow(int64(1), "AAPL", now, "John Doe", "Goldman Sachs",
		"Buy", 4.0, 200.0, "Hold",
		180.0, "upgrade", "Strong growth", "GS Research", now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/admin/analyst-ratings", handler.GetAnalystRatings)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/analyst-ratings", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Goldman Sachs")
}

// ---------------------------------------------------------------------------
// GetInsiderTrades — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetInsiderTrades_Mock_DBError(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/admin/insider-trades", handler.GetInsiderTrades)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/insider-trades", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetInsiderTrades_Mock_Success(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"id", "ticker", "filing_date", "transaction_date", "insider_name",
		"insider_title", "transaction_type", "shares", "price_per_share",
		"total_value", "shares_owned_after", "is_derivative", "form_type",
		"sec_filing_url", "created_at",
	}).AddRow(int64(1), "AAPL", now, now, "Tim Cook",
		"CEO", "Purchase", int64(10000), 150.25,
		int64(1502500), int64(1000000), false, "4",
		"https://sec.gov/filing", now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/admin/insider-trades", handler.GetInsiderTrades)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/insider-trades", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Tim Cook")
}

// ---------------------------------------------------------------------------
// GetInstitutionalHoldings — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetInstitutionalHoldings_Mock_DBError(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/admin/holdings", handler.GetInstitutionalHoldings)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/holdings", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetInstitutionalHoldings_Mock_Success(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"id", "ticker", "filing_date", "quarter_end_date", "institution_name",
		"institution_cik", "shares", "market_value", "percent_of_portfolio",
		"position_change", "shares_change", "percent_change", "sec_filing_url",
		"created_at",
	}).AddRow(int64(1), "AAPL", now, now, "Berkshire Hathaway",
		"0001067983", int64(900000), int64(135000000), 5.5,
		"increased", int64(100000), 12.5, "https://sec.gov/filing",
		now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/admin/holdings", handler.GetInstitutionalHoldings)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/holdings", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Berkshire Hathaway")
}

// ---------------------------------------------------------------------------
// GetTechnicalIndicators — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetTechnicalIndicators_Mock_DBError(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/admin/indicators", handler.GetTechnicalIndicators)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/indicators", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetTechnicalIndicators_Mock_Success(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"time", "ticker", "indicator_name", "value", "metadata",
	}).AddRow(now, "AAPL", "RSI", 65.5, `{"period":14}`)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/admin/indicators", handler.GetTechnicalIndicators)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/indicators", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "RSI")
}

// ---------------------------------------------------------------------------
// GetCompanies — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetCompanies_Mock_DBError(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/admin/companies", handler.GetCompanies)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/companies", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetCompanies_Mock_Success(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"id", "ticker", "name", "sector", "industry", "market_cap", "country",
		"exchange", "currency", "website", "description", "employees",
		"founded_year", "hq_location", "logo_url", "is_active",
		"last_updated", "created_at",
	}).AddRow(int64(1), "AAPL", "Apple Inc.", "Technology", "Consumer Electronics",
		int64(3000000000000), "US", "NASDAQ", "USD", "https://apple.com",
		"Tech company", int64(160000), int64(1976), "Cupertino, CA",
		"https://logo.com/aapl.png", true, now, now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/admin/companies", handler.GetCompanies)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/companies", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Apple Inc.")
}

// ---------------------------------------------------------------------------
// GetRiskMetrics — DB-backed tests via sqlmock
// ---------------------------------------------------------------------------

func TestGetRiskMetrics_Mock_DBError(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/admin/risk-metrics", handler.GetRiskMetrics)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/risk-metrics", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetRiskMetrics_Mock_Success(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"time", "ticker", "period", "alpha", "beta", "sharpe_ratio", "sortino_ratio",
		"std_dev", "max_drawdown", "var_5", "annualized_return", "downside_deviation",
		"data_points", "calculation_date",
	}).AddRow(now, "AAPL", "1Y", 0.05, 1.2, 1.5, 2.0,
		0.18, -0.15, -0.03, 0.12, 0.10,
		int64(252), now)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/admin/risk-metrics", handler.GetRiskMetrics)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/risk-metrics", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "AAPL")
}

// ---------------------------------------------------------------------------
// GetDatabaseStats — DB-backed tests via sqlmock (increase coverage beyond 70%)
// ---------------------------------------------------------------------------

func TestGetDatabaseStats_Mock_Success(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	// The handler queries COUNT(*) for each table
	tables := []string{
		"tickers", "stock_prices", "fundamentals", "ttm_financials",
		"valuation_ratios", "ic_scores", "news_articles", "insider_trading",
		"analyst_ratings", "technical_indicators", "users", "watch_lists",
		"alert_rules", "user_subscriptions", "reddit_heatmap_daily",
	}
	for range tables {
		mock.ExpectQuery("SELECT COUNT").WillReturnRows(
			sqlmock.NewRows([]string{"count"}).AddRow(100),
		)
	}

	r := setupMockRouterNoAuth()
	r.GET("/admin/stats", handler.GetDatabaseStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "stats")
}

func TestGetDatabaseStats_Mock_SomeTablesFail(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	// Some tables succeed, some fail
	for i := 0; i < 15; i++ {
		if i%3 == 0 {
			mock.ExpectQuery("SELECT COUNT").WillReturnError(fmt.Errorf("table not found"))
		} else {
			mock.ExpectQuery("SELECT COUNT").WillReturnRows(
				sqlmock.NewRows([]string{"count"}).AddRow(42),
			)
		}
	}

	r := setupMockRouterNoAuth()
	r.GET("/admin/stats", handler.GetDatabaseStats)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
	r.ServeHTTP(w, req)

	// Should still return 200, with 0 for failed tables
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// GetStocks with search — to increase from 44% to higher
// ---------------------------------------------------------------------------

func TestGetStocks_Mock_WithSearch(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT .+ FROM tickers").WillReturnRows(
		sqlmock.NewRows([]string{
			"symbol", "name", "exchange", "sector", "industry", "market_cap",
			"description", "country", "currency", "active", "created_at", "updated_at",
		}),
	)

	r := setupMockRouterNoAuth()
	r.GET("/admin/stocks", handler.GetStocks)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/stocks?search=AAPL", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetStocks_Mock_WithSortAndOrder(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"symbol", "name", "exchange", "sector", "industry", "market_cap",
		"description", "country", "currency", "active", "created_at", "updated_at",
	}).AddRow("AAPL", "Apple Inc.", "NASDAQ", "Technology", "Consumer Electronics",
		3000000000000.0, "Tech company", "US", "USD", true, now, now)

	mock.ExpectQuery("SELECT .+ FROM tickers").WillReturnRows(rows)

	r := setupMockRouterNoAuth()
	r.GET("/admin/stocks", handler.GetStocks)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/stocks?sort=market_cap&order=desc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Apple Inc.")
}

func TestGetStocks_Mock_InvalidSort(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT .+ FROM tickers").WillReturnRows(
		sqlmock.NewRows([]string{
			"symbol", "name", "exchange", "sector", "industry", "market_cap",
			"description", "country", "currency", "active", "created_at", "updated_at",
		}),
	)

	r := setupMockRouterNoAuth()
	r.GET("/admin/stocks", handler.GetStocks)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/stocks?sort=invalid_col&order=invalid", nil)
	r.ServeHTTP(w, req)

	// Should fallback to default sort (symbol asc)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetStocks_Mock_DBError(t *testing.T) {
	handler, mock, cleanup := newAdminHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT .+ FROM tickers").WillReturnError(fmt.Errorf("db error"))

	r := setupMockRouterNoAuth()
	r.GET("/admin/stocks", handler.GetStocks)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/stocks", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
