package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"task-service/database"
)

var validTaskTypeName = regexp.MustCompile(`^[a-z0-9_]{1,100}$`)
var validSkillPath = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,199}$`)

// TaskType represents a task type that maps to a skill.
type TaskType struct {
	ID          int              `json:"id"`
	Name        string           `json:"name"`
	SkillPath   *string          `json:"skill_path,omitempty"`
	ParamSchema *json.RawMessage `json:"param_schema,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// ListTaskTypes handles GET /task-types
func ListTaskTypes(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	rows, err := database.DB.Query(`
		SELECT id, name, skill_path, param_schema, created_at, updated_at
		FROM task_types
		ORDER BY name ASC
	`)
	if err != nil {
		log.Printf("Error fetching task types: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch task types"})
		return
	}
	defer rows.Close()

	taskTypes := []TaskType{}
	for rows.Next() {
		var t TaskType
		err := rows.Scan(&t.ID, &t.Name, &t.SkillPath, &t.ParamSchema, &t.CreatedAt, &t.UpdatedAt)
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

// CreateTaskType handles POST /task-types
func CreateTaskType(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	var req struct {
		Name        string           `json:"name" binding:"required"`
		SkillPath   *string          `json:"skill_path"`
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

	if req.SkillPath != nil && *req.SkillPath != "" && !validSkillPath.MatchString(*req.SkillPath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Skill path must be lowercase alphanumeric with hyphens (1-200 chars)"})
		return
	}

	var taskType TaskType
	err := database.DB.QueryRow(
		`INSERT INTO task_types (name, skill_path, param_schema)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, skill_path, param_schema, created_at, updated_at`,
		req.Name, req.SkillPath, req.ParamSchema,
	).Scan(&taskType.ID, &taskType.Name, &taskType.SkillPath, &taskType.ParamSchema,
		&taskType.CreatedAt, &taskType.UpdatedAt)
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

// UpdateTaskType handles PUT /task-types/:id
func UpdateTaskType(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	id := c.Param("id")

	var req struct {
		SkillPath   *string          `json:"skill_path"`
		ParamSchema *json.RawMessage `json:"param_schema"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.SkillPath != nil && *req.SkillPath != "" && !validSkillPath.MatchString(*req.SkillPath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Skill path must be lowercase alphanumeric with hyphens (1-200 chars)"})
		return
	}

	var taskType TaskType
	err := database.DB.QueryRow(
		`UPDATE task_types SET
			skill_path = COALESCE($2, skill_path),
			param_schema = COALESCE($3, param_schema)
		WHERE id = $1
		RETURNING id, name, skill_path, param_schema, created_at, updated_at`,
		id, req.SkillPath, req.ParamSchema,
	).Scan(&taskType.ID, &taskType.Name, &taskType.SkillPath, &taskType.ParamSchema,
		&taskType.CreatedAt, &taskType.UpdatedAt)
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

// DeleteTaskType handles DELETE /task-types/:id
func DeleteTaskType(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	id := c.Param("id")

	result, err := database.DB.Exec("DELETE FROM task_types WHERE id = $1", id)
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
		"message": "Task type deleted",
	})
}
