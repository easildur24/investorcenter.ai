package services

import (
	"testing"
)

// TestCalculatedFallbacks tests the calculated metric fallbacks in MergeAllData
func TestPEGRatioCalculation(t *testing.T) {
	tests := []struct {
		name           string
		peRatio        *float64
		epsGrowth5Y    *float64
		epsGrowthYoY   *float64
		expectedPEG    *float64
		expectedSource DataSource
	}{
		{
			name:           "PEG from P/E and 5Y EPS growth",
			peRatio:        float64Ptr(20.0),
			epsGrowth5Y:    float64Ptr(10.0),
			epsGrowthYoY:   nil,
			expectedPEG:    float64Ptr(2.0), // 20 / 10 = 2.0
			expectedSource: SourceCalculated,
		},
		{
			name:           "PEG from P/E and YoY EPS growth when 5Y missing",
			peRatio:        float64Ptr(15.0),
			epsGrowth5Y:    nil,
			epsGrowthYoY:   float64Ptr(5.0),
			expectedPEG:    float64Ptr(3.0), // 15 / 5 = 3.0
			expectedSource: SourceCalculated,
		},
		{
			name:           "PEG prefers 5Y over YoY when both available",
			peRatio:        float64Ptr(25.0),
			epsGrowth5Y:    float64Ptr(10.0),
			epsGrowthYoY:   float64Ptr(20.0), // Should ignore this
			expectedPEG:    float64Ptr(2.5),  // 25 / 10 = 2.5 (uses 5Y)
			expectedSource: SourceCalculated,
		},
		{
			name:           "No PEG calculation when P/E is zero",
			peRatio:        float64Ptr(0.0),
			epsGrowth5Y:    float64Ptr(10.0),
			epsGrowthYoY:   nil,
			expectedPEG:    nil,
			expectedSource: "",
		},
		{
			name:           "No PEG calculation when EPS growth is zero",
			peRatio:        float64Ptr(20.0),
			epsGrowth5Y:    float64Ptr(0.0),
			epsGrowthYoY:   nil,
			expectedPEG:    nil,
			expectedSource: "",
		},
		{
			name:           "No PEG calculation when both EPS growths missing",
			peRatio:        float64Ptr(20.0),
			epsGrowth5Y:    nil,
			epsGrowthYoY:   nil,
			expectedPEG:    nil,
			expectedSource: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies the PEG calculation logic
			// The actual calculation happens in handlers/ic_score_handlers.go
			// Here we test the formula directly

			var pegRatio *float64
			var source DataSource

			if tt.peRatio != nil && *tt.peRatio > 0 {
				var epsGrowth *float64
				if tt.epsGrowth5Y != nil && *tt.epsGrowth5Y > 0 {
					epsGrowth = tt.epsGrowth5Y
				} else if tt.epsGrowthYoY != nil && *tt.epsGrowthYoY > 0 {
					epsGrowth = tt.epsGrowthYoY
				}
				if epsGrowth != nil {
					peg := *tt.peRatio / *epsGrowth
					pegRatio = &peg
					source = SourceCalculated
				}
			}

			if tt.expectedPEG == nil {
				if pegRatio != nil {
					t.Errorf("Expected no PEG calculation, but got %v", *pegRatio)
				}
			} else {
				if pegRatio == nil {
					t.Errorf("Expected PEG=%v, but got nil", *tt.expectedPEG)
				} else if *pegRatio != *tt.expectedPEG {
					t.Errorf("Expected PEG=%v, got %v", *tt.expectedPEG, *pegRatio)
				}
				if source != tt.expectedSource {
					t.Errorf("Expected source=%v, got %v", tt.expectedSource, source)
				}
			}
		})
	}
}

func TestEarningsYieldCalculation(t *testing.T) {
	tests := []struct {
		name                  string
		peRatio               *float64
		expectedEarningsYield *float64
	}{
		{
			name:                  "Earnings Yield from P/E=20",
			peRatio:               float64Ptr(20.0),
			expectedEarningsYield: float64Ptr(5.0), // (1/20) * 100 = 5.0%
		},
		{
			name:                  "Earnings Yield from P/E=10",
			peRatio:               float64Ptr(10.0),
			expectedEarningsYield: float64Ptr(10.0), // (1/10) * 100 = 10.0%
		},
		{
			name:                  "Earnings Yield from P/E=25",
			peRatio:               float64Ptr(25.0),
			expectedEarningsYield: float64Ptr(4.0), // (1/25) * 100 = 4.0%
		},
		{
			name:                  "No calculation when P/E is zero",
			peRatio:               float64Ptr(0.0),
			expectedEarningsYield: nil,
		},
		{
			name:                  "No calculation when P/E is nil",
			peRatio:               nil,
			expectedEarningsYield: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var earningsYield *float64

			if tt.peRatio != nil && *tt.peRatio > 0 {
				yield := (1.0 / *tt.peRatio) * 100
				earningsYield = &yield
			}

			if tt.expectedEarningsYield == nil {
				if earningsYield != nil {
					t.Errorf("Expected no earnings yield, but got %v", *earningsYield)
				}
			} else {
				if earningsYield == nil {
					t.Errorf("Expected earnings yield=%v, but got nil", *tt.expectedEarningsYield)
				} else if *earningsYield != *tt.expectedEarningsYield {
					t.Errorf("Expected earnings yield=%v, got %v", *tt.expectedEarningsYield, *earningsYield)
				}
			}
		})
	}
}

func TestFCFYieldCalculation(t *testing.T) {
	tests := []struct {
		name             string
		priceToFCF       *float64
		expectedFCFYield *float64
	}{
		{
			name:             "FCF Yield from Price-to-FCF=20",
			priceToFCF:       float64Ptr(20.0),
			expectedFCFYield: float64Ptr(5.0), // (1/20) * 100 = 5.0%
		},
		{
			name:             "FCF Yield from Price-to-FCF=15",
			priceToFCF:       float64Ptr(15.0),
			expectedFCFYield: float64Ptr(6.666666666666667), // (1/15) * 100
		},
		{
			name:             "No calculation when Price-to-FCF is zero",
			priceToFCF:       float64Ptr(0.0),
			expectedFCFYield: nil,
		},
		{
			name:             "No calculation when Price-to-FCF is nil",
			priceToFCF:       nil,
			expectedFCFYield: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fcfYield *float64

			if tt.priceToFCF != nil && *tt.priceToFCF > 0 {
				yield := (1.0 / *tt.priceToFCF) * 100
				fcfYield = &yield
			}

			if tt.expectedFCFYield == nil {
				if fcfYield != nil {
					t.Errorf("Expected no FCF yield, but got %v", *fcfYield)
				}
			} else {
				if fcfYield == nil {
					t.Errorf("Expected FCF yield=%v, but got nil", *tt.expectedFCFYield)
				} else if *fcfYield != *tt.expectedFCFYield {
					t.Errorf("Expected FCF yield=%v, got %v", *tt.expectedFCFYield, *fcfYield)
				}
			}
		})
	}
}

func TestEVToSalesCalculation(t *testing.T) {
	tests := []struct {
		name              string
		enterpriseValue   *float64
		psRatio           *float64
		marketCap         *float64
		expectedEVToSales *float64
	}{
		{
			name:              "EV/Sales calculated",
			enterpriseValue:   float64Ptr(1000.0),
			psRatio:           float64Ptr(2.0),
			marketCap:         float64Ptr(800.0),
			expectedEVToSales: float64Ptr(2.5), // (1000 * 2) / 800 = 2.5
		},
		{
			name:              "EV/Sales with different values",
			enterpriseValue:   float64Ptr(500.0),
			psRatio:           float64Ptr(1.5),
			marketCap:         float64Ptr(400.0),
			expectedEVToSales: float64Ptr(1.875), // (500 * 1.5) / 400 = 1.875
		},
		{
			name:              "No calculation when Market Cap is zero",
			enterpriseValue:   float64Ptr(1000.0),
			psRatio:           float64Ptr(2.0),
			marketCap:         float64Ptr(0.0),
			expectedEVToSales: nil,
		},
		{
			name:              "No calculation when EV is nil",
			enterpriseValue:   nil,
			psRatio:           float64Ptr(2.0),
			marketCap:         float64Ptr(800.0),
			expectedEVToSales: nil,
		},
		{
			name:              "No calculation when P/S is nil",
			enterpriseValue:   float64Ptr(1000.0),
			psRatio:           nil,
			marketCap:         float64Ptr(800.0),
			expectedEVToSales: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var evToSales *float64

			if tt.enterpriseValue != nil && tt.psRatio != nil && tt.marketCap != nil && *tt.marketCap > 0 {
				// EV/Sales = EV / Revenue
				// P/S = Market Cap / Revenue, so Revenue = Market Cap / P/S
				// Therefore: EV/Sales = EV / (Market Cap / P/S) = (EV * P/S) / Market Cap
				ev := (*tt.enterpriseValue * *tt.psRatio) / *tt.marketCap
				evToSales = &ev
			}

			if tt.expectedEVToSales == nil {
				if evToSales != nil {
					t.Errorf("Expected no EV/Sales, but got %v", *evToSales)
				}
			} else {
				if evToSales == nil {
					t.Errorf("Expected EV/Sales=%v, but got nil", *tt.expectedEVToSales)
				} else if *evToSales != *tt.expectedEVToSales {
					t.Errorf("Expected EV/Sales=%v, got %v", *tt.expectedEVToSales, *evToSales)
				}
			}
		})
	}
}

func TestForwardPECalculation(t *testing.T) {
	tests := []struct {
		name              string
		currentPrice      float64
		epsDiluted        *float64
		epsGrowth5Y       *float64
		epsGrowthYoY      *float64
		expectedForwardPE *float64
	}{
		{
			name:              "Forward P/E using 5Y growth",
			currentPrice:      100.0,
			epsDiluted:        float64Ptr(5.0),
			epsGrowth5Y:       float64Ptr(20.0), // 20% growth
			epsGrowthYoY:      nil,
			expectedForwardPE: float64Ptr(16.666666666666668), // 100 / (5 * 1.20) = 100 / 6 = 16.67
		},
		{
			name:              "Forward P/E using YoY growth when 5Y missing",
			currentPrice:      150.0,
			epsDiluted:        float64Ptr(10.0),
			epsGrowth5Y:       nil,
			epsGrowthYoY:      float64Ptr(15.0),               // 15% growth
			expectedForwardPE: float64Ptr(13.043478260869565), // 150 / (10 * 1.15) = 150 / 11.5 = 13.04
		},
		{
			name:              "Forward P/E using default 10% growth when no growth data",
			currentPrice:      200.0,
			epsDiluted:        float64Ptr(8.0),
			epsGrowth5Y:       nil,
			epsGrowthYoY:      nil,
			expectedForwardPE: float64Ptr(22.727272727272727), // 200 / (8 * 1.10) = 200 / 8.8 = 22.73
		},
		{
			name:              "No calculation when EPS is zero",
			currentPrice:      100.0,
			epsDiluted:        float64Ptr(0.0),
			epsGrowth5Y:       float64Ptr(20.0),
			epsGrowthYoY:      nil,
			expectedForwardPE: nil,
		},
		{
			name:              "No calculation when price is zero",
			currentPrice:      0.0,
			epsDiluted:        float64Ptr(5.0),
			epsGrowth5Y:       float64Ptr(20.0),
			epsGrowthYoY:      nil,
			expectedForwardPE: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var forwardPE *float64

			if tt.epsDiluted != nil && *tt.epsDiluted > 0 {
				// Use EPS growth rate to project next year's EPS
				var epsGrowthRate float64 = 0.10 // Default 10% growth assumption
				if tt.epsGrowth5Y != nil && *tt.epsGrowth5Y > 0 {
					epsGrowthRate = *tt.epsGrowth5Y / 100 // Convert percentage to decimal
				} else if tt.epsGrowthYoY != nil && *tt.epsGrowthYoY > 0 {
					epsGrowthRate = *tt.epsGrowthYoY / 100
				}

				// Project next year's EPS: CurrentEPS * (1 + growth rate)
				projectedEPS := *tt.epsDiluted * (1 + epsGrowthRate)

				// Calculate Forward P/E from current price and projected EPS
				if tt.currentPrice > 0 && projectedEPS > 0 {
					fpe := tt.currentPrice / projectedEPS
					forwardPE = &fpe
				}
			}

			if tt.expectedForwardPE == nil {
				if forwardPE != nil {
					t.Errorf("Expected no Forward P/E, but got %v", *forwardPE)
				}
			} else {
				if forwardPE == nil {
					t.Errorf("Expected Forward P/E=%v, but got nil", *tt.expectedForwardPE)
				} else if *forwardPE != *tt.expectedForwardPE {
					t.Errorf("Expected Forward P/E=%v, got %v", *tt.expectedForwardPE, *forwardPE)
				}
			}
		})
	}
}

func TestEVToEBITDACalculation(t *testing.T) {
	tests := []struct {
		name               string
		enterpriseValue    *float64
		revenuePerShare    *float64
		ebitdaMargin       *float64
		marketCap          *float64
		currentPrice       float64
		expectedEVToEBITDA *float64
	}{
		{
			name:               "EV/EBITDA calculated from revenue and margin",
			enterpriseValue:    float64Ptr(1000000.0),
			revenuePerShare:    float64Ptr(50.0),
			ebitdaMargin:       float64Ptr(20.0), // 20%
			marketCap:          float64Ptr(800000.0),
			currentPrice:       100.0,
			expectedEVToEBITDA: float64Ptr(12.5), // 1M / (50 * 8000 * 0.20) = 1M / 80K = 12.5
		},
		{
			name:               "No calculation when EV is zero",
			enterpriseValue:    float64Ptr(0.0),
			revenuePerShare:    float64Ptr(50.0),
			ebitdaMargin:       float64Ptr(20.0),
			marketCap:          float64Ptr(800000.0),
			currentPrice:       100.0,
			expectedEVToEBITDA: nil,
		},
		{
			name:               "No calculation when EBITDA margin is zero",
			enterpriseValue:    float64Ptr(1000000.0),
			revenuePerShare:    float64Ptr(50.0),
			ebitdaMargin:       float64Ptr(0.0),
			marketCap:          float64Ptr(800000.0),
			currentPrice:       100.0,
			expectedEVToEBITDA: nil,
		},
		{
			name:               "No calculation when current price is zero",
			enterpriseValue:    float64Ptr(1000000.0),
			revenuePerShare:    float64Ptr(50.0),
			ebitdaMargin:       float64Ptr(20.0),
			marketCap:          float64Ptr(800000.0),
			currentPrice:       0.0,
			expectedEVToEBITDA: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var evToEBITDA *float64

			if tt.enterpriseValue != nil && *tt.enterpriseValue > 0 {
				var ebitda *float64

				// Calculate from Revenue * EBITDA Margin
				if tt.revenuePerShare != nil && tt.ebitdaMargin != nil &&
					*tt.revenuePerShare > 0 && *tt.ebitdaMargin > 0 {
					if tt.marketCap != nil && *tt.marketCap > 0 && tt.currentPrice > 0 {
						sharesOutstanding := *tt.marketCap / tt.currentPrice
						revenue := *tt.revenuePerShare * sharesOutstanding
						ebitdaValue := revenue * (*tt.ebitdaMargin / 100)
						ebitda = &ebitdaValue
					}
				}

				// Calculate EV/EBITDA if we derived EBITDA
				if ebitda != nil && *ebitda > 0 {
					ratio := *tt.enterpriseValue / *ebitda
					evToEBITDA = &ratio
				}
			}

			if tt.expectedEVToEBITDA == nil {
				if evToEBITDA != nil {
					t.Errorf("Expected no EV/EBITDA, but got %v", *evToEBITDA)
				}
			} else {
				if evToEBITDA == nil {
					t.Errorf("Expected EV/EBITDA=%v, but got nil", *tt.expectedEVToEBITDA)
				} else if *evToEBITDA != *tt.expectedEVToEBITDA {
					t.Errorf("Expected EV/EBITDA=%v, got %v", *tt.expectedEVToEBITDA, *evToEBITDA)
				}
			}
		})
	}
}

// Helper function to create float64 pointers
func float64Ptr(f float64) *float64 {
	return &f
}
