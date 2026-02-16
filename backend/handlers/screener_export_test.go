package handlers

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"investorcenter-api/database"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// ExportScreenerCSV handler tests
// ---------------------------------------------------------------------------

func TestExportScreenerCSV_NoDB(t *testing.T) {
	original := database.DB
	database.DB = nil
	defer func() { database.DB = original }()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/screener/stocks/export", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	ExportScreenerCSV(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if _, ok := body["error"]; !ok {
		t.Error("expected error field in response")
	}
}

func TestExportScreenerCSV_Headers(t *testing.T) {
	if database.DB == nil {
		t.Skip("database not available")
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/screener/stocks/export?limit=1", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	ExportScreenerCSV(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/csv" {
		t.Errorf("expected Content-Type text/csv, got %q", contentType)
	}

	disposition := w.Header().Get("Content-Disposition")
	if !strings.HasPrefix(disposition, "attachment; filename=screener-export-") {
		t.Errorf("unexpected Content-Disposition: %q", disposition)
	}
	if !strings.HasSuffix(disposition, ".csv") {
		t.Errorf("filename should end with .csv: %q", disposition)
	}

	// Parse CSV and verify header row
	reader := csv.NewReader(strings.NewReader(w.Body.String()))
	header, err := reader.Read()
	if err != nil {
		t.Fatalf("failed to read CSV header: %v", err)
	}
	expectedCols := 35
	if len(header) != expectedCols {
		t.Errorf("expected %d CSV columns, got %d: %v", expectedCols, len(header), header)
	}
	if header[0] != "Symbol" {
		t.Errorf("expected first column 'Symbol', got %q", header[0])
	}
	if header[len(header)-1] != "Technical Score" {
		t.Errorf("expected last column 'Technical Score', got %q", header[len(header)-1])
	}
}

func TestExportScreenerCSV_RespectsFilters(t *testing.T) {
	if database.DB == nil {
		t.Skip("database not available")
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/screener/stocks/export?sectors=Technology&ic_score_min=90", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	ExportScreenerCSV(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	reader := csv.NewReader(strings.NewReader(w.Body.String()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to read CSV: %v", err)
	}

	// Should have at least the header row
	if len(records) < 1 {
		t.Fatal("expected at least 1 row (header)")
	}

	// Every data row should have sector=Technology (column index 2)
	for i, row := range records[1:] {
		if row[2] != "Technology" {
			t.Errorf("row %d: expected sector 'Technology', got %q", i+1, row[2])
		}
	}
}

// ---------------------------------------------------------------------------
// GetScreenerIndustries handler tests
// ---------------------------------------------------------------------------

func TestGetScreenerIndustries_NoDB(t *testing.T) {
	original := database.DB
	database.DB = nil
	defer func() { database.DB = original }()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/screener/industries", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	GetScreenerIndustries(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}
}

func TestGetScreenerIndustries_AllIndustries(t *testing.T) {
	if database.DB == nil {
		t.Skip("database not available")
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/screener/industries", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	GetScreenerIndustries(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.Data) == 0 {
		t.Error("expected at least one industry")
	}

	// Verify sorted order
	for i := 1; i < len(resp.Data); i++ {
		if resp.Data[i] < resp.Data[i-1] {
			t.Errorf("industries not sorted: %q came after %q", resp.Data[i], resp.Data[i-1])
			break
		}
	}
}

func TestGetScreenerIndustries_FilteredBySector(t *testing.T) {
	if database.DB == nil {
		t.Skip("database not available")
	}

	gin.SetMode(gin.TestMode)

	// Get all industries
	wAll := httptest.NewRecorder()
	reqAll := httptest.NewRequest(http.MethodGet, "/api/v1/screener/industries", nil)
	cAll, _ := gin.CreateTestContext(wAll)
	cAll.Request = reqAll
	GetScreenerIndustries(cAll)

	var allResp struct {
		Data []string `json:"data"`
	}
	json.Unmarshal(wAll.Body.Bytes(), &allResp)

	// Get industries filtered by one sector
	wFiltered := httptest.NewRecorder()
	reqFiltered := httptest.NewRequest(http.MethodGet, "/api/v1/screener/industries?sectors=Technology", nil)
	cFiltered, _ := gin.CreateTestContext(wFiltered)
	cFiltered.Request = reqFiltered
	GetScreenerIndustries(cFiltered)

	var filteredResp struct {
		Data []string `json:"data"`
	}
	json.Unmarshal(wFiltered.Body.Bytes(), &filteredResp)

	if wFiltered.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", wFiltered.Code)
	}

	// Filtered should be a subset (fewer or equal)
	if len(filteredResp.Data) > len(allResp.Data) {
		t.Errorf("filtered industries (%d) should not exceed all industries (%d)",
			len(filteredResp.Data), len(allResp.Data))
	}
}

// ---------------------------------------------------------------------------
// CSV helper unit tests
// ---------------------------------------------------------------------------

// TestPtrStr verifies the nil-safe string pointer helper.
func TestPtrStr(t *testing.T) {
	cases := []struct {
		name string
		in   *string
		want string
	}{
		{"nil", nil, ""},
		{"non-nil", strPtr("Technology"), "Technology"},
		{"empty", strPtr(""), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ptrStr(tc.in)
			if got != tc.want {
				t.Errorf("ptrStr(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestFmtFloat verifies the nil-safe float formatting helper.
func TestFmtFloat(t *testing.T) {
	cases := []struct {
		name     string
		in       *float64
		decimals int
		want     string
	}{
		{"nil", nil, 2, ""},
		{"zero", floatPtr(0.0), 2, "0.00"},
		{"positive", floatPtr(123.456), 2, "123.46"},
		{"negative", floatPtr(-5.5), 1, "-5.5"},
		{"large", floatPtr(1e12), 0, "1000000000000"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := fmtFloat(tc.in, tc.decimals)
			if got != tc.want {
				t.Errorf("fmtFloat(%v, %d) = %q, want %q", tc.in, tc.decimals, got, tc.want)
			}
		})
	}
}

// TestFmtInt verifies the nil-safe int formatting helper.
func TestFmtInt(t *testing.T) {
	cases := []struct {
		name string
		in   *int
		want string
	}{
		{"nil", nil, ""},
		{"zero", intPtr(0), "0"},
		{"positive", intPtr(42), "42"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := fmtInt(tc.in)
			if got != tc.want {
				t.Errorf("fmtInt(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func strPtr(s string) *string     { return &s }
func floatPtr(f float64) *float64 { return &f }
func intPtr(i int) *int           { return &i }
