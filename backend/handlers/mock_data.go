package handlers

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/shopspring/decimal"
	"investorcenter-api/models"
)

// Helper functions for mock data
func decimalPtr(f float64) *decimal.Decimal {
	d := decimal.NewFromFloat(f)
	return &d
}

func int64Ptr(i int64) *int64 {
	return &i
}

// generateMockChartData creates realistic chart data
func generateMockChartData(symbol string, period string) models.ChartData {
	var dataPoints []models.ChartDataPoint
	var days int
	var interval time.Duration

	switch period {
	case "1D":
		days = 1
		interval = 5 * time.Minute
	case "5D":
		days = 5
		interval = 15 * time.Minute
	case "1M":
		days = 30
		interval = 1 * time.Hour
	case "3M":
		days = 90
		interval = 4 * time.Hour
	case "6M":
		days = 180
		interval = 1 * 24 * time.Hour
	case "1Y":
		days = 365
		interval = 1 * 24 * time.Hour
	case "5Y":
		days = 1825
		interval = 7 * 24 * time.Hour
	default:
		days = 365
		interval = 1 * 24 * time.Hour
	}

	basePrice := 175.0
	currentTime := time.Now()

	for i := 0; i < days; i++ {
		timestamp := currentTime.Add(-time.Duration(days-i) * interval)

		// Generate realistic price movement
		change := (rand.Float64() - 0.5) * 0.05 // ±2.5% daily change
		basePrice = basePrice * (1 + change)

		// Ensure price doesn't go negative or too extreme
		if basePrice < 50 {
			basePrice = 50
		}
		if basePrice > 300 {
			basePrice = 300
		}

		open := basePrice
		high := open * (1 + rand.Float64()*0.03)
		low := open * (1 - rand.Float64()*0.03)
		close := low + rand.Float64()*(high-low)
		volume := int64(30000000 + rand.Intn(50000000))

		dataPoints = append(dataPoints, models.ChartDataPoint{
			Timestamp: timestamp,
			Open:      decimal.NewFromFloat(open),
			High:      decimal.NewFromFloat(high),
			Low:       decimal.NewFromFloat(low),
			Close:     decimal.NewFromFloat(close),
			Volume:    volume,
		})
	}

	return models.ChartData{
		Symbol:      symbol,
		Period:      period,
		DataPoints:  dataPoints,
		LastUpdated: time.Now(),
	}
}

// generateMockNews creates mock news articles
func generateMockNews(symbol string) []models.NewsArticle {
	articles := []models.NewsArticle{
		{
			ID:          1,
			Symbol:      symbol,
			Title:       fmt.Sprintf("%s Reports Strong Q4 Earnings, Beats Estimates", symbol),
			Summary:     "Company exceeded analyst expectations with strong revenue growth and improved margins.",
			Author:      "Financial Analyst",
			Source:      "MarketWatch",
			URL:         "https://example.com/news/1",
			Sentiment:   "Positive",
			PublishedAt: time.Now().Add(-2 * time.Hour),
			CreatedAt:   time.Now(),
		},
		{
			ID:          2,
			Symbol:      symbol,
			Title:       fmt.Sprintf("Analysts Upgrade %s on Innovation Pipeline", symbol),
			Summary:     "Multiple analysts raise price targets citing strong product roadmap and market position.",
			Author:      "Tech Reporter",
			Source:      "TechCrunch",
			URL:         "https://example.com/news/2",
			Sentiment:   "Positive",
			PublishedAt: time.Now().Add(-6 * time.Hour),
			CreatedAt:   time.Now(),
		},
		{
			ID:          3,
			Symbol:      symbol,
			Title:       fmt.Sprintf("%s Faces Regulatory Headwinds in EU", symbol),
			Summary:     "New regulations may impact European operations and revenue growth.",
			Author:      "Policy Analyst",
			Source:      "Reuters",
			URL:         "https://example.com/news/3",
			Sentiment:   "Negative",
			PublishedAt: time.Now().Add(-12 * time.Hour),
			CreatedAt:   time.Now(),
		},
	}

	return articles
}

// generateMockEarnings creates mock earnings history
func generateMockEarnings(symbol string) []models.Earnings {
	earnings := []models.Earnings{}

	for year := 2024; year >= 2020; year-- {
		quarters := []string{"Q4", "Q3", "Q2", "Q1"}
		for _, quarter := range quarters {
			if year == 2024 && (quarter == "Q1" || quarter == "Q2") {
				continue // Future quarters
			}

			baseEPS := 1.5 + rand.Float64()*2.0
			estimate := baseEPS * (0.95 + rand.Float64()*0.1)
			surprise := baseEPS - estimate

			earnings = append(earnings, models.Earnings{
				Symbol:             symbol,
				Quarter:            quarter,
				Year:               year,
				ReportDate:         getQuarterEndDate(year, quarter),
				EPSActual:          decimalPtr(baseEPS),
				EPSEstimate:        decimalPtr(estimate),
				EPSSurprise:        decimalPtr(surprise),
				EPSSurprisePercent: decimalPtr((surprise / estimate) * 100),
				RevenueActual:      decimalPtr(80000000000 + rand.Float64()*20000000000),
				RevenueEstimate:    decimalPtr(78000000000 + rand.Float64()*20000000000),
				CreatedAt:          time.Now(),
			})
		}
	}

	return earnings
}

// generateMockAnalystRatings creates mock analyst ratings
func generateMockAnalystRatings(symbol string) []models.AnalystRating {
	firms := []string{"Goldman Sachs", "Morgan Stanley", "JPMorgan", "Bank of America", "Citigroup", "Wells Fargo", "Barclays", "Deutsche Bank"}
	ratings := []string{"Strong Buy", "Buy", "Hold", "Sell"}

	var analystRatings []models.AnalystRating

	for i, firm := range firms {
		rating := ratings[rand.Intn(len(ratings))]
		priceTarget := 160.0 + rand.Float64()*80.0

		analystRatings = append(analystRatings, models.AnalystRating{
			Symbol:      symbol,
			Firm:        firm,
			Analyst:     fmt.Sprintf("Analyst %d", i+1),
			Rating:      rating,
			PriceTarget: decimalPtr(priceTarget),
			RatingDate:  time.Now().AddDate(0, 0, -rand.Intn(30)),
			CreatedAt:   time.Now(),
		})
	}

	return analystRatings
}

// getQuarterEndDate returns the end date for a given quarter
func getQuarterEndDate(year int, quarter string) time.Time {
	switch quarter {
	case "Q1":
		return time.Date(year, 3, 31, 0, 0, 0, 0, time.UTC)
	case "Q2":
		return time.Date(year, 6, 30, 0, 0, 0, 0, time.UTC)
	case "Q3":
		return time.Date(year, 9, 30, 0, 0, 0, 0, time.UTC)
	case "Q4":
		return time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC)
	default:
		return time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC)
	}
}
