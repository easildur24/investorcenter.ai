package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"investorcenter-api/database"
)

// FeatureGroup represents a top-level group of features
type FeatureGroup struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Notes     string    `json:"notes" db:"notes"`
	SortOrder int       `json:"sort_order" db:"sort_order"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Feature represents a feature within a group
type Feature struct {
	ID        string    `json:"id" db:"id"`
	GroupID   string    `json:"group_id" db:"group_id"`
	Name      string    `json:"name" db:"name"`
	Notes     string    `json:"notes" db:"notes"`
	SortOrder int       `json:"sort_order" db:"sort_order"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// FeatureNote represents a note under a feature section
type FeatureNote struct {
	ID        string    `json:"id" db:"id"`
	FeatureID string    `json:"feature_id" db:"feature_id"`
	Section   string    `json:"section" db:"section"`
	Title     string    `json:"title" db:"title"`
	Content   string    `json:"content" db:"content"`
	SortOrder int       `json:"sort_order" db:"sort_order"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// FeatureWithCounts is used for the tree endpoint
type FeatureWithCounts struct {
	Feature
	NoteCounts map[string]int `json:"note_counts"`
}

// GroupWithFeatures is used for the tree endpoint
type GroupWithFeatures struct {
	FeatureGroup
	Features []FeatureWithCounts `json:"features"`
}

// ==================== Tree ====================

// GetNotesTree returns the full hierarchy for the sidebar
func GetNotesTree(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	// Get all groups
	var groups []FeatureGroup
	err := database.DB.Select(&groups, "SELECT * FROM feature_groups ORDER BY sort_order, created_at")
	if err != nil {
		log.Printf("Error fetching feature groups: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
		return
	}

	// Get all features
	var features []Feature
	err = database.DB.Select(&features, "SELECT * FROM features ORDER BY sort_order, created_at")
	if err != nil {
		log.Printf("Error fetching features: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch features"})
		return
	}

	// Get note counts per feature per section
	type NoteCount struct {
		FeatureID string `db:"feature_id"`
		Section   string `db:"section"`
		Count     int    `db:"count"`
	}
	var noteCounts []NoteCount
	err = database.DB.Select(&noteCounts, "SELECT feature_id, section, COUNT(*) as count FROM feature_notes GROUP BY feature_id, section")
	if err != nil {
		log.Printf("Error fetching note counts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch note counts"})
		return
	}

	// Build count map: feature_id -> section -> count
	countMap := make(map[string]map[string]int)
	for _, nc := range noteCounts {
		if countMap[nc.FeatureID] == nil {
			countMap[nc.FeatureID] = make(map[string]int)
		}
		countMap[nc.FeatureID][nc.Section] = nc.Count
	}

	// Build feature map: group_id -> []FeatureWithCounts
	featureMap := make(map[string][]FeatureWithCounts)
	for _, f := range features {
		counts := map[string]int{"ui": 0, "backend": 0, "data": 0, "infra": 0}
		if fc, ok := countMap[f.ID]; ok {
			for section, count := range fc {
				counts[section] = count
			}
		}
		featureMap[f.GroupID] = append(featureMap[f.GroupID], FeatureWithCounts{
			Feature:    f,
			NoteCounts: counts,
		})
	}

	// Assemble tree
	result := make([]GroupWithFeatures, len(groups))
	for i, g := range groups {
		feats := featureMap[g.ID]
		if feats == nil {
			feats = []FeatureWithCounts{}
		}
		result[i] = GroupWithFeatures{
			FeatureGroup: g,
			Features:     feats,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// ==================== Groups ====================

// ListFeatureGroups handles GET /admin/notes/groups
func ListFeatureGroups(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	var groups []FeatureGroup
	err := database.DB.Select(&groups, "SELECT * FROM feature_groups ORDER BY sort_order, created_at")
	if err != nil {
		log.Printf("Error fetching feature groups: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch groups"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    groups,
	})
}

// CreateFeatureGroup handles POST /admin/notes/groups
func CreateFeatureGroup(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	var req struct {
		Name  string `json:"name" binding:"required,max=255"`
		Notes string `json:"notes" binding:"max=10000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var group FeatureGroup
	err := database.DB.QueryRowx(
		`INSERT INTO feature_groups (name, notes) VALUES ($1, $2) RETURNING *`,
		req.Name, req.Notes,
	).StructScan(&group)
	if err != nil {
		log.Printf("Error creating feature group: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    group,
	})
}

// UpdateFeatureGroup handles PUT /admin/notes/groups/:id
func UpdateFeatureGroup(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	id := c.Param("id")

	var req struct {
		Name      *string `json:"name" binding:"omitempty,max=255"`
		Notes     *string `json:"notes" binding:"omitempty,max=10000"`
		SortOrder *int    `json:"sort_order" binding:"omitempty,min=0,max=10000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var group FeatureGroup
	err := database.DB.QueryRowx(
		`UPDATE feature_groups SET
			name = COALESCE($2, name),
			notes = COALESCE($3, notes),
			sort_order = COALESCE($4, sort_order)
		WHERE id = $1 RETURNING *`,
		id, req.Name, req.Notes, req.SortOrder,
	).StructScan(&group)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
			return
		}
		log.Printf("Error updating feature group: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    group,
	})
}

// DeleteFeatureGroup handles DELETE /admin/notes/groups/:id
func DeleteFeatureGroup(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	id := c.Param("id")

	result, err := database.DB.Exec("DELETE FROM feature_groups WHERE id = $1", id)
	if err != nil {
		log.Printf("Error deleting feature group: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete group"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Group deleted successfully",
	})
}

// ==================== Features ====================

// ListFeatures handles GET /admin/notes/groups/:groupId/features
func ListFeatures(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	groupID := c.Param("groupId")

	var features []Feature
	err := database.DB.Select(&features,
		"SELECT * FROM features WHERE group_id = $1 ORDER BY sort_order, created_at",
		groupID,
	)
	if err != nil {
		log.Printf("Error fetching features: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch features"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    features,
	})
}

// CreateFeature handles POST /admin/notes/groups/:groupId/features
func CreateFeature(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	groupID := c.Param("groupId")

	var req struct {
		Name  string `json:"name" binding:"required,max=255"`
		Notes string `json:"notes" binding:"max=10000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var feature Feature
	err := database.DB.QueryRowx(
		`INSERT INTO features (group_id, name, notes) VALUES ($1, $2, $3) RETURNING *`,
		groupID, req.Name, req.Notes,
	).StructScan(&feature)
	if err != nil {
		log.Printf("Error creating feature: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create feature"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    feature,
	})
}

// UpdateFeature handles PUT /admin/notes/features/:id
func UpdateFeature(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	id := c.Param("id")

	var req struct {
		Name      *string `json:"name" binding:"omitempty,max=255"`
		Notes     *string `json:"notes" binding:"omitempty,max=10000"`
		SortOrder *int    `json:"sort_order" binding:"omitempty,min=0,max=10000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var feature Feature
	err := database.DB.QueryRowx(
		`UPDATE features SET
			name = COALESCE($2, name),
			notes = COALESCE($3, notes),
			sort_order = COALESCE($4, sort_order)
		WHERE id = $1 RETURNING *`,
		id, req.Name, req.Notes, req.SortOrder,
	).StructScan(&feature)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Feature not found"})
			return
		}
		log.Printf("Error updating feature: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update feature"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    feature,
	})
}

// DeleteFeature handles DELETE /admin/notes/features/:id
func DeleteFeature(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	id := c.Param("id")

	result, err := database.DB.Exec("DELETE FROM features WHERE id = $1", id)
	if err != nil {
		log.Printf("Error deleting feature: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete feature"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Feature not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Feature deleted successfully",
	})
}

// ==================== Notes ====================

// ListFeatureNotes handles GET /admin/notes/features/:featureId/notes
func ListFeatureNotes(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	featureID := c.Param("featureId")
	section := c.Query("section")

	var notes []FeatureNote
	var err error

	if section != "" {
		err = database.DB.Select(&notes,
			"SELECT * FROM feature_notes WHERE feature_id = $1 AND section = $2 ORDER BY sort_order, created_at",
			featureID, section,
		)
	} else {
		err = database.DB.Select(&notes,
			"SELECT * FROM feature_notes WHERE feature_id = $1 ORDER BY section, sort_order, created_at",
			featureID,
		)
	}

	if err != nil {
		log.Printf("Error fetching feature notes: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    notes,
	})
}

// CreateFeatureNote handles POST /admin/notes/features/:featureId/notes
func CreateFeatureNote(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	featureID := c.Param("featureId")

	var req struct {
		Section string `json:"section" binding:"required,max=50"`
		Title   string `json:"title" binding:"max=500"`
		Content string `json:"content" binding:"max=50000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate section
	validSections := map[string]bool{"ui": true, "backend": true, "data": true, "infra": true}
	if !validSections[req.Section] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid section. Must be one of: ui, backend, data, infra"})
		return
	}

	title := req.Title
	if title == "" {
		title = "Untitled"
	}

	var note FeatureNote
	err := database.DB.QueryRowx(
		`INSERT INTO feature_notes (feature_id, section, title, content) VALUES ($1, $2, $3, $4) RETURNING *`,
		featureID, req.Section, title, req.Content,
	).StructScan(&note)
	if err != nil {
		log.Printf("Error creating feature note: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create note"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    note,
	})
}

// UpdateFeatureNote handles PUT /admin/notes/notes/:id
func UpdateFeatureNote(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	id := c.Param("id")

	var req struct {
		Title     *string `json:"title" binding:"omitempty,max=500"`
		Content   *string `json:"content" binding:"omitempty,max=50000"`
		SortOrder *int    `json:"sort_order" binding:"omitempty,min=0,max=10000"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var note FeatureNote
	err := database.DB.QueryRowx(
		`UPDATE feature_notes SET
			title = COALESCE($2, title),
			content = COALESCE($3, content),
			sort_order = COALESCE($4, sort_order)
		WHERE id = $1 RETURNING *`,
		id, req.Title, req.Content, req.SortOrder,
	).StructScan(&note)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
			return
		}
		log.Printf("Error updating feature note: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update note"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    note,
	})
}

// DeleteFeatureNote handles DELETE /admin/notes/notes/:id
func DeleteFeatureNote(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	id := c.Param("id")

	result, err := database.DB.Exec("DELETE FROM feature_notes WHERE id = $1", id)
	if err != nil {
		log.Printf("Error deleting feature note: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete note"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Note not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Note deleted successfully",
	})
}
