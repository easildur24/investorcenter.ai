package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
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
		SELECT id, ticker, title, summary, source, url, sentiment,
		       author, published_at, created_at
		FROM news_articles
	`
	countQuery := "SELECT COUNT(*) FROM news_articles"
	args := []interface{}{}

	if search != "" {
		query += " WHERE ticker ILIKE $1 OR title ILIKE $1"
		countQuery += " WHERE ticker ILIKE $1 OR title ILIKE $1"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch news"})
		return
	}
	defer rows.Close()

	var articles []map[string]interface{}
	for rows.Next() {
		var id int
		var ticker, title, summary, source, url, sentiment, author sql.NullString
		var publishedAt, createdAt sql.NullTime

		err := rows.Scan(&id, &ticker, &title, &summary, &source, &url, &sentiment,
			&author, &publishedAt, &createdAt)
		if err != nil {
			continue
		}

		article := map[string]interface{}{
			"id":           id,
			"ticker":       ticker.String,
			"title":        title.String,
			"summary":      summary.String,
			"source":       source.String,
			"url":          url.String,
			"sentiment":    sentiment.String,
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
		SELECT ticker, period, year, pe_ratio, pb_ratio, ps_ratio,
		       revenue, eps, market_cap, created_at, updated_at
		FROM fundamentals
	`
	countQuery := "SELECT COUNT(*) FROM fundamentals"
	args := []interface{}{}

	if search != "" {
		query += " WHERE ticker ILIKE $1"
		countQuery += " WHERE ticker ILIKE $1"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY ticker, year DESC, period DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch fundamentals"})
		return
	}
	defer rows.Close()

	var fundamentals []map[string]interface{}
	for rows.Next() {
		var ticker, period sql.NullString
		var year sql.NullInt64
		var peRatio, pbRatio, psRatio, revenue, eps, marketCap sql.NullFloat64
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(&ticker, &period, &year, &peRatio, &pbRatio, &psRatio,
			&revenue, &eps, &marketCap, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		fundamental := map[string]interface{}{
			"ticker":     ticker.String,
			"period":     period.String,
			"year":       year.Int64,
			"pe_ratio":   peRatio.Float64,
			"pb_ratio":   pbRatio.Float64,
			"ps_ratio":   psRatio.Float64,
			"revenue":    revenue.Float64,
			"eps":        eps.Float64,
			"market_cap": marketCap.Float64,
			"created_at": createdAt.Time,
			"updated_at": updatedAt.Time,
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

// GetDatabaseStats returns database statistics
func (h *AdminDataHandler) GetDatabaseStats(c *gin.Context) {
	stats := make(map[string]interface{})

	// Count all tables
	tables := []string{
		"tickers", "stock_prices", "fundamentals", "ic_scores",
		"news_articles", "insider_trading", "analyst_ratings",
		"technical_indicators", "users", "watch_lists", "alert_rules",
		"user_subscriptions", "reddit_heatmap_daily",
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
