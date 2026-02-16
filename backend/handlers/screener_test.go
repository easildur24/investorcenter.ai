package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"investorcenter-api/database"

	"github.com/gin-gonic/gin"
)

// createTestContext builds a gin.Context from a URL query string.
func createTestContext(queryString string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/screener/stocks?"+queryString, nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	return c, w
}

// ---------------------------------------------------------------------------
// parseScreenerParams — defaults
// ---------------------------------------------------------------------------

func TestParseScreenerParamsDefaults(t *testing.T) {
	c, _ := createTestContext("")
	params := parseScreenerParams(c)

	if params.Page != 1 {
		t.Errorf("expected default page=1, got %d", params.Page)
	}
	if params.Limit != defaultScreenerLimit {
		t.Errorf("expected default limit=%d, got %d", defaultScreenerLimit, params.Limit)
	}
	if params.Sort != "market_cap" {
		t.Errorf("expected default sort=market_cap, got %q", params.Sort)
	}
	if params.Order != "desc" {
		t.Errorf("expected default order=desc, got %q", params.Order)
	}
	if params.AssetType != "CS" {
		t.Errorf("expected default asset_type=CS, got %q", params.AssetType)
	}
	if len(params.Sectors) != 0 {
		t.Errorf("expected no sectors, got %v", params.Sectors)
	}
}

// ---------------------------------------------------------------------------
// parseScreenerParams — pagination
// ---------------------------------------------------------------------------

func TestParseScreenerParamsPage(t *testing.T) {
	tests := []struct {
		query    string
		wantPage int
	}{
		{"page=5", 5},
		{"page=0", 1},   // invalid, falls back to default
		{"page=-1", 1},  // invalid
		{"page=abc", 1}, // non-numeric
	}
	for _, tc := range tests {
		t.Run(tc.query, func(t *testing.T) {
			c, _ := createTestContext(tc.query)
			params := parseScreenerParams(c)
			if params.Page != tc.wantPage {
				t.Errorf("query=%q: expected page=%d, got %d", tc.query, tc.wantPage, params.Page)
			}
		})
	}
}

func TestParseScreenerParamsLimit(t *testing.T) {
	tests := []struct {
		query     string
		wantLimit int
	}{
		{"limit=25", 25},
		{"limit=100", 100},
		{"limit=50000", maxScreenerLimit},   // capped at max
		{"limit=0", defaultScreenerLimit},   // invalid, falls back to default
		{"limit=-5", defaultScreenerLimit},  // invalid
		{"limit=abc", defaultScreenerLimit}, // non-numeric
	}
	for _, tc := range tests {
		t.Run(tc.query, func(t *testing.T) {
			c, _ := createTestContext(tc.query)
			params := parseScreenerParams(c)
			if params.Limit != tc.wantLimit {
				t.Errorf("query=%q: expected limit=%d, got %d", tc.query, tc.wantLimit, params.Limit)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseScreenerParams — sort validation
// ---------------------------------------------------------------------------

func TestParseScreenerParamsSortValid(t *testing.T) {
	c, _ := createTestContext("sort=ic_score")
	params := parseScreenerParams(c)
	if params.Sort != "ic_score" {
		t.Errorf("expected sort=ic_score, got %q", params.Sort)
	}
}

func TestParseScreenerParamsSortInvalid(t *testing.T) {
	// SQL injection attempt should fall back to default
	c, _ := createTestContext("sort=market_cap%3BDROP%20TABLE%20stocks")
	params := parseScreenerParams(c)
	if params.Sort != "market_cap" {
		t.Errorf("expected sort to fall back to market_cap, got %q", params.Sort)
	}
}

func TestParseScreenerParamsSortAllValid(t *testing.T) {
	for key := range database.ValidScreenerSortColumns {
		t.Run(key, func(t *testing.T) {
			c, _ := createTestContext("sort=" + key)
			params := parseScreenerParams(c)
			if params.Sort != key {
				t.Errorf("expected sort=%q, got %q", key, params.Sort)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseScreenerParams — order validation
// ---------------------------------------------------------------------------

func TestParseScreenerParamsOrder(t *testing.T) {
	tests := []struct {
		query     string
		wantOrder string
	}{
		{"order=asc", "asc"},
		{"order=desc", "desc"},
		{"order=ASC", "asc"}, // lowercased
		{"order=DESC", "desc"},
		{"order=invalid", "desc"}, // falls back to default
		{"order=", "desc"},
	}
	for _, tc := range tests {
		t.Run(tc.query, func(t *testing.T) {
			c, _ := createTestContext(tc.query)
			params := parseScreenerParams(c)
			if params.Order != tc.wantOrder {
				t.Errorf("query=%q: expected order=%q, got %q", tc.query, tc.wantOrder, params.Order)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseScreenerParams — sectors & industries
// ---------------------------------------------------------------------------

func TestParseScreenerParamsSectors(t *testing.T) {
	c, _ := createTestContext("sectors=Technology,Healthcare,Energy")
	params := parseScreenerParams(c)
	if len(params.Sectors) != 3 {
		t.Fatalf("expected 3 sectors, got %d: %v", len(params.Sectors), params.Sectors)
	}
	if params.Sectors[0] != "Technology" || params.Sectors[1] != "Healthcare" || params.Sectors[2] != "Energy" {
		t.Errorf("unexpected sectors: %v", params.Sectors)
	}
}

func TestParseScreenerParamsSectorsWhitespaceTrimmed(t *testing.T) {
	c, _ := createTestContext("sectors=Technology%2C+Healthcare+%2C+Energy")
	params := parseScreenerParams(c)
	for _, s := range params.Sectors {
		if s != "Technology" && s != "Healthcare" && s != "Energy" {
			t.Errorf("sector not trimmed properly: %q", s)
		}
	}
}

func TestParseScreenerParamsIndustries(t *testing.T) {
	c, _ := createTestContext("industries=Software,Semiconductors")
	params := parseScreenerParams(c)
	if len(params.Industries) != 2 {
		t.Fatalf("expected 2 industries, got %d: %v", len(params.Industries), params.Industries)
	}
	if params.Industries[0] != "Software" || params.Industries[1] != "Semiconductors" {
		t.Errorf("unexpected industries: %v", params.Industries)
	}
}

// ---------------------------------------------------------------------------
// parseScreenerParams — range filters
// ---------------------------------------------------------------------------

func TestParseScreenerParamsRangeFilters(t *testing.T) {
	c, _ := createTestContext("pe_min=5&pe_max=25&roe_min=10&beta_max=1.5")
	params := parseScreenerParams(c)

	if params.PEMin == nil || *params.PEMin != 5.0 {
		t.Errorf("expected pe_min=5, got %v", params.PEMin)
	}
	if params.PEMax == nil || *params.PEMax != 25.0 {
		t.Errorf("expected pe_max=25, got %v", params.PEMax)
	}
	if params.ROEMin == nil || *params.ROEMin != 10.0 {
		t.Errorf("expected roe_min=10, got %v", params.ROEMin)
	}
	if params.BetaMax == nil || *params.BetaMax != 1.5 {
		t.Errorf("expected beta_max=1.5, got %v", params.BetaMax)
	}
}

func TestParseScreenerParamsRangeFiltersInvalid(t *testing.T) {
	// Non-numeric values should be ignored (field stays nil)
	c, _ := createTestContext("pe_min=abc&pe_max=&roe_min=not_a_number")
	params := parseScreenerParams(c)

	if params.PEMin != nil {
		t.Errorf("expected pe_min to be nil for invalid input, got %v", *params.PEMin)
	}
	if params.PEMax != nil {
		t.Errorf("expected pe_max to be nil for empty input, got %v", *params.PEMax)
	}
	if params.ROEMin != nil {
		t.Errorf("expected roe_min to be nil for invalid input, got %v", *params.ROEMin)
	}
}

func TestParseScreenerParamsNegativeRangeValues(t *testing.T) {
	c, _ := createTestContext("revenue_growth_min=-50&dcf_upside_min=-25.5")
	params := parseScreenerParams(c)

	if params.RevenueGrowthMin == nil || *params.RevenueGrowthMin != -50.0 {
		t.Errorf("expected revenue_growth_min=-50, got %v", params.RevenueGrowthMin)
	}
	if params.DCFUpsideMin == nil || *params.DCFUpsideMin != -25.5 {
		t.Errorf("expected dcf_upside_min=-25.5, got %v", params.DCFUpsideMin)
	}
}

func TestParseScreenerParamsICScoreSubFactors(t *testing.T) {
	c, _ := createTestContext("value_score_min=60&growth_score_max=90&momentum_score_min=50&technical_score_max=80")
	params := parseScreenerParams(c)

	if params.ValueScoreMin == nil || *params.ValueScoreMin != 60.0 {
		t.Errorf("expected value_score_min=60, got %v", params.ValueScoreMin)
	}
	if params.GrowthScoreMax == nil || *params.GrowthScoreMax != 90.0 {
		t.Errorf("expected growth_score_max=90, got %v", params.GrowthScoreMax)
	}
	if params.MomentumScoreMin == nil || *params.MomentumScoreMin != 50.0 {
		t.Errorf("expected momentum_score_min=50, got %v", params.MomentumScoreMin)
	}
	if params.TechnicalScoreMax == nil || *params.TechnicalScoreMax != 80.0 {
		t.Errorf("expected technical_score_max=80, got %v", params.TechnicalScoreMax)
	}
}

// ---------------------------------------------------------------------------
// parseScreenerParams — asset type validation
// ---------------------------------------------------------------------------

func TestParseScreenerParamsAssetType(t *testing.T) {
	tests := []struct {
		query         string
		wantAssetType string
	}{
		{"asset_type=CS", "CS"},
		{"asset_type=ETF", "ETF"},
		{"asset_type=crypto", "crypto"},
		{"asset_type=INVALID", "CS"},               // falls back to default
		{"asset_type=%27%3B%20DROP%20TABLE", "CS"}, // SQL injection falls back
		{"", "CS"}, // no param, default
	}
	for _, tc := range tests {
		t.Run(tc.query, func(t *testing.T) {
			c, _ := createTestContext(tc.query)
			params := parseScreenerParams(c)
			if params.AssetType != tc.wantAssetType {
				t.Errorf("query=%q: expected asset_type=%q, got %q", tc.query, tc.wantAssetType, params.AssetType)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseScreenerParams — combined params
// ---------------------------------------------------------------------------

func TestParseScreenerParamsCombined(t *testing.T) {
	c, _ := createTestContext(
		"page=2&limit=25&sort=ic_score&order=asc&sectors=Technology&pe_max=30&ic_score_min=70&asset_type=CS",
	)
	params := parseScreenerParams(c)

	if params.Page != 2 {
		t.Errorf("expected page=2, got %d", params.Page)
	}
	if params.Limit != 25 {
		t.Errorf("expected limit=25, got %d", params.Limit)
	}
	if params.Sort != "ic_score" {
		t.Errorf("expected sort=ic_score, got %q", params.Sort)
	}
	if params.Order != "asc" {
		t.Errorf("expected order=asc, got %q", params.Order)
	}
	if len(params.Sectors) != 1 || params.Sectors[0] != "Technology" {
		t.Errorf("expected sectors=[Technology], got %v", params.Sectors)
	}
	if params.PEMax == nil || *params.PEMax != 30.0 {
		t.Errorf("expected pe_max=30, got %v", params.PEMax)
	}
	if params.ICScoreMin == nil || *params.ICScoreMin != 70.0 {
		t.Errorf("expected ic_score_min=70, got %v", params.ICScoreMin)
	}
}

// ---------------------------------------------------------------------------
// rangeParams registry — completeness
// ---------------------------------------------------------------------------

func TestRangeParamsNoDuplicateKeys(t *testing.T) {
	seen := map[string]int{}
	for i, fp := range rangeParams {
		if prev, ok := seen[fp.key]; ok {
			t.Errorf("duplicate key %q at index %d and %d", fp.key, prev, i)
		}
		seen[fp.key] = i
	}
}

func TestRangeParamsAllSetters(t *testing.T) {
	// Verify that every setter actually sets a value (doesn't panic)
	for _, fp := range rangeParams {
		t.Run(fp.key, func(t *testing.T) {
			var params = parseScreenerParams(func() *gin.Context {
				c, _ := createTestContext(fp.key + "=42.5")
				return c
			}())
			_ = params // just verify no panic
		})
	}
}
