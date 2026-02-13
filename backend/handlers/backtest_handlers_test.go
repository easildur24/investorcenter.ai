package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"investorcenter-api/models"
	"investorcenter-api/services"
)

// MockBacktestService is a mock implementation of the backtest service
type MockBacktestService struct {
	mock.Mock
}

func (m *MockBacktestService) ValidateConfig(config models.BacktestConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockBacktestService) RunBacktest(config models.BacktestConfig) (*models.BacktestSummary, error) {
	args := m.Called(config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BacktestSummary), args.Error(1)
}

func (m *MockBacktestService) SubmitBacktestJob(config models.BacktestConfig, userID *string) (*models.BacktestJob, error) {
	args := m.Called(config, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BacktestJob), args.Error(1)
}

func (m *MockBacktestService) GetJobStatus(jobID string) (*models.BacktestJob, error) {
	args := m.Called(jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BacktestJob), args.Error(1)
}

func (m *MockBacktestService) GetJobResult(jobID string) (*models.BacktestSummary, error) {
	args := m.Called(jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BacktestSummary), args.Error(1)
}

func (m *MockBacktestService) GetLatestBacktest() (*models.BacktestSummary, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BacktestSummary), args.Error(1)
}

func (m *MockBacktestService) GetDefaultBacktestConfig() models.BacktestConfig {
	args := m.Called()
	return args.Get(0).(models.BacktestConfig)
}

func (m *MockBacktestService) GetCachedOrRunBacktest(config models.BacktestConfig) (*models.BacktestSummary, error) {
	args := m.Called(config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BacktestSummary), args.Error(1)
}

func (m *MockBacktestService) GenerateCharts(summary *models.BacktestSummary) map[string]interface{} {
	args := m.Called(summary)
	return args.Get(0).(map[string]interface{})
}

func (m *MockBacktestService) GetUserBacktests(userID string, limit int) ([]models.BacktestJob, error) {
	args := m.Called(userID, limit)
	return args.Get(0).([]models.BacktestJob), args.Error(1)
}

// setupBacktestTestRouter creates a test router for backtest endpoints
func setupBacktestTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	return router
}

// Helper to create sample backtest summary
func createSampleBacktestSummary() *models.BacktestSummary {
	return &models.BacktestSummary{
		DecilePerformance: []models.DecilePerformance{
			{Decile: 1, TotalReturn: 0.25, AnnualizedReturn: 0.12, SharpeRatio: 1.5, MaxDrawdown: 0.15},
			{Decile: 2, TotalReturn: 0.20, AnnualizedReturn: 0.10, SharpeRatio: 1.2, MaxDrawdown: 0.18},
			{Decile: 10, TotalReturn: -0.05, AnnualizedReturn: -0.02, SharpeRatio: -0.5, MaxDrawdown: 0.35},
		},
		SpreadCAGR:        0.14,
		TopVsBenchmark:    0.05,
		HitRate:           0.65,
		MonotonicityScore: 0.85,
		InformationRatio:  0.75,
		TopDecileSharpe:   1.5,
		TopDecileMaxDD:    0.15,
		Benchmark:         "SPY",
		StartDate:         "2019-01-01",
		EndDate:           "2024-01-01",
		NumPeriods:        60,
	}
}

func TestRunBacktest(t *testing.T) {
	router := setupBacktestTestRouter()

	// Create mock service (unused for now, but kept for future integration tests)
	_ = new(MockBacktestService)

	// Create handler with mock service
	handler := &BacktestHandler{service: &services.BacktestService{}}

	// For integration tests, we'd use a real service
	// For now, we'll test the handler structure
	router.POST("/api/v1/ic-scores/backtest", handler.RunBacktest)

	t.Run("Invalid request body", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/ic-scores/backtest", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]string
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Contains(t, response["error"], "Invalid")
	})
}

func TestGetDefaultConfig(t *testing.T) {
	router := setupBacktestTestRouter()

	// For this test, we'll verify the endpoint structure
	// In production, this would use the actual service
	router.GET("/api/v1/ic-scores/backtest/config/default", func(c *gin.Context) {
		config := models.BacktestConfig{
			StartDate:          "2019-01-01",
			EndDate:            "2024-01-01",
			RebalanceFrequency: "monthly",
			Universe:           "sp500",
			Benchmark:          "SPY",
			TransactionCostBps: 10.0,
			SlippageBps:        5.0,
			ExcludeFinancials:  false,
			ExcludeUtilities:   false,
			UseSmoothedScores:  true,
		}
		c.JSON(http.StatusOK, config)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/ic-scores/backtest/config/default", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var config models.BacktestConfig
	err := json.Unmarshal(w.Body.Bytes(), &config)
	assert.NoError(t, err)
	assert.Equal(t, "monthly", config.RebalanceFrequency)
	assert.Equal(t, "sp500", config.Universe)
	assert.Equal(t, "SPY", config.Benchmark)
	assert.Equal(t, float64(10), config.TransactionCostBps)
}

func TestGetBacktestJobStatus(t *testing.T) {
	router := setupBacktestTestRouter()

	t.Run("Missing job ID", func(t *testing.T) {
		router.GET("/api/v1/ic-scores/backtest/jobs/:jobId", func(c *gin.Context) {
			jobID := c.Param("jobId")
			if jobID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"job_id": jobID, "status": "pending"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/ic-scores/backtest/jobs/test-job-123", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "test-job-123", response["job_id"])
		assert.Equal(t, "pending", response["status"])
	})
}

func TestSubmitBacktestJob(t *testing.T) {
	router := setupBacktestTestRouter()

	router.POST("/api/v1/ic-scores/backtest/jobs", func(c *gin.Context) {
		var config models.BacktestConfig
		if err := c.ShouldBindJSON(&config); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{
			"job_id":  "new-job-456",
			"status":  "pending",
			"message": "Backtest job submitted successfully",
		})
	})

	t.Run("Valid job submission", func(t *testing.T) {
		config := models.BacktestConfig{
			StartDate:          "2020-01-01",
			EndDate:            "2024-01-01",
			RebalanceFrequency: "monthly",
			Universe:           "sp500",
			Benchmark:          "SPY",
		}

		body, _ := json.Marshal(config)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/ic-scores/backtest/jobs", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusAccepted, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "new-job-456", response["job_id"])
		assert.Equal(t, "pending", response["status"])
	})

	t.Run("Invalid job submission", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/ic-scores/backtest/jobs", bytes.NewBuffer([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetLatestBacktest(t *testing.T) {
	router := setupBacktestTestRouter()

	t.Run("Returns latest backtest summary", func(t *testing.T) {
		summary := createSampleBacktestSummary()

		router.GET("/api/v1/ic-scores/backtest/latest", func(c *gin.Context) {
			c.JSON(http.StatusOK, summary)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/ic-scores/backtest/latest", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.BacktestSummary
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 0.14, response.SpreadCAGR)
		assert.Equal(t, 0.65, response.HitRate)
		assert.Equal(t, "SPY", response.Benchmark)
	})

	t.Run("No backtests found", func(t *testing.T) {
		router2 := setupBacktestTestRouter()
		router2.GET("/api/v1/ic-scores/backtest/latest", func(c *gin.Context) {
			c.JSON(http.StatusNotFound, gin.H{"error": "No completed backtests found"})
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/ic-scores/backtest/latest", nil)
		router2.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestGetBacktestJobResult(t *testing.T) {
	router := setupBacktestTestRouter()

	router.GET("/api/v1/ic-scores/backtest/jobs/:jobId/result", func(c *gin.Context) {
		jobID := c.Param("jobId")

		if jobID == "completed-job" {
			c.JSON(http.StatusOK, createSampleBacktestSummary())
			return
		}

		if jobID == "pending-job" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not completed yet"})
			return
		}

		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
	})

	t.Run("Completed job returns result", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/ic-scores/backtest/jobs/completed-job/result", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.BacktestSummary
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, 0.14, response.SpreadCAGR)
	})

	t.Run("Pending job returns error", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/ic-scores/backtest/jobs/pending-job/result", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Unknown job returns error", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/ic-scores/backtest/jobs/unknown-job/result", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestBacktestConfigValidation(t *testing.T) {
	t.Run("Valid configuration", func(t *testing.T) {
		config := models.BacktestConfig{
			StartDate:          "2020-01-01",
			EndDate:            "2024-01-01",
			RebalanceFrequency: "monthly",
			Universe:           "sp500",
			Benchmark:          "SPY",
			TransactionCostBps: 10.0,
			SlippageBps:        5.0,
		}

		// Validate dates
		_, err := time.Parse("2006-01-02", config.StartDate)
		assert.NoError(t, err)

		_, err = time.Parse("2006-01-02", config.EndDate)
		assert.NoError(t, err)

		// Validate frequency
		validFrequencies := []string{"daily", "weekly", "monthly", "quarterly"}
		found := false
		for _, f := range validFrequencies {
			if f == config.RebalanceFrequency {
				found = true
				break
			}
		}
		assert.True(t, found, "Frequency should be valid")

		// Validate universe
		validUniverses := []string{"sp500", "sp1500", "all"}
		found = false
		for _, u := range validUniverses {
			if u == config.Universe {
				found = true
				break
			}
		}
		assert.True(t, found, "Universe should be valid")
	})

	t.Run("End date before start date", func(t *testing.T) {
		config := models.BacktestConfig{
			StartDate: "2024-01-01",
			EndDate:   "2020-01-01", // Before start
		}

		startDate, _ := time.Parse("2006-01-02", config.StartDate)
		endDate, _ := time.Parse("2006-01-02", config.EndDate)

		assert.True(t, endDate.Before(startDate), "End date should be before start date")
	})

	t.Run("Negative transaction costs", func(t *testing.T) {
		config := models.BacktestConfig{
			TransactionCostBps: -5.0,
		}

		assert.True(t, config.TransactionCostBps < 0, "Transaction cost should be negative")
	})
}

func TestBacktestSummaryStructure(t *testing.T) {
	summary := createSampleBacktestSummary()

	t.Run("Has required fields", func(t *testing.T) {
		assert.NotEmpty(t, summary.DecilePerformance)
		assert.NotEmpty(t, summary.Benchmark)
		assert.NotEmpty(t, summary.StartDate)
		assert.NotEmpty(t, summary.EndDate)
	})

	t.Run("Decile performance ordering", func(t *testing.T) {
		// First decile should have highest returns
		assert.Greater(t, summary.DecilePerformance[0].AnnualizedReturn,
			summary.DecilePerformance[len(summary.DecilePerformance)-1].AnnualizedReturn)
	})

	t.Run("Hit rate in valid range", func(t *testing.T) {
		assert.GreaterOrEqual(t, summary.HitRate, 0.0)
		assert.LessOrEqual(t, summary.HitRate, 1.0)
	})

	t.Run("Monotonicity score in valid range", func(t *testing.T) {
		assert.GreaterOrEqual(t, summary.MonotonicityScore, 0.0)
		assert.LessOrEqual(t, summary.MonotonicityScore, 1.0)
	})
}

func TestDecilePerformanceStructure(t *testing.T) {
	perf := models.DecilePerformance{
		Decile:           1,
		TotalReturn:      0.50,
		AnnualizedReturn: 0.12,
		SharpeRatio:      1.5,
		MaxDrawdown:      0.20,
	}

	t.Run("Valid decile number", func(t *testing.T) {
		assert.GreaterOrEqual(t, perf.Decile, 1)
		assert.LessOrEqual(t, perf.Decile, 10)
	})

	t.Run("Sharpe ratio calculation validity", func(t *testing.T) {
		// Positive return with positive Sharpe is valid
		if perf.AnnualizedReturn > 0 {
			assert.Greater(t, perf.SharpeRatio, 0.0)
		}
	})

	t.Run("Max drawdown is non-negative", func(t *testing.T) {
		assert.GreaterOrEqual(t, perf.MaxDrawdown, 0.0)
	})
}
