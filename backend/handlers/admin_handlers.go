package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// AdminDataHandler handles admin queries for all data types
type AdminDataHandler struct {
	db *sqlx.DB
}

// NewAdminDataHandler creates a new admin data handler
func NewAdminDataHandler(db *sqlx.DB) *AdminDataHandler {
	return &AdminDataHandler{db: db}
}

// GetStocks returns all stocks with pagination and search
func (h *AdminDataHandler) GetStocks(c *gin.Context) {
	limit := parseQueryInt(c, "limit", 50)
	offset := parseQueryInt(c, "offset", 0)
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort", "symbol")
	order := c.DefaultQuery("order", "asc")

	// Build query
	query := `
		SELECT symbol, name, exchange, sector, industry, market_cap, description,
		       country, currency, active, created_at, updated_at
		FROM tickers
	`
	countQuery := "SELECT COUNT(*) FROM tickers"
	args := []interface{}{}

	if search != "" {
		query += " WHERE symbol ILIKE $1 OR name ILIKE $1"
		countQuery += " WHERE symbol ILIKE $1 OR name ILIKE $1"
		args = append(args, "%"+search+"%")
	}

	// Validate sort column
	validSortColumns := map[string]bool{
		"symbol": true, "name": true, "exchange": true,
		"sector": true, "market_cap": true, "created_at": true,
	}
	if !validSortColumns[sortBy] {
		sortBy = "symbol"
	}
	if order != "asc" && order != "desc" {
		order = "asc"
	}

	query += " ORDER BY " + sortBy + " " + order
	query += " LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	// Get total count
	var total int
	countArgs := args[:len(args)-2] // Exclude limit and offset
	if len(countArgs) == 0 {
		h.db.QueryRow(countQuery).Scan(&total)
	} else {
		h.db.QueryRow(countQuery, countArgs...).Scan(&total)
	}

	// Execute query
	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stocks"})
		return
	}
	defer rows.Close()

	var stocks []map[string]interface{}
	for rows.Next() {
		var symbol, name, exchange, sector, industry, country, currency sql.NullString
		var description sql.NullString
		var marketCap sql.NullFloat64
		var active sql.NullBool
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(&symbol, &name, &exchange, &sector, &industry, &marketCap,
			&description, &country, &currency, &active, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		stock := map[string]interface{}{
			"symbol":      symbol.String,
			"name":        name.String,
			"exchange":    exchange.String,
			"sector":      sector.String,
			"industry":    industry.String,
			"market_cap":  marketCap.Float64,
			"description": description.String,
			"country":     country.String,
			"currency":    currency.String,
			"active":      active.Bool,
			"created_at":  createdAt.Time,
			"updated_at":  updatedAt.Time,
		}
		stocks = append(stocks, stock)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": stocks,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetUsers returns all users (admin only)
func (h *AdminDataHandler) GetUsers(c *gin.Context) {
	limit := parseQueryInt(c, "limit", 50)
	offset := parseQueryInt(c, "offset", 0)
	search := c.Query("search")

	query := `
		SELECT id, email, full_name, timezone, created_at, updated_at,
		       last_login_at, email_verified, is_premium, is_active, is_admin
		FROM users
	`
	countQuery := "SELECT COUNT(*) FROM users"
	args := []interface{}{}

	if search != "" {
		query += " WHERE email ILIKE $1 OR full_name ILIKE $1"
		countQuery += " WHERE email ILIKE $1 OR full_name ILIKE $1"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	var total int
	countArgs := args[:len(args)-2]
	if len(countArgs) == 0 {
		h.db.QueryRow(countQuery).Scan(&total)
	} else {
		h.db.QueryRow(countQuery, countArgs...).Scan(&total)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id, email, fullName, timezone string
		var createdAt, updatedAt sql.NullTime
		var lastLoginAt sql.NullTime
		var emailVerified, isPremium, isActive, isAdmin bool

		err := rows.Scan(&id, &email, &fullName, &timezone, &createdAt, &updatedAt,
			&lastLoginAt, &emailVerified, &isPremium, &isActive, &isAdmin)
		if err != nil {
			continue
		}

		user := map[string]interface{}{
			"id":             id,
			"email":          email,
			"full_name":      fullName,
			"timezone":       timezone,
			"created_at":     createdAt.Time,
			"updated_at":     updatedAt.Time,
			"last_login_at":  nil,
			"email_verified": emailVerified,
			"is_premium":     isPremium,
			"is_active":      isActive,
			"is_admin":       isAdmin,
		}
		if lastLoginAt.Valid {
			user["last_login_at"] = lastLoginAt.Time
		}
		users = append(users, user)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": users,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetNewsArticles returns all news articles with pagination
func (h *AdminDataHandler) GetNewsArticles(c *gin.Context) {
	limit := parseQueryInt(c, "limit", 50)
	offset := parseQueryInt(c, "offset", 0)
	search := c.Query("search")

	query := `
		SELECT id, tickers, title, summary, source, url, sentiment_label,
		       author, published_at, created_at
		FROM news_articles
	`
	countQuery := "SELECT COUNT(*) FROM news_articles"
	args := []interface{}{}

	if search != "" {
		query += " WHERE title ILIKE $1"
		countQuery += " WHERE title ILIKE $1"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY published_at DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	var total int
	countArgs := args[:len(args)-2]
	if len(countArgs) == 0 {
		h.db.QueryRow(countQuery).Scan(&total)
	} else {
		h.db.QueryRow(countQuery, countArgs...).Scan(&total)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch news", "details": err.Error()})
		return
	}
	defer rows.Close()

	var articles []map[string]interface{}
	for rows.Next() {
		var id int
		var title, summary, source, url, sentimentLabel, author sql.NullString
		var tickers pq.StringArray
		var publishedAt, createdAt sql.NullTime

		err := rows.Scan(&id, &tickers, &title, &summary, &source, &url, &sentimentLabel,
			&author, &publishedAt, &createdAt)
		if err != nil {
			continue
		}

		article := map[string]interface{}{
			"id":           id,
			"tickers":      tickers,
			"title":        title.String,
			"summary":      summary.String,
			"source":       source.String,
			"url":          url.String,
			"sentiment":    sentimentLabel.String,
			"author":       author.String,
			"published_at": publishedAt.Time,
			"created_at":   createdAt.Time,
		}
		articles = append(articles, article)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": articles,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetFundamentals returns all fundamentals data
func (h *AdminDataHandler) GetFundamentals(c *gin.Context) {
	limit := parseQueryInt(c, "limit", 50)
	offset := parseQueryInt(c, "offset", 0)
	search := c.Query("search")

	query := `
		SELECT
			COALESCE(t.ticker, v.ticker) as ticker,
			COALESCE(t.calculation_date, v.calculation_date) as calculation_date,
			t.ttm_period_start,
			t.ttm_period_end,
			v.ttm_pe_ratio,
			v.ttm_pb_ratio,
			v.ttm_ps_ratio,
			t.revenue,
			t.eps_diluted,
			v.ttm_market_cap,
			COALESCE(t.created_at, v.created_at) as created_at
		FROM ttm_financials t
		FULL OUTER JOIN valuation_ratios v ON t.ticker = v.ticker AND t.calculation_date = v.calculation_date
	`
	countQuery := `
		SELECT COUNT(*)
		FROM ttm_financials t
		FULL OUTER JOIN valuation_ratios v ON t.ticker = v.ticker AND t.calculation_date = v.calculation_date
	`
	args := []interface{}{}

	if search != "" {
		query += " WHERE COALESCE(t.ticker, v.ticker) ILIKE $1"
		countQuery += " WHERE COALESCE(t.ticker, v.ticker) ILIKE $1"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY COALESCE(t.ticker, v.ticker), COALESCE(t.created_at, v.created_at) DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	var total int
	countArgs := args[:len(args)-2]
	if len(countArgs) == 0 {
		h.db.QueryRow(countQuery).Scan(&total)
	} else {
		h.db.QueryRow(countQuery, countArgs...).Scan(&total)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch fundamentals", "details": err.Error()})
		return
	}
	defer rows.Close()

	var fundamentals []map[string]interface{}
	for rows.Next() {
		var ticker sql.NullString
		var calculationDate, ttmPeriodStart, ttmPeriodEnd sql.NullTime
		var peRatio, pbRatio, psRatio sql.NullFloat64
		var revenue, marketCap sql.NullInt64
		var epsDiluted sql.NullFloat64
		var createdAt sql.NullTime

		err := rows.Scan(&ticker, &calculationDate, &ttmPeriodStart, &ttmPeriodEnd,
			&peRatio, &pbRatio, &psRatio, &revenue, &epsDiluted, &marketCap, &createdAt)
		if err != nil {
			continue
		}

		fundamental := map[string]interface{}{
			"ticker":           ticker.String,
			"calculation_date": calculationDate.Time,
			"ttm_period_start": ttmPeriodStart.Time,
			"ttm_period_end":   ttmPeriodEnd.Time,
			"pe_ratio":         peRatio.Float64,
			"pb_ratio":         pbRatio.Float64,
			"ps_ratio":         psRatio.Float64,
			"revenue":          revenue.Int64,
			"eps":              epsDiluted.Float64,
			"market_cap":       marketCap.Int64,
			"created_at":       createdAt.Time,
		}
		fundamentals = append(fundamentals, fundamental)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": fundamentals,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetAlerts returns all alert rules
func (h *AdminDataHandler) GetAlerts(c *gin.Context) {
	limit := parseQueryInt(c, "limit", 50)
	offset := parseQueryInt(c, "offset", 0)

	query := `
		SELECT id, user_id, watch_list_id, symbol, alert_type,
		       frequency, notify_email, notify_in_app, is_active, created_at
		FROM alert_rules
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	var total int
	h.db.QueryRow("SELECT COUNT(*) FROM alert_rules").Scan(&total)

	rows, err := h.db.Query(query, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch alerts"})
		return
	}
	defer rows.Close()

	var alerts []map[string]interface{}
	for rows.Next() {
		var id, userID sql.NullString
		var watchListID sql.NullString
		var symbol, alertType, frequency sql.NullString
		var notifyEmail, notifyInApp, isActive sql.NullBool
		var createdAt sql.NullTime

		err := rows.Scan(&id, &userID, &watchListID, &symbol, &alertType,
			&frequency, &notifyEmail, &notifyInApp, &isActive, &createdAt)
		if err != nil {
			continue
		}

		alert := map[string]interface{}{
			"id":            id.String,
			"user_id":       userID.String,
			"watch_list_id": watchListID.String,
			"symbol":        symbol.String,
			"alert_type":    alertType.String,
			"frequency":     frequency.String,
			"notify_email":  notifyEmail.Bool,
			"notify_in_app": notifyInApp.Bool,
			"is_active":     isActive.Bool,
			"created_at":    createdAt.Time,
		}
		alerts = append(alerts, alert)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": alerts,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetWatchLists returns all watch lists
func (h *AdminDataHandler) GetWatchLists(c *gin.Context) {
	limit := parseQueryInt(c, "limit", 50)
	offset := parseQueryInt(c, "offset", 0)

	query := `
		SELECT id, user_id, name, description, is_default,
		       is_public, created_at, updated_at
		FROM watch_lists
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	var total int
	h.db.QueryRow("SELECT COUNT(*) FROM watch_lists").Scan(&total)

	rows, err := h.db.Query(query, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch watch lists"})
		return
	}
	defer rows.Close()

	var watchLists []map[string]interface{}
	for rows.Next() {
		var id, userID, name sql.NullString
		var description sql.NullString
		var isDefault, isPublic sql.NullBool
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(&id, &userID, &name, &description, &isDefault,
			&isPublic, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		watchList := map[string]interface{}{
			"id":          id.String,
			"user_id":     userID.String,
			"name":        name.String,
			"description": description.String,
			"is_default":  isDefault.Bool,
			"is_public":   isPublic.Bool,
			"created_at":  createdAt.Time,
			"updated_at":  updatedAt.Time,
		}
		watchLists = append(watchLists, watchList)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": watchLists,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetSECFinancials returns raw quarterly SEC financial data
func (h *AdminDataHandler) GetSECFinancials(c *gin.Context) {
	limit := parseQueryInt(c, "limit", 50)
	offset := parseQueryInt(c, "offset", 0)
	search := c.Query("search")

	query := `
		SELECT
			id, ticker, period_end_date, fiscal_year, fiscal_quarter,
			revenue, cost_of_revenue, gross_profit, operating_expenses,
			operating_income, net_income, eps_basic, eps_diluted,
			shares_outstanding, total_assets, total_liabilities,
			shareholders_equity, cash_and_equivalents, short_term_debt,
			long_term_debt, roa, roe, roic, gross_margin,
			operating_margin, net_margin, created_at
		FROM financials
	`
	countQuery := "SELECT COUNT(*) FROM financials"
	args := []interface{}{}

	if search != "" {
		query += " WHERE ticker ILIKE $1"
		countQuery += " WHERE ticker ILIKE $1"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY period_end_date DESC, ticker LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	var total int
	countArgs := args[:len(args)-2]
	if len(countArgs) == 0 {
		h.db.QueryRow(countQuery).Scan(&total)
	} else {
		h.db.QueryRow(countQuery, countArgs...).Scan(&total)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch SEC financials", "details": err.Error()})
		return
	}
	defer rows.Close()

	var financials []map[string]interface{}
	for rows.Next() {
		var id int
		var ticker sql.NullString
		var periodEndDate sql.NullTime
		var fiscalYear, fiscalQuarter sql.NullInt64
		var revenue, costOfRevenue, grossProfit, operatingExpenses sql.NullInt64
		var operatingIncome, netIncome sql.NullInt64
		var epsBasic, epsDiluted sql.NullFloat64
		var sharesOutstanding sql.NullInt64
		var totalAssets, totalLiabilities, shareholdersEquity sql.NullInt64
		var cash, shortTermDebt, longTermDebt sql.NullInt64
		var roa, roe, roic sql.NullFloat64
		var grossMargin, operatingMargin, netMargin sql.NullFloat64
		var createdAt sql.NullTime

		err := rows.Scan(&id, &ticker, &periodEndDate, &fiscalYear, &fiscalQuarter,
			&revenue, &costOfRevenue, &grossProfit, &operatingExpenses,
			&operatingIncome, &netIncome, &epsBasic, &epsDiluted,
			&sharesOutstanding, &totalAssets, &totalLiabilities,
			&shareholdersEquity, &cash, &shortTermDebt, &longTermDebt,
			&roa, &roe, &roic, &grossMargin, &operatingMargin, &netMargin, &createdAt)
		if err != nil {
			continue
		}

		financial := map[string]interface{}{
			"id":                  id,
			"ticker":              ticker.String,
			"period_end_date":     periodEndDate.Time,
			"fiscal_year":         fiscalYear.Int64,
			"fiscal_quarter":      fiscalQuarter.Int64,
			"revenue":             revenue.Int64,
			"cost_of_revenue":     costOfRevenue.Int64,
			"gross_profit":        grossProfit.Int64,
			"operating_expenses":  operatingExpenses.Int64,
			"operating_income":    operatingIncome.Int64,
			"net_income":          netIncome.Int64,
			"eps_basic":           epsBasic.Float64,
			"eps_diluted":         epsDiluted.Float64,
			"shares_outstanding":  sharesOutstanding.Int64,
			"total_assets":        totalAssets.Int64,
			"total_liabilities":   totalLiabilities.Int64,
			"shareholders_equity": shareholdersEquity.Int64,
			"cash":                cash.Int64,
			"short_term_debt":     shortTermDebt.Int64,
			"long_term_debt":      longTermDebt.Int64,
			"roa":                 roa.Float64,
			"roe":                 roe.Float64,
			"roic":                roic.Float64,
			"gross_margin":        grossMargin.Float64,
			"operating_margin":    operatingMargin.Float64,
			"net_margin":          netMargin.Float64,
			"created_at":          createdAt.Time,
		}
		financials = append(financials, financial)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": financials,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetTTMFinancials returns TTM financial data
func (h *AdminDataHandler) GetTTMFinancials(c *gin.Context) {
	limit := parseQueryInt(c, "limit", 50)
	offset := parseQueryInt(c, "offset", 0)
	search := c.Query("search")

	query := `
		SELECT
			id, ticker, calculation_date, ttm_period_start, ttm_period_end,
			revenue, cost_of_revenue, gross_profit, operating_expenses,
			operating_income, net_income, eps_basic, eps_diluted,
			shares_outstanding, total_assets, total_liabilities,
			shareholders_equity, cash_and_equivalents, short_term_debt,
			long_term_debt, operating_cash_flow, investing_cash_flow,
			financing_cash_flow, free_cash_flow, capex, created_at
		FROM ttm_financials
	`
	countQuery := "SELECT COUNT(*) FROM ttm_financials"
	args := []interface{}{}

	if search != "" {
		query += " WHERE ticker ILIKE $1"
		countQuery += " WHERE ticker ILIKE $1"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY calculation_date DESC, ticker LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	var total int
	countArgs := args[:len(args)-2]
	if len(countArgs) == 0 {
		h.db.QueryRow(countQuery).Scan(&total)
	} else {
		h.db.QueryRow(countQuery, countArgs...).Scan(&total)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch TTM financials", "details": err.Error()})
		return
	}
	defer rows.Close()

	var ttmFinancials []map[string]interface{}
	for rows.Next() {
		var id int
		var ticker sql.NullString
		var calculationDate, ttmPeriodStart, ttmPeriodEnd sql.NullTime
		var revenue, costOfRevenue, grossProfit, operatingExpenses sql.NullInt64
		var operatingIncome, netIncome sql.NullInt64
		var epsBasic, epsDiluted sql.NullFloat64
		var sharesOutstanding sql.NullInt64
		var totalAssets, totalLiabilities, shareholdersEquity sql.NullInt64
		var cash, shortTermDebt, longTermDebt sql.NullInt64
		var operatingCashFlow, investingCashFlow, financingCashFlow sql.NullInt64
		var freeCashFlow, capex sql.NullInt64
		var createdAt sql.NullTime

		err := rows.Scan(&id, &ticker, &calculationDate, &ttmPeriodStart, &ttmPeriodEnd,
			&revenue, &costOfRevenue, &grossProfit, &operatingExpenses,
			&operatingIncome, &netIncome, &epsBasic, &epsDiluted,
			&sharesOutstanding, &totalAssets, &totalLiabilities,
			&shareholdersEquity, &cash, &shortTermDebt, &longTermDebt,
			&operatingCashFlow, &investingCashFlow, &financingCashFlow,
			&freeCashFlow, &capex, &createdAt)
		if err != nil {
			continue
		}

		ttmFinancial := map[string]interface{}{
			"id":                   id,
			"ticker":               ticker.String,
			"calculation_date":     calculationDate.Time,
			"ttm_period_start":     ttmPeriodStart.Time,
			"ttm_period_end":       ttmPeriodEnd.Time,
			"revenue":              revenue.Int64,
			"cost_of_revenue":      costOfRevenue.Int64,
			"gross_profit":         grossProfit.Int64,
			"operating_expenses":   operatingExpenses.Int64,
			"operating_income":     operatingIncome.Int64,
			"net_income":           netIncome.Int64,
			"eps_basic":            epsBasic.Float64,
			"eps_diluted":          epsDiluted.Float64,
			"shares_outstanding":   sharesOutstanding.Int64,
			"total_assets":         totalAssets.Int64,
			"total_liabilities":    totalLiabilities.Int64,
			"shareholders_equity":  shareholdersEquity.Int64,
			"cash_and_equivalents": cash.Int64,
			"short_term_debt":      shortTermDebt.Int64,
			"long_term_debt":       longTermDebt.Int64,
			"operating_cash_flow":  operatingCashFlow.Int64,
			"investing_cash_flow":  investingCashFlow.Int64,
			"financing_cash_flow":  financingCashFlow.Int64,
			"free_cash_flow":       freeCashFlow.Int64,
			"capex":                capex.Int64,
			"created_at":           createdAt.Time,
		}
		ttmFinancials = append(ttmFinancials, ttmFinancial)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": ttmFinancials,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetValuationRatios returns valuation ratios data
func (h *AdminDataHandler) GetValuationRatios(c *gin.Context) {
	limit := parseQueryInt(c, "limit", 50)
	offset := parseQueryInt(c, "offset", 0)
	search := c.Query("search")

	query := `
		SELECT
			id, ticker, ttm_financial_id, calculation_date,
			stock_price, ttm_market_cap, ttm_pe_ratio, ttm_pb_ratio,
			ttm_ps_ratio, ttm_period_start, ttm_period_end,
			created_at
		FROM valuation_ratios
	`
	countQuery := "SELECT COUNT(*) FROM valuation_ratios"
	args := []interface{}{}

	if search != "" {
		query += " WHERE ticker ILIKE $1"
		countQuery += " WHERE ticker ILIKE $1"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY calculation_date DESC, ticker LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	var total int
	countArgs := args[:len(args)-2]
	if len(countArgs) == 0 {
		h.db.QueryRow(countQuery).Scan(&total)
	} else {
		h.db.QueryRow(countQuery, countArgs...).Scan(&total)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch valuation ratios", "details": err.Error()})
		return
	}
	defer rows.Close()

	var valuationRatios []map[string]interface{}
	for rows.Next() {
		var id, ttmFinancialID sql.NullInt64
		var ticker sql.NullString
		var calculationDate, ttmPeriodStart, ttmPeriodEnd sql.NullTime
		var stockPrice sql.NullFloat64
		var marketCap sql.NullInt64
		var peRatio, pbRatio, psRatio sql.NullFloat64
		var createdAt sql.NullTime

		err := rows.Scan(&id, &ticker, &ttmFinancialID, &calculationDate,
			&stockPrice, &marketCap, &peRatio, &pbRatio, &psRatio,
			&ttmPeriodStart, &ttmPeriodEnd, &createdAt)
		if err != nil {
			continue
		}

		valuationRatio := map[string]interface{}{
			"id":               id.Int64,
			"ticker":           ticker.String,
			"ttm_financial_id": ttmFinancialID.Int64,
			"calculation_date": calculationDate.Time,
			"stock_price":      stockPrice.Float64,
			"ttm_market_cap":   marketCap.Int64,
			"ttm_pe_ratio": func() interface{} {
				if peRatio.Valid {
					return peRatio.Float64
				} else {
					return nil
				}
			}(),
			"ttm_pb_ratio": func() interface{} {
				if pbRatio.Valid {
					return pbRatio.Float64
				} else {
					return nil
				}
			}(),
			"ttm_ps_ratio": func() interface{} {
				if psRatio.Valid {
					return psRatio.Float64
				} else {
					return nil
				}
			}(),
			"ttm_period_start": ttmPeriodStart.Time,
			"ttm_period_end":   ttmPeriodEnd.Time,
			"created_at":       createdAt.Time,
		}
		valuationRatios = append(valuationRatios, valuationRatio)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": valuationRatios,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetDatabaseStats returns database statistics
func (h *AdminDataHandler) GetDatabaseStats(c *gin.Context) {
	stats := make(map[string]interface{})

	// Count all tables
	tables := []string{
		"tickers", "stock_prices", "fundamentals", "ttm_financials",
		"valuation_ratios", "ic_scores", "news_articles", "insider_trading",
		"analyst_ratings", "technical_indicators", "users", "watch_lists",
		"alert_rules", "user_subscriptions", "reddit_heatmap_daily",
	}

	for _, table := range tables {
		var count int
		query := "SELECT COUNT(*) FROM " + table
		err := h.db.QueryRow(query).Scan(&count)
		if err == nil {
			stats[table] = count
		} else {
			stats[table] = 0
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

// GetAnalystRatings returns analyst ratings data
func (h *AdminDataHandler) GetAnalystRatings(c *gin.Context) {
	limit := parseQueryInt(c, "limit", 50)
	offset := parseQueryInt(c, "offset", 0)
	search := c.Query("search")

	query := `
		SELECT
			id, ticker, rating_date, analyst_name, analyst_firm,
			rating, rating_numeric, price_target, prior_rating,
			prior_price_target, action, notes, source, created_at
		FROM analyst_ratings
	`
	countQuery := "SELECT COUNT(*) FROM analyst_ratings"
	args := []interface{}{}

	if search != "" {
		query += " WHERE ticker ILIKE $1"
		countQuery += " WHERE ticker ILIKE $1"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY rating_date DESC, ticker LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	var total int
	countArgs := args[:len(args)-2]
	if len(countArgs) == 0 {
		h.db.QueryRow(countQuery).Scan(&total)
	} else {
		h.db.QueryRow(countQuery, countArgs...).Scan(&total)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch analyst ratings", "details": err.Error()})
		return
	}
	defer rows.Close()

	var ratings []map[string]interface{}
	for rows.Next() {
		var id sql.NullInt64
		var ticker, analystName, analystFirm, rating, action, notes, source sql.NullString
		var ratingDate sql.NullTime
		var ratingNumeric, priceTarget, priorPriceTarget sql.NullFloat64
		var priorRating sql.NullString
		var createdAt sql.NullTime

		err := rows.Scan(&id, &ticker, &ratingDate, &analystName, &analystFirm,
			&rating, &ratingNumeric, &priceTarget, &priorRating,
			&priorPriceTarget, &action, &notes, &source, &createdAt)
		if err != nil {
			continue
		}

		ratingData := map[string]interface{}{
			"id":           id.Int64,
			"ticker":       ticker.String,
			"rating_date":  ratingDate.Time,
			"analyst_name": analystName.String,
			"analyst_firm": analystFirm.String,
			"rating":       rating.String,
			"rating_numeric": func() interface{} {
				if ratingNumeric.Valid {
					return ratingNumeric.Float64
				} else {
					return nil
				}
			}(),
			"price_target": func() interface{} {
				if priceTarget.Valid {
					return priceTarget.Float64
				} else {
					return nil
				}
			}(),
			"prior_rating": priorRating.String,
			"prior_price_target": func() interface{} {
				if priorPriceTarget.Valid {
					return priorPriceTarget.Float64
				} else {
					return nil
				}
			}(),
			"action":     action.String,
			"notes":      notes.String,
			"source":     source.String,
			"created_at": createdAt.Time,
		}
		ratings = append(ratings, ratingData)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": ratings,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetInsiderTrades returns insider trading data
func (h *AdminDataHandler) GetInsiderTrades(c *gin.Context) {
	limit := parseQueryInt(c, "limit", 50)
	offset := parseQueryInt(c, "offset", 0)
	search := c.Query("search")

	query := `
		SELECT
			id, ticker, filing_date, transaction_date, insider_name,
			insider_title, transaction_type, shares, price_per_share,
			total_value, shares_owned_after, is_derivative, form_type,
			sec_filing_url, created_at
		FROM insider_trades
	`
	countQuery := "SELECT COUNT(*) FROM insider_trades"
	args := []interface{}{}

	if search != "" {
		query += " WHERE ticker ILIKE $1"
		countQuery += " WHERE ticker ILIKE $1"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY transaction_date DESC, ticker LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	var total int
	countArgs := args[:len(args)-2]
	if len(countArgs) == 0 {
		h.db.QueryRow(countQuery).Scan(&total)
	} else {
		h.db.QueryRow(countQuery, countArgs...).Scan(&total)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch insider trades", "details": err.Error()})
		return
	}
	defer rows.Close()

	var trades []map[string]interface{}
	for rows.Next() {
		var id sql.NullInt64
		var ticker, insiderName, insiderTitle, transactionType, formType, secFilingURL sql.NullString
		var filingDate, transactionDate sql.NullTime
		var shares, totalValue, sharesOwnedAfter sql.NullInt64
		var pricePerShare sql.NullFloat64
		var isDerivative sql.NullBool
		var createdAt sql.NullTime

		err := rows.Scan(&id, &ticker, &filingDate, &transactionDate, &insiderName,
			&insiderTitle, &transactionType, &shares, &pricePerShare,
			&totalValue, &sharesOwnedAfter, &isDerivative, &formType,
			&secFilingURL, &createdAt)
		if err != nil {
			continue
		}

		trade := map[string]interface{}{
			"id":               id.Int64,
			"ticker":           ticker.String,
			"filing_date":      filingDate.Time,
			"transaction_date": transactionDate.Time,
			"insider_name":     insiderName.String,
			"insider_title":    insiderTitle.String,
			"transaction_type": transactionType.String,
			"shares":           shares.Int64,
			"price_per_share": func() interface{} {
				if pricePerShare.Valid {
					return pricePerShare.Float64
				} else {
					return nil
				}
			}(),
			"total_value":        totalValue.Int64,
			"shares_owned_after": sharesOwnedAfter.Int64,
			"is_derivative":      isDerivative.Bool,
			"form_type":          formType.String,
			"sec_filing_url":     secFilingURL.String,
			"created_at":         createdAt.Time,
		}
		trades = append(trades, trade)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": trades,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetInstitutionalHoldings returns institutional holdings data (13F filings)
func (h *AdminDataHandler) GetInstitutionalHoldings(c *gin.Context) {
	limit := parseQueryInt(c, "limit", 50)
	offset := parseQueryInt(c, "offset", 0)
	search := c.Query("search")

	query := `
		SELECT
			id, ticker, filing_date, quarter_end_date, institution_name,
			institution_cik, shares, market_value, percent_of_portfolio,
			position_change, shares_change, percent_change, sec_filing_url,
			created_at
		FROM institutional_holdings
	`
	countQuery := "SELECT COUNT(*) FROM institutional_holdings"
	args := []interface{}{}

	if search != "" {
		query += " WHERE ticker ILIKE $1"
		countQuery += " WHERE ticker ILIKE $1"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY quarter_end_date DESC, ticker LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	var total int
	countArgs := args[:len(args)-2]
	if len(countArgs) == 0 {
		h.db.QueryRow(countQuery).Scan(&total)
	} else {
		h.db.QueryRow(countQuery, countArgs...).Scan(&total)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch institutional holdings", "details": err.Error()})
		return
	}
	defer rows.Close()

	var holdings []map[string]interface{}
	for rows.Next() {
		var id sql.NullInt64
		var ticker, institutionName, institutionCIK, positionChange, secFilingURL sql.NullString
		var filingDate, quarterEndDate sql.NullTime
		var shares, marketValue, sharesChange sql.NullInt64
		var percentOfPortfolio, percentChange sql.NullFloat64
		var createdAt sql.NullTime

		err := rows.Scan(&id, &ticker, &filingDate, &quarterEndDate, &institutionName,
			&institutionCIK, &shares, &marketValue, &percentOfPortfolio,
			&positionChange, &sharesChange, &percentChange, &secFilingURL,
			&createdAt)
		if err != nil {
			continue
		}

		holding := map[string]interface{}{
			"id":               id.Int64,
			"ticker":           ticker.String,
			"filing_date":      filingDate.Time,
			"quarter_end_date": quarterEndDate.Time,
			"institution_name": institutionName.String,
			"institution_cik":  institutionCIK.String,
			"shares":           shares.Int64,
			"market_value":     marketValue.Int64,
			"percent_of_portfolio": func() interface{} {
				if percentOfPortfolio.Valid {
					return percentOfPortfolio.Float64
				} else {
					return nil
				}
			}(),
			"position_change": positionChange.String,
			"shares_change":   sharesChange.Int64,
			"percent_change": func() interface{} {
				if percentChange.Valid {
					return percentChange.Float64
				} else {
					return nil
				}
			}(),
			"sec_filing_url": secFilingURL.String,
			"created_at":     createdAt.Time,
		}
		holdings = append(holdings, holding)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": holdings,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetTechnicalIndicators returns technical indicators data
func (h *AdminDataHandler) GetTechnicalIndicators(c *gin.Context) {
	limit := parseQueryInt(c, "limit", 50)
	offset := parseQueryInt(c, "offset", 0)
	search := c.Query("search")

	query := `
		SELECT
			time, ticker, indicator_name, value, metadata
		FROM technical_indicators
	`
	countQuery := "SELECT COUNT(*) FROM technical_indicators"
	args := []interface{}{}

	if search != "" {
		query += " WHERE ticker ILIKE $1"
		countQuery += " WHERE ticker ILIKE $1"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY time DESC, ticker LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	var total int
	countArgs := args[:len(args)-2]
	if len(countArgs) == 0 {
		h.db.QueryRow(countQuery).Scan(&total)
	} else {
		h.db.QueryRow(countQuery, countArgs...).Scan(&total)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch technical indicators", "details": err.Error()})
		return
	}
	defer rows.Close()

	var indicators []map[string]interface{}
	for rows.Next() {
		var time sql.NullTime
		var ticker, indicatorName sql.NullString
		var value sql.NullFloat64
		var metadata sql.NullString

		err := rows.Scan(&time, &ticker, &indicatorName, &value, &metadata)
		if err != nil {
			continue
		}

		indicator := map[string]interface{}{
			"time":           time.Time,
			"ticker":         ticker.String,
			"indicator_name": indicatorName.String,
			"value": func() interface{} {
				if value.Valid {
					return value.Float64
				} else {
					return nil
				}
			}(),
			"metadata": metadata.String,
		}
		indicators = append(indicators, indicator)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": indicators,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetCompanies returns companies master data
func (h *AdminDataHandler) GetCompanies(c *gin.Context) {
	limit := parseQueryInt(c, "limit", 50)
	offset := parseQueryInt(c, "offset", 0)
	search := c.Query("search")

	query := `
		SELECT
			id, ticker, name, sector, industry, market_cap, country,
			exchange, currency, website, description, employees,
			founded_year, hq_location, logo_url, is_active,
			last_updated, created_at
		FROM companies
	`
	countQuery := "SELECT COUNT(*) FROM companies"
	args := []interface{}{}

	if search != "" {
		query += " WHERE ticker ILIKE $1 OR name ILIKE $1"
		countQuery += " WHERE ticker ILIKE $1 OR name ILIKE $1"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY market_cap DESC NULLS LAST, ticker LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	var total int
	countArgs := args[:len(args)-2]
	if len(countArgs) == 0 {
		h.db.QueryRow(countQuery).Scan(&total)
	} else {
		h.db.QueryRow(countQuery, countArgs...).Scan(&total)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch companies", "details": err.Error()})
		return
	}
	defer rows.Close()

	var companies []map[string]interface{}
	for rows.Next() {
		var id sql.NullInt64
		var ticker, name, sector, industry, country, exchange, currency sql.NullString
		var website, description, hqLocation, logoURL sql.NullString
		var marketCap sql.NullInt64
		var employees, foundedYear sql.NullInt64
		var isActive sql.NullBool
		var lastUpdated, createdAt sql.NullTime

		err := rows.Scan(&id, &ticker, &name, &sector, &industry, &marketCap, &country,
			&exchange, &currency, &website, &description, &employees,
			&foundedYear, &hqLocation, &logoURL, &isActive,
			&lastUpdated, &createdAt)
		if err != nil {
			continue
		}

		company := map[string]interface{}{
			"id":           id.Int64,
			"ticker":       ticker.String,
			"name":         name.String,
			"sector":       sector.String,
			"industry":     industry.String,
			"market_cap":   marketCap.Int64,
			"country":      country.String,
			"exchange":     exchange.String,
			"currency":     currency.String,
			"website":      website.String,
			"description":  description.String,
			"employees":    employees.Int64,
			"founded_year": foundedYear.Int64,
			"hq_location":  hqLocation.String,
			"logo_url":     logoURL.String,
			"is_active":    isActive.Bool,
			"last_updated": lastUpdated.Time,
			"created_at":   createdAt.Time,
		}
		companies = append(companies, company)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": companies,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// Helper function to parse query integer parameters
func parseQueryInt(c *gin.Context, key string, defaultValue int) int {
	val := c.Query(key)
	if val == "" {
		return defaultValue
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	if intVal < 0 {
		return defaultValue
	}
	// Cap maximum values
	if key == "limit" && intVal > 200 {
		return 200
	}
	return intVal
}
