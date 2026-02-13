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

var dataIngestionProxy *httputil.ReverseProxy

func init() {
	dataIngestionURL := os.Getenv("DATA_INGESTION_SERVICE_URL")
	if dataIngestionURL == "" {
		dataIngestionURL = "http://localhost:8002"
	}

	target, err := url.Parse(dataIngestionURL)
	if err != nil {
		log.Printf("Warning: invalid DATA_INGESTION_SERVICE_URL %q: %v", dataIngestionURL, err)
		return
	}

	dataIngestionProxy = httputil.NewSingleHostReverseProxy(target)

	originalDirector := dataIngestionProxy.Director
	dataIngestionProxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Strip /api/v1 prefix — data ingestion service routes start at /ingest
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/api/v1")
		req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, "/api/v1")
	}
}

// DataIngestionProxy returns a Gin handler that proxies requests to the data ingestion service.
func DataIngestionProxy() gin.HandlerFunc {
	return func(c *gin.Context) {
		if dataIngestionProxy == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Data ingestion service not configured"})
			c.Abort()
			return
		}
		dataIngestionProxy.ServeHTTP(c.Writer, c.Request)
	}
}
