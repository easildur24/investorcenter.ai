package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"investorcenter-api/auth"
	"investorcenter-api/database"
	"investorcenter-api/models"
	"investorcenter-api/services"
)

var heatmapService = services.NewHeatmapService()

// GetHeatmapData generates heatmap data for a watch list
func GetHeatmapData(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	watchListID := c.Param("id")
	configID := c.Query("config_id") // Optional

	// Check for override parameters in query
	var overrides *models.GetHeatmapDataRequest
	if c.Query("size_metric") != "" || c.Query("color_metric") != "" || c.Query("time_period") != "" {
		overrides = &models.GetHeatmapDataRequest{
			SizeMetric:  c.Query("size_metric"),
			ColorMetric: c.Query("color_metric"),
			TimePeriod:  c.Query("time_period"),
		}
	}

	// Verify ownership
	if err := heatmapService.ValidateWatchListOwnership(watchListID, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access to watch list"})
		return
	}

	// Generate heatmap data
	heatmapData, err := heatmapService.GenerateHeatmapData(watchListID, userID, configID, overrides)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, heatmapData)
}

// ListHeatmapConfigs retrieves all configs for a watch list
func ListHeatmapConfigs(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	watchListID := c.Param("id")

	// Verify ownership
	if err := heatmapService.ValidateWatchListOwnership(watchListID, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access to watch list"})
		return
	}

	configs, err := database.GetHeatmapConfigsByWatchListID(watchListID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch heatmap configs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"configs": configs})
}

// CreateHeatmapConfig saves a new heatmap configuration
func CreateHeatmapConfig(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.CreateHeatmapConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	watchListID := c.Param("id")

	// Ensure the watch list ID in the URL matches the request body
	if req.WatchListID != watchListID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Watch list ID mismatch"})
		return
	}

	// Verify ownership
	if err := heatmapService.ValidateWatchListOwnership(req.WatchListID, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access to watch list"})
		return
	}

	config := &models.HeatmapConfig{
		UserID:            userID,
		WatchListID:       req.WatchListID,
		Name:              req.Name,
		SizeMetric:        req.SizeMetric,
		ColorMetric:       req.ColorMetric,
		TimePeriod:        req.TimePeriod,
		ColorScheme:       req.ColorScheme,
		LabelDisplay:      req.LabelDisplay,
		LayoutType:        req.LayoutType,
		FiltersJSON:       req.Filters,
		ColorGradientJSON: req.ColorGradient,
		IsDefault:         req.IsDefault,
	}

	err := database.CreateHeatmapConfig(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create heatmap config"})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// UpdateHeatmapConfig updates an existing configuration
func UpdateHeatmapConfig(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	watchListID := c.Param("id")
	configID := c.Param("configId")

	var req models.UpdateHeatmapConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify ownership of watch list
	if err := heatmapService.ValidateWatchListOwnership(watchListID, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access to watch list"})
		return
	}

	// Get existing config to verify ownership
	existingConfig, err := database.GetHeatmapConfigByID(configID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Heatmap config not found"})
		return
	}

	// Verify the config belongs to the specified watch list
	if existingConfig.WatchListID != watchListID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Config does not belong to this watch list"})
		return
	}

	// Update fields
	if req.Name != "" {
		existingConfig.Name = req.Name
	}
	if req.SizeMetric != "" {
		existingConfig.SizeMetric = req.SizeMetric
	}
	if req.ColorMetric != "" {
		existingConfig.ColorMetric = req.ColorMetric
	}
	if req.TimePeriod != "" {
		existingConfig.TimePeriod = req.TimePeriod
	}
	if req.ColorScheme != "" {
		existingConfig.ColorScheme = req.ColorScheme
	}
	if req.LabelDisplay != "" {
		existingConfig.LabelDisplay = req.LabelDisplay
	}
	if req.LayoutType != "" {
		existingConfig.LayoutType = req.LayoutType
	}
	if req.Filters != nil {
		existingConfig.FiltersJSON = req.Filters
	}
	if req.ColorGradient != nil {
		existingConfig.ColorGradientJSON = req.ColorGradient
	}
	existingConfig.IsDefault = req.IsDefault

	err = database.UpdateHeatmapConfig(existingConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update heatmap config"})
		return
	}

	c.JSON(http.StatusOK, existingConfig)
}

// DeleteHeatmapConfig deletes a configuration
func DeleteHeatmapConfig(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	watchListID := c.Param("id")
	configID := c.Param("configId")

	// Verify ownership of watch list
	if err := heatmapService.ValidateWatchListOwnership(watchListID, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access to watch list"})
		return
	}

	// Get config to verify it belongs to the watch list
	config, err := database.GetHeatmapConfigByID(configID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Heatmap config not found"})
		return
	}

	if config.WatchListID != watchListID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Config does not belong to this watch list"})
		return
	}

	err = database.DeleteHeatmapConfig(configID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Heatmap config deleted successfully"})
}
