package database

import (
	"database/sql"
	"fmt"
	"time"
)

// RedditHeatmapData represents aggregated daily Reddit metrics
type RedditHeatmapData struct {
	TickerSymbol    string    `json:"tickerSymbol"`
	Date            time.Time `json:"date"`
	AvgRank         float64   `json:"avgRank"`
	MinRank         int       `json:"minRank"`
	MaxRank         int       `json:"maxRank"`
	TotalMentions   int       `json:"totalMentions"`
	TotalUpvotes    int       `json:"totalUpvotes"`
	RankVolatility  float64   `json:"rankVolatility"`
	TrendDirection  string    `json:"trendDirection"`
	PopularityScore float64   `json:"popularityScore"`
	DataSource      string    `json:"dataSource"`
}

// RedditTickerHistory represents a ticker's Reddit history over time
type RedditTickerHistory struct {
	TickerSymbol string               `json:"tickerSymbol"`
	History      []RedditHeatmapData  `json:"history"`
	Summary      RedditHistorySummary `json:"summary"`
}

// RedditHistorySummary provides aggregate statistics
type RedditHistorySummary struct {
	DaysAppeared  int       `json:"daysAppeared"`
	AvgPopularity float64   `json:"avgPopularity"`
	AvgRank       float64   `json:"avgRank"`
	BestRank      int       `json:"bestRank"`
	WorstRank     int       `json:"worstRank"`
	TotalMentions int       `json:"totalMentions"`
	TotalUpvotes  int       `json:"totalUpvotes"`
	PeriodStart   time.Time `json:"periodStart"`
	PeriodEnd     time.Time `json:"periodEnd"`
}

// GetRedditHeatmap retrieves top trending tickers for the specified number of days
// Returns aggregated data sorted by average popularity score
func GetRedditHeatmap(days int, limit int) ([]RedditHeatmapData, error) {
	if days <= 0 {
		days = 7 // Default to 7 days
	}
	if limit <= 0 {
		limit = 50 // Default to top 50
	}

	query := `
		SELECT
			ticker_symbol,
			AVG(popularity_score) as avg_popularity,
			AVG(avg_rank) as avg_rank,
			MIN(min_rank) as best_rank,
			MAX(max_rank) as worst_rank,
			SUM(total_mentions) as total_mentions,
			SUM(total_upvotes) as total_upvotes,
			AVG(rank_volatility) as avg_volatility,
			COUNT(*) as days_appeared,
			MIN(date) as period_start,
			MAX(date) as period_end,
			data_source
		FROM reddit_heatmap_daily
		WHERE date >= CURRENT_DATE - $1::INTEGER * INTERVAL '1 day'
		GROUP BY ticker_symbol, data_source
		HAVING COUNT(*) >= LEAST($1::INTEGER / 2, 3)
		ORDER BY avg_popularity DESC
		LIMIT $2
	`

	rows, err := DB.Query(query, days, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query reddit heatmap: %w", err)
	}
	defer rows.Close()

	var results []RedditHeatmapData
	for rows.Next() {
		var data RedditHeatmapData
		var periodStart, periodEnd time.Time
		var daysAppeared int
		var rankVolatility sql.NullFloat64

		err := rows.Scan(
			&data.TickerSymbol,
			&data.PopularityScore,
			&data.AvgRank,
			&data.MinRank,
			&data.MaxRank,
			&data.TotalMentions,
			&data.TotalUpvotes,
			&rankVolatility,
			&daysAppeared,
			&periodStart,
			&periodEnd,
			&data.DataSource,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reddit heatmap row: %w", err)
		}

		// Handle NULL rank_volatility
		if rankVolatility.Valid {
			data.RankVolatility = rankVolatility.Float64
		} else {
			data.RankVolatility = 0
		}

		// Use the period end date as the representative date
		data.Date = periodEnd
		results = append(results, data)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reddit heatmap rows: %w", err)
	}

	return results, nil
}

// GetTickerRedditHistory retrieves Reddit metrics history for a specific ticker
func GetTickerRedditHistory(symbol string, days int) (*RedditTickerHistory, error) {
	if days <= 0 {
		days = 30 // Default to 30 days
	}

	query := `
		SELECT
			ticker_symbol,
			date,
			avg_rank,
			min_rank,
			max_rank,
			total_mentions,
			total_upvotes,
			rank_volatility,
			trend_direction,
			popularity_score,
			data_source
		FROM reddit_heatmap_daily
		WHERE ticker_symbol = $1
			AND date >= CURRENT_DATE - $2::INTEGER * INTERVAL '1 day'
		ORDER BY date DESC
	`

	rows, err := DB.Query(query, symbol, days)
	if err != nil {
		return nil, fmt.Errorf("failed to query ticker reddit history: %w", err)
	}
	defer rows.Close()

	var history []RedditHeatmapData
	for rows.Next() {
		var data RedditHeatmapData
		var rankVolatility sql.NullFloat64
		var trendDirection sql.NullString

		err := rows.Scan(
			&data.TickerSymbol,
			&data.Date,
			&data.AvgRank,
			&data.MinRank,
			&data.MaxRank,
			&data.TotalMentions,
			&data.TotalUpvotes,
			&rankVolatility,
			&trendDirection,
			&data.PopularityScore,
			&data.DataSource,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ticker reddit history row: %w", err)
		}

		// Handle NULL rank_volatility
		if rankVolatility.Valid {
			data.RankVolatility = rankVolatility.Float64
		} else {
			data.RankVolatility = 0
		}

		// Handle NULL trend_direction
		if trendDirection.Valid {
			data.TrendDirection = trendDirection.String
		} else {
			data.TrendDirection = "stable"
		}

		history = append(history, data)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ticker reddit history rows: %w", err)
	}

	if len(history) == 0 {
		return nil, sql.ErrNoRows
	}

	// Calculate summary statistics
	summary := calculateHistorySummary(history)

	return &RedditTickerHistory{
		TickerSymbol: symbol,
		History:      history,
		Summary:      summary,
	}, nil
}

// calculateHistorySummary computes aggregate statistics from history data
func calculateHistorySummary(history []RedditHeatmapData) RedditHistorySummary {
	if len(history) == 0 {
		return RedditHistorySummary{}
	}

	summary := RedditHistorySummary{
		DaysAppeared: len(history),
		BestRank:     int(^uint(0) >> 1), // Max int
		WorstRank:    0,
		PeriodStart:  history[len(history)-1].Date,
		PeriodEnd:    history[0].Date,
	}

	totalPopularity := 0.0
	totalRank := 0.0

	for _, day := range history {
		totalPopularity += day.PopularityScore
		totalRank += day.AvgRank
		summary.TotalMentions += day.TotalMentions
		summary.TotalUpvotes += day.TotalUpvotes

		if day.MinRank < summary.BestRank {
			summary.BestRank = day.MinRank
		}
		if day.MaxRank > summary.WorstRank {
			summary.WorstRank = day.MaxRank
		}
	}

	summary.AvgPopularity = totalPopularity / float64(len(history))
	summary.AvgRank = totalRank / float64(len(history))

	return summary
}

// GetLatestRedditDate returns the most recent date with Reddit data
func GetLatestRedditDate() (time.Time, error) {
	query := `SELECT MAX(date) FROM reddit_heatmap_daily`

	var latestDate sql.NullTime
	err := DB.QueryRow(query).Scan(&latestDate)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get latest reddit date: %w", err)
	}

	if !latestDate.Valid {
		return time.Time{}, sql.ErrNoRows
	}

	return latestDate.Time, nil
}
