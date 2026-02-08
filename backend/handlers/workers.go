package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"investorcenter-api/auth"
	"investorcenter-api/database"
)

// WorkerTask represents a task assigned to a worker
type WorkerTask struct {
	ID             string     `json:"id" db:"id"`
	Title          string     `json:"title" db:"title"`
	Description    string     `json:"description" db:"description"`
	AssignedTo     *string    `json:"assigned_to" db:"assigned_to"`
	AssignedToName *string    `json:"assigned_to_name,omitempty" db:"assigned_to_name"`
	Status         string     `json:"status" db:"status"`
	Priority       string     `json:"priority" db:"priority"`
	CreatedBy      *string    `json:"created_by" db:"created_by"`
	CreatedByName  *string    `json:"created_by_name,omitempty" db:"created_by_name"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
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

	query := `
		SELECT t.id, t.title, t.description, t.assigned_to, t.status, t.priority,
		       t.created_by, t.created_at, t.updated_at,
		       a.full_name as assigned_to_name, cr.full_name as created_by_name
		FROM worker_tasks t
		LEFT JOIN users a ON t.assigned_to = a.id
		LEFT JOIN users cr ON t.created_by = cr.id
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
		err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.AssignedTo, &t.Status, &t.Priority,
			&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt, &t.AssignedToName, &t.CreatedByName)
		if err != nil {
			log.Printf("Error scanning task: %v", err)
			continue
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
		       t.created_by, t.created_at, t.updated_at,
		       a.full_name as assigned_to_name, cr.full_name as created_by_name
		FROM worker_tasks t
		LEFT JOIN users a ON t.assigned_to = a.id
		LEFT JOIN users cr ON t.created_by = cr.id
		WHERE t.id = $1
	`

	var t WorkerTask
	err := database.DB.QueryRow(query, id).Scan(&t.ID, &t.Title, &t.Description, &t.AssignedTo, &t.Status, &t.Priority,
		&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt, &t.AssignedToName, &t.CreatedByName)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		log.Printf("Error fetching task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch task"})
		return
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
		Title       string  `json:"title" binding:"required"`
		Description string  `json:"description"`
		AssignedTo  *string `json:"assigned_to"`
		Priority    string  `json:"priority"`
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
		`INSERT INTO worker_tasks (title, description, assigned_to, priority, created_by)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, title, description, assigned_to, status, priority, created_by, created_at, updated_at`,
		req.Title, req.Description, req.AssignedTo, priority, userID,
	).Scan(&task.ID, &task.Title, &task.Description, &task.AssignedTo, &task.Status, &task.Priority,
		&task.CreatedBy, &task.CreatedAt, &task.UpdatedAt)
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
		Title       *string `json:"title"`
		Description *string `json:"description"`
		AssignedTo  *string `json:"assigned_to"`
		Status      *string `json:"status"`
		Priority    *string `json:"priority"`
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
			priority = COALESCE($6, priority)
		WHERE id = $1
		RETURNING id, title, description, assigned_to, status, priority, created_by, created_at, updated_at`,
		id, req.Title, req.Description, req.AssignedTo, req.Status, req.Priority,
	).Scan(&task.ID, &task.Title, &task.Description, &task.AssignedTo, &task.Status, &task.Priority,
		&task.CreatedBy, &task.CreatedAt, &task.UpdatedAt)
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
