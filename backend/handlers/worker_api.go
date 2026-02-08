package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"investorcenter-api/auth"
	"investorcenter-api/database"
)

// verifyWorker checks if the authenticated user is a worker and returns their ID
func verifyWorker(c *gin.Context) (string, bool) {
	userID, ok := auth.GetUserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return "", false
	}

	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not available"})
		return "", false
	}

	var isWorker bool
	err := database.DB.QueryRow("SELECT COALESCE(is_worker, FALSE) FROM users WHERE id = $1", userID).Scan(&isWorker)
	if err != nil || !isWorker {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized as a worker"})
		return "", false
	}

	// Update last activity
	database.DB.Exec("UPDATE users SET last_activity_at = $1 WHERE id = $2", time.Now(), userID)

	return userID, true
}

// WorkerGetMyTasks handles GET /worker/tasks
func WorkerGetMyTasks(c *gin.Context) {
	userID, ok := verifyWorker(c)
	if !ok {
		return
	}

	status := c.Query("status")

	query := `
		SELECT id, title, description, assigned_to, status, priority, created_by, created_at, updated_at
		FROM worker_tasks
		WHERE assigned_to = $1
	`
	args := []interface{}{userID}

	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}

	query += " ORDER BY CASE priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 END, created_at DESC"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		log.Printf("Error fetching worker tasks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
		return
	}
	defer rows.Close()

	tasks := []WorkerTask{}
	for rows.Next() {
		var t WorkerTask
		err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.AssignedTo, &t.Status, &t.Priority,
			&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
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

// WorkerGetTask handles GET /worker/tasks/:id
func WorkerGetTask(c *gin.Context) {
	userID, ok := verifyWorker(c)
	if !ok {
		return
	}

	taskID := c.Param("id")

	var t WorkerTask
	err := database.DB.QueryRow(
		`SELECT id, title, description, assigned_to, status, priority, created_by, created_at, updated_at
		 FROM worker_tasks
		 WHERE id = $1 AND assigned_to = $2`,
		taskID, userID,
	).Scan(&t.ID, &t.Title, &t.Description, &t.AssignedTo, &t.Status, &t.Priority,
		&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found or not assigned to you"})
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

// WorkerUpdateTaskStatus handles PUT /worker/tasks/:id/status
func WorkerUpdateTaskStatus(c *gin.Context) {
	userID, ok := verifyWorker(c)
	if !ok {
		return
	}

	taskID := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate status
	validStatuses := map[string]bool{"in_progress": true, "completed": true, "failed": true}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status. Must be one of: in_progress, completed, failed"})
		return
	}

	var t WorkerTask
	err := database.DB.QueryRow(
		`UPDATE worker_tasks SET status = $1
		 WHERE id = $2 AND assigned_to = $3
		 RETURNING id, title, description, assigned_to, status, priority, created_by, created_at, updated_at`,
		req.Status, taskID, userID,
	).Scan(&t.ID, &t.Title, &t.Description, &t.AssignedTo, &t.Status, &t.Priority,
		&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found or not assigned to you"})
			return
		}
		log.Printf("Error updating task status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    t,
	})
}

// WorkerPostUpdate handles POST /worker/tasks/:id/updates
func WorkerPostUpdate(c *gin.Context) {
	userID, ok := verifyWorker(c)
	if !ok {
		return
	}

	taskID := c.Param("id")

	// Verify task is assigned to this worker
	var exists bool
	err := database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM worker_tasks WHERE id = $1 AND assigned_to = $2)", taskID, userID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found or not assigned to you"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var update WorkerTaskUpdate
	err = database.DB.QueryRow(
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

// WorkerGetTaskUpdates handles GET /worker/tasks/:id/updates
func WorkerGetTaskUpdates(c *gin.Context) {
	userID, ok := verifyWorker(c)
	if !ok {
		return
	}

	taskID := c.Param("id")

	// Verify task is assigned to this worker
	var exists bool
	err := database.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM worker_tasks WHERE id = $1 AND assigned_to = $2)", taskID, userID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found or not assigned to you"})
		return
	}

	rows, err := database.DB.Query(
		`SELECT wtu.id, wtu.task_id, wtu.content, wtu.created_by,
		        COALESCE(u.full_name, 'Unknown') as created_by_name,
		        wtu.created_at
		 FROM worker_task_updates wtu
		 LEFT JOIN users u ON u.id = wtu.created_by
		 WHERE wtu.task_id = $1
		 ORDER BY wtu.created_at ASC`,
		taskID,
	)
	if err != nil {
		log.Printf("Error fetching task updates: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updates"})
		return
	}
	defer rows.Close()

	updates := []WorkerTaskUpdate{}
	for rows.Next() {
		var u WorkerTaskUpdate
		err := rows.Scan(&u.ID, &u.TaskID, &u.Content, &u.CreatedBy, &u.CreatedByName, &u.CreatedAt)
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

// WorkerHeartbeat handles POST /worker/heartbeat
func WorkerHeartbeat(c *gin.Context) {
	_, ok := verifyWorker(c)
	if !ok {
		return
	}
	// verifyWorker already updates last_activity_at
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Heartbeat received",
	})
}
