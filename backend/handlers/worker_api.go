package handlers

import (
	"database/sql"
	"encoding/json"
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
		SELECT t.id, t.title, t.description, t.assigned_to, t.status, t.priority,
		       t.task_type_id, t.params, t.result,
		       t.created_by, t.created_at, t.updated_at, t.started_at, t.completed_at,
		       tt.id, tt.name, tt.label, tt.sop
		FROM worker_tasks t
		LEFT JOIN task_types tt ON t.task_type_id = tt.id
		WHERE t.assigned_to = $1
	`
	args := []interface{}{userID}

	if status != "" {
		query += " AND t.status = $2"
		args = append(args, status)
	}

	query += " ORDER BY CASE t.priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 END, t.created_at DESC"

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
		var ttID *int
		var ttName, ttLabel, ttSOP *string
		err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.AssignedTo, &t.Status, &t.Priority,
			&t.TaskTypeID, &t.Params, &t.Result,
			&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt, &t.StartedAt, &t.CompletedAt,
			&ttID, &ttName, &ttLabel, &ttSOP)
		if err != nil {
			log.Printf("Error scanning task: %v", err)
			continue
		}
		if ttID != nil {
			tt := &TaskType{ID: *ttID, Name: *ttName, Label: *ttLabel}
			if ttSOP != nil {
				tt.SOP = *ttSOP
			}
			t.TaskType = tt
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
	var ttID *int
	var ttName, ttLabel, ttSOP *string
	err := database.DB.QueryRow(
		`SELECT t.id, t.title, t.description, t.assigned_to, t.status, t.priority,
		        t.task_type_id, t.params, t.result,
		        t.created_by, t.created_at, t.updated_at, t.started_at, t.completed_at,
		        tt.id, tt.name, tt.label, tt.sop
		 FROM worker_tasks t
		 LEFT JOIN task_types tt ON t.task_type_id = tt.id
		 WHERE t.id = $1 AND t.assigned_to = $2`,
		taskID, userID,
	).Scan(&t.ID, &t.Title, &t.Description, &t.AssignedTo, &t.Status, &t.Priority,
		&t.TaskTypeID, &t.Params, &t.Result,
		&t.CreatedBy, &t.CreatedAt, &t.UpdatedAt, &t.StartedAt, &t.CompletedAt,
		&ttID, &ttName, &ttLabel, &ttSOP)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found or not assigned to you"})
			return
		}
		log.Printf("Error fetching task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch task"})
		return
	}
	if ttID != nil {
		tt := &TaskType{ID: *ttID, Name: *ttName, Label: *ttLabel}
		if ttSOP != nil {
			tt.SOP = *ttSOP
		}
		t.TaskType = tt
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

	// Build query with timestamp updates based on status
	var query string
	switch req.Status {
	case "in_progress":
		query = `UPDATE worker_tasks SET status = $1, started_at = NOW()
			 WHERE id = $2 AND assigned_to = $3
			 RETURNING id, title, description, assigned_to, status, priority, task_type_id, params, result, created_by, created_at, updated_at, started_at, completed_at`
	case "completed", "failed":
		query = `UPDATE worker_tasks SET status = $1, completed_at = NOW()
			 WHERE id = $2 AND assigned_to = $3
			 RETURNING id, title, description, assigned_to, status, priority, task_type_id, params, result, created_by, created_at, updated_at, started_at, completed_at`
	default:
		query = `UPDATE worker_tasks SET status = $1
			 WHERE id = $2 AND assigned_to = $3
			 RETURNING id, title, description, assigned_to, status, priority, task_type_id, params, result, created_by, created_at, updated_at, started_at, completed_at`
	}

	var t WorkerTask
	err := database.DB.QueryRow(query, req.Status, taskID, userID).Scan(&t.ID, &t.Title, &t.Description, &t.AssignedTo, &t.Status, &t.Priority,
		&t.TaskTypeID, &t.Params, &t.Result, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
		&t.StartedAt, &t.CompletedAt)
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

// WorkerPostResult handles POST /worker/tasks/:id/result
func WorkerPostResult(c *gin.Context) {
	userID, ok := verifyWorker(c)
	if !ok {
		return
	}

	taskID := c.Param("id")

	var req struct {
		Result json.RawMessage `json:"result" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify task exists, is assigned to this worker, and is in_progress
	var currentStatus string
	err := database.DB.QueryRow(
		"SELECT status FROM worker_tasks WHERE id = $1 AND assigned_to = $2",
		taskID, userID,
	).Scan(&currentStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found or not assigned to you"})
			return
		}
		log.Printf("Error checking task status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check task status"})
		return
	}
	if currentStatus != "in_progress" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can only post results to tasks with status 'in_progress'"})
		return
	}

	var t WorkerTask
	err = database.DB.QueryRow(
		`UPDATE worker_tasks SET result = $1
		 WHERE id = $2 AND assigned_to = $3
		 RETURNING id, title, description, assigned_to, status, priority, task_type_id, params, result, created_by, created_at, updated_at, started_at, completed_at`,
		req.Result, taskID, userID,
	).Scan(&t.ID, &t.Title, &t.Description, &t.AssignedTo, &t.Status, &t.Priority,
		&t.TaskTypeID, &t.Params, &t.Result, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
		&t.StartedAt, &t.CompletedAt)
	if err != nil {
		log.Printf("Error posting task result: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save result"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    t,
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
