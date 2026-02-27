package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"investorcenter-api/models"
)

// mockCronjobService implements CronjobServicer for testing.
type mockCronjobService struct {
	overviewResp  *models.CronjobOverviewResponse
	overviewErr   error
	historyResp   *models.CronjobHistoryResponse
	historyErr    error
	detailsResp   *models.CronjobExecutionLog
	detailsErr    error
	metricsResp   *models.CronjobMetricsResponse
	metricsErr    error
	schedulesResp []models.CronjobSchedule
	schedulesErr  error
}

func (m *mockCronjobService) GetOverview() (*models.CronjobOverviewResponse, error) {
	return m.overviewResp, m.overviewErr
}
func (m *mockCronjobService) GetJobHistory(jobName string, limit, offset int) (*models.CronjobHistoryResponse, error) {
	return m.historyResp, m.historyErr
}
func (m *mockCronjobService) GetJobDetails(executionID string) (*models.CronjobExecutionLog, error) {
	return m.detailsResp, m.detailsErr
}
func (m *mockCronjobService) GetMetrics(period int) (*models.CronjobMetricsResponse, error) {
	return m.metricsResp, m.metricsErr
}
func (m *mockCronjobService) GetAllSchedules() ([]models.CronjobSchedule, error) {
	return m.schedulesResp, m.schedulesErr
}

// ---------------------------------------------------------------------------
// GetOverview — mock service tests
// ---------------------------------------------------------------------------

func TestGetOverview_Mock_Success(t *testing.T) {
	svc := &mockCronjobService{
		overviewResp: &models.CronjobOverviewResponse{
			Summary: models.CronjobSummary{
				TotalJobs: 5,
			},
		},
	}
	handler := NewCronjobHandler(svc)

	r := setupMockRouterNoAuth()
	r.GET("/cronjobs/overview", handler.GetOverview)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cronjobs/overview", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetOverview_Mock_Error(t *testing.T) {
	svc := &mockCronjobService{
		overviewErr: fmt.Errorf("service error"),
	}
	handler := NewCronjobHandler(svc)

	r := setupMockRouterNoAuth()
	r.GET("/cronjobs/overview", handler.GetOverview)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cronjobs/overview", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// GetJobHistory — mock service tests
// ---------------------------------------------------------------------------

func TestGetJobHistory_Mock_Success(t *testing.T) {
	svc := &mockCronjobService{
		historyResp: &models.CronjobHistoryResponse{
			JobName: "test-job",
		},
	}
	handler := NewCronjobHandler(svc)

	r := setupMockRouterNoAuth()
	r.GET("/cronjobs/:jobName/history", handler.GetJobHistory)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cronjobs/test-job/history", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetJobHistory_Mock_Error(t *testing.T) {
	svc := &mockCronjobService{
		historyErr: fmt.Errorf("history error"),
	}
	handler := NewCronjobHandler(svc)

	r := setupMockRouterNoAuth()
	r.GET("/cronjobs/:jobName/history", handler.GetJobHistory)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cronjobs/test-job/history?limit=10&offset=5", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// GetJobDetails — mock service tests
// ---------------------------------------------------------------------------

func TestGetJobDetails_Mock_Success(t *testing.T) {
	now := time.Now()
	svc := &mockCronjobService{
		detailsResp: &models.CronjobExecutionLog{
			ID:          1,
			ExecutionID: "exec-1",
			StartedAt:   now,
		},
	}
	handler := NewCronjobHandler(svc)

	r := setupMockRouterNoAuth()
	r.GET("/cronjobs/details/:executionId", handler.GetJobDetails)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cronjobs/details/exec-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetJobDetails_Mock_NotFound(t *testing.T) {
	svc := &mockCronjobService{
		detailsErr: fmt.Errorf("execution not found"),
	}
	handler := NewCronjobHandler(svc)

	r := setupMockRouterNoAuth()
	r.GET("/cronjobs/details/:executionId", handler.GetJobDetails)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cronjobs/details/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetJobDetails_Mock_InternalError(t *testing.T) {
	svc := &mockCronjobService{
		detailsErr: fmt.Errorf("some db error"),
	}
	handler := NewCronjobHandler(svc)

	r := setupMockRouterNoAuth()
	r.GET("/cronjobs/details/:executionId", handler.GetJobDetails)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cronjobs/details/exec-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// GetMetrics — mock service tests
// ---------------------------------------------------------------------------

func TestGetMetrics_Mock_Success(t *testing.T) {
	svc := &mockCronjobService{
		metricsResp: &models.CronjobMetricsResponse{
			DailySuccessRate: []models.CronjobDailySummary{},
		},
	}
	handler := NewCronjobHandler(svc)

	r := setupMockRouterNoAuth()
	r.GET("/cronjobs/metrics", handler.GetMetrics)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cronjobs/metrics", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetMetrics_Mock_Error(t *testing.T) {
	svc := &mockCronjobService{
		metricsErr: fmt.Errorf("metrics error"),
	}
	handler := NewCronjobHandler(svc)

	r := setupMockRouterNoAuth()
	r.GET("/cronjobs/metrics", handler.GetMetrics)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cronjobs/metrics?period=30", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// GetAllSchedules — mock service tests
// ---------------------------------------------------------------------------

func TestGetAllSchedules_Mock_Success(t *testing.T) {
	svc := &mockCronjobService{
		schedulesResp: []models.CronjobSchedule{
			{JobName: "test-job", ScheduleCron: "*/5 * * * *"},
		},
	}
	handler := NewCronjobHandler(svc)

	r := setupMockRouterNoAuth()
	r.GET("/cronjobs/schedules", handler.GetAllSchedules)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cronjobs/schedules", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetAllSchedules_Mock_Error(t *testing.T) {
	svc := &mockCronjobService{
		schedulesErr: fmt.Errorf("schedule error"),
	}
	handler := NewCronjobHandler(svc)

	r := setupMockRouterNoAuth()
	r.GET("/cronjobs/schedules", handler.GetAllSchedules)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cronjobs/schedules", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
