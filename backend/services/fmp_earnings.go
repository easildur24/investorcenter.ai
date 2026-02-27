package services

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
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

// BeatRate summarizes how often a stock beats estimates.
// TotalQuarters counts quarters with EPS data; TotalRevenueQuarters counts
// quarters with revenue data (may differ for small-caps without coverage).
type BeatRate struct {
	EPSBeats             int `json:"epsBeats"`
	RevenueBeats         int `json:"revenueBeats"`
	TotalQuarters        int `json:"totalQuarters"`
	TotalRevenueQuarters int `json:"totalRevenueQuarters"`
}

// EarningsResponse is the full response returned by the earnings endpoint.
// NextEarnings is non-nil only when a future-dated record exists.
// MostRecentEarnings is always the most recent past record (if any).
type EarningsResponse struct {
	Earnings           []EarningsResult `json:"earnings"`
	NextEarnings       *EarningsResult  `json:"nextEarnings"`
	MostRecentEarnings *EarningsResult  `json:"mostRecentEarnings"`
	BeatRate           *BeatRate        `json:"beatRate"`
}

// ============================================================================
// FMP API Fetch Methods
// ============================================================================

// GetEarnings fetches earnings history for a single ticker from FMP
func (c *FMPClient) GetEarnings(ticker string) ([]FMPEarningsRecord, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("FMP API key not configured")
	}

	params := url.Values{}
	params.Set("symbol", ticker)
	params.Set("apikey", c.APIKey)
	reqURL := fmt.Sprintf("%s/earnings?%s", FMPBaseURL, params.Encode())

	resp, err := c.Client.Get(reqURL)
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

	params := url.Values{}
	params.Set("from", from)
	params.Set("to", to)
	params.Set("apikey", c.APIKey)
	reqURL := fmt.Sprintf("%s/earnings-calendar?%s", FMPBaseURL, params.Encode())

	resp, err := c.Client.Get(reqURL)
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
// Returns nil if either input is nil or if estimated is zero (division
// undefined — callers should handle the zero-estimate case explicitly).
func ComputeSurprisePercent(actual, estimated *float64) *float64 {
	if actual == nil || estimated == nil {
		return nil
	}
	if *estimated == 0 {
		return nil
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
	var mostRecentEarnings *EarningsResult
	epsBeats := 0
	revenueBeats := 0
	qualifiedQuarters := 0
	qualifiedRevenueQuarters := 0

	for _, r := range records {
		// Determine upcoming purely by date; a future date means the report
		// hasn't happened yet regardless of whether EPSActual is nil.
		isUpcoming := r.Date > today

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
			snapshot := result
			nextEarnings = &snapshot
		}

		// Track most recent past record
		if !isUpcoming && mostRecentEarnings == nil {
			snapshot := result
			mostRecentEarnings = &snapshot
		}

		// Count EPS beats (only quarters with both actual and estimate, max 8)
		if !isUpcoming && r.EPSActual != nil && r.EPSEstimated != nil && qualifiedQuarters < 8 {
			qualifiedQuarters++
			if *r.EPSActual > *r.EPSEstimated {
				epsBeats++
			}
		}

		// Count revenue beats with a separate denominator
		if !isUpcoming && r.RevenueActual != nil && r.RevenueEstimated != nil && qualifiedRevenueQuarters < 8 {
			qualifiedRevenueQuarters++
			if *r.RevenueActual > *r.RevenueEstimated {
				revenueBeats++
			}
		}
	}

	var beatRate *BeatRate
	if qualifiedQuarters > 0 || qualifiedRevenueQuarters > 0 {
		beatRate = &BeatRate{
			EPSBeats:             epsBeats,
			RevenueBeats:         revenueBeats,
			TotalQuarters:        qualifiedQuarters,
			TotalRevenueQuarters: qualifiedRevenueQuarters,
		}
	}

	return &EarningsResponse{
		Earnings:           results,
		NextEarnings:       nextEarnings,
		MostRecentEarnings: mostRecentEarnings,
		BeatRate:           beatRate,
	}
}
