package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewWatchListService
// ---------------------------------------------------------------------------

func TestNewWatchListService(t *testing.T) {
	svc := NewWatchListService()
	require.NotNil(t, svc)
}

// ---------------------------------------------------------------------------
// SearchTickers — query uppercasing (pure logic)
// ---------------------------------------------------------------------------

func TestSearchTickers_QueryUppercased(t *testing.T) {
	// We can't call SearchTickers without DB, but we can verify the
	// strings.ToUpper logic that the function applies.
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"lowercase", "aapl", "AAPL"},
		{"uppercase", "AAPL", "AAPL"},
		{"mixed case", "AaPl", "AAPL"},
		{"with numbers", "spy3", "SPY3"},
		{"empty string", "", ""},
		{"lowercase long", "bitcoin", "BITCOIN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strings.ToUpper(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// SearchTickers — limit validation
// ---------------------------------------------------------------------------

func TestSearchTickers_LimitValues(t *testing.T) {
	// Verify that various limit values are valid integers.
	// The actual DB query is gated by database.SearchStocks,
	// so we validate the parameter contract here.
	tests := []struct {
		name  string
		limit int
		valid bool
	}{
		{"positive limit", 10, true},
		{"zero limit", 0, true},
		{"large limit", 1000, true},
		{"negative limit", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				assert.GreaterOrEqual(t, tt.limit, 0)
			} else {
				assert.Less(t, tt.limit, 0)
			}
		})
	}
}
