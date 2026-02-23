package services

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

var taskServiceProxy *httputil.ReverseProxy

func init() {
	taskServiceURL := os.Getenv("TASK_SERVICE_URL")
	if taskServiceURL == "" {
		taskServiceURL = "http://localhost:8001"
	}

	target, err := url.Parse(taskServiceURL)
	if err != nil {
		log.Printf("Warning: invalid TASK_SERVICE_URL %q: %v", taskServiceURL, err)
		return
	}

	taskServiceProxy = httputil.NewSingleHostReverseProxy(target)

	originalDirector := taskServiceProxy.Director
	taskServiceProxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Strip /api/v1 prefix — task service routes start at /tasks or /task-types
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/api/v1")
		req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, "/api/v1")
	}
}

// TaskServiceProxy returns a Gin handler that proxies requests to the task service.
func TaskServiceProxy() gin.HandlerFunc {
	return func(c *gin.Context) {
		if taskServiceProxy == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Task service not configured"})
			c.Abort()
			return
		}
		taskServiceProxy.ServeHTTP(c.Writer, c.Request)
	}
}
