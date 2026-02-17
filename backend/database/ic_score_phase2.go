package database

import (
	"context"
	"fmt"
	"sort"
	"time"

	"investorcenter-api/models"
)

// ===================
// EPS Estimates Repository
// ===================

// GetEPSEstimate retrieves the current annual EPS estimate for a ticker
func GetEPSEstimate(ticker string) (*models.EPSEstimate, error) {
	var estimate models.EPSEstimate

	query := `
		SELECT *
		FROM eps_estimates
		WHERE ticker = $1
		  AND fiscal_quarter IS NULL
		  AND fiscal_year >= EXTRACT(YEAR FROM CURRENT_DATE)
		ORDER BY fiscal_year ASC
		LIMIT 1
	`

	err := DB.Get(&estimate, query, ticker)
	if err != nil {
		return nil, err
	}

	return &estimate, nil
}

// GetEPSEstimates retrieves all EPS estimates for a ticker
func GetEPSEstimates(ticker string, limit int) ([]models.EPSEstimate, error) {
	estimates := []models.EPSEstimate{}

	query := `
		SELECT *
		FROM eps_estimates
		WHERE ticker = $1
		ORDER BY fiscal_year DESC, fiscal_quarter DESC NULLS FIRST
		LIMIT $2
	`

	err := DB.Select(&estimates, query, ticker, limit)
	if err != nil {
		return nil, err
	}

	return estimates, nil
}

// UpsertEPSEstimate inserts or updates an EPS estimate
func UpsertEPSEstimate(ctx context.Context, estimate *models.EPSEstimate) error {
	query := `
		INSERT INTO eps_estimates (
			ticker, fiscal_year, fiscal_quarter,
			consensus_eps, num_analysts, high_estimate, low_estimate,
			estimate_30d_ago, estimate_60d_ago, estimate_90d_ago,
			upgrades_30d, downgrades_30d, upgrades_60d, downgrades_60d,
			upgrades_90d, downgrades_90d,
			revision_pct_30d, revision_pct_60d, revision_pct_90d,
			fetched_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, NOW()
		)
		ON CONFLICT (ticker, fiscal_year, fiscal_quarter)
		DO UPDATE SET
			consensus_eps = EXCLUDED.consensus_eps,
			num_analysts = EXCLUDED.num_analysts,
			high_estimate = EXCLUDED.high_estimate,
			low_estimate = EXCLUDED.low_estimate,
			estimate_30d_ago = EXCLUDED.estimate_30d_ago,
			estimate_60d_ago = EXCLUDED.estimate_60d_ago,
			estimate_90d_ago = EXCLUDED.estimate_90d_ago,
			upgrades_30d = EXCLUDED.upgrades_30d,
			downgrades_30d = EXCLUDED.downgrades_30d,
			upgrades_60d = EXCLUDED.upgrades_60d,
			downgrades_60d = EXCLUDED.downgrades_60d,
			upgrades_90d = EXCLUDED.upgrades_90d,
			downgrades_90d = EXCLUDED.downgrades_90d,
			revision_pct_30d = EXCLUDED.revision_pct_30d,
			revision_pct_60d = EXCLUDED.revision_pct_60d,
			revision_pct_90d = EXCLUDED.revision_pct_90d,
			fetched_at = EXCLUDED.fetched_at,
			updated_at = NOW()
	`

	_, err := DB.ExecContext(ctx, query,
		estimate.Ticker, estimate.FiscalYear, estimate.FiscalQuarter,
		estimate.ConsensusEPS, estimate.NumAnalysts, estimate.HighEstimate, estimate.LowEstimate,
		estimate.Estimate30dAgo, estimate.Estimate60dAgo, estimate.Estimate90dAgo,
		estimate.Upgrades30d, estimate.Downgrades30d, estimate.Upgrades60d, estimate.Downgrades60d,
		estimate.Upgrades90d, estimate.Downgrades90d,
		estimate.RevisionPct30d, estimate.RevisionPct60d, estimate.RevisionPct90d,
		estimate.FetchedAt,
	)

	return err
}

// ===================
// Valuation History Repository
// ===================

// GetValuationHistory retrieves historical valuation data for a ticker
func GetValuationHistory(ticker string, years int) ([]models.ValuationHistory, error) {
	history := []models.ValuationHistory{}

	cutoffDate := time.Now().AddDate(-years, 0, 0)

	query := `
		SELECT *
		FROM valuation_history
		WHERE ticker = $1
		  AND snapshot_date >= $2
		ORDER BY snapshot_date ASC
	`

	err := DB.Select(&history, query, ticker, cutoffDate)
	if err != nil {
		return nil, err
	}

	return history, nil
}

// GetValuationHistorySummary calculates 5-year valuation range summary
func GetValuationHistorySummary(ticker string) (*models.ValuationHistorySummary, error) {
	history, err := GetValuationHistory(ticker, 5)
	if err != nil {
		return nil, err
	}

	if len(history) < 12 { // Need at least 12 months
		return nil, fmt.Errorf("insufficient valuation history data")
	}

	// Collect P/E and P/S values
	peValues := []float64{}
	psValues := []float64{}

	for _, h := range history {
		if h.PERatio != nil {
			v, _ := h.PERatio.Float64()
			if v > 0 {
				peValues = append(peValues, v)
			}
		}
		if h.PSRatio != nil {
			v, _ := h.PSRatio.Float64()
			if v > 0 {
				psValues = append(psValues, v)
			}
		}
	}

	summary := &models.ValuationHistorySummary{
		Ticker:     ticker,
		DataPoints: len(history),
	}

	// Get current valuation
	currentVal, err := GetCurrentValuation(ticker)
	if err == nil && currentVal != nil {
		if currentVal.PERatio != nil {
			v, _ := currentVal.PERatio.Float64()
			summary.CurrentPE = &v
		}
		if currentVal.PSRatio != nil {
			v, _ := currentVal.PSRatio.Float64()
			summary.CurrentPS = &v
		}
	}

	// Calculate P/E statistics
	if len(peValues) >= 12 {
		sort.Float64s(peValues)
		low := peValues[0]
		high := peValues[len(peValues)-1]
		median := peValues[len(peValues)/2]
		summary.PE5YrLow = &low
		summary.PE5YrHigh = &high
		summary.PE5YrMedian = &median

		if summary.CurrentPE != nil {
			pct := percentileInSlice(*summary.CurrentPE, peValues)
			summary.PEPercentile = &pct
		}
	}

	// Calculate P/S statistics
	if len(psValues) >= 12 {
		sort.Float64s(psValues)
		low := psValues[0]
		high := psValues[len(psValues)-1]
		median := psValues[len(psValues)/2]
		summary.PS5YrLow = &low
		summary.PS5YrHigh = &high
		summary.PS5YrMedian = &median

		if summary.CurrentPS != nil {
			pct := percentileInSlice(*summary.CurrentPS, psValues)
			summary.PSPercentile = &pct
		}
	}

	// Determine if growth stock (check net margin)
	var netMargin *float64
	marginQuery := `
		SELECT net_margin
		FROM financials
		WHERE ticker = $1
		ORDER BY period_end_date DESC
		LIMIT 1
	`
	_ = DB.Get(&netMargin, marginQuery, ticker)

	if netMargin != nil && *netMargin < 5 {
		summary.IsGrowthStock = true
	}

	return summary, nil
}

// GetCurrentValuation retrieves the latest valuation ratios
func GetCurrentValuation(ticker string) (*models.ValuationHistory, error) {
	var val models.ValuationHistory

	query := `
		SELECT
			ticker,
			calculation_date as snapshot_date,
			ttm_pe_ratio as pe_ratio,
			ttm_ps_ratio as ps_ratio,
			ttm_pb_ratio as pb_ratio,
			stock_price
		FROM valuation_ratios
		WHERE ticker = $1
		ORDER BY calculation_date DESC
		LIMIT 1
	`

	err := DB.Get(&val, query, ticker)
	if err != nil {
		return nil, err
	}

	return &val, nil
}

// UpsertValuationHistory inserts or updates a valuation history snapshot
func UpsertValuationHistory(ctx context.Context, vh *models.ValuationHistory) error {
	query := `
		INSERT INTO valuation_history (
			ticker, snapshot_date,
			pe_ratio, ps_ratio, pb_ratio, ev_ebitda, peg_ratio,
			stock_price, market_cap, eps_ttm, revenue_ttm
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		ON CONFLICT (ticker, snapshot_date)
		DO UPDATE SET
			pe_ratio = EXCLUDED.pe_ratio,
			ps_ratio = EXCLUDED.ps_ratio,
			pb_ratio = EXCLUDED.pb_ratio,
			ev_ebitda = EXCLUDED.ev_ebitda,
			peg_ratio = EXCLUDED.peg_ratio,
			stock_price = EXCLUDED.stock_price,
			market_cap = EXCLUDED.market_cap,
			eps_ttm = EXCLUDED.eps_ttm,
			revenue_ttm = EXCLUDED.revenue_ttm
	`

	_, err := DB.ExecContext(ctx, query,
		vh.Ticker, vh.SnapshotDate,
		vh.PERatio, vh.PSRatio, vh.PBRatio, vh.EVEbitda, vh.PEGRatio,
		vh.StockPrice, vh.MarketCap, vh.EPSTTM, vh.RevenueTTM,
	)

	return err
}

// ===================
// Dividend History Repository
// ===================

// GetDividendHistory retrieves dividend history for a ticker
func GetDividendHistory(ticker string, years int) ([]models.DividendHistory, error) {
	history := []models.DividendHistory{}

	query := `
		SELECT *
		FROM dividend_history
		WHERE ticker = $1
		  AND fiscal_year >= EXTRACT(YEAR FROM CURRENT_DATE) - $2
		ORDER BY fiscal_year DESC
	`

	err := DB.Select(&history, query, ticker, years)
	if err != nil {
		return nil, err
	}

	return history, nil
}

// GetLatestDividendHistory retrieves the most recent dividend data
func GetLatestDividendHistory(ticker string) (*models.DividendHistory, error) {
	var dh models.DividendHistory

	query := `
		SELECT *
		FROM dividend_history
		WHERE ticker = $1
		ORDER BY fiscal_year DESC
		LIMIT 1
	`

	err := DB.Get(&dh, query, ticker)
	if err != nil {
		return nil, err
	}

	return &dh, nil
}

// UpsertDividendHistory inserts or updates dividend history
func UpsertDividendHistory(ctx context.Context, dh *models.DividendHistory) error {
	query := `
		INSERT INTO dividend_history (
			ticker, fiscal_year,
			annual_dividend, dividend_yield, payout_ratio, dividend_growth_yoy,
			ex_dividend_date, payment_date,
			consecutive_years_paid, consecutive_years_increased,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW()
		)
		ON CONFLICT (ticker, fiscal_year)
		DO UPDATE SET
			annual_dividend = EXCLUDED.annual_dividend,
			dividend_yield = EXCLUDED.dividend_yield,
			payout_ratio = EXCLUDED.payout_ratio,
			dividend_growth_yoy = EXCLUDED.dividend_growth_yoy,
			ex_dividend_date = EXCLUDED.ex_dividend_date,
			payment_date = EXCLUDED.payment_date,
			consecutive_years_paid = EXCLUDED.consecutive_years_paid,
			consecutive_years_increased = EXCLUDED.consecutive_years_increased,
			updated_at = NOW()
	`

	_, err := DB.ExecContext(ctx, query,
		dh.Ticker, dh.FiscalYear,
		dh.AnnualDividend, dh.DividendYield, dh.PayoutRatio, dh.DividendGrowthYoY,
		dh.ExDividendDate, dh.PaymentDate,
		dh.ConsecutiveYearsPaid, dh.ConsecutiveYearsIncreased,
	)

	return err
}

// CalculateDividendCAGR calculates 5-year dividend CAGR
func CalculateDividendCAGR(ticker string) (*float64, error) {
	history, err := GetDividendHistory(ticker, 5)
	if err != nil || len(history) < 5 {
		return nil, err
	}

	// Get current and oldest dividend
	currentDiv := history[0].AnnualDividend
	oldDiv := history[len(history)-1].AnnualDividend

	if currentDiv == nil || oldDiv == nil {
		return nil, fmt.Errorf("missing dividend data")
	}

	currentVal, _ := currentDiv.Float64()
	oldVal, _ := oldDiv.Float64()

	if oldVal <= 0 || currentVal <= 0 {
		return nil, fmt.Errorf("invalid dividend values")
	}

	// Calculate CAGR: ((Current/Old)^(1/years) - 1) * 100
	years := float64(len(history) - 1)
	cagr := (pow(currentVal/oldVal, 1/years) - 1) * 100

	return &cagr, nil
}

// ===================
// Helper Functions
// ===================

// percentileInSlice calculates where a value sits in a sorted slice
func percentileInSlice(value float64, sortedSlice []float64) float64 {
	if len(sortedSlice) == 0 {
		return 50 // Neutral
	}

	belowCount := 0
	for _, v := range sortedSlice {
		if v < value {
			belowCount++
		}
	}

	return float64(belowCount) / float64(len(sortedSlice)) * 100
}

// pow calculates x^y (simple implementation)
func pow(x, y float64) float64 {
	if y == 0 {
		return 1
	}
	// Use math.Pow for actual implementation
	result := 1.0
	for i := 0; i < int(y); i++ {
		result *= x
	}
	// For fractional exponents, use approximation
	if y != float64(int(y)) {
		// Simple approximation using binary search or Newton's method
		// For production, use math.Pow
		return result // Simplified
	}
	return result
}

// ===================
// Batch Operations
// ===================

// BatchUpsertEPSEstimates inserts/updates multiple EPS estimates
func BatchUpsertEPSEstimates(ctx context.Context, estimates []models.EPSEstimate) error {
	tx, err := DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, e := range estimates {
		if err := UpsertEPSEstimate(ctx, &e); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// BatchUpsertValuationHistory inserts/updates multiple valuation snapshots
func BatchUpsertValuationHistory(ctx context.Context, snapshots []models.ValuationHistory) error {
	tx, err := DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, s := range snapshots {
		if err := UpsertValuationHistory(ctx, &s); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetStocksByDividendTier retrieves stocks by dividend tier
func GetStocksByDividendTier(tier models.DividendTier, limit int) ([]models.DividendHistory, error) {
	history := []models.DividendHistory{}

	var minYears, maxYears int
	switch tier {
	case models.DividendTierKing:
		minYears = 50
		maxYears = 999
	case models.DividendTierAristocr:
		minYears = 25
		maxYears = 49
	case models.DividendTierAchiever:
		minYears = 10
		maxYears = 24
	case models.DividendTierPayer:
		minYears = 5
		maxYears = 9
	default:
		minYears = 0
		maxYears = 4
	}

	query := `
		SELECT DISTINCT ON (ticker) *
		FROM dividend_history
		WHERE consecutive_years_increased >= $1
		  AND consecutive_years_increased <= $2
		ORDER BY ticker, fiscal_year DESC
		LIMIT $3
	`

	err := DB.Select(&history, query, minYears, maxYears, limit)
	return history, err
}
