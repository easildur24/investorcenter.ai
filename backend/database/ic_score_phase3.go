package database

import (
	"context"
	"encoding/json"
	"time"

	"investorcenter-api/models"
)

// ===================
// IC Score Events Repository
// ===================

// GetICScoreEvents retrieves events for a ticker within a date range
func GetICScoreEvents(ticker string, days int) ([]models.ICScoreEvent, error) {
	events := []models.ICScoreEvent{}

	query := `
		SELECT *
		FROM ic_score_events
		WHERE ticker = $1
		  AND event_date >= CURRENT_DATE - $2 * INTERVAL '1 day'
		ORDER BY event_date DESC
	`

	err := DB.Select(&events, query, ticker, days)
	if err != nil {
		return nil, err
	}

	return events, nil
}

// GetRecentICScoreEvents retrieves recent events that may trigger score recalculation
func GetRecentICScoreEvents(ticker string) ([]models.ICScoreEvent, error) {
	events := []models.ICScoreEvent{}

	query := `
		SELECT *
		FROM ic_score_events
		WHERE ticker = $1
		  AND event_date >= CURRENT_DATE - INTERVAL '7 days'
		ORDER BY event_date DESC
		LIMIT 10
	`

	err := DB.Select(&events, query, ticker)
	if err != nil {
		return nil, err
	}

	return events, nil
}

// UpsertICScoreEvent inserts or updates an IC Score event
func UpsertICScoreEvent(ctx context.Context, event *models.ICScoreEvent) error {
	metadataJSON, _ := json.Marshal(event.Metadata)

	query := `
		INSERT INTO ic_score_events (
			ticker, event_type, event_date, description,
			impact_direction, impact_magnitude, source, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (ticker, event_type, event_date)
		DO UPDATE SET
			description = EXCLUDED.description,
			impact_direction = EXCLUDED.impact_direction,
			impact_magnitude = EXCLUDED.impact_magnitude,
			source = EXCLUDED.source,
			metadata = EXCLUDED.metadata
	`

	_, err := DB.ExecContext(ctx, query,
		event.Ticker, event.EventType, event.EventDate, event.Description,
		event.ImpactDirection, event.ImpactMagnitude, event.Source, metadataJSON,
	)

	return err
}

// DeleteOldICScoreEvents removes events older than the specified days
func DeleteOldICScoreEvents(ctx context.Context, days int) (int64, error) {
	query := `
		DELETE FROM ic_score_events
		WHERE event_date < CURRENT_DATE - $1 * INTERVAL '1 day'
	`

	result, err := DB.ExecContext(ctx, query, days)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// ===================
// Stock Peers Repository
// ===================

// GetStockPeers retrieves peer stocks for a ticker
func GetStockPeers(ticker string, limit int) ([]models.StockPeer, error) {
	peers := []models.StockPeer{}

	query := `
		SELECT *
		FROM stock_peers
		WHERE ticker = $1
		  AND calculated_at = (
			SELECT MAX(calculated_at) FROM stock_peers WHERE ticker = $1
		  )
		ORDER BY similarity_score DESC
		LIMIT $2
	`

	err := DB.Select(&peers, query, ticker, limit)
	if err != nil {
		return nil, err
	}

	return peers, nil
}

// GetStockPeersWithScores retrieves peers with their IC Scores
func GetStockPeersWithScores(ticker string, limit int) ([]models.StockPeerResponse, error) {
	peers := []models.StockPeerResponse{}

	query := `
		SELECT
			sp.ticker,
			sp.peer_ticker,
			t.company_name,
			i.overall_score as ic_score,
			sp.similarity_score,
			sp.similarity_factors
		FROM stock_peers sp
		JOIN tickers t ON sp.peer_ticker = t.symbol
		LEFT JOIN ic_scores i ON sp.peer_ticker = i.ticker AND i.date = CURRENT_DATE
		WHERE sp.ticker = $1
		  AND sp.calculated_at = (
			SELECT MAX(calculated_at) FROM stock_peers WHERE ticker = $1
		  )
		ORDER BY sp.similarity_score DESC
		LIMIT $2
	`

	err := DB.Select(&peers, query, ticker, limit)
	if err != nil {
		return nil, err
	}

	return peers, nil
}

// UpsertStockPeer inserts or updates a stock peer relationship
func UpsertStockPeer(ctx context.Context, peer *models.StockPeer) error {
	factorsJSON, _ := json.Marshal(peer.SimilarityFactors)

	query := `
		INSERT INTO stock_peers (
			ticker, peer_ticker, similarity_score, similarity_factors, calculated_at
		) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (ticker, peer_ticker, calculated_at)
		DO UPDATE SET
			similarity_score = EXCLUDED.similarity_score,
			similarity_factors = EXCLUDED.similarity_factors
	`

	_, err := DB.ExecContext(ctx, query,
		peer.Ticker, peer.PeerTicker, peer.SimilarityScore, factorsJSON, peer.CalculatedAt,
	)

	return err
}

// BatchUpsertStockPeers inserts multiple stock peer relationships
func BatchUpsertStockPeers(ctx context.Context, peers []models.StockPeer) error {
	tx, err := DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, peer := range peers {
		if err := UpsertStockPeer(ctx, &peer); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetPeerComparisonSummary gets peer comparison statistics for a ticker
func GetPeerComparisonSummary(ticker string) (*models.PeerComparisonResponse, error) {
	var summary models.PeerComparisonResponse

	// Get stock's own IC Score
	var icScore float64
	scoreQuery := `SELECT overall_score FROM ic_scores WHERE ticker = $1 AND date = CURRENT_DATE LIMIT 1`
	if err := DB.Get(&icScore, scoreQuery, ticker); err != nil {
		return nil, err
	}

	summary.Ticker = ticker
	summary.ICScore = icScore

	// Get peers with scores
	peers, err := GetStockPeersWithScores(ticker, 5)
	if err != nil {
		return nil, err
	}
	summary.Peers = peers

	// Calculate average peer score
	if len(peers) > 0 {
		var totalScore float64
		var count int
		for _, p := range peers {
			if p.ICScore != nil {
				totalScore += *p.ICScore
				count++
			}
		}
		if count > 0 {
			avgScore := totalScore / float64(count)
			summary.AvgPeerScore = &avgScore
			delta := icScore - avgScore
			summary.VsPeersDelta = &delta
		}
	}

	return &summary, nil
}

// ===================
// Catalyst Events Repository
// ===================

// GetCatalystEvents retrieves upcoming catalysts for a ticker
func GetCatalystEvents(ticker string, limit int) ([]models.CatalystEvent, error) {
	catalysts := []models.CatalystEvent{}

	query := `
		SELECT *
		FROM catalyst_events
		WHERE ticker = $1
		  AND (days_until >= -7 OR days_until IS NULL)
		  AND (expires_at IS NULL OR expires_at >= CURRENT_DATE)
		ORDER BY COALESCE(days_until, 999) ASC
		LIMIT $2
	`

	err := DB.Select(&catalysts, query, ticker, limit)
	if err != nil {
		return nil, err
	}

	return catalysts, nil
}

// GetUpcomingCatalysts retrieves all upcoming catalysts across stocks
func GetUpcomingCatalysts(days int, limit int) ([]models.CatalystEvent, error) {
	catalysts := []models.CatalystEvent{}

	query := `
		SELECT *
		FROM catalyst_events
		WHERE days_until >= 0
		  AND days_until <= $1
		  AND (expires_at IS NULL OR expires_at >= CURRENT_DATE)
		ORDER BY days_until ASC, confidence DESC NULLS LAST
		LIMIT $2
	`

	err := DB.Select(&catalysts, query, days, limit)
	if err != nil {
		return nil, err
	}

	return catalysts, nil
}

// UpsertCatalystEvent inserts or updates a catalyst event
func UpsertCatalystEvent(ctx context.Context, catalyst *models.CatalystEvent) error {
	metadataJSON, _ := json.Marshal(catalyst.Metadata)

	query := `
		INSERT INTO catalyst_events (
			ticker, event_type, title, event_date, icon, impact,
			confidence, days_until, metadata, updated_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), $10)
		ON CONFLICT (ticker, event_type, event_date)
		DO UPDATE SET
			title = EXCLUDED.title,
			icon = EXCLUDED.icon,
			impact = EXCLUDED.impact,
			confidence = EXCLUDED.confidence,
			days_until = EXCLUDED.days_until,
			metadata = EXCLUDED.metadata,
			updated_at = NOW(),
			expires_at = EXCLUDED.expires_at
	`

	_, err := DB.ExecContext(ctx, query,
		catalyst.Ticker, catalyst.EventType, catalyst.Title, catalyst.EventDate,
		catalyst.Icon, catalyst.Impact, catalyst.Confidence, catalyst.DaysUntil,
		metadataJSON, catalyst.ExpiresAt,
	)

	return err
}

// RefreshCatalystDaysUntil updates days_until for all catalysts
func RefreshCatalystDaysUntil(ctx context.Context) error {
	query := `
		UPDATE catalyst_events
		SET days_until = CASE
			WHEN event_date IS NOT NULL THEN event_date - CURRENT_DATE
			ELSE NULL
		END,
		updated_at = NOW()
		WHERE event_date IS NOT NULL
	`

	_, err := DB.ExecContext(ctx, query)
	return err
}

// DeleteExpiredCatalysts removes expired catalyst events
func DeleteExpiredCatalysts(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM catalyst_events
		WHERE expires_at < CURRENT_DATE
		   OR (days_until < -30 AND days_until IS NOT NULL)
	`

	result, err := DB.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// ===================
// IC Score Changes Repository
// ===================

// GetICScoreChanges retrieves score change history for a ticker
func GetICScoreChanges(ticker string, limit int) ([]models.ICScoreChange, error) {
	changes := []models.ICScoreChange{}

	query := `
		SELECT *
		FROM ic_score_changes
		WHERE ticker = $1
		ORDER BY calculated_at DESC
		LIMIT $2
	`

	err := DB.Select(&changes, query, ticker, limit)
	if err != nil {
		return nil, err
	}

	return changes, nil
}

// GetSignificantScoreChanges retrieves significant score changes across stocks
func GetSignificantScoreChanges(minDelta float64, days int, limit int) ([]models.ICScoreChange, error) {
	changes := []models.ICScoreChange{}

	query := `
		SELECT *
		FROM ic_score_changes
		WHERE ABS(delta) >= $1
		  AND calculated_at >= CURRENT_DATE - $2 * INTERVAL '1 day'
		ORDER BY ABS(delta) DESC, calculated_at DESC
		LIMIT $3
	`

	err := DB.Select(&changes, query, minDelta, days, limit)
	if err != nil {
		return nil, err
	}

	return changes, nil
}

// UpsertICScoreChange inserts or updates a score change record
func UpsertICScoreChange(ctx context.Context, change *models.ICScoreChange) error {
	factorChangesJSON, _ := json.Marshal(change.FactorChanges)
	triggerEventsJSON, _ := json.Marshal(change.TriggerEvents)

	query := `
		INSERT INTO ic_score_changes (
			ticker, calculated_at, previous_score, current_score, delta,
			factor_changes, trigger_events, smoothing_applied, summary
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (ticker, calculated_at)
		DO UPDATE SET
			previous_score = EXCLUDED.previous_score,
			current_score = EXCLUDED.current_score,
			delta = EXCLUDED.delta,
			factor_changes = EXCLUDED.factor_changes,
			trigger_events = EXCLUDED.trigger_events,
			smoothing_applied = EXCLUDED.smoothing_applied,
			summary = EXCLUDED.summary
	`

	_, err := DB.ExecContext(ctx, query,
		change.Ticker, change.CalculatedAt, change.PreviousScore, change.CurrentScore,
		change.Delta, factorChangesJSON, triggerEventsJSON, change.SmoothingApplied, change.Summary,
	)

	return err
}

// GetLatestScoreChange retrieves the most recent score change for a ticker
func GetLatestScoreChange(ticker string) (*models.ICScoreChange, error) {
	var change models.ICScoreChange

	query := `
		SELECT *
		FROM ic_score_changes
		WHERE ticker = $1
		ORDER BY calculated_at DESC
		LIMIT 1
	`

	err := DB.Get(&change, query, ticker)
	if err != nil {
		return nil, err
	}

	return &change, nil
}

// ===================
// IC Score Settings Repository
// ===================

// GetICScoreSetting retrieves a setting by key
func GetICScoreSetting(key string) (*models.ICScoreSetting, error) {
	var setting models.ICScoreSetting

	query := `SELECT * FROM ic_score_settings WHERE setting_key = $1`

	err := DB.Get(&setting, query, key)
	if err != nil {
		return nil, err
	}

	return &setting, nil
}

// GetAllICScoreSettings retrieves all settings
func GetAllICScoreSettings() ([]models.ICScoreSetting, error) {
	settings := []models.ICScoreSetting{}

	query := `SELECT * FROM ic_score_settings ORDER BY setting_key`

	err := DB.Select(&settings, query)
	if err != nil {
		return nil, err
	}

	return settings, nil
}

// UpdateICScoreSetting updates a setting value
func UpdateICScoreSetting(ctx context.Context, key string, value map[string]any) error {
	valueJSON, _ := json.Marshal(value)

	query := `
		UPDATE ic_score_settings
		SET setting_value = $2, updated_at = NOW()
		WHERE setting_key = $1
	`

	_, err := DB.ExecContext(ctx, query, key, valueJSON)
	return err
}

// ===================
// Previous Score Helpers
// ===================

// GetPreviousICScore retrieves the previous day's IC Score for a ticker
func GetPreviousICScore(ticker string) (*float64, error) {
	var score float64

	query := `
		SELECT overall_score
		FROM ic_scores
		WHERE ticker = $1
		  AND date < CURRENT_DATE
		ORDER BY date DESC
		LIMIT 1
	`

	err := DB.Get(&score, query, ticker)
	if err != nil {
		return nil, err
	}

	return &score, nil
}

// UpdateICScoreWithPhase3Fields updates IC Score record with Phase 3 fields
func UpdateICScoreWithPhase3Fields(ctx context.Context, ticker string, date time.Time, fields map[string]any) error {
	// Build dynamic update query
	query := `
		UPDATE ic_scores
		SET previous_score = COALESCE($3, previous_score),
		    raw_score = COALESCE($4, raw_score),
		    smoothing_applied = COALESCE($5, smoothing_applied),
		    peer_avg_score = COALESCE($6, peer_avg_score),
		    vs_peers_delta = COALESCE($7, vs_peers_delta)
		WHERE ticker = $1 AND date = $2
	`

	_, err := DB.ExecContext(ctx, query,
		ticker, date,
		fields["previous_score"],
		fields["raw_score"],
		fields["smoothing_applied"],
		fields["peer_avg_score"],
		fields["vs_peers_delta"],
	)

	return err
}
