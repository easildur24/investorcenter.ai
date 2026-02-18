package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"investorcenter-api/models"
)

// MockCronjobService satisfies the CronjobServicer interface for testing.
type MockCronjobService struct {
	mock.Mock
}

func (m *MockCronjobService) GetOverview() (*models.CronjobOverviewResponse, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CronjobOverviewResponse), args.Error(1)
}

func (m *MockCronjobService) GetJobHistory(jobName string, limit, offset int) (*models.CronjobHistoryResponse, error) {
	args := m.Called(jobName, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CronjobHistoryResponse), args.Error(1)
}

func (m *MockCronjobService) GetJobDetails(executionID string) (*models.CronjobExecutionLog, error) {
	args := m.Called(executionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CronjobExecutionLog), args.Error(1)
}

func (m *MockCronjobService) GetMetrics(period int) (*models.CronjobMetricsResponse, error) {
	args := m.Called(period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CronjobMetricsResponse), args.Error(1)
}

func (m *MockCronjobService) GetAllSchedules() ([]models.CronjobSchedule, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.CronjobSchedule), args.Error(1)
}

func setupCronjobRouter(handler *CronjobHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/cronjobs/overview", handler.GetOverview)
	r.GET("/cronjobs/:jobName/history", handler.GetJobHistory)
	r.GET("/cronjobs/details/:executionId", handler.GetJobDetails)
	r.GET("/cronjobs/metrics", handler.GetMetrics)
	r.GET("/cronjobs/schedules", handler.GetAllSchedules)
	return r
}

func TestCronjobHandler_GetOverview_Success(t *testing.T) {
	mockSvc := new(MockCronjobService)
	handler := NewCronjobHandler(mockSvc)
	router := setupCronjobRouter(handler)

	expected := &models.CronjobOverviewResponse{
		Summary: models.CronjobSummary{
			TotalJobs:  10,
			ActiveJobs: 8,
		},
		Jobs: []models.CronjobStatusWithInfo{},
	}
	mockSvc.On("GetOverview").Return(expected, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/cronjobs/overview", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp models.CronjobOverviewResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 10, resp.Summary.TotalJobs)
	mockSvc.AssertExpectations(t)
}

func TestCronjobHandler_GetOverview_Error(t *testing.T) {
	mockSvc := new(MockCronjobService)
	handler := NewCronjobHandler(mockSvc)
	router := setupCronjobRouter(handler)

	mockSvc.On("GetOverview").Return(nil, errors.New("db error"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/cronjobs/overview", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestCronjobHandler_GetJobHistory_Success(t *testing.T) {
	mockSvc := new(MockCronjobService)
	handler := NewCronjobHandler(mockSvc)
	router := setupCronjobRouter(handler)

	expected := &models.CronjobHistoryResponse{
		JobName:         "daily-price-update",
		TotalExecutions: 100,
		Executions:      []models.CronjobExecutionLog{},
	}
	mockSvc.On("GetJobHistory", "daily-price-update", 50, 0).Return(expected, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/cronjobs/daily-price-update/history", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp models.CronjobHistoryResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "daily-price-update", resp.JobName)
	mockSvc.AssertExpectations(t)
}

func TestCronjobHandler_GetJobHistory_CustomPagination(t *testing.T) {
	mockSvc := new(MockCronjobService)
	handler := NewCronjobHandler(mockSvc)
	router := setupCronjobRouter(handler)

	expected := &models.CronjobHistoryResponse{
		JobName:         "sec-financials",
		TotalExecutions: 50,
		Executions:      []models.CronjobExecutionLog{},
	}
	mockSvc.On("GetJobHistory", "sec-financials", 10, 20).Return(expected, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/cronjobs/sec-financials/history?limit=10&offset=20", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestCronjobHandler_GetJobDetails_Success(t *testing.T) {
	mockSvc := new(MockCronjobService)
	handler := NewCronjobHandler(mockSvc)
	router := setupCronjobRouter(handler)

	now := time.Now()
	expected := &models.CronjobExecutionLog{
		ID:          1,
		JobName:     "daily-price-update",
		ExecutionID: "exec-123",
		Status:      "success",
		StartedAt:   now,
	}
	mockSvc.On("GetJobDetails", "exec-123").Return(expected, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/cronjobs/details/exec-123", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestCronjobHandler_GetJobDetails_NotFound(t *testing.T) {
	mockSvc := new(MockCronjobService)
	handler := NewCronjobHandler(mockSvc)
	router := setupCronjobRouter(handler)

	mockSvc.On("GetJobDetails", "nonexistent").Return(nil, errors.New("execution not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/cronjobs/details/nonexistent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestCronjobHandler_GetMetrics_Success(t *testing.T) {
	mockSvc := new(MockCronjobService)
	handler := NewCronjobHandler(mockSvc)
	router := setupCronjobRouter(handler)

	expected := &models.CronjobMetricsResponse{
		DailySuccessRate: []models.CronjobDailySummary{},
		JobPerformance:   []models.CronjobPerformance{},
		FailureBreakdown: map[string]int{"timeout": 3, "error": 1},
	}
	mockSvc.On("GetMetrics", 7).Return(expected, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/cronjobs/metrics", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestCronjobHandler_GetAllSchedules_Success(t *testing.T) {
	mockSvc := new(MockCronjobService)
	handler := NewCronjobHandler(mockSvc)
	router := setupCronjobRouter(handler)

	expected := []models.CronjobSchedule{
		{ID: 1, JobName: "daily-price-update", ScheduleCron: "30 22 * * *", IsActive: true},
		{ID: 2, JobName: "sec-financials", ScheduleCron: "0 2 * * 0", IsActive: true},
	}
	mockSvc.On("GetAllSchedules").Return(expected, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/cronjobs/schedules", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []models.CronjobSchedule
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
	mockSvc.AssertExpectations(t)
}
