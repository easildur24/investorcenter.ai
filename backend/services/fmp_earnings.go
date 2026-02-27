package services

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"
)

// ============================================================================
// FMP Earnings API Response Structs
// ============================================================================

// FMPEarningsRecord represents a single record from the FMP earnings endpoint
type FMPEarningsRecord struct {
	Symbol           string   `json:"symbol"`
	Date             string   `json:"date"`
	EPSActual        *float64 `json:"epsActual"`
	EPSEstimated     *float64 `json:"epsEstimated"`
	RevenueActual    *float64 `json:"revenueActual"`
	RevenueEstimated *float64 `json:"revenueEstimated"`
	LastUpdated      string   `json:"lastUpdated"`
}

// ============================================================================
// Enriched Response Structs (returned to frontend)
// ============================================================================

// EarningsResult is an enriched earnings record with computed fields
type EarningsResult struct {
	Symbol                 string   `json:"symbol"`
	Date                   string   `json:"date"`
	FiscalQuarter          string   `json:"fiscalQuarter"`
	EPSEstimated           *float64 `json:"epsEstimated"`
	EPSActual              *float64 `json:"epsActual"`
	EPSSurprisePercent     *float64 `json:"epsSurprisePercent"`
	EPSBeat                *bool    `json:"epsBeat"`
	RevenueEstimated       *float64 `json:"revenueEstimated"`
	RevenueActual          *float64 `json:"revenueActual"`
	RevenueSurprisePercent *float64 `json:"revenueSurprisePercent"`
	RevenueBeat            *bool    `json:"revenueBeat"`
	IsUpcoming             bool     `json:"isUpcoming"`
}

// BeatRate summarizes how often a stock beats estimates
type BeatRate struct {
	EPSBeats      int `json:"epsBeats"`
	RevenueBeats  int `json:"revenueBeats"`
	TotalQuarters int `json:"totalQuarters"`
}

// EarningsResponse is the full response returned by the earnings endpoint
type EarningsResponse struct {
	Earnings     []EarningsResult `json:"earnings"`
	NextEarnings *EarningsResult  `json:"nextEarnings"`
	BeatRate     *BeatRate        `json:"beatRate"`
}

// ============================================================================
// FMP API Fetch Methods
// ============================================================================

// GetEarnings fetches earnings history for a single ticker from FMP
func (c *FMPClient) GetEarnings(ticker string) ([]FMPEarningsRecord, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("FMP API key not configured")
	}

	url := fmt.Sprintf("%s/earnings?symbol=%s&apikey=%s", FMPBaseURL, ticker, c.APIKey)

	resp, err := c.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("FMP earnings request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FMP earnings returned status %d", resp.StatusCode)
	}

	var results []FMPEarningsRecord
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode FMP earnings response: %w", err)
	}

	return results, nil
}

// GetEarningsCalendar fetches earnings calendar for a date range from FMP
func (c *FMPClient) GetEarningsCalendar(from, to string) ([]FMPEarningsRecord, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("FMP API key not configured")
	}

	url := fmt.Sprintf("%s/earnings-calendar?from=%s&to=%s&apikey=%s",
		FMPBaseURL, from, to, c.APIKey)

	resp, err := c.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("FMP earnings-calendar request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FMP earnings-calendar returned status %d", resp.StatusCode)
	}

	var results []FMPEarningsRecord
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode FMP earnings-calendar response: %w", err)
	}

	return results, nil
}

// ============================================================================
// Computation Functions
// ============================================================================

// ComputeSurprisePercent calculates (actual - estimated) / |estimated| * 100.
// Returns nil if either input is nil. Returns absolute difference if estimated is zero.
func ComputeSurprisePercent(actual, estimated *float64) *float64 {
	if actual == nil || estimated == nil {
		return nil
	}
	if *estimated == 0 {
		diff := *actual - *estimated
		return &diff
	}
	surprise := (*actual - *estimated) / math.Abs(*estimated) * 100
	rounded := math.Round(surprise*100) / 100 // round to 2 decimal places
	return &rounded
}

// ComputeBeat returns true if actual > estimated, nil if either is nil.
func ComputeBeat(actual, estimated *float64) *bool {
	if actual == nil || estimated == nil {
		return nil
	}
	beat := *actual > *estimated
	return &beat
}

// ToFiscalQuarter converts a date string "YYYY-MM-DD" to a label like "Q1 '26".
// Returns the original string if parsing fails.
func ToFiscalQuarter(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	month := t.Month()
	year := t.Year() % 100
	var q int
	switch {
	case month >= 1 && month <= 3:
		q = 1
	case month >= 4 && month <= 6:
		q = 2
	case month >= 7 && month <= 9:
		q = 3
	default:
		q = 4
	}
	return fmt.Sprintf("Q%d '%02d", q, year)
}

// ============================================================================
// Transformation
// ============================================================================

// TransformEarnings converts raw FMP records into an enriched EarningsResponse.
// It computes surprise percentages, beat flags, identifies the next earnings,
// and calculates the beat rate over qualifying quarters.
func TransformEarnings(records []FMPEarningsRecord) *EarningsResponse {
	if len(records) == 0 {
		return &EarningsResponse{
			Earnings: []EarningsResult{},
		}
	}

	today := time.Now().Format("2006-01-02")
	results := make([]EarningsResult, 0, len(records))

	var nextEarnings *EarningsResult
	epsBeats := 0
	revenueBeats := 0
	qualifiedQuarters := 0

	for _, r := range records {
		isUpcoming := r.EPSActual == nil && r.Date > today

		result := EarningsResult{
			Symbol:                 r.Symbol,
			Date:                   r.Date,
			FiscalQuarter:          ToFiscalQuarter(r.Date),
			EPSEstimated:           r.EPSEstimated,
			EPSActual:              r.EPSActual,
			EPSSurprisePercent:     ComputeSurprisePercent(r.EPSActual, r.EPSEstimated),
			EPSBeat:                ComputeBeat(r.EPSActual, r.EPSEstimated),
			RevenueEstimated:       r.RevenueEstimated,
			RevenueActual:          r.RevenueActual,
			RevenueSurprisePercent: ComputeSurprisePercent(r.RevenueActual, r.RevenueEstimated),
			RevenueBeat:            ComputeBeat(r.RevenueActual, r.RevenueEstimated),
			IsUpcoming:             isUpcoming,
		}

		results = append(results, result)

		// Track next earnings (first upcoming record)
		if isUpcoming && nextEarnings == nil {
			copy := result
			nextEarnings = &copy
		}

		// Count beats for beat rate (only quarters with both actual and estimate)
		if !isUpcoming && r.EPSActual != nil && r.EPSEstimated != nil && qualifiedQuarters < 8 {
			qualifiedQuarters++
			if *r.EPSActual > *r.EPSEstimated {
				epsBeats++
			}
			if r.RevenueActual != nil && r.RevenueEstimated != nil && *r.RevenueActual > *r.RevenueEstimated {
				revenueBeats++
			}
		}
	}

	// If no upcoming earnings found, use most recent past as nextEarnings
	if nextEarnings == nil && len(results) > 0 {
		for i := range results {
			if !results[i].IsUpcoming {
				copy := results[i]
				nextEarnings = &copy
				break
			}
		}
	}

	var beatRate *BeatRate
	if qualifiedQuarters > 0 {
		beatRate = &BeatRate{
			EPSBeats:      epsBeats,
			RevenueBeats:  revenueBeats,
			TotalQuarters: qualifiedQuarters,
		}
	}

	return &EarningsResponse{
		Earnings:     results,
		NextEarnings: nextEarnings,
		BeatRate:     beatRate,
	}
}
