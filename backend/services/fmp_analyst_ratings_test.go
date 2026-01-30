package services

import (
	"testing"
)

// TestFMPGradesSummaryMerge tests that grades summary data is correctly merged
func TestFMPGradesSummaryMerge(t *testing.T) {
	tests := []struct {
		name               string
		gradesSummary      *FMPGradesSummary
		expectedStrongBuy  *int
		expectedBuy        *int
		expectedHold       *int
		expectedSell       *int
		expectedStrongSell *int
		expectedConsensus  *string
	}{
		{
			name: "Strong Buy consensus",
			gradesSummary: &FMPGradesSummary{
				Symbol:     "AAPL",
				StrongBuy:  10,
				Buy:        5,
				Hold:       2,
				Sell:       0,
				StrongSell: 0,
				Consensus:  "Strong Buy",
			},
			expectedStrongBuy:  intPtr(10),
			expectedBuy:        intPtr(5),
			expectedHold:       intPtr(2),
			expectedSell:       intPtr(0),
			expectedStrongSell: intPtr(0),
			expectedConsensus:  stringPtr("Strong Buy"),
		},
		{
			name: "Buy consensus",
			gradesSummary: &FMPGradesSummary{
				Symbol:     "MSFT",
				StrongBuy:  5,
				Buy:        10,
				Hold:       3,
				Sell:       1,
				StrongSell: 0,
				Consensus:  "Buy",
			},
			expectedStrongBuy:  intPtr(5),
			expectedBuy:        intPtr(10),
			expectedHold:       intPtr(3),
			expectedSell:       intPtr(1),
			expectedStrongSell: intPtr(0),
			expectedConsensus:  stringPtr("Buy"),
		},
		{
			name: "Hold consensus",
			gradesSummary: &FMPGradesSummary{
				Symbol:     "IBM",
				StrongBuy:  2,
				Buy:        3,
				Hold:       10,
				Sell:       2,
				StrongSell: 1,
				Consensus:  "Hold",
			},
			expectedStrongBuy:  intPtr(2),
			expectedBuy:        intPtr(3),
			expectedHold:       intPtr(10),
			expectedSell:       intPtr(2),
			expectedStrongSell: intPtr(1),
			expectedConsensus:  stringPtr("Hold"),
		},
		{
			name: "Sell consensus",
			gradesSummary: &FMPGradesSummary{
				Symbol:     "XYZ",
				StrongBuy:  0,
				Buy:        1,
				Hold:       3,
				Sell:       8,
				StrongSell: 2,
				Consensus:  "Sell",
			},
			expectedStrongBuy:  intPtr(0),
			expectedBuy:        intPtr(1),
			expectedHold:       intPtr(3),
			expectedSell:       intPtr(8),
			expectedStrongSell: intPtr(2),
			expectedConsensus:  stringPtr("Sell"),
		},
		{
			name:               "Nil grades summary",
			gradesSummary:      nil,
			expectedStrongBuy:  nil,
			expectedBuy:        nil,
			expectedHold:       nil,
			expectedSell:       nil,
			expectedStrongSell: nil,
			expectedConsensus:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmpMetrics := &FMPAllMetrics{
				GradesSummary: tt.gradesSummary,
				Errors:        make(map[string]error),
			}

			merged := MergeAllData(fmpMetrics, 100.0)

			// Verify Strong Buy
			if tt.expectedStrongBuy == nil {
				if merged.AnalystRatingStrongBuy != nil {
					t.Errorf("Expected no Strong Buy, but got %v", *merged.AnalystRatingStrongBuy)
				}
			} else {
				if merged.AnalystRatingStrongBuy == nil {
					t.Errorf("Expected Strong Buy=%v, but got nil", *tt.expectedStrongBuy)
				} else if *merged.AnalystRatingStrongBuy != *tt.expectedStrongBuy {
					t.Errorf("Expected Strong Buy=%v, got %v", *tt.expectedStrongBuy, *merged.AnalystRatingStrongBuy)
				}
			}

			// Verify Buy
			if tt.expectedBuy == nil {
				if merged.AnalystRatingBuy != nil {
					t.Errorf("Expected no Buy, but got %v", *merged.AnalystRatingBuy)
				}
			} else {
				if merged.AnalystRatingBuy == nil {
					t.Errorf("Expected Buy=%v, but got nil", *tt.expectedBuy)
				} else if *merged.AnalystRatingBuy != *tt.expectedBuy {
					t.Errorf("Expected Buy=%v, got %v", *tt.expectedBuy, *merged.AnalystRatingBuy)
				}
			}

			// Verify Hold
			if tt.expectedHold == nil {
				if merged.AnalystRatingHold != nil {
					t.Errorf("Expected no Hold, but got %v", *merged.AnalystRatingHold)
				}
			} else {
				if merged.AnalystRatingHold == nil {
					t.Errorf("Expected Hold=%v, but got nil", *tt.expectedHold)
				} else if *merged.AnalystRatingHold != *tt.expectedHold {
					t.Errorf("Expected Hold=%v, got %v", *tt.expectedHold, *merged.AnalystRatingHold)
				}
			}

			// Verify Sell
			if tt.expectedSell == nil {
				if merged.AnalystRatingSell != nil {
					t.Errorf("Expected no Sell, but got %v", *merged.AnalystRatingSell)
				}
			} else {
				if merged.AnalystRatingSell == nil {
					t.Errorf("Expected Sell=%v, but got nil", *tt.expectedSell)
				} else if *merged.AnalystRatingSell != *tt.expectedSell {
					t.Errorf("Expected Sell=%v, got %v", *tt.expectedSell, *merged.AnalystRatingSell)
				}
			}

			// Verify Strong Sell
			if tt.expectedStrongSell == nil {
				if merged.AnalystRatingStrongSell != nil {
					t.Errorf("Expected no Strong Sell, but got %v", *merged.AnalystRatingStrongSell)
				}
			} else {
				if merged.AnalystRatingStrongSell == nil {
					t.Errorf("Expected Strong Sell=%v, but got nil", *tt.expectedStrongSell)
				} else if *merged.AnalystRatingStrongSell != *tt.expectedStrongSell {
					t.Errorf("Expected Strong Sell=%v, got %v", *tt.expectedStrongSell, *merged.AnalystRatingStrongSell)
				}
			}

			// Verify Consensus
			if tt.expectedConsensus == nil {
				if merged.AnalystConsensus != nil {
					t.Errorf("Expected no Consensus, but got %v", *merged.AnalystConsensus)
				}
			} else {
				if merged.AnalystConsensus == nil {
					t.Errorf("Expected Consensus=%v, but got nil", *tt.expectedConsensus)
				} else if *merged.AnalystConsensus != *tt.expectedConsensus {
					t.Errorf("Expected Consensus=%v, got %v", *tt.expectedConsensus, *merged.AnalystConsensus)
				}
			}
		})
	}
}

// TestFMPPriceTargetConsensusMerge tests that price target consensus data is correctly merged
func TestFMPPriceTargetConsensusMerge(t *testing.T) {
	tests := []struct {
		name                    string
		priceTargetConsensus    *FMPPriceTargetConsensus
		expectedTargetHigh      *float64
		expectedTargetLow       *float64
		expectedTargetConsensus *float64
		expectedTargetMedian    *float64
	}{
		{
			name: "Normal price targets",
			priceTargetConsensus: &FMPPriceTargetConsensus{
				Symbol:          "AAPL",
				TargetHigh:      float64Ptr(250.0),
				TargetLow:       float64Ptr(180.0),
				TargetConsensus: float64Ptr(220.0),
				TargetMedian:    float64Ptr(215.0),
			},
			expectedTargetHigh:      float64Ptr(250.0),
			expectedTargetLow:       float64Ptr(180.0),
			expectedTargetConsensus: float64Ptr(220.0),
			expectedTargetMedian:    float64Ptr(215.0),
		},
		{
			name: "Wide price target range",
			priceTargetConsensus: &FMPPriceTargetConsensus{
				Symbol:          "TSLA",
				TargetHigh:      float64Ptr(500.0),
				TargetLow:       float64Ptr(100.0),
				TargetConsensus: float64Ptr(300.0),
				TargetMedian:    float64Ptr(280.0),
			},
			expectedTargetHigh:      float64Ptr(500.0),
			expectedTargetLow:       float64Ptr(100.0),
			expectedTargetConsensus: float64Ptr(300.0),
			expectedTargetMedian:    float64Ptr(280.0),
		},
		{
			name:                    "Nil price target consensus",
			priceTargetConsensus:    nil,
			expectedTargetHigh:      nil,
			expectedTargetLow:       nil,
			expectedTargetConsensus: nil,
			expectedTargetMedian:    nil,
		},
		{
			name: "Partial data - only consensus",
			priceTargetConsensus: &FMPPriceTargetConsensus{
				Symbol:          "TEST",
				TargetHigh:      nil,
				TargetLow:       nil,
				TargetConsensus: float64Ptr(150.0),
				TargetMedian:    nil,
			},
			expectedTargetHigh:      nil,
			expectedTargetLow:       nil,
			expectedTargetConsensus: float64Ptr(150.0),
			expectedTargetMedian:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmpMetrics := &FMPAllMetrics{
				PriceTargetConsensus: tt.priceTargetConsensus,
				Errors:               make(map[string]error),
			}

			merged := MergeAllData(fmpMetrics, 100.0)

			// Verify Target High
			if tt.expectedTargetHigh == nil {
				if merged.TargetHigh != nil {
					t.Errorf("Expected no Target High, but got %v", *merged.TargetHigh)
				}
			} else {
				if merged.TargetHigh == nil {
					t.Errorf("Expected Target High=%v, but got nil", *tt.expectedTargetHigh)
				} else if *merged.TargetHigh != *tt.expectedTargetHigh {
					t.Errorf("Expected Target High=%v, got %v", *tt.expectedTargetHigh, *merged.TargetHigh)
				}
			}

			// Verify Target Low
			if tt.expectedTargetLow == nil {
				if merged.TargetLow != nil {
					t.Errorf("Expected no Target Low, but got %v", *merged.TargetLow)
				}
			} else {
				if merged.TargetLow == nil {
					t.Errorf("Expected Target Low=%v, but got nil", *tt.expectedTargetLow)
				} else if *merged.TargetLow != *tt.expectedTargetLow {
					t.Errorf("Expected Target Low=%v, got %v", *tt.expectedTargetLow, *merged.TargetLow)
				}
			}

			// Verify Target Consensus
			if tt.expectedTargetConsensus == nil {
				if merged.TargetConsensus != nil {
					t.Errorf("Expected no Target Consensus, but got %v", *merged.TargetConsensus)
				}
			} else {
				if merged.TargetConsensus == nil {
					t.Errorf("Expected Target Consensus=%v, but got nil", *tt.expectedTargetConsensus)
				} else if *merged.TargetConsensus != *tt.expectedTargetConsensus {
					t.Errorf("Expected Target Consensus=%v, got %v", *tt.expectedTargetConsensus, *merged.TargetConsensus)
				}
			}

			// Verify Target Median
			if tt.expectedTargetMedian == nil {
				if merged.TargetMedian != nil {
					t.Errorf("Expected no Target Median, but got %v", *merged.TargetMedian)
				}
			} else {
				if merged.TargetMedian == nil {
					t.Errorf("Expected Target Median=%v, but got nil", *tt.expectedTargetMedian)
				} else if *merged.TargetMedian != *tt.expectedTargetMedian {
					t.Errorf("Expected Target Median=%v, got %v", *tt.expectedTargetMedian, *merged.TargetMedian)
				}
			}
		})
	}
}

// TestCombinedAnalystRatingsData tests that both grades summary and price targets are merged correctly together
func TestCombinedAnalystRatingsData(t *testing.T) {
	fmpMetrics := &FMPAllMetrics{
		GradesSummary: &FMPGradesSummary{
			Symbol:     "AAPL",
			StrongBuy:  15,
			Buy:        8,
			Hold:       5,
			Sell:       1,
			StrongSell: 0,
			Consensus:  "Strong Buy",
		},
		PriceTargetConsensus: &FMPPriceTargetConsensus{
			Symbol:          "AAPL",
			TargetHigh:      float64Ptr(250.0),
			TargetLow:       float64Ptr(190.0),
			TargetConsensus: float64Ptr(225.0),
			TargetMedian:    float64Ptr(220.0),
		},
		Errors: make(map[string]error),
	}

	merged := MergeAllData(fmpMetrics, 200.0)

	// Verify ratings
	if merged.AnalystRatingStrongBuy == nil || *merged.AnalystRatingStrongBuy != 15 {
		t.Errorf("Expected Strong Buy=15, got %v", merged.AnalystRatingStrongBuy)
	}
	if merged.AnalystRatingBuy == nil || *merged.AnalystRatingBuy != 8 {
		t.Errorf("Expected Buy=8, got %v", merged.AnalystRatingBuy)
	}
	if merged.AnalystRatingHold == nil || *merged.AnalystRatingHold != 5 {
		t.Errorf("Expected Hold=5, got %v", merged.AnalystRatingHold)
	}
	if merged.AnalystConsensus == nil || *merged.AnalystConsensus != "Strong Buy" {
		t.Errorf("Expected Consensus='Strong Buy', got %v", merged.AnalystConsensus)
	}

	// Verify price targets
	if merged.TargetHigh == nil || *merged.TargetHigh != 250.0 {
		t.Errorf("Expected Target High=250.0, got %v", merged.TargetHigh)
	}
	if merged.TargetLow == nil || *merged.TargetLow != 190.0 {
		t.Errorf("Expected Target Low=190.0, got %v", merged.TargetLow)
	}
	if merged.TargetConsensus == nil || *merged.TargetConsensus != 225.0 {
		t.Errorf("Expected Target Consensus=225.0, got %v", merged.TargetConsensus)
	}
	if merged.TargetMedian == nil || *merged.TargetMedian != 220.0 {
		t.Errorf("Expected Target Median=220.0, got %v", merged.TargetMedian)
	}
}

// TestEmptyConsensusString tests that empty consensus strings are handled correctly
func TestEmptyConsensusString(t *testing.T) {
	fmpMetrics := &FMPAllMetrics{
		GradesSummary: &FMPGradesSummary{
			Symbol:     "TEST",
			StrongBuy:  0,
			Buy:        0,
			Hold:       0,
			Sell:       0,
			StrongSell: 0,
			Consensus:  "", // Empty consensus
		},
		Errors: make(map[string]error),
	}

	merged := MergeAllData(fmpMetrics, 100.0)

	// Verify that empty consensus string results in nil
	if merged.AnalystConsensus != nil {
		t.Errorf("Expected nil Consensus for empty string, got %v", *merged.AnalystConsensus)
	}

	// Ratings should still be set (even if all zero)
	if merged.AnalystRatingStrongBuy == nil || *merged.AnalystRatingStrongBuy != 0 {
		t.Errorf("Expected Strong Buy=0, got %v", merged.AnalystRatingStrongBuy)
	}
}

// TestCalculateTargetUpside tests the upside/downside calculation logic
func TestCalculateTargetUpside(t *testing.T) {
	tests := []struct {
		name           string
		currentPrice   float64
		targetPrice    *float64
		expectedUpside *float64
	}{
		{
			name:           "25% upside",
			currentPrice:   100.0,
			targetPrice:    float64Ptr(125.0),
			expectedUpside: float64Ptr(25.0),
		},
		{
			name:           "20% downside",
			currentPrice:   100.0,
			targetPrice:    float64Ptr(80.0),
			expectedUpside: float64Ptr(-20.0),
		},
		{
			name:           "No change",
			currentPrice:   100.0,
			targetPrice:    float64Ptr(100.0),
			expectedUpside: float64Ptr(0.0),
		},
		{
			name:           "100% upside",
			currentPrice:   50.0,
			targetPrice:    float64Ptr(100.0),
			expectedUpside: float64Ptr(100.0),
		},
		{
			name:           "Nil target",
			currentPrice:   100.0,
			targetPrice:    nil,
			expectedUpside: nil,
		},
		{
			name:           "Zero current price",
			currentPrice:   0.0,
			targetPrice:    float64Ptr(100.0),
			expectedUpside: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var upside *float64

			if tt.targetPrice != nil && tt.currentPrice > 0 {
				result := ((*tt.targetPrice - tt.currentPrice) / tt.currentPrice) * 100
				upside = &result
			}

			if tt.expectedUpside == nil {
				if upside != nil {
					t.Errorf("Expected no upside calculation, but got %v", *upside)
				}
			} else {
				if upside == nil {
					t.Errorf("Expected upside=%v, but got nil", *tt.expectedUpside)
				} else if *upside != *tt.expectedUpside {
					t.Errorf("Expected upside=%v, got %v", *tt.expectedUpside, *upside)
				}
			}
		})
	}
}

// Helper function to create int pointers
func intPtr(i int) *int {
	return &i
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
