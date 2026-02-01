package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"investorcenter/backend/models"
)

// GetSectorPercentile retrieves the latest percentile statistics for a sector/metric combination
func GetSectorPercentile(sector, metricName string) (*models.SectorPercentile, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT
			id, sector, metric_name, calculated_at,
			min_value, p10_value, p25_value, p50_value,
			p75_value, p90_value, max_value,
			mean_value, std_dev, sample_count,
			created_at
		FROM mv_latest_sector_percentiles
		WHERE sector = $1 AND metric_name = $2
	`

	var sp models.SectorPercentile
	err := DB.Get(&sp, query, sector, metricName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get sector percentile: %w", err)
	}

	return &sp, nil
}

// GetSectorPercentiles retrieves all percentile statistics for a sector
func GetSectorPercentiles(sector string) ([]models.SectorPercentile, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT
			id, sector, metric_name, calculated_at,
			min_value, p10_value, p25_value, p50_value,
			p75_value, p90_value, max_value,
			mean_value, std_dev, sample_count,
			created_at
		FROM mv_latest_sector_percentiles
		WHERE sector = $1
		ORDER BY metric_name
	`

	var percentiles []models.SectorPercentile
	err := DB.Select(&percentiles, query, sector)
	if err != nil {
		return nil, fmt.Errorf("failed to get sector percentiles: %w", err)
	}

	return percentiles, nil
}

// GetAllSectors retrieves list of all sectors with percentile data
func GetAllSectors() ([]string, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT DISTINCT sector
		FROM mv_latest_sector_percentiles
		ORDER BY sector
	`

	var sectors []string
	err := DB.Select(&sectors, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get sectors: %w", err)
	}

	return sectors, nil
}

// CalculatePercentile calculates the percentile score for a value within sector distribution
// Returns a score 0-100 where higher is better (inverted for lower-is-better metrics)
func CalculatePercentile(sector, metricName string, value float64) (*float64, error) {
	sp, err := GetSectorPercentile(sector, metricName)
	if err != nil {
		return nil, err
	}
	if sp == nil {
		return nil, nil
	}

	// Get the distribution values
	var minVal, p10, p25, p50, p75, p90, maxVal float64

	if sp.MinValue != nil {
		minVal, _ = sp.MinValue.Float64()
	}
	if sp.P10Value != nil {
		p10, _ = sp.P10Value.Float64()
	}
	if sp.P25Value != nil {
		p25, _ = sp.P25Value.Float64()
	}
	if sp.P50Value != nil {
		p50, _ = sp.P50Value.Float64()
	}
	if sp.P75Value != nil {
		p75, _ = sp.P75Value.Float64()
	}
	if sp.P90Value != nil {
		p90, _ = sp.P90Value.Float64()
	}
	if sp.MaxValue != nil {
		maxVal, _ = sp.MaxValue.Float64()
	}

	// Calculate raw percentile using piecewise linear interpolation
	var rawPct float64

	switch {
	case value <= minVal:
		rawPct = 0
	case value >= maxVal:
		rawPct = 100
	case value <= p10:
		rawPct = interpolate(value, minVal, p10, 0, 10)
	case value <= p25:
		rawPct = interpolate(value, p10, p25, 10, 25)
	case value <= p50:
		rawPct = interpolate(value, p25, p50, 25, 50)
	case value <= p75:
		rawPct = interpolate(value, p50, p75, 50, 75)
	case value <= p90:
		rawPct = interpolate(value, p75, p90, 75, 90)
	default:
		rawPct = interpolate(value, p90, maxVal, 90, 100)
	}

	// Invert for "lower is better" metrics
	if models.LowerIsBetterMetrics[metricName] {
		rawPct = 100 - rawPct
	}

	return &rawPct, nil
}

// interpolate performs linear interpolation between two points
func interpolate(value, lowVal, highVal, lowPct, highPct float64) float64 {
	if highVal == lowVal {
		return lowPct
	}
	ratio := (value - lowVal) / (highVal - lowVal)
	return lowPct + ratio*(highPct-lowPct)
}

// GetLifecycleClassification retrieves the latest lifecycle classification for a ticker
func GetLifecycleClassification(ticker string) (*models.LifecycleClassification, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT
			id, ticker, classified_at, lifecycle_stage,
			revenue_growth_yoy, net_margin, pe_ratio, market_cap,
			weights_applied, created_at
		FROM lifecycle_classifications
		WHERE ticker = $1
		ORDER BY classified_at DESC
		LIMIT 1
	`

	var lc models.LifecycleClassification
	err := DB.Get(&lc, query, ticker)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get lifecycle classification: %w", err)
	}

	return &lc, nil
}

// GetSectorRank retrieves a stock's rank within its sector by IC Score
func GetSectorRank(sector, ticker string, overallScore float64) (rank int, total int, err error) {
	if DB == nil {
		return 0, 0, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT
			COUNT(*) FILTER (WHERE overall_score > $1) + 1 as rank,
			COUNT(*) as total
		FROM ic_scores ics
		JOIN tickers t ON ics.ticker = t.symbol
		WHERE t.sector = $2
		AND ics.date = (SELECT MAX(date) FROM ic_scores WHERE ticker = ics.ticker)
	`

	var result struct {
		Rank  int `db:"rank"`
		Total int `db:"total"`
	}

	err = DB.Get(&result, query, overallScore, sector)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get sector rank: %w", err)
	}

	return result.Rank, result.Total, nil
}

// RefreshSectorPercentilesMaterializedView refreshes the materialized view
func RefreshSectorPercentilesMaterializedView(ctx context.Context) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	_, err := DB.ExecContext(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY mv_latest_sector_percentiles")
	if err != nil {
		return fmt.Errorf("failed to refresh materialized view: %w", err)
	}

	return nil
}

// GetSectorPercentileStats retrieves summary statistics about sector percentile data
func GetSectorPercentileStats() (*SectorPercentileStats, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT
			COUNT(DISTINCT sector) as sector_count,
			COUNT(DISTINCT metric_name) as metric_count,
			COUNT(*) as total_records,
			MAX(calculated_at) as last_calculated
		FROM mv_latest_sector_percentiles
	`

	var stats SectorPercentileStats
	err := DB.Get(&stats, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &stats, nil
}

// SectorPercentileStats contains summary statistics
type SectorPercentileStats struct {
	SectorCount    int        `json:"sector_count" db:"sector_count"`
	MetricCount    int        `json:"metric_count" db:"metric_count"`
	TotalRecords   int        `json:"total_records" db:"total_records"`
	LastCalculated *time.Time `json:"last_calculated" db:"last_calculated"`
}
