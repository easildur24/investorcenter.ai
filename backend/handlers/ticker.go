package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"investorcenter-api/models"
)

// GetTickerOverview returns comprehensive ticker data
func GetTickerOverview(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	// Mock data - replace with database queries
	tickerData := models.TickerPageData{
		Summary: models.StockSummary{
			Stock: models.Stock{
				ID:          1,
				Symbol:      symbol,
				Name:        getCompanyName(symbol),
				Exchange:    "NASDAQ",
				Sector:      "Technology",
				Industry:    "Consumer Electronics",
				Country:     "US",
				Currency:    "USD",
				MarketCap:   decimalPtr(2800000000000), // $2.8T
				Description: "Technology company focused on consumer electronics and services.",
				Website:     "https://www.apple.com",
				CreatedAt:   time.Now().Add(-365 * 24 * time.Hour),
				UpdatedAt:   time.Now(),
			},
			Price: models.StockPrice{
				Symbol:        symbol,
				Price:         decimal.NewFromFloat(175.43),
				Open:          decimal.NewFromFloat(174.20),
				High:          decimal.NewFromFloat(177.50),
				Low:           decimal.NewFromFloat(173.80),
				Close:         decimal.NewFromFloat(175.43),
				Volume:        45678901,
				Change:        decimal.NewFromFloat(2.34),
				ChangePercent: decimal.NewFromFloat(1.35),
				Timestamp:     time.Now(),
			},
			Fundamentals: &models.Fundamentals{
				Symbol:       symbol,
				Period:       "Q4",
				Year:         2024,
				PE:           decimalPtr(28.5),
				PB:           decimalPtr(8.2),
				PS:           decimalPtr(7.1),
				ROE:          decimalPtr(0.285),
				ROA:          decimalPtr(0.185),
				Revenue:      decimalPtr(394328000000), // $394.3B
				NetIncome:    decimalPtr(97000000000),  // $97B
				EPS:          decimalPtr(6.15),
				DebtToEquity: decimalPtr(1.73),
				CurrentRatio: decimalPtr(1.05),
				UpdatedAt:    time.Now(),
			},
			TechnicalIndicators: &models.TechnicalIndicators{
				Symbol:         symbol,
				RSI:            decimalPtr(58.3),
				SMA20:          decimalPtr(172.45),
				SMA50:          decimalPtr(168.90),
				SMA200:         decimalPtr(155.20),
				BollingerUpper: decimalPtr(180.50),
				BollingerLower: decimalPtr(165.30),
				Beta:           decimalPtr(1.24),
				Volatility:     decimalPtr(0.28),
				Timestamp:      time.Now(),
			},
			AnalystConsensus: &models.AnalystConsensus{
				Rating:            "Buy",
				RatingScore:       decimal.NewFromFloat(4.2),
				PriceTarget:       decimalPtr(195.50),
				PriceTargetHigh:   decimalPtr(220.00),
				PriceTargetLow:    decimalPtr(165.00),
				PriceTargetMedian: decimalPtr(190.00),
				Upside:            decimalPtr(11.44),
				NumberOfAnalysts:  25,
				StrongBuy:         8,
				Buy:               12,
				Hold:              4,
				Sell:              1,
				StrongSell:        0,
				LastUpdated:       time.Now(),
			},
			KeyMetrics: &models.KeyMetrics{
				Symbol:            symbol,
				Week52High:        decimalPtr(198.23),
				Week52Low:         decimalPtr(124.17),
				Week52Change:      decimalPtr(0.185),
				YTDChange:         decimalPtr(0.124),
				MarketCap:         decimalPtr(2800000000000),
				TrailingPE:        decimalPtr(28.5),
				ForwardPE:         decimalPtr(25.8),
				PriceToBook:       decimalPtr(8.2),
				PriceToSales:      decimalPtr(7.1),
				RevenueGrowth1Y:   decimalPtr(0.028),
				EarningsGrowth1Y:  decimalPtr(0.135),
				DebtToEquity:      decimalPtr(1.73),
				CurrentRatio:      decimalPtr(1.05),
				Beta:              decimalPtr(1.24),
				SharesOutstanding: int64Ptr(15728000000),
				LastUpdated:       time.Now(),
			},
		},
		ChartData:           generateMockChartData(symbol, "1Y"),
		RecentNews:          generateMockNews(symbol),
		EarningsHistory:     generateMockEarnings(symbol),
		DividendHistory:     generateMockDividends(symbol),
		AnalystRatings:      generateMockAnalystRatings(symbol),
		InsiderActivity:     generateMockInsiderActivity(symbol),
		PeerComparisons:     generateMockPeerComparisons(symbol),
		FundamentalsHistory: generateMockFundamentalsHistory(symbol),
	}

	c.JSON(http.StatusOK, gin.H{
		"data": tickerData,
		"meta": gin.H{
			"symbol":    symbol,
			"timestamp": time.Now().UTC(),
		},
	})
}

// GetTickerChart returns chart data for a specific period
func GetTickerChart(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	period := c.DefaultQuery("period", "1Y")
	interval := c.DefaultQuery("interval", "1d")

	chartData := generateMockChartData(symbol, period)

	c.JSON(http.StatusOK, gin.H{
		"data": chartData,
		"meta": gin.H{
			"symbol":    symbol,
			"period":    period,
			"interval":  interval,
			"timestamp": time.Now().UTC(),
		},
	})
}

// GetTickerFundamentals returns fundamental data
func GetTickerFundamentals(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	years := c.DefaultQuery("years", "5")

	yearsInt, _ := strconv.Atoi(years)
	fundamentals := generateMockFundamentalsHistory(symbol)

	// Limit to requested years
	if len(fundamentals) > yearsInt*4 { // 4 quarters per year
		fundamentals = fundamentals[:yearsInt*4]
	}

	c.JSON(http.StatusOK, gin.H{
		"data": fundamentals,
		"meta": gin.H{
			"symbol":    symbol,
			"years":     yearsInt,
			"count":     len(fundamentals),
			"timestamp": time.Now().UTC(),
		},
	})
}

// GetTickerNews returns recent news for a ticker
func GetTickerNews(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))
	limit := c.DefaultQuery("limit", "20")

	limitInt, _ := strconv.Atoi(limit)
	news := generateMockNews(symbol)

	if len(news) > limitInt {
		news = news[:limitInt]
	}

	c.JSON(http.StatusOK, gin.H{
		"data": news,
		"meta": gin.H{
			"symbol":    symbol,
			"count":     len(news),
			"timestamp": time.Now().UTC(),
		},
	})
}

// GetTickerEarnings returns earnings history
func GetTickerEarnings(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	earnings := generateMockEarnings(symbol)

	c.JSON(http.StatusOK, gin.H{
		"data": earnings,
		"meta": gin.H{
			"symbol":    symbol,
			"count":     len(earnings),
			"timestamp": time.Now().UTC(),
		},
	})
}

// GetTickerDividends returns dividend history
func GetTickerDividends(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	dividends := generateMockDividends(symbol)

	c.JSON(http.StatusOK, gin.H{
		"data": dividends,
		"meta": gin.H{
			"symbol":    symbol,
			"count":     len(dividends),
			"timestamp": time.Now().UTC(),
		},
	})
}

// GetTickerAnalysts returns analyst ratings
func GetTickerAnalysts(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	ratings := generateMockAnalystRatings(symbol)

	c.JSON(http.StatusOK, gin.H{
		"data": ratings,
		"meta": gin.H{
			"symbol":    symbol,
			"count":     len(ratings),
			"timestamp": time.Now().UTC(),
		},
	})
}

// GetTickerInsiders returns insider trading activity
func GetTickerInsiders(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	insiders := generateMockInsiderActivity(symbol)

	c.JSON(http.StatusOK, gin.H{
		"data": insiders,
		"meta": gin.H{
			"symbol":    symbol,
			"count":     len(insiders),
			"timestamp": time.Now().UTC(),
		},
	})
}

// GetTickerPeers returns peer comparison data
func GetTickerPeers(c *gin.Context) {
	symbol := strings.ToUpper(c.Param("symbol"))

	peers := generateMockPeerComparisons(symbol)

	c.JSON(http.StatusOK, gin.H{
		"data": peers,
		"meta": gin.H{
			"symbol":    symbol,
			"count":     len(peers),
			"timestamp": time.Now().UTC(),
		},
	})
}

// Helper functions
func decimalPtr(f float64) *decimal.Decimal {
	d := decimal.NewFromFloat(f)
	return &d
}

func int64Ptr(i int64) *int64 {
	return &i
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func getCompanyName(symbol string) string {
	companies := map[string]string{
		"AAPL":  "Apple Inc.",
		"GOOGL": "Alphabet Inc.",
		"MSFT":  "Microsoft Corporation",
		"TSLA":  "Tesla Inc.",
		"AMZN":  "Amazon.com Inc.",
		"NVDA":  "NVIDIA Corporation",
		"META":  "Meta Platforms Inc.",
		"NFLX":  "Netflix Inc.",
	}

	if name, exists := companies[symbol]; exists {
		return name
	}
	return symbol + " Inc."
}
