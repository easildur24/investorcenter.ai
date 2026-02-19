package database

import (
	"database/sql"
	"errors"
	"fmt"
	"investorcenter-api/models"
	"strings"

	"github.com/lib/pq"
)

// Sentinel errors for watchlist operations
var (
	ErrWatchListNotFound         = errors.New("watch list not found")
	ErrWatchListItemNotFound     = errors.New("watch list item not found")
	ErrTickerNotFound            = errors.New("ticker not found in database")
	ErrTickerAlreadyExists       = errors.New("ticker already exists in this watch list")
	ErrWatchListItemLimitReached = errors.New("watch list item limit reached")
)

// Watchlist limits (keep in sync with DB trigger check_watch_list_item_limit)
const (
	MaxWatchListsPerUser = 3
	MaxItemsPerWatchList = 10 // enforced by DB trigger for free tier
)

// Watch List Operations

// CreateWatchList creates a new watch list
func CreateWatchList(watchList *models.WatchList) error {
	query := `
		INSERT INTO watch_lists (user_id, name, description, is_default, display_order)
		VALUES ($1, $2, $3, $4, COALESCE((SELECT MAX(display_order) + 1 FROM watch_lists WHERE user_id = $1), 0))
		RETURNING id, created_at, updated_at, display_order
	`
	err := DB.QueryRow(
		query,
		watchList.UserID,
		watchList.Name,
		watchList.Description,
		watchList.IsDefault,
	).Scan(&watchList.ID, &watchList.CreatedAt, &watchList.UpdatedAt, &watchList.DisplayOrder)

	if err != nil {
		return fmt.Errorf("failed to create watch list: %w", err)
	}
	return nil
}

// GetWatchListsByUserID retrieves all watch lists for a user
func GetWatchListsByUserID(userID string) ([]models.WatchListSummary, error) {
	query := `
		SELECT
			wl.id, wl.name, wl.description, wl.is_default, wl.created_at, wl.updated_at,
			COUNT(wli.id) as item_count
		FROM watch_lists wl
		LEFT JOIN watch_list_items wli ON wl.id = wli.watch_list_id
		WHERE wl.user_id = $1
		GROUP BY wl.id, wl.name, wl.description, wl.is_default, wl.created_at, wl.updated_at, wl.display_order
		ORDER BY wl.display_order ASC, wl.created_at ASC
	`
	rows, err := DB.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get watch lists: %w", err)
	}
	defer rows.Close()

	watchLists := []models.WatchListSummary{}
	for rows.Next() {
		var wl models.WatchListSummary
		err := rows.Scan(
			&wl.ID,
			&wl.Name,
			&wl.Description,
			&wl.IsDefault,
			&wl.CreatedAt,
			&wl.UpdatedAt,
			&wl.ItemCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan watch list: %w", err)
		}
		watchLists = append(watchLists, wl)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating watch lists: %w", err)
	}

	return watchLists, nil
}

// GetWatchListByID retrieves a single watch list by ID
func GetWatchListByID(watchListID string, userID string) (*models.WatchList, error) {
	query := `
		SELECT id, user_id, name, description, is_default, display_order, is_public, public_slug, created_at, updated_at
		FROM watch_lists
		WHERE id = $1 AND user_id = $2
	`
	watchList := &models.WatchList{}
	err := DB.QueryRow(query, watchListID, userID).Scan(
		&watchList.ID,
		&watchList.UserID,
		&watchList.Name,
		&watchList.Description,
		&watchList.IsDefault,
		&watchList.DisplayOrder,
		&watchList.IsPublic,
		&watchList.PublicSlug,
		&watchList.CreatedAt,
		&watchList.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrWatchListNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get watch list: %w", err)
	}
	return watchList, nil
}

// UpdateWatchList updates watch list metadata
func UpdateWatchList(watchList *models.WatchList) error {
	query := `
		UPDATE watch_lists
		SET name = $1, description = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3 AND user_id = $4
	`
	result, err := DB.Exec(query, watchList.Name, watchList.Description, watchList.ID, watchList.UserID)
	if err != nil {
		return fmt.Errorf("failed to update watch list: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrWatchListNotFound
	}
	return nil
}

// DeleteWatchList deletes a watch list
func DeleteWatchList(watchListID string, userID string) error {
	query := `DELETE FROM watch_lists WHERE id = $1 AND user_id = $2`
	result, err := DB.Exec(query, watchListID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete watch list: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrWatchListNotFound
	}
	return nil
}

// Watch List Item Operations

// AddTickerToWatchList adds a ticker to a watch list
func AddTickerToWatchList(item *models.WatchListItem) error {
	// Verify ticker exists in tickers table
	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tickers WHERE symbol = $1)", item.Symbol).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to verify ticker: %w", err)
	}
	if !exists {
		return ErrTickerNotFound
	}

	query := `
		INSERT INTO watch_list_items (watch_list_id, symbol, notes, tags, target_buy_price, target_sell_price, display_order)
		VALUES ($1, $2, $3, $4, $5, $6, COALESCE((SELECT MAX(display_order) + 1 FROM watch_list_items WHERE watch_list_id = $1), 0))
		RETURNING id, added_at, display_order
	`
	err = DB.QueryRow(
		query,
		item.WatchListID,
		item.Symbol,
		item.Notes,
		pq.Array(item.Tags),
		item.TargetBuyPrice,
		item.TargetSellPrice,
	).Scan(&item.ID, &item.AddedAt, &item.DisplayOrder)

	if err != nil {
		// Check for unique constraint violation
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return ErrTickerAlreadyExists
		}
		// Check for limit trigger (fired by check_watch_list_item_limit)
		if strings.Contains(err.Error(), "Watch list limit reached") {
			return ErrWatchListItemLimitReached
		}
		return fmt.Errorf("failed to add ticker to watch list: %w", err)
	}
	return nil
}

// GetWatchListItems retrieves all items in a watch list
func GetWatchListItems(watchListID string) ([]models.WatchListItem, error) {
	query := `
		SELECT id, watch_list_id, symbol, notes, tags, target_buy_price, target_sell_price, added_at, display_order
		FROM watch_list_items
		WHERE watch_list_id = $1
		ORDER BY display_order ASC, added_at DESC
	`
	rows, err := DB.Query(query, watchListID)
	if err != nil {
		return nil, fmt.Errorf("failed to get watch list items: %w", err)
	}
	defer rows.Close()

	items := []models.WatchListItem{}
	for rows.Next() {
		var item models.WatchListItem
		err := rows.Scan(
			&item.ID,
			&item.WatchListID,
			&item.Symbol,
			&item.Notes,
			pq.Array(&item.Tags),
			&item.TargetBuyPrice,
			&item.TargetSellPrice,
			&item.AddedAt,
			&item.DisplayOrder,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan watch list item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating watch list items: %w", err)
	}

	return items, nil
}

// GetWatchListItemsWithData retrieves items with ticker data, Reddit metrics,
// screener_data (IC Score, fundamentals, valuation), and active alert counts.
//
// Performance: the LATERAL JOINs execute per-row and require these indexes:
//   - reddit_heatmap_daily(ticker_symbol, date DESC)
//   - reddit_ticker_rankings(ticker_symbol, snapshot_time DESC)
//   - alert_rules(watch_list_id, symbol, is_active)
//   - screener_data(symbol) — unique in prod materialized view
func GetWatchListItemsWithData(watchListID string) ([]models.WatchListItemDetail, error) {
	query := `
		SELECT
			-- watch_list_items (9 cols)
			wli.id, wli.watch_list_id, wli.symbol, wli.notes, wli.tags,
			wli.target_buy_price, wli.target_sell_price, wli.added_at, wli.display_order,

			-- tickers (4 cols; COALESCE guards against orphaned items with no ticker row)
			COALESCE(t.name, wli.symbol), COALESCE(t.exchange, ''), COALESCE(t.asset_type, 'stock'), t.logo_url,

			-- reddit_heatmap_daily via LATERAL (4 cols)
			rhd.avg_rank, rhd.total_mentions, rhd.popularity_score, rhd.trend_direction,

			-- reddit_ticker_rankings via LATERAL (1 col)
			rtr.rank_change,

			-- screener_data (28 cols)
			sd.ic_score, sd.ic_rating,
			sd.value_score, sd.growth_score, sd.profitability_score,
			sd.financial_health_score, sd.momentum_score,
			sd.analyst_consensus_score, sd.insider_activity_score,
			sd.institutional_score, sd.news_sentiment_score, sd.technical_score,
			sd.ic_sector_percentile, sd.lifecycle_stage,
			sd.pe_ratio, sd.pb_ratio, sd.ps_ratio,
			sd.roe, sd.roa, sd.gross_margin, sd.operating_margin, sd.net_margin,
			sd.debt_to_equity, sd.current_ratio,
			sd.revenue_growth, sd.eps_growth_yoy,
			sd.dividend_yield, sd.payout_ratio,

			-- alert count (1 col)
			COALESCE(ac.alert_count, 0)
		FROM watch_list_items wli
		LEFT JOIN tickers t ON wli.symbol = t.symbol
		LEFT JOIN LATERAL (
			SELECT avg_rank, total_mentions, popularity_score, trend_direction
			FROM reddit_heatmap_daily
			WHERE ticker_symbol = wli.symbol
			ORDER BY date DESC
			LIMIT 1
		) rhd ON true
		LEFT JOIN LATERAL (
			SELECT CASE WHEN rank_24h_ago IS NOT NULL
				THEN rank - rank_24h_ago
				ELSE NULL
			END as rank_change
			FROM reddit_ticker_rankings
			WHERE ticker_symbol = wli.symbol
			ORDER BY snapshot_time DESC
			LIMIT 1
		) rtr ON true
		LEFT JOIN screener_data sd ON wli.symbol = sd.symbol
		LEFT JOIN LATERAL (
			SELECT COUNT(*) as alert_count
			FROM alert_rules
			WHERE watch_list_id = wli.watch_list_id
			  AND symbol = wli.symbol
			  AND is_active = true
		) ac ON true
		WHERE wli.watch_list_id = $1
		ORDER BY wli.display_order ASC, wli.added_at DESC
	`
	rows, err := DB.Query(query, watchListID)
	if err != nil {
		return nil, fmt.Errorf("failed to get watch list items with data: %w", err)
	}
	defer rows.Close()

	items := []models.WatchListItemDetail{}
	for rows.Next() {
		item, err := scanWatchListItemDetail(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating watch list items: %w", err)
	}

	return items, nil
}

// scanWatchListItemDetail scans a single row from the GetWatchListItemsWithData query
// into a WatchListItemDetail struct. Centralises the 47-column positional scan so
// callers only need `scanWatchListItemDetail(rows)` instead of 150+ lines of inline code.
func scanWatchListItemDetail(rows *sql.Rows) (models.WatchListItemDetail, error) {
	var item models.WatchListItemDetail

	// Reddit fields (nullable)
	var redditRank sql.NullFloat64
	var redditMentions sql.NullInt32
	var redditPopularity sql.NullFloat64
	var redditTrend sql.NullString
	var redditRankChange sql.NullFloat64

	// Screener fields (nullable)
	var icScore, valueScore, growthScore, profitabilityScore sql.NullFloat64
	var financialHealthScore, momentumScore sql.NullFloat64
	var analystConsensusScore, insiderActivityScore sql.NullFloat64
	var institutionalScore, newsSentimentScore, technicalScore sql.NullFloat64
	var sectorPercentile sql.NullFloat64
	var icRating, lifecycleStage sql.NullString
	var peRatio, pbRatio, psRatio sql.NullFloat64
	var roe, roa, grossMargin, operatingMargin, netMargin sql.NullFloat64
	var debtToEquity, currentRatio sql.NullFloat64
	var revenueGrowth, epsGrowth sql.NullFloat64
	var dividendYield, payoutRatio sql.NullFloat64

	err := rows.Scan(
		// watch_list_items (9 cols)
		&item.ID, &item.WatchListID, &item.Symbol, &item.Notes,
		pq.Array(&item.Tags),
		&item.TargetBuyPrice, &item.TargetSellPrice, &item.AddedAt, &item.DisplayOrder,

		// tickers (4 cols)
		&item.Name, &item.Exchange, &item.AssetType, &item.LogoURL,

		// reddit (5 cols)
		&redditRank, &redditMentions, &redditPopularity, &redditTrend,
		&redditRankChange,

		// screener_data (28 cols)
		&icScore, &icRating,
		&valueScore, &growthScore, &profitabilityScore,
		&financialHealthScore, &momentumScore,
		&analystConsensusScore, &insiderActivityScore,
		&institutionalScore, &newsSentimentScore, &technicalScore,
		&sectorPercentile, &lifecycleStage,
		&peRatio, &pbRatio, &psRatio,
		&roe, &roa, &grossMargin, &operatingMargin, &netMargin,
		&debtToEquity, &currentRatio,
		&revenueGrowth, &epsGrowth,
		&dividendYield, &payoutRatio,

		// alert count (1 col)
		&item.AlertCount,
	)
	if err != nil {
		return item, fmt.Errorf("failed to scan watch list item detail: %w", err)
	}

	// Convert Reddit nullable fields
	if redditRank.Valid {
		rank := int(redditRank.Float64)
		item.RedditRank = &rank
	}
	if redditMentions.Valid {
		mentions := int(redditMentions.Int32)
		item.RedditMentions = &mentions
	}
	if redditPopularity.Valid {
		item.RedditPopularity = &redditPopularity.Float64
	}
	if redditTrend.Valid {
		item.RedditTrend = &redditTrend.String
	}
	if redditRankChange.Valid {
		change := int(redditRankChange.Float64)
		item.RedditRankChange = &change
	}

	// Convert screener nullable fields
	if icScore.Valid {
		item.ICScore = &icScore.Float64
	}
	if icRating.Valid {
		item.ICRating = &icRating.String
	}
	if valueScore.Valid {
		item.ValueScore = &valueScore.Float64
	}
	if growthScore.Valid {
		item.GrowthScore = &growthScore.Float64
	}
	if profitabilityScore.Valid {
		item.ProfitabilityScore = &profitabilityScore.Float64
	}
	if financialHealthScore.Valid {
		item.FinancialHealthScore = &financialHealthScore.Float64
	}
	if momentumScore.Valid {
		item.MomentumScore = &momentumScore.Float64
	}
	if analystConsensusScore.Valid {
		item.AnalystConsensusScore = &analystConsensusScore.Float64
	}
	if insiderActivityScore.Valid {
		item.InsiderActivityScore = &insiderActivityScore.Float64
	}
	if institutionalScore.Valid {
		item.InstitutionalScore = &institutionalScore.Float64
	}
	if newsSentimentScore.Valid {
		item.NewsSentimentScore = &newsSentimentScore.Float64
	}
	if technicalScore.Valid {
		item.TechnicalScore = &technicalScore.Float64
	}
	if sectorPercentile.Valid {
		item.SectorPercentile = &sectorPercentile.Float64
	}
	if lifecycleStage.Valid {
		item.LifecycleStage = &lifecycleStage.String
	}
	if peRatio.Valid {
		item.PERatio = &peRatio.Float64
	}
	if pbRatio.Valid {
		item.PBRatio = &pbRatio.Float64
	}
	if psRatio.Valid {
		item.PSRatio = &psRatio.Float64
	}
	if roe.Valid {
		item.ROE = &roe.Float64
	}
	if roa.Valid {
		item.ROA = &roa.Float64
	}
	if grossMargin.Valid {
		item.GrossMargin = &grossMargin.Float64
	}
	if operatingMargin.Valid {
		item.OperatingMargin = &operatingMargin.Float64
	}
	if netMargin.Valid {
		item.NetMargin = &netMargin.Float64
	}
	if debtToEquity.Valid {
		item.DebtToEquity = &debtToEquity.Float64
	}
	if currentRatio.Valid {
		item.CurrentRatio = &currentRatio.Float64
	}
	if revenueGrowth.Valid {
		item.RevenueGrowth = &revenueGrowth.Float64
	}
	if epsGrowth.Valid {
		item.EPSGrowth = &epsGrowth.Float64
	}
	if dividendYield.Valid {
		item.DividendYield = &dividendYield.Float64
	}
	if payoutRatio.Valid {
		item.PayoutRatio = &payoutRatio.Float64
	}

	return item, nil
}

// UpdateWatchListItem updates ticker metadata
func UpdateWatchListItem(item *models.WatchListItem) error {
	query := `
		UPDATE watch_list_items
		SET notes = $1, tags = $2, target_buy_price = $3, target_sell_price = $4
		WHERE id = $5
	`
	result, err := DB.Exec(
		query,
		item.Notes,
		pq.Array(item.Tags),
		item.TargetBuyPrice,
		item.TargetSellPrice,
		item.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update watch list item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrWatchListItemNotFound
	}
	return nil
}

// RemoveTickerFromWatchList removes a ticker from watch list
func RemoveTickerFromWatchList(watchListID string, symbol string) error {
	query := `DELETE FROM watch_list_items WHERE watch_list_id = $1 AND symbol = $2`
	result, err := DB.Exec(query, watchListID, symbol)
	if err != nil {
		return fmt.Errorf("failed to remove ticker from watch list: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrWatchListItemNotFound
	}
	return nil
}

// BulkAddTickers adds multiple tickers to a watch list
func BulkAddTickers(watchListID string, symbols []string) ([]string, []string, error) {
	added := []string{}
	failed := []string{}

	for _, symbol := range symbols {
		item := &models.WatchListItem{
			WatchListID: watchListID,
			Symbol:      symbol,
			Tags:        []string{},
		}
		err := AddTickerToWatchList(item)
		if err == nil {
			added = append(added, symbol)
		} else {
			failed = append(failed, symbol)
		}
	}

	return added, failed, nil
}

// GetWatchListItemByID retrieves a single watch list item
func GetWatchListItemByID(itemID string) (*models.WatchListItem, error) {
	query := `
		SELECT id, watch_list_id, symbol, notes, tags, target_buy_price, target_sell_price, added_at, display_order
		FROM watch_list_items
		WHERE id = $1
	`
	item := &models.WatchListItem{}
	err := DB.QueryRow(query, itemID).Scan(
		&item.ID,
		&item.WatchListID,
		&item.Symbol,
		&item.Notes,
		pq.Array(&item.Tags),
		&item.TargetBuyPrice,
		&item.TargetSellPrice,
		&item.AddedAt,
		&item.DisplayOrder,
	)

	if err == sql.ErrNoRows {
		return nil, ErrWatchListItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get watch list item: %w", err)
	}
	return item, nil
}

// ValidateItemsBelongToWatchList checks that all given item IDs belong to the specified watchlist.
// Returns an error if any item ID is not found in the watchlist.
func ValidateItemsBelongToWatchList(watchListID string, itemIDs []string) error {
	if len(itemIDs) == 0 {
		return nil
	}

	// Build parameterized query for the item IDs
	placeholders := make([]string, len(itemIDs))
	args := make([]interface{}, 0, len(itemIDs)+1)
	args = append(args, watchListID)
	for i, id := range itemIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM watch_list_items
		WHERE watch_list_id = $1 AND id IN (%s)
	`, strings.Join(placeholders, ", "))

	var count int
	err := DB.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to validate item ownership: %w", err)
	}
	if count != len(itemIDs) {
		return errors.New("one or more items do not belong to this watch list")
	}
	return nil
}

// CreateWatchListAtomic creates a new watch list with an atomic count check to prevent TOCTOU races.
// Returns ErrWatchListNotFound (repurposed as limit error) if the user already has maxLists.
func CreateWatchListAtomic(watchList *models.WatchList, maxLists int) error {
	query := `
		INSERT INTO watch_lists (user_id, name, description, is_default, display_order)
		SELECT $1, $2, $3, $4, COALESCE((SELECT MAX(display_order) + 1 FROM watch_lists WHERE user_id = $1), 0)
		WHERE (SELECT COUNT(*) FROM watch_lists WHERE user_id = $1) < $5
		RETURNING id, created_at, updated_at, display_order
	`
	err := DB.QueryRow(
		query,
		watchList.UserID,
		watchList.Name,
		watchList.Description,
		watchList.IsDefault,
		maxLists,
	).Scan(&watchList.ID, &watchList.CreatedAt, &watchList.UpdatedAt, &watchList.DisplayOrder)

	if err == sql.ErrNoRows {
		return fmt.Errorf("watch list limit reached: maximum %d allowed", maxLists)
	}
	if err != nil {
		return fmt.Errorf("failed to create watch list: %w", err)
	}
	return nil
}

// UpdateItemDisplayOrder updates display order for items
func UpdateItemDisplayOrder(itemID string, displayOrder int) error {
	query := `UPDATE watch_list_items SET display_order = $1 WHERE id = $2`
	_, err := DB.Exec(query, displayOrder, itemID)
	return err
}
