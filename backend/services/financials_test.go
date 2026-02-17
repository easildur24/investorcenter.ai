package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// CalculateFreeCashFlow — pure function
// ---------------------------------------------------------------------------

func TestCalculateFreeCashFlow_Normal(t *testing.T) {
	data := map[string]interface{}{
		"net_cash_flow_from_operating_activities": float64(110000000000),
		"capital_expenditure":                     float64(-10000000000), // typically negative
	}

	result := CalculateFreeCashFlow(data)
	require.NotNil(t, result)
	// FCF = 110B + (-10B) = 100B
	assert.InDelta(t, 100000000000.0, *result, 0.01)
}

func TestCalculateFreeCashFlow_NoCapex(t *testing.T) {
	data := map[string]interface{}{
		"net_cash_flow_from_operating_activities": float64(50000000000),
		// no capital_expenditure
	}

	result := CalculateFreeCashFlow(data)
	require.NotNil(t, result)
	// When capex is missing, FCF = operating cash flow
	assert.InDelta(t, 50000000000.0, *result, 0.01)
}

func TestCalculateFreeCashFlow_NoOperatingCF(t *testing.T) {
	data := map[string]interface{}{
		"capital_expenditure": float64(-5000000000),
		// no operating cash flow
	}

	result := CalculateFreeCashFlow(data)
	assert.Nil(t, result, "should return nil when operating cash flow is missing")
}

func TestCalculateFreeCashFlow_EmptyData(t *testing.T) {
	result := CalculateFreeCashFlow(map[string]interface{}{})
	assert.Nil(t, result)
}

func TestCalculateFreeCashFlow_NegativeOperating(t *testing.T) {
	data := map[string]interface{}{
		"net_cash_flow_from_operating_activities": float64(-5000000000),
		"capital_expenditure":                     float64(-2000000000),
	}

	result := CalculateFreeCashFlow(data)
	require.NotNil(t, result)
	// FCF = -5B + (-2B) = -7B
	assert.InDelta(t, -7000000000.0, *result, 0.01)
}

func TestCalculateFreeCashFlow_ZeroCapex(t *testing.T) {
	data := map[string]interface{}{
		"net_cash_flow_from_operating_activities": float64(20000000000),
		"capital_expenditure":                     float64(0),
	}

	result := CalculateFreeCashFlow(data)
	require.NotNil(t, result)
	assert.InDelta(t, 20000000000.0, *result, 0.01)
}

func TestCalculateFreeCashFlow_WrongType(t *testing.T) {
	data := map[string]interface{}{
		"net_cash_flow_from_operating_activities": "not a number",
		"capital_expenditure":                     float64(-5000000000),
	}

	result := CalculateFreeCashFlow(data)
	assert.Nil(t, result, "should return nil when operating CF is wrong type")
}

// ---------------------------------------------------------------------------
// EnrichCashFlowData — pure function
// ---------------------------------------------------------------------------

func TestEnrichCashFlowData_AddsFCF(t *testing.T) {
	data := map[string]interface{}{
		"net_cash_flow_from_operating_activities": float64(110000000000),
		"capital_expenditure":                     float64(-10000000000),
	}

	enriched := EnrichCashFlowData(data)

	// Should have original fields
	assert.Equal(t, float64(110000000000), enriched["net_cash_flow_from_operating_activities"])
	assert.Equal(t, float64(-10000000000), enriched["capital_expenditure"])

	// Should have calculated FCF
	assert.InDelta(t, 100000000000.0, enriched["free_cash_flow"], 0.01)
}

func TestEnrichCashFlowData_DoesNotMutateOriginal(t *testing.T) {
	data := map[string]interface{}{
		"net_cash_flow_from_operating_activities": float64(50000000000),
		"capital_expenditure":                     float64(-5000000000),
	}

	enriched := EnrichCashFlowData(data)

	// Original should not have free_cash_flow
	_, exists := data["free_cash_flow"]
	assert.False(t, exists, "original data should not be mutated")

	// Enriched should have it
	_, exists = enriched["free_cash_flow"]
	assert.True(t, exists, "enriched data should have free_cash_flow")
}

func TestEnrichCashFlowData_NoFCFWhenMissingOperating(t *testing.T) {
	data := map[string]interface{}{
		"capital_expenditure": float64(-5000000000),
	}

	enriched := EnrichCashFlowData(data)

	_, exists := enriched["free_cash_flow"]
	assert.False(t, exists, "should not add FCF when operating CF is missing")
}

func TestEnrichCashFlowData_EmptyInput(t *testing.T) {
	enriched := EnrichCashFlowData(map[string]interface{}{})
	assert.NotNil(t, enriched)
	assert.Empty(t, enriched)
}

func TestEnrichCashFlowData_PreservesExtraFields(t *testing.T) {
	data := map[string]interface{}{
		"net_cash_flow_from_operating_activities": float64(50000000000),
		"net_cash_flow_from_investing_activities": float64(-15000000000),
		"net_cash_flow_from_financing_activities": float64(-20000000000),
		"capital_expenditure":                     float64(-8000000000),
	}

	enriched := EnrichCashFlowData(data)

	assert.Equal(t, float64(-15000000000), enriched["net_cash_flow_from_investing_activities"])
	assert.Equal(t, float64(-20000000000), enriched["net_cash_flow_from_financing_activities"])
	assert.InDelta(t, 42000000000.0, enriched["free_cash_flow"], 0.01)
}

// ---------------------------------------------------------------------------
// NewFinancialsService — constructor
// ---------------------------------------------------------------------------

func TestNewFinancialsService(t *testing.T) {
	svc := NewFinancialsService()
	require.NotNil(t, svc)
	assert.NotNil(t, svc.polygonClient)
	assert.NotNil(t, svc.icScoreClient)
}
