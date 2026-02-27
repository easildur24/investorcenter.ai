package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"investorcenter-api/services"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// earningsCacheTTL is the Redis cache TTL for per-stock earnings data.
const earningsCacheTTL = 1 * time.Hour

// calendarCacheTTL is the Redis cache TTL for earnings calendar data.
const calendarCacheTTL = 4 * time.Hour

// earningsCacheVersion is bumped when the response shape changes to avoid
// serving stale cached responses with an outdated schema after deploys.
const earningsCacheVersion = "v1"

// validTickerRe matches 1-10 uppercase alphanumeric characters, dots, and hyphens.
var validTickerRe = regexp.MustCompile(`^[A-Z0-9.\-]{1,10}$`)

// isFMPReady returns true if the FMP client is configured and available.
func isFMPReady() bool {
	return fmpClient != nil && fmpClient.APIKey != ""
}

// GetStockEarnings handles GET /api/v1/stocks/:ticker/earnings
// Returns earnings history with computed surprise %, beat rate, and next earnings.
func GetStockEarnings(c *gin.Context) {
	ticker := strings.ToUpper(c.Param("ticker"))
	if !validTickerRe.MatchString(ticker) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticker symbol"})
		return
	}

	ctx := c.Request.Context()
	cacheKey := fmt.Sprintf("earnings:%s:stock:%s", earningsCacheVersion, ticker)

	// Check Redis cache
	if redisClient != nil {
		cached, err := redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			c.Data(http.StatusOK, "application/json", []byte(cached))
			return
		}
		if err != redis.Nil {
			log.Printf("Redis GET error for %s: %v", cacheKey, err)
		}
	}

	// Fetch from FMP
	if !isFMPReady() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "FMP not configured",
			"message": "Earnings data is not available at this time",
		})
		return
	}

	records, err := fmpClient.GetEarnings(ticker)
	if err != nil {
		log.Printf("FMP earnings fetch error for %s: %v", ticker, err)
		c.JSON(http.StatusBadGateway, gin.H{
			"error":   "Upstream service unavailable",
			"message": "Failed to fetch earnings data",
		})
		return
	}

	// Transform
	transformed := services.TransformEarnings(records)

	response := gin.H{
		"data": transformed,
		"meta": gin.H{
			"ticker":    ticker,
			"timestamp": time.Now().UTC(),
		},
	}

	// Cache in Redis
	if redisClient != nil {
		responseJSON, err := json.Marshal(response)
		if err != nil {
			log.Printf("JSON marshal error for earnings %s: %v", ticker, err)
		} else {
			if err := redisClient.Set(ctx, cacheKey, responseJSON, earningsCacheTTL).Err(); err != nil {
				log.Printf("Redis SET error for %s: %v", cacheKey, err)
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetEarningsCalendar handles GET /api/v1/earnings-calendar?from=YYYY-MM-DD&to=YYYY-MM-DD
// Returns earnings for all tickers in the given date range.
// Default range: current Monday through the following Friday (12 days).
func GetEarningsCalendar(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")

	// Default: current Monday through next week's Friday (12-day window).
	// This gives users a two-week lookahead of upcoming earnings.
	if from == "" || to == "" {
		now := time.Now()
		weekday := now.Weekday()
		daysToMonday := int(weekday - time.Monday)
		if daysToMonday < 0 {
			daysToMonday += 7
		}
		monday := now.AddDate(0, 0, -daysToMonday)
		// Next week's Friday = Monday + 11 days (Mon=0, Tue=1, ..., Fri_next=11)
		nextFriday := monday.AddDate(0, 0, 11)

		from = monday.Format("2006-01-02")
		to = nextFriday.Format("2006-01-02")
	}

	// Validate date format
	fromDate, err := time.Parse("2006-01-02", from)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'from' date format. Use YYYY-MM-DD"})
		return
	}
	toDate, err := time.Parse("2006-01-02", to)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'to' date format. Use YYYY-MM-DD"})
		return
	}

	// Validate max 14-day window (inclusive: 14 days exactly is the limit)
	if toDate.Sub(fromDate).Hours() >= 15*24 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Date range must not exceed 14 days",
		})
		return
	}

	ctx := c.Request.Context()
	cacheKey := fmt.Sprintf("earnings:%s:calendar:%s:%s", earningsCacheVersion, from, to)

	// Check Redis cache
	if redisClient != nil {
		cached, err := redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			c.Data(http.StatusOK, "application/json", []byte(cached))
			return
		}
		if err != redis.Nil {
			log.Printf("Redis GET error for %s: %v", cacheKey, err)
		}
	}

	// Fetch from FMP
	if !isFMPReady() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "FMP not configured",
			"message": "Earnings calendar is not available at this time",
		})
		return
	}

	records, err := fmpClient.GetEarningsCalendar(from, to)
	if err != nil {
		log.Printf("FMP earnings calendar fetch error: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{
			"error":   "Upstream service unavailable",
			"message": "Failed to fetch earnings calendar",
		})
		return
	}

	// Transform each record and build counts map
	today := time.Now().Format("2006-01-02")
	earnings := make([]services.EarningsResult, 0, len(records))
	earningsCounts := make(map[string]int)

	for _, r := range records {
		result := services.EarningsResult{
			Symbol:                 r.Symbol,
			Date:                   r.Date,
			FiscalQuarter:          services.ToFiscalQuarter(r.Date),
			EPSEstimated:           r.EPSEstimated,
			EPSActual:              r.EPSActual,
			EPSSurprisePercent:     services.ComputeSurprisePercent(r.EPSActual, r.EPSEstimated),
			EPSBeat:                services.ComputeBeat(r.EPSActual, r.EPSEstimated),
			RevenueEstimated:       r.RevenueEstimated,
			RevenueActual:          r.RevenueActual,
			RevenueSurprisePercent: services.ComputeSurprisePercent(r.RevenueActual, r.RevenueEstimated),
			RevenueBeat:            services.ComputeBeat(r.RevenueActual, r.RevenueEstimated),
			IsUpcoming:             r.Date > today,
		}
		earnings = append(earnings, result)
		earningsCounts[r.Date]++
	}

	response := gin.H{
		"data": gin.H{
			"earnings":       earnings,
			"earningsCounts": earningsCounts,
		},
		"meta": gin.H{
			"from":      from,
			"to":        to,
			"total":     len(earnings),
			"timestamp": time.Now().UTC(),
		},
	}

	// Cache in Redis
	if redisClient != nil {
		responseJSON, err := json.Marshal(response)
		if err != nil {
			log.Printf("JSON marshal error for earnings calendar %s-%s: %v", from, to, err)
		} else {
			if err := redisClient.Set(ctx, cacheKey, responseJSON, calendarCacheTTL).Err(); err != nil {
				log.Printf("Redis SET error for %s: %v", cacheKey, err)
			}
		}
	}

	c.JSON(http.StatusOK, response)
}
