package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"investorcenter-api/models"
)

// Heatmap Config Operations

// CreateHeatmapConfig creates a new heatmap configuration
func CreateHeatmapConfig(config *models.HeatmapConfig) error {
	// If setting as default, unset other defaults for this watch list
	if config.IsDefault {
		_, err := DB.Exec(`
			UPDATE heatmap_configs
			SET is_default = FALSE
			WHERE user_id = $1 AND watch_list_id = $2
		`, config.UserID, config.WatchListID)
		if err != nil {
			return fmt.Errorf("failed to unset previous default: %w", err)
		}
	}

	filtersJSON, _ := json.Marshal(config.FiltersJSON)
	gradientJSON, _ := json.Marshal(config.ColorGradientJSON)

	query := `
		INSERT INTO heatmap_configs (
			user_id, watch_list_id, name, size_metric, color_metric, time_period,
			color_scheme, label_display, layout_type, filters_json, color_gradient_json, is_default
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`

	err := DB.QueryRow(
		query,
		config.UserID,
		config.WatchListID,
		config.Name,
		config.SizeMetric,
		config.ColorMetric,
		config.TimePeriod,
		config.ColorScheme,
		config.LabelDisplay,
		config.LayoutType,
		filtersJSON,
		gradientJSON,
		config.IsDefault,
	).Scan(&config.ID, &config.CreatedAt, &config.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create heatmap config: %w", err)
	}
	return nil
}

// GetHeatmapConfigsByWatchListID retrieves all configs for a watch list
func GetHeatmapConfigsByWatchListID(watchListID string, userID string) ([]models.HeatmapConfig, error) {
	query := `
		SELECT
			id, user_id, watch_list_id, name, size_metric, color_metric, time_period,
			color_scheme, label_display, layout_type, filters_json, color_gradient_json,
			is_default, created_at, updated_at
		FROM heatmap_configs
		WHERE watch_list_id = $1 AND user_id = $2
		ORDER BY is_default DESC, created_at DESC
	`

	rows, err := DB.Query(query, watchListID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get heatmap configs: %w", err)
	}
	defer rows.Close()

	configs := []models.HeatmapConfig{}
	for rows.Next() {
		var config models.HeatmapConfig
		var filtersJSON, gradientJSON []byte

		err := rows.Scan(
			&config.ID,
			&config.UserID,
			&config.WatchListID,
			&config.Name,
			&config.SizeMetric,
			&config.ColorMetric,
			&config.TimePeriod,
			&config.ColorScheme,
			&config.LabelDisplay,
			&config.LayoutType,
			&filtersJSON,
			&gradientJSON,
			&config.IsDefault,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan heatmap config: %w", err)
		}

		// Parse JSON fields
		if len(filtersJSON) > 0 {
			_ = json.Unmarshal(filtersJSON, &config.FiltersJSON)
		}
		if len(gradientJSON) > 0 {
			_ = json.Unmarshal(gradientJSON, &config.ColorGradientJSON)
		}

		configs = append(configs, config)
	}

	return configs, nil
}

// GetHeatmapConfigByID retrieves a single config by ID
func GetHeatmapConfigByID(configID string, userID string) (*models.HeatmapConfig, error) {
	query := `
		SELECT
			id, user_id, watch_list_id, name, size_metric, color_metric, time_period,
			color_scheme, label_display, layout_type, filters_json, color_gradient_json,
			is_default, created_at, updated_at
		FROM heatmap_configs
		WHERE id = $1 AND user_id = $2
	`

	var config models.HeatmapConfig
	var filtersJSON, gradientJSON []byte

	err := DB.QueryRow(query, configID, userID).Scan(
		&config.ID,
		&config.UserID,
		&config.WatchListID,
		&config.Name,
		&config.SizeMetric,
		&config.ColorMetric,
		&config.TimePeriod,
		&config.ColorScheme,
		&config.LabelDisplay,
		&config.LayoutType,
		&filtersJSON,
		&gradientJSON,
		&config.IsDefault,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("heatmap config not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get heatmap config: %w", err)
	}

	// Parse JSON fields
	if len(filtersJSON) > 0 {
		_ = json.Unmarshal(filtersJSON, &config.FiltersJSON)
	}
	if len(gradientJSON) > 0 {
		_ = json.Unmarshal(gradientJSON, &config.ColorGradientJSON)
	}

	return &config, nil
}

// GetDefaultHeatmapConfig gets the default config for a watch list
func GetDefaultHeatmapConfig(watchListID string, userID string) (*models.HeatmapConfig, error) {
	query := `
		SELECT
			id, user_id, watch_list_id, name, size_metric, color_metric, time_period,
			color_scheme, label_display, layout_type, filters_json, color_gradient_json,
			is_default, created_at, updated_at
		FROM heatmap_configs
		WHERE watch_list_id = $1 AND user_id = $2 AND is_default = TRUE
		LIMIT 1
	`

	var config models.HeatmapConfig
	var filtersJSON, gradientJSON []byte

	err := DB.QueryRow(query, watchListID, userID).Scan(
		&config.ID,
		&config.UserID,
		&config.WatchListID,
		&config.Name,
		&config.SizeMetric,
		&config.ColorMetric,
		&config.TimePeriod,
		&config.ColorScheme,
		&config.LabelDisplay,
		&config.LayoutType,
		&filtersJSON,
		&gradientJSON,
		&config.IsDefault,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// No default found, create one
		defaultConfig := &models.HeatmapConfig{
			UserID:            userID,
			WatchListID:       watchListID,
			Name:              "Default Heatmap",
			SizeMetric:        "market_cap",
			ColorMetric:       "price_change_pct",
			TimePeriod:        "1D",
			ColorScheme:       "red_green",
			LabelDisplay:      "symbol_change",
			LayoutType:        "treemap",
			FiltersJSON:       map[string]interface{}{},
			ColorGradientJSON: nil,
			IsDefault:         true,
		}
		err = CreateHeatmapConfig(defaultConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return defaultConfig, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get default heatmap config: %w", err)
	}

	// Parse JSON fields
	if len(filtersJSON) > 0 {
		_ = json.Unmarshal(filtersJSON, &config.FiltersJSON)
	}
	if len(gradientJSON) > 0 {
		_ = json.Unmarshal(gradientJSON, &config.ColorGradientJSON)
	}

	return &config, nil
}

// UpdateHeatmapConfig updates an existing config
func UpdateHeatmapConfig(config *models.HeatmapConfig) error {
	// If setting as default, unset other defaults for this watch list
	if config.IsDefault {
		_, err := DB.Exec(`
			UPDATE heatmap_configs
			SET is_default = FALSE
			WHERE user_id = $1 AND watch_list_id = $2 AND id != $3
		`, config.UserID, config.WatchListID, config.ID)
		if err != nil {
			return fmt.Errorf("failed to unset previous default: %w", err)
		}
	}

	filtersJSON, _ := json.Marshal(config.FiltersJSON)
	gradientJSON, _ := json.Marshal(config.ColorGradientJSON)

	query := `
		UPDATE heatmap_configs
		SET name = $1, size_metric = $2, color_metric = $3, time_period = $4,
		    color_scheme = $5, label_display = $6, layout_type = $7,
		    filters_json = $8, color_gradient_json = $9, is_default = $10,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $11 AND user_id = $12
	`

	result, err := DB.Exec(
		query,
		config.Name,
		config.SizeMetric,
		config.ColorMetric,
		config.TimePeriod,
		config.ColorScheme,
		config.LabelDisplay,
		config.LayoutType,
		filtersJSON,
		gradientJSON,
		config.IsDefault,
		config.ID,
		config.UserID,
	)
	if err != nil {
		return fmt.Errorf("failed to update heatmap config: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("heatmap config not found or unauthorized")
	}

	return nil
}

// DeleteHeatmapConfig deletes a config
func DeleteHeatmapConfig(configID string, userID string) error {
	query := `DELETE FROM heatmap_configs WHERE id = $1 AND user_id = $2`
	result, err := DB.Exec(query, configID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete heatmap config: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("heatmap config not found or unauthorized")
	}

	return nil
}
