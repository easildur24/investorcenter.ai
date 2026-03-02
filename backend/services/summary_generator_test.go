package services

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateTemplate_WithFullData(t *testing.T) {
	sg := &SummaryGenerator{}

	data := &summaryPromptData{
		Date: "2026-03-01",
		Indices: []marketDataPoint{
			{Name: "S&P 500", Price: 5000.00, Change: 75.00, ChangePercent: 1.50},
			{Name: "Dow Jones", Price: 40000.00, Change: -200.00, ChangePercent: -0.50},
		},
		Gainers: []marketDataPoint{
			{Name: "AAPL", Price: 200.00, Change: 10.00, ChangePercent: 5.0},
			{Name: "MSFT", Price: 400.00, Change: 12.00, ChangePercent: 3.0},
		},
		Losers: []marketDataPoint{
			{Name: "TSLA", Price: 180.00, Change: -6.00, ChangePercent: -3.0},
			{Name: "NVDA", Price: 800.00, Change: -16.00, ChangePercent: -2.0},
		},
		Sentiment: "mixed",
	}

	result := sg.generateTemplate(data)

	assert.Equal(t, "template", result.Method)
	assert.NotEmpty(t, result.Summary)
	assert.NotEmpty(t, result.Timestamp)
	assert.NotEmpty(t, result.Title)
	assert.Equal(t, "Mixed session as indices close with varied results", result.Title)

	assert.Contains(t, result.Summary, "S&P 500")
	assert.Contains(t, result.Summary, "Dow Jones")
	assert.Contains(t, result.Summary, "AAPL")
	assert.Contains(t, result.Summary, "TSLA")
	assert.Contains(t, result.Summary, "Top gainers")
	assert.Contains(t, result.Summary, "Notable decliners")
}

func TestGenerateTemplate_EmptyData(t *testing.T) {
	sg := &SummaryGenerator{}

	data := &summaryPromptData{}

	result := sg.generateTemplate(data)

	assert.Equal(t, "template", result.Method)
	assert.Equal(t, "Markets closed for the weekend", result.Title)
	assert.Contains(t, result.Summary, "markets are closed")
}

func TestGenerateTemplate_IndicesOnly(t *testing.T) {
	sg := &SummaryGenerator{}

	data := &summaryPromptData{
		Date: "2026-03-01",
		Indices: []marketDataPoint{
			{Name: "S&P 500", Price: 5000.00, Change: 75.00, ChangePercent: 1.50},
			{Name: "Nasdaq", Price: 16000.00, Change: 120.00, ChangePercent: 0.75},
		},
		Gainers:   []marketDataPoint{},
		Losers:    []marketDataPoint{},
		Sentiment: "bullish",
	}

	result := sg.generateTemplate(data)

	assert.Equal(t, "template", result.Method)
	assert.Equal(t, "Markets rally as major indices close higher", result.Title)
	assert.Contains(t, result.Summary, "S&P 500")
	assert.Contains(t, result.Summary, "Nasdaq")
	assert.NotContains(t, result.Summary, "Top gainers")
	assert.NotContains(t, result.Summary, "Notable decliners")
}

func TestGenerateTemplate_VIXExcluded(t *testing.T) {
	sg := &SummaryGenerator{}

	data := &summaryPromptData{
		Date: "2026-03-01",
		Indices: []marketDataPoint{
			{Name: "S&P 500", Price: 5000.00, Change: 75.00, ChangePercent: 1.50},
			{Name: "VIX", Price: 18.50, Change: -0.50, ChangePercent: -2.63},
		},
		Sentiment: "bullish",
	}

	result := sg.generateTemplate(data)

	assert.Equal(t, "template", result.Method)
	assert.Contains(t, result.Summary, "S&P 500")
	assert.NotContains(t, result.Summary, "VIX")
}

func TestCacheSummary_RoundTrip(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	sg := &SummaryGenerator{redisClient: rdb}
	ctx := context.Background()

	original := &MarketSummaryResult{
		Title:     "Markets rally on strong earnings",
		Summary:   "Markets rallied today with the S&P 500 up 1.5%.",
		Timestamp: "2026-03-01T16:00:00Z",
		Method:    "template",
	}

	sg.cacheSummary(ctx, original)

	cached, err := sg.GetCachedSummary(ctx)
	require.NoError(t, err)
	require.NotNil(t, cached)

	assert.Equal(t, original.Title, cached.Title)
	assert.Equal(t, original.Summary, cached.Summary)
	assert.Equal(t, original.Timestamp, cached.Timestamp)
	assert.Equal(t, original.Method, cached.Method)
}

func TestGetCachedSummary_Miss(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	sg := &SummaryGenerator{redisClient: rdb}
	ctx := context.Background()

	cached, err := sg.GetCachedSummary(ctx)
	assert.NoError(t, err)
	assert.Nil(t, cached)
}
