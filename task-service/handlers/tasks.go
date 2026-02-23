package handlers

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"task-service/database"
)

// JSONB handles nullable JSON columns from PostgreSQL.
type JSONB json.RawMessage

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("JSONB.Scan: expected []byte or string, got %T", value)
	}
	if len(bytes) == 0 {
		*j = nil
		return nil
	}
	if !json.Valid(bytes) {
		*j = nil
		return nil
	}
	*j = JSONB(bytes)
	return nil
}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return []byte(j), nil
}

func (j JSONB) MarshalJSON() ([]byte, error) {
	if j == nil || len(j) == 0 {
		return []byte("null"), nil
	}
	if !json.Valid([]byte(j)) {
		return []byte("null"), nil
	}
	return []byte(j), nil
}

func (j *JSONB) UnmarshalJSON(data []byte) error {
	if data == nil || string(data) == "null" {
		*j = nil
		return nil
	}
	*j = JSONB(data)
	return nil
}

// Task represents a task in the queue.
type Task struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	TaskTypeID  int        `json:"task_type_id"`
	TaskType    *TaskType  `json:"task_type,omitempty"`
	Params      JSONB      `json:"params"`
	RetryCount  int        `json:"retry_count"`
	CreatedBy   *string    `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	ClaimedBy   *string    `json:"claimed_by"`
}

// taskColumns is the standard column list for task queries.
// params is cast to text to avoid PostgreSQL JSONB binary format issues with lib/pq.
const taskColumns = `id, status, priority, task_type_id, params::text, retry_count,
	created_by, created_at, updated_at, started_at, completed_at, claimed_by`

func scanTask(row interface{ Scan(dest ...interface{}) error }, t *Task) error {
	return row.Scan(
		&t.ID, &t.Status, &t.Priority, &t.TaskTypeID,
		&t.Params, &t.RetryCount, &t.CreatedBy,
		&t.CreatedAt, &t.UpdatedAt, &t.StartedAt, &t.CompletedAt, &t.ClaimedBy,
	)
}

func fetchTaskType(taskTypeID int) *TaskType {
	var tt TaskType
	err := database.DB.QueryRow(
		`SELECT id, name, skill_path, param_schema FROM task_types WHERE id = $1`,
		taskTypeID,
	).Scan(&tt.ID, &tt.Name, &tt.SkillPath, &tt.ParamSchema)
	if err != nil {
		return nil
	}
	return &tt
}

// ClaimNextTask handles POST /tasks/next
// Atomically claims the highest-priority pending task.
func ClaimNextTask(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	userID, _ := getUserID(c)

	var req struct {
		TaskType *string `json:"task_type"`
	}
	c.ShouldBindJSON(&req)

	query := `
		UPDATE tasks SET
			status = 'in_progress',
			claimed_by = $1,
			started_at = NOW()
		WHERE id = (
			SELECT id FROM tasks
			WHERE status = 'pending'
	`
	args := []interface{}{userID}

	if req.TaskType != nil && *req.TaskType != "" {
		query += ` AND task_type_id = (SELECT id FROM task_types WHERE name = $2)`
		args = append(args, *req.TaskType)
	}

	query += fmt.Sprintf(`
			ORDER BY
				CASE priority
					WHEN 'urgent' THEN 0
					WHEN 'high' THEN 1
					WHEN 'medium' THEN 2
					WHEN 'low' THEN 3
				END,
				created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING %s
	`, taskColumns)

	var t Task
	err := scanTask(database.DB.QueryRow(query, args...), &t)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNoContent, nil)
			return
		}
		log.Printf("Error claiming next task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to claim task"})
		return
	}

	t.TaskType = fetchTaskType(t.TaskTypeID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    t,
	})
}

// CreateTask handles POST /tasks
func CreateTask(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	userID, _ := getUserID(c)

	var req struct {
		TaskTypeID int             `json:"task_type_id" binding:"required"`
		Priority   string          `json:"priority"`
		Params     json.RawMessage `json:"params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}
	validPriorities := map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}
	if !validPriorities[priority] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid priority. Must be one of: low, medium, high, urgent"})
		return
	}

	query := fmt.Sprintf(
		`INSERT INTO tasks (task_type_id, priority, created_by, params)
		 VALUES ($1, $2, $3, $4)
		 RETURNING %s`, taskColumns,
	)

	var t Task
	err := scanTask(database.DB.QueryRow(query, req.TaskTypeID, priority, userID, req.Params), &t)
	if err != nil {
		log.Printf("Error creating task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    t,
	})
}

// UpdateTask handles PUT /tasks/:id
// Supports updating status (to in_progress, completed, failed, pending).
func UpdateTask(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	taskID := c.Param("id")

	var req struct {
		Status    *string `json:"status"`
		IncrRetry bool    `json:"increment_retry"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Status == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Must provide status to update"})
		return
	}

	validStatuses := map[string]bool{"pending": true, "in_progress": true, "completed": true, "failed": true}
	if !validStatuses[*req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status. Must be one of: pending, in_progress, completed, failed"})
		return
	}

	setClauses := []string{fmt.Sprintf("status = $%d", 1)}
	args := []interface{}{*req.Status}
	argIdx := 2

	switch *req.Status {
	case "in_progress":
		setClauses = append(setClauses, "started_at = NOW()")
	case "completed", "failed":
		setClauses = append(setClauses, "completed_at = NOW()")
		if req.IncrRetry {
			setClauses = append(setClauses, "retry_count = retry_count + 1")
		}
	case "pending":
		setClauses = append(setClauses, "started_at = NULL", "completed_at = NULL", "claimed_by = NULL")
		if req.IncrRetry {
			setClauses = append(setClauses, "retry_count = retry_count + 1")
		}
	}

	query := fmt.Sprintf(
		`UPDATE tasks SET %s WHERE id = $%d
		 RETURNING %s`,
		joinStrings(setClauses, ", "), argIdx, taskColumns,
	)
	args = append(args, taskID)

	var t Task
	err := scanTask(database.DB.QueryRow(query, args...), &t)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		log.Printf("Error updating task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    t,
	})
}

// DeleteTask handles DELETE /tasks/:id
func DeleteTask(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	taskID := c.Param("id")

	result, err := database.DB.Exec("DELETE FROM tasks WHERE id = $1", taskID)
	if err != nil {
		log.Printf("Error deleting task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Task deleted",
	})
}

// getUserID extracts the user ID from the JWT context.
func getUserID(c *gin.Context) (string, bool) {
	id, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	return id.(string), true
}

func joinStrings(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}
