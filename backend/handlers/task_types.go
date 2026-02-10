package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"investorcenter-api/database"
)

var validTaskTypeName = regexp.MustCompile(`^[a-z0-9_]{1,100}$`)

// TaskType represents a task type definition with SOP
type TaskType struct {
	ID          int              `json:"id"`
	Name        string           `json:"name"`
	Label       string           `json:"label"`
	SOP         string           `json:"sop,omitempty"`
	ParamSchema *json.RawMessage `json:"param_schema,omitempty"`
	IsActive    bool             `json:"is_active"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// ==================== Admin Task Type Handlers ====================

// ListTaskTypes handles GET /admin/workers/task-types
func ListTaskTypes(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	query := `
		SELECT id, name, label, sop, param_schema, is_active, created_at, updated_at
		FROM task_types
		WHERE is_active = TRUE
		ORDER BY name ASC
	`

	rows, err := database.DB.Query(query)
	if err != nil {
		log.Printf("Error fetching task types: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch task types"})
		return
	}
	defer rows.Close()

	taskTypes := []TaskType{}
	for rows.Next() {
		var t TaskType
		err := rows.Scan(&t.ID, &t.Name, &t.Label, &t.SOP, &t.ParamSchema, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning task type: %v", err)
			continue
		}
		taskTypes = append(taskTypes, t)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    taskTypes,
	})
}

// CreateTaskType handles POST /admin/workers/task-types
func CreateTaskType(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	var req struct {
		Name        string           `json:"name" binding:"required"`
		Label       string           `json:"label" binding:"required"`
		SOP         string           `json:"sop"`
		ParamSchema *json.RawMessage `json:"param_schema"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !validTaskTypeName.MatchString(req.Name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name must be lowercase alphanumeric with underscores only (1-100 chars)"})
		return
	}

	var taskType TaskType
	err := database.DB.QueryRow(
		`INSERT INTO task_types (name, label, sop, param_schema)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, label, sop, param_schema, is_active, created_at, updated_at`,
		req.Name, req.Label, req.SOP, req.ParamSchema,
	).Scan(&taskType.ID, &taskType.Name, &taskType.Label, &taskType.SOP, &taskType.ParamSchema,
		&taskType.IsActive, &taskType.CreatedAt, &taskType.UpdatedAt)
	if err != nil {
		log.Printf("Error creating task type: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task type. Name may already be in use."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    taskType,
	})
}

// UpdateTaskType handles PUT /admin/workers/task-types/:id
func UpdateTaskType(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	id := c.Param("id")

	var req struct {
		Label       *string          `json:"label"`
		SOP         *string          `json:"sop"`
		ParamSchema *json.RawMessage `json:"param_schema"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var taskType TaskType
	err := database.DB.QueryRow(
		`UPDATE task_types SET
			label = COALESCE($2, label),
			sop = COALESCE($3, sop),
			param_schema = COALESCE($4, param_schema)
		WHERE id = $1 AND is_active = TRUE
		RETURNING id, name, label, sop, param_schema, is_active, created_at, updated_at`,
		id, req.Label, req.SOP, req.ParamSchema,
	).Scan(&taskType.ID, &taskType.Name, &taskType.Label, &taskType.SOP, &taskType.ParamSchema,
		&taskType.IsActive, &taskType.CreatedAt, &taskType.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task type not found"})
			return
		}
		log.Printf("Error updating task type: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task type"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    taskType,
	})
}

// DeleteTaskType handles DELETE /admin/workers/task-types/:id
func DeleteTaskType(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	id := c.Param("id")

	result, err := database.DB.Exec("UPDATE task_types SET is_active = FALSE WHERE id = $1 AND is_active = TRUE", id)
	if err != nil {
		log.Printf("Error deleting task type: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task type"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task type not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Task type deleted successfully",
	})
}

// ==================== Worker Task Type Handlers ====================

// WorkerGetTaskType handles GET /worker/task-types/:id
func WorkerGetTaskType(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	id := c.Param("id")

	query := `
		SELECT id, name, label, sop, param_schema, is_active, created_at, updated_at
		FROM task_types
		WHERE id = $1 AND is_active = TRUE
	`

	var t TaskType
	err := database.DB.QueryRow(query, id).Scan(&t.ID, &t.Name, &t.Label, &t.SOP, &t.ParamSchema,
		&t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task type not found"})
			return
		}
		log.Printf("Error fetching task type: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch task type"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    t,
	})
}
