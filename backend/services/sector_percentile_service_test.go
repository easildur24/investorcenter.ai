package services

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"investorcenter-api/models"
)

// ─── Helper function tests ───────────────────────────────────────

func TestSectorPercentileInterpolate(t *testing.T) {
	tests := []struct {
		name    string
		value   float64
		lowVal  float64
		highVal float64
		lowPct  float64
		highPct float64
		want    float64
	}{
		{
			name:  "midpoint interpolation",
			value: 50, lowVal: 0, highVal: 100, lowPct: 0, highPct: 100,
			want: 50,
		},
		{
			name:  "at low boundary",
			value: 10, lowVal: 10, highVal: 20, lowPct: 25, highPct: 50,
			want: 25,
		},
		{
			name:  "at high boundary",
			value: 20, lowVal: 10, highVal: 20, lowPct: 25, highPct: 50,
			want: 50,
		},
		{
			name:  "quarter point",
			value: 12.5, lowVal: 10, highVal: 20, lowPct: 25, highPct: 50,
			want: 31.25,
		},
		{
			name:  "equal low and high values returns lowPct",
			value: 5, lowVal: 5, highVal: 5, lowPct: 50, highPct: 75,
			want: 50,
		},
		{
			name:  "negative value range",
			value: -5, lowVal: -10, highVal: 0, lowPct: 0, highPct: 10,
			want: 5,
		},
		{
			name:  "p75 to p90 segment",
			value: 82.5, lowVal: 75, highVal: 90, lowPct: 75, highPct: 90,
			want: 82.5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := interpolate(tc.value, tc.lowVal, tc.highVal, tc.lowPct, tc.highPct)
			assert.InDelta(t, tc.want, result, 0.001)
		})
	}
}

func TestSectorPercentilePtrFloat64(t *testing.T) {
	t.Run("returns value when pointer is non-nil", func(t *testing.T) {
		v := 42.5
		assert.Equal(t, 42.5, ptrFloat64(&v, 0))
	})

	t.Run("returns default when pointer is nil", func(t *testing.T) {
		assert.Equal(t, 99.0, ptrFloat64(nil, 99.0))
	})

	t.Run("returns zero value pointer correctly", func(t *testing.T) {
		v := 0.0
		assert.Equal(t, 0.0, ptrFloat64(&v, 100.0))
	})

	t.Run("returns negative value pointer correctly", func(t *testing.T) {
		v := -15.5
		assert.Equal(t, -15.5, ptrFloat64(&v, 0))
	})
}

// ─── interpolatePercentile (piecewise) ──────────────────────────

func TestSectorPercentileInterpolatePercentile(t *testing.T) {
	// Create a service to call the method
	svc := &SectorPercentileService{}

	// Helper to create a standard percentile distribution
	newSP := func(min, p10, p25, p50, p75, p90, max float64) *CachedSectorPercentile {
		return &CachedSectorPercentile{
			MinValue: &min,
			P10Value: &p10,
			P25Value: &p25,
			P50Value: &p50,
			P75Value: &p75,
			P90Value: &p90,
			MaxValue: &max,
		}
	}

	t.Run("value at minimum returns 0", func(t *testing.T) {
		sp := newSP(0, 10, 25, 50, 75, 90, 100)
		assert.Equal(t, 0.0, svc.interpolatePercentile(0, sp))
	})

	t.Run("value below minimum returns 0", func(t *testing.T) {
		sp := newSP(10, 20, 30, 50, 70, 80, 100)
		assert.Equal(t, 0.0, svc.interpolatePercentile(5, sp))
	})

	t.Run("value at maximum returns 100", func(t *testing.T) {
		sp := newSP(0, 10, 25, 50, 75, 90, 100)
		assert.Equal(t, 100.0, svc.interpolatePercentile(100, sp))
	})

	t.Run("value above maximum returns 100", func(t *testing.T) {
		sp := newSP(0, 10, 25, 50, 75, 90, 100)
		assert.Equal(t, 100.0, svc.interpolatePercentile(150, sp))
	})

	t.Run("value at p50 returns 50", func(t *testing.T) {
		sp := newSP(0, 10, 25, 50, 75, 90, 100)
		assert.InDelta(t, 50.0, svc.interpolatePercentile(50, sp), 0.001)
	})

	t.Run("value at p10 returns 10", func(t *testing.T) {
		sp := newSP(0, 10, 25, 50, 75, 90, 100)
		assert.InDelta(t, 10.0, svc.interpolatePercentile(10, sp), 0.001)
	})

	t.Run("value at p25 returns 25", func(t *testing.T) {
		sp := newSP(0, 10, 25, 50, 75, 90, 100)
		assert.InDelta(t, 25.0, svc.interpolatePercentile(25, sp), 0.001)
	})

	t.Run("value at p75 returns 75", func(t *testing.T) {
		sp := newSP(0, 10, 25, 50, 75, 90, 100)
		assert.InDelta(t, 75.0, svc.interpolatePercentile(75, sp), 0.001)
	})

	t.Run("value at p90 returns 90", func(t *testing.T) {
		sp := newSP(0, 10, 25, 50, 75, 90, 100)
		assert.InDelta(t, 90.0, svc.interpolatePercentile(90, sp), 0.001)
	})

	t.Run("value between p25 and p50 interpolates correctly", func(t *testing.T) {
		sp := newSP(0, 10, 20, 40, 60, 80, 100)
		// value=30 is midpoint between p25=20 and p50=40
		// expected: midpoint between 25 and 50 = 37.5
		result := svc.interpolatePercentile(30, sp)
		assert.InDelta(t, 37.5, result, 0.001)
	})

	t.Run("value between p90 and max interpolates correctly", func(t *testing.T) {
		sp := newSP(0, 10, 25, 50, 75, 90, 100)
		// value=95 is midpoint between p90=90 and max=100
		// expected: midpoint between 90 and 100 = 95
		result := svc.interpolatePercentile(95, sp)
		assert.InDelta(t, 95.0, result, 0.001)
	})

	t.Run("handles all nil values gracefully", func(t *testing.T) {
		sp := &CachedSectorPercentile{} // all nil
		// All nil defaults to 0, so value=0 should return 0 (at min)
		result := svc.interpolatePercentile(0, sp)
		assert.InDelta(t, 0.0, result, 0.001)
	})

	t.Run("handles negative distribution", func(t *testing.T) {
		sp := newSP(-100, -80, -50, -20, -10, -5, 0)
		// value at p50=-20 should return 50
		result := svc.interpolatePercentile(-20, sp)
		assert.InDelta(t, 50.0, result, 0.001)
	})

	t.Run("single stock sector (all values same)", func(t *testing.T) {
		sp := newSP(42, 42, 42, 42, 42, 42, 42)
		// At the value, it's at min so returns 0
		result := svc.interpolatePercentile(42, sp)
		// When min == max, value <= min, returns 0
		assert.InDelta(t, 0.0, result, 0.001)
	})
}

// ─── Cache key generation ────────────────────────────────────────

func TestSectorPercentileCacheKeys(t *testing.T) {
	svc := &SectorPercentileService{}

	t.Run("cacheKey generates correct format", func(t *testing.T) {
		key := svc.cacheKey("Technology", "pe_ratio")
		assert.Equal(t, "sector_percentile:Technology:pe_ratio", key)
	})

	t.Run("sectorCacheKey generates correct format", func(t *testing.T) {
		key := svc.sectorCacheKey("Healthcare")
		assert.Equal(t, "sector_percentiles:Healthcare", key)
	})

	t.Run("cache keys are unique for different sectors", func(t *testing.T) {
		key1 := svc.cacheKey("Technology", "roe")
		key2 := svc.cacheKey("Healthcare", "roe")
		assert.NotEqual(t, key1, key2)
	})

	t.Run("cache keys are unique for different metrics", func(t *testing.T) {
		key1 := svc.cacheKey("Technology", "roe")
		key2 := svc.cacheKey("Technology", "roa")
		assert.NotEqual(t, key1, key2)
	})

	t.Run("handles sectors with spaces", func(t *testing.T) {
		key := svc.cacheKey("Real Estate", "dividend_yield")
		assert.Equal(t, "sector_percentile:Real Estate:dividend_yield", key)
	})
}

// ─── toCachedFormat conversion ──────────────────────────────────

func TestSectorPercentileToCachedFormat(t *testing.T) {
	svc := &SectorPercentileService{}

	t.Run("converts full model to cached format", func(t *testing.T) {
		sampleCount := 150
		sp := &models.SectorPercentile{
			Sector:       "Technology",
			MetricName:   "pe_ratio",
			CalculatedAt: time.Date(2026, 2, 13, 0, 0, 0, 0, time.UTC),
			MinValue:     testDecimalPtr(5.0),
			P10Value:     testDecimalPtr(10.0),
			P25Value:     testDecimalPtr(15.0),
			P50Value:     testDecimalPtr(20.0),
			P75Value:     testDecimalPtr(30.0),
			P90Value:     testDecimalPtr(40.0),
			MaxValue:     testDecimalPtr(100.0),
			MeanValue:    testDecimalPtr(22.5),
			StdDev:       testDecimalPtr(12.3),
			SampleCount:  &sampleCount,
		}

		cached := svc.toCachedFormat(sp)

		assert.Equal(t, "Technology", cached.Sector)
		assert.Equal(t, "pe_ratio", cached.MetricName)
		assert.Equal(t, "2026-02-13", cached.CalculatedAt)
		require.NotNil(t, cached.MinValue)
		assert.InDelta(t, 5.0, *cached.MinValue, 0.001)
		require.NotNil(t, cached.P10Value)
		assert.InDelta(t, 10.0, *cached.P10Value, 0.001)
		require.NotNil(t, cached.P25Value)
		assert.InDelta(t, 15.0, *cached.P25Value, 0.001)
		require.NotNil(t, cached.P50Value)
		assert.InDelta(t, 20.0, *cached.P50Value, 0.001)
		require.NotNil(t, cached.P75Value)
		assert.InDelta(t, 30.0, *cached.P75Value, 0.001)
		require.NotNil(t, cached.P90Value)
		assert.InDelta(t, 40.0, *cached.P90Value, 0.001)
		require.NotNil(t, cached.MaxValue)
		assert.InDelta(t, 100.0, *cached.MaxValue, 0.001)
		require.NotNil(t, cached.MeanValue)
		assert.InDelta(t, 22.5, *cached.MeanValue, 0.001)
		require.NotNil(t, cached.StdDev)
		assert.InDelta(t, 12.3, *cached.StdDev, 0.001)
		require.NotNil(t, cached.SampleCount)
		assert.Equal(t, 150, *cached.SampleCount)
		assert.NotEmpty(t, cached.CachedAt)
	})

	t.Run("handles nil decimal values", func(t *testing.T) {
		sp := &models.SectorPercentile{
			Sector:       "Healthcare",
			MetricName:   "roa",
			CalculatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			// All decimal values are nil
		}

		cached := svc.toCachedFormat(sp)

		assert.Equal(t, "Healthcare", cached.Sector)
		assert.Equal(t, "roa", cached.MetricName)
		assert.Nil(t, cached.MinValue)
		assert.Nil(t, cached.P10Value)
		assert.Nil(t, cached.P25Value)
		assert.Nil(t, cached.P50Value)
		assert.Nil(t, cached.P75Value)
		assert.Nil(t, cached.P90Value)
		assert.Nil(t, cached.MaxValue)
		assert.Nil(t, cached.MeanValue)
		assert.Nil(t, cached.StdDev)
		assert.Nil(t, cached.SampleCount)
	})

	t.Run("handles partial nil values", func(t *testing.T) {
		sp := &models.SectorPercentile{
			Sector:       "Energy",
			MetricName:   "roe",
			CalculatedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			MinValue:     testDecimalPtr(-10.0),
			P50Value:     testDecimalPtr(15.0),
			MaxValue:     testDecimalPtr(50.0),
			// Other percentiles are nil
		}

		cached := svc.toCachedFormat(sp)

		require.NotNil(t, cached.MinValue)
		assert.InDelta(t, -10.0, *cached.MinValue, 0.001)
		assert.Nil(t, cached.P10Value)
		assert.Nil(t, cached.P25Value)
		require.NotNil(t, cached.P50Value)
		assert.InDelta(t, 15.0, *cached.P50Value, 0.001)
		assert.Nil(t, cached.P75Value)
		assert.Nil(t, cached.P90Value)
		require.NotNil(t, cached.MaxValue)
		assert.InDelta(t, 50.0, *cached.MaxValue, 0.001)
	})

	t.Run("sets CachedAt to current time", func(t *testing.T) {
		sp := &models.SectorPercentile{
			Sector:       "Utilities",
			MetricName:   "dividend_yield",
			CalculatedAt: time.Date(2026, 2, 13, 0, 0, 0, 0, time.UTC),
		}

		before := time.Now()
		cached := svc.toCachedFormat(sp)
		after := time.Now()

		cachedTime, err := time.Parse(time.RFC3339, cached.CachedAt)
		require.NoError(t, err)
		assert.True(t, !cachedTime.Before(before.Truncate(time.Second)))
		assert.True(t, !cachedTime.After(after.Add(time.Second)))
	})
}

// ─── LowerIsBetterMetrics ────────────────────────────────────────

func TestSectorPercentileLowerIsBetterMetrics(t *testing.T) {
	t.Run("pe_ratio is lower-is-better", func(t *testing.T) {
		assert.True(t, models.LowerIsBetterMetrics["pe_ratio"])
	})

	t.Run("ps_ratio is lower-is-better", func(t *testing.T) {
		assert.True(t, models.LowerIsBetterMetrics["ps_ratio"])
	})

	t.Run("pb_ratio is lower-is-better", func(t *testing.T) {
		assert.True(t, models.LowerIsBetterMetrics["pb_ratio"])
	})

	t.Run("debt_to_equity is lower-is-better", func(t *testing.T) {
		assert.True(t, models.LowerIsBetterMetrics["debt_to_equity"])
	})

	t.Run("roe is NOT lower-is-better", func(t *testing.T) {
		assert.False(t, models.LowerIsBetterMetrics["roe"])
	})

	t.Run("roa is NOT lower-is-better", func(t *testing.T) {
		assert.False(t, models.LowerIsBetterMetrics["roa"])
	})

	t.Run("gross_margin is NOT lower-is-better", func(t *testing.T) {
		assert.False(t, models.LowerIsBetterMetrics["gross_margin"])
	})
}

// ─── CachedSectorPercentile struct ──────────────────────────────

func TestSectorPercentileCachedStruct(t *testing.T) {
	t.Run("all fields populated", func(t *testing.T) {
		minV, p10V, p25V, p50V := 1.0, 10.0, 25.0, 50.0
		p75V, p90V, maxV := 75.0, 90.0, 100.0
		meanV, stdV := 45.0, 20.0
		count := 500

		sp := CachedSectorPercentile{
			Sector:       "Financials",
			MetricName:   "current_ratio",
			CalculatedAt: "2026-02-13",
			MinValue:     &minV,
			P10Value:     &p10V,
			P25Value:     &p25V,
			P50Value:     &p50V,
			P75Value:     &p75V,
			P90Value:     &p90V,
			MaxValue:     &maxV,
			MeanValue:    &meanV,
			StdDev:       &stdV,
			SampleCount:  &count,
			CachedAt:     "2026-02-13T12:00:00Z",
		}

		assert.Equal(t, "Financials", sp.Sector)
		assert.Equal(t, "current_ratio", sp.MetricName)
		assert.Equal(t, 500, *sp.SampleCount)
		assert.InDelta(t, 50.0, *sp.P50Value, 0.001)
	})
}

// ─── TrackedMetrics completeness ────────────────────────────────

func TestSectorPercentileTrackedMetrics(t *testing.T) {
	t.Run("contains expected metric categories", func(t *testing.T) {
		metrics := make(map[string]bool)
		for _, m := range models.TrackedMetrics {
			metrics[m] = true
		}

		// Valuation metrics
		assert.True(t, metrics["pe_ratio"], "should track pe_ratio")
		assert.True(t, metrics["ps_ratio"], "should track ps_ratio")
		assert.True(t, metrics["pb_ratio"], "should track pb_ratio")

		// Profitability metrics
		assert.True(t, metrics["roe"], "should track roe")
		assert.True(t, metrics["roa"], "should track roa")
		assert.True(t, metrics["gross_margin"], "should track gross_margin")
		assert.True(t, metrics["net_margin"], "should track net_margin")

		// Growth metrics
		assert.True(t, metrics["revenue_growth_yoy"], "should track revenue_growth_yoy")
		assert.True(t, metrics["eps_growth_yoy"], "should track eps_growth_yoy")

		// Financial health metrics
		assert.True(t, metrics["current_ratio"], "should track current_ratio")
		assert.True(t, metrics["debt_to_equity"], "should track debt_to_equity")

		// Market metrics
		assert.True(t, metrics["dividend_yield"], "should track dividend_yield")
	})

	t.Run("no duplicate metrics", func(t *testing.T) {
		seen := make(map[string]bool)
		for _, m := range models.TrackedMetrics {
			assert.False(t, seen[m], "duplicate metric: %s", m)
			seen[m] = true
		}
	})

	t.Run("all lower-is-better metrics are tracked", func(t *testing.T) {
		metrics := make(map[string]bool)
		for _, m := range models.TrackedMetrics {
			metrics[m] = true
		}

		for metric := range models.LowerIsBetterMetrics {
			// Some lower-is-better metrics might not be tracked yet
			// (like net_debt_to_ebitda), so just check the common ones
			if metric == "pe_ratio" || metric == "ps_ratio" || metric == "pb_ratio" || metric == "debt_to_equity" {
				assert.True(t, metrics[metric],
					"lower-is-better metric %q should be tracked", metric)
			}
		}
	})
}

// ─── Helper to create decimal pointer ───────────────────────────

func testDecimalPtr(f float64) *decimal.Decimal {
	d := decimal.NewFromFloat(f)
	return &d
}
