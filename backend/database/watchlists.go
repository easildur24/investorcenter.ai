package database

import (
	"database/sql"
	"errors"
	"fmt"
	"investorcenter-api/models"
	"strings"

	"github.com/lib/pq"
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
		return nil, errors.New("watch list not found")
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
		return errors.New("watch list not found or unauthorized")
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
		return errors.New("watch list not found or unauthorized")
	}
	return nil
}

// Watch List Item Operations

// AddTickerToWatchList adds a ticker to a watch list
func AddTickerToWatchList(item *models.WatchListItem) error {
	// Verify ticker exists in stocks table
	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM stocks WHERE symbol = $1)", item.Symbol).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to verify ticker: %w", err)
	}
	if !exists {
		return errors.New("ticker not found in database")
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
			return errors.New("ticker already exists in this watch list")
		}
		// Check for limit trigger
		if strings.Contains(err.Error(), "Watch list limit reached") {
			return errors.New("watch list limit reached. Free tier allows maximum 10 tickers per watch list")
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

	return items, nil
}

// GetWatchListItemsWithData retrieves items with ticker data and real-time prices
func GetWatchListItemsWithData(watchListID string) ([]models.WatchListItemWithData, error) {
	query := `
		SELECT
			wli.id, wli.watch_list_id, wli.symbol, wli.notes, wli.tags,
			wli.target_buy_price, wli.target_sell_price, wli.added_at, wli.display_order,
			s.name, s.exchange, s.asset_type, s.logo_url
		FROM watch_list_items wli
		JOIN stocks s ON wli.symbol = s.symbol
		WHERE wli.watch_list_id = $1
		ORDER BY wli.display_order ASC, wli.added_at DESC
	`
	rows, err := DB.Query(query, watchListID)
	if err != nil {
		return nil, fmt.Errorf("failed to get watch list items: %w", err)
	}
	defer rows.Close()

	items := []models.WatchListItemWithData{}
	for rows.Next() {
		var item models.WatchListItemWithData
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
			&item.Name,
			&item.Exchange,
			&item.AssetType,
			&item.LogoURL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan watch list item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
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
		return errors.New("watch list item not found")
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
		return errors.New("ticker not found in watch list")
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
		return nil, errors.New("watch list item not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get watch list item: %w", err)
	}
	return item, nil
}

// UpdateItemDisplayOrder updates display order for items
func UpdateItemDisplayOrder(itemID string, displayOrder int) error {
	query := `UPDATE watch_list_items SET display_order = $1 WHERE id = $2`
	_, err := DB.Exec(query, displayOrder, itemID)
	return err
}
