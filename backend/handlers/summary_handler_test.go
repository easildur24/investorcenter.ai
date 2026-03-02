package handlers

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetMarketSummary_HandlerExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Should not panic — proves the handler has the correct gin.HandlerFunc signature
	r.GET("/api/v1/markets/summary", GetMarketSummary)
}
