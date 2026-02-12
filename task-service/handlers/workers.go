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
	"task-service/auth"
	"task-service/database"
)

// JSONB handles nullable JSON columns from PostgreSQL.
// It properly scans NULL values and serializes to/from JSON.
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
	// PostgreSQL JSONB binary format may have a leading version byte (0x01)
	// or null byte (0x00). Strip any leading non-JSON bytes.
	for len(bytes) > 0 && bytes[0] != '{' && bytes[0] != '[' && bytes[0] != '"' && bytes[0] != 't' && bytes[0] != 'f' && bytes[0] != 'n' && !(bytes[0] >= '0' && bytes[0] <= '9') && bytes[0] != '-' {
		bytes = bytes[1:]
	}
	if len(bytes) == 0 {
		*j = nil
		return nil
	}
	// Validate it's actually valid JSON before storing
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
	// Validate before returning to prevent broken JSON responses
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

// WorkerTask represents a task assigned to a worker
type WorkerTask struct {
	ID             string     `json:"id" db:"id"`
	Title          string     `json:"title" db:"title"`
	Description    string     `json:"description" db:"description"`
	AssignedTo     *string    `json:"assigned_to" db:"assigned_to"`
	AssignedToName *string    `json:"assigned_to_name,omitempty" db:"assigned_to_name"`
	Status         string     `json:"status" db:"status"`
	Priority       string     `json:"priority" db:"priority"`
	TaskTypeID     *int       `json:"task_type_id" db:"task_type_id"`
	TaskType       *TaskType  `json:"task_type,omitempty"`
	Params         JSONB      `json:"params" db:"params"`
	Result         JSONB      `json:"result" db:"result"`
	CreatedBy      *string    `json:"created_by" db:"created_by"`
	CreatedByName  *string    `json:"created_by_name,omitempty" db:"created_by_name"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	StartedAt      *time.Time `json:"started_at" db:"started_at"`
	CompletedAt    *time.Time `json:"completed_at" db:"completed_at"`
	RetryCount     int        `json:"retry_count" db:"retry_count"`
}

// WorkerTaskUpdate represents an update/log entry on a task
type WorkerTaskUpdate struct {
	ID            string    `json:"id" db:"id"`
	TaskID        string    `json:"task_id" db:"task_id"`
	Content       string    `json:"content" db:"content"`
	CreatedBy     *string   `json:"created_by" db:"created_by"`
	CreatedByName *string   `json:"created_by_name,omitempty" db:"created_by_name"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// WorkerInfo represents a worker for the admin list
type WorkerInfo struct {
	ID             string     `json:"id" db:"id"`
	Email          string     `json:"email" db:"email"`
	FullName       string     `json:"full_name" db:"full_name"`
	LastLoginAt    *time.Time `json:"last_login_at" db:"last_login_at"`
	LastActivityAt *time.Time `json:"last_activity_at" db:"last_activity_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	TaskCount      int        `json:"task_count" db:"task_count"`
	IsOnline       bool       `json:"is_online"`
}

// ==================== Workers ====================

// ListWorkers handles GET /admin/workers
func ListWorkers(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	query := `
		SELECT u.id, u.email, u.full_name, u.last_login_at, u.last_activity_at, u.created_at,
		       COALESCE((SELECT COUNT(*) FROM worker_tasks wt WHERE wt.assigned_to = u.id), 0) as task_count
		FROM users u
		WHERE u.is_worker = TRUE AND u.is_active = TRUE
		ORDER BY u.created_at DESC
	`

	rows, err := database.DB.Query(query)
	if err != nil {
		log.Printf("Error fetching workers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch workers"})
		return
	}
	defer rows.Close()

	now := time.Now()
	workers := []WorkerInfo{}
	for rows.Next() {
		var w WorkerInfo
		err := rows.Scan(&w.ID, &w.Email, &w.FullName, &w.LastLoginAt, &w.LastActivityAt, &w.CreatedAt, &w.TaskCount)
		if err != nil {
			log.Printf("Error scanning worker: %v", err)
			continue
		}
		// Online if last activity within 5 minutes
		if w.LastActivityAt != nil {
			w.IsOnline = now.Sub(*w.LastActivityAt) < 5*time.Minute
		}
		workers = append(workers, w)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    workers,
	})
}

// RegisterWorker handles POST /admin/workers
func RegisterWorker(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		FullName string `json:"full_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create worker"})
		return
	}

	// Insert user with is_worker=true, email_verified=true
	var worker WorkerInfo
	err = database.DB.QueryRow(
		`INSERT INTO users (email, password_hash, full_name, is_worker, email_verified)
		 VALUES ($1, $2, $3, TRUE, TRUE)
		 RETURNING id, email, full_name, last_login_at, last_activity_at, created_at`,
		req.Email, hashedPassword, req.FullName,
	).Scan(&worker.ID, &worker.Email, &worker.FullName, &worker.LastLoginAt, &worker.LastActivityAt, &worker.CreatedAt)
	if err != nil {
		log.Printf("Error creating worker: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create worker. Email may already be in use."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    worker,
	})
}

// DeleteWorker handles DELETE /admin/workers/:id
func DeleteWorker(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	id := c.Param("id")

	// Just unset is_worker flag, don't delete the user
	result, err := database.DB.Exec("UPDATE users SET is_worker = FALSE WHERE id = $1 AND is_worker = TRUE", id)
	if err != nil {
		log.Printf("Error removing worker: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove worker"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Worker removed successfully",
	})
}

// ==================== Tasks ====================

// ListTasks handles GET /admin/workers/tasks
func ListTasks(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	status := c.Query("status")
	assignedTo := c.Query("assigned_to")
	taskType := c.Query("task_type")

	query := `
		SELECT t.id, t.title, t.description, t.assigned_to, t.status, t.priority,
		       t.task_type_id, t.params, t.result, t.retry_count,
		       t.created_by, t.created_at, t.updated_at, t.started_at, t.completed_at,
		       a.full_name as assigned_to_name, cr.full_name as created_by_name,
		       tt.id, tt.name, tt.label
		FROM worker_tasks t
		LEFT JOIN users a ON t.assigned_to = a.id
		LEFT JOIN users cr ON t.created_by = cr.id
		LEFT JOIN task_types tt ON t.task_type_id = tt.id
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if status != "" {
		query += fmt.Sprintf(" AND t.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if assignedTo != "" {
		query += fmt.Sprintf(" AND t.assigned_to = $%d", argIdx)
		args = append(args, assignedTo)
		argIdx++
	}
	if taskType != "" {
		query += fmt.Sprintf(" AND tt.name = $%d", argIdx)
		args = append(args, taskType)
		argIdx++
	}

	query += " ORDER BY CASE t.priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 END, t.created_at DESC"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		log.Printf("Error fetching tasks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
		return
	}
	defer rows.Close()

	tasks := []WorkerTask{}
	for rows.Next() {
		var t WorkerTask
		var ttID *int
		var ttName, ttLabel *string
		err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.AssignedTo, &t.Status, &t.Priority,
			&t.TaskTypeID, &t.Params, &t.Result, &t.RetryCount,
			&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt, &t.StartedAt, &t.CompletedAt,
			&t.AssignedToName, &t.CreatedByName,
			&ttID, &ttName, &ttLabel)
		if err != nil {
			log.Printf("Error scanning task: %v", err)
			continue
		}
		if ttID != nil {
			t.TaskType = &TaskType{ID: *ttID, Name: *ttName, Label: *ttLabel}
		}
		tasks = append(tasks, t)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tasks,
	})
}

// GetTask handles GET /admin/workers/tasks/:id
func GetTask(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	id := c.Param("id")

	query := `
		SELECT t.id, t.title, t.description, t.assigned_to, t.status, t.priority,
		       t.task_type_id, t.params, t.result, t.retry_count,
		       t.created_by, t.created_at, t.updated_at, t.started_at, t.completed_at,
		       a.full_name as assigned_to_name, cr.full_name as created_by_name,
		       tt.id, tt.name, tt.label
		FROM worker_tasks t
		LEFT JOIN users a ON t.assigned_to = a.id
		LEFT JOIN users cr ON t.created_by = cr.id
		LEFT JOIN task_types tt ON t.task_type_id = tt.id
		WHERE t.id = $1
	`

	var t WorkerTask
	var ttID *int
	var ttName, ttLabel *string
	err := database.DB.QueryRow(query, id).Scan(&t.ID, &t.Title, &t.Description, &t.AssignedTo, &t.Status, &t.Priority,
		&t.TaskTypeID, &t.Params, &t.Result, &t.RetryCount,
		&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt, &t.StartedAt, &t.CompletedAt,
		&t.AssignedToName, &t.CreatedByName,
		&ttID, &ttName, &ttLabel)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		log.Printf("Error fetching task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch task"})
		return
	}
	if ttID != nil {
		t.TaskType = &TaskType{ID: *ttID, Name: *ttName, Label: *ttLabel}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    t,
	})
}

// CreateTask handles POST /admin/workers/tasks
func CreateTask(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	userID, _ := auth.GetUserIDFromContext(c)

	var req struct {
		Title       string          `json:"title" binding:"required"`
		Description string          `json:"description"`
		AssignedTo  *string         `json:"assigned_to"`
		Priority    string          `json:"priority"`
		TaskTypeID  *int            `json:"task_type_id"`
		Params      json.RawMessage `json:"params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}

	var task WorkerTask
	err := database.DB.QueryRow(
		`INSERT INTO worker_tasks (title, description, assigned_to, priority, created_by, task_type_id, params)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, title, description, assigned_to, status, priority, task_type_id, params, result, retry_count, created_by, created_at, updated_at, started_at, completed_at`,
		req.Title, req.Description, req.AssignedTo, priority, userID, req.TaskTypeID, req.Params,
	).Scan(&task.ID, &task.Title, &task.Description, &task.AssignedTo, &task.Status, &task.Priority,
		&task.TaskTypeID, &task.Params, &task.Result, &task.RetryCount, &task.CreatedBy, &task.CreatedAt, &task.UpdatedAt,
		&task.StartedAt, &task.CompletedAt)
	if err != nil {
		log.Printf("Error creating task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    task,
	})
}

// UpdateTask handles PUT /admin/workers/tasks/:id
func UpdateTask(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	id := c.Param("id")

	var req struct {
		Title       *string          `json:"title"`
		Description *string          `json:"description"`
		AssignedTo  *string          `json:"assigned_to"`
		Status      *string          `json:"status"`
		Priority    *string          `json:"priority"`
		TaskTypeID  *int             `json:"task_type_id"`
		Params      *json.RawMessage `json:"params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var task WorkerTask
	err := database.DB.QueryRow(
		`UPDATE worker_tasks SET
			title = COALESCE($2, title),
			description = COALESCE($3, description),
			assigned_to = COALESCE($4, assigned_to),
			status = COALESCE($5, status),
			priority = COALESCE($6, priority),
			task_type_id = COALESCE($7, task_type_id),
			params = COALESCE($8, params)
		WHERE id = $1
		RETURNING id, title, description, assigned_to, status, priority, task_type_id, params, result, retry_count, created_by, created_at, updated_at, started_at, completed_at`,
		id, req.Title, req.Description, req.AssignedTo, req.Status, req.Priority, req.TaskTypeID, req.Params,
	).Scan(&task.ID, &task.Title, &task.Description, &task.AssignedTo, &task.Status, &task.Priority,
		&task.TaskTypeID, &task.Params, &task.Result, &task.RetryCount, &task.CreatedBy, &task.CreatedAt, &task.UpdatedAt,
		&task.StartedAt, &task.CompletedAt)
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
		"data":    task,
	})
}

// DeleteTask handles DELETE /admin/workers/tasks/:id
func DeleteTask(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	id := c.Param("id")

	result, err := database.DB.Exec("DELETE FROM worker_tasks WHERE id = $1", id)
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
		"message": "Task deleted successfully",
	})
}

// ==================== Task Updates ====================

// ListTaskUpdates handles GET /admin/workers/tasks/:id/updates
func ListTaskUpdates(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	taskID := c.Param("id")

	query := `
		SELECT u.id, u.task_id, u.content, u.created_by, u.created_at,
		       usr.full_name as created_by_name
		FROM worker_task_updates u
		LEFT JOIN users usr ON u.created_by = usr.id
		WHERE u.task_id = $1
		ORDER BY u.created_at ASC
	`

	rows, err := database.DB.Query(query, taskID)
	if err != nil {
		log.Printf("Error fetching task updates: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updates"})
		return
	}
	defer rows.Close()

	updates := []WorkerTaskUpdate{}
	for rows.Next() {
		var u WorkerTaskUpdate
		err := rows.Scan(&u.ID, &u.TaskID, &u.Content, &u.CreatedBy, &u.CreatedAt, &u.CreatedByName)
		if err != nil {
			log.Printf("Error scanning update: %v", err)
			continue
		}
		updates = append(updates, u)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    updates,
	})
}

// CreateTaskUpdate handles POST /admin/workers/tasks/:id/updates
func CreateTaskUpdate(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return
	}

	taskID := c.Param("id")
	userID, _ := auth.GetUserIDFromContext(c)

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var update WorkerTaskUpdate
	err := database.DB.QueryRow(
		`INSERT INTO worker_task_updates (task_id, content, created_by)
		 VALUES ($1, $2, $3)
		 RETURNING id, task_id, content, created_by, created_at`,
		taskID, req.Content, userID,
	).Scan(&update.ID, &update.TaskID, &update.Content, &update.CreatedBy, &update.CreatedAt)
	if err != nil {
		log.Printf("Error creating task update: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create update"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    update,
	})
}

// AdminGetTaskData handles GET /admin/workers/tasks/:id/data
func AdminGetTaskData(c *gin.Context) {
	taskID := c.Param("id")
	dataType := c.Query("data_type")
	ticker := c.Query("ticker")

	limit := 100
	offset := 0
	if v := c.Query("limit"); v != "" {
		fmt.Sscanf(v, "%d", &limit)
	}
	if v := c.Query("offset"); v != "" {
		fmt.Sscanf(v, "%d", &offset)
	}

	items, total, err := database.GetTaskData(taskID, dataType, ticker, limit, offset)
	if err != nil {
		log.Printf("Error fetching task data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch task data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":  items,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}
