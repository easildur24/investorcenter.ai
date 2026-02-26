package evaluator

import (
	"encoding/json"
	"fmt"

	"notification-service/models"
)

// evaluateVolumeAbove returns true if the current volume >= threshold.
func evaluateVolumeAbove(alert *models.AlertRule, quote *models.SymbolQuote) (bool, error) {
	var cond models.ThresholdCondition
	if err := json.Unmarshal(alert.Conditions, &cond); err != nil {
		return false, fmt.Errorf("parse volume_above conditions: %w", err)
	}
	if cond.Threshold <= 0 {
		return false, fmt.Errorf("invalid threshold: %f", cond.Threshold)
	}
	return float64(quote.Volume) >= cond.Threshold, nil
}

// evaluateVolumeBelow returns true if the current volume <= threshold.
func evaluateVolumeBelow(alert *models.AlertRule, quote *models.SymbolQuote) (bool, error) {
	var cond models.ThresholdCondition
	if err := json.Unmarshal(alert.Conditions, &cond); err != nil {
		return false, fmt.Errorf("parse volume_below conditions: %w", err)
	}
	if cond.Threshold <= 0 {
		return false, fmt.Errorf("invalid threshold: %f", cond.Threshold)
	}
	return float64(quote.Volume) <= cond.Threshold, nil
}

// Note: volume_spike evaluation is not yet implemented. It requires
// historical volume baselines (e.g., 30-day average) which are stored
// in the database. This will be added in Phase 5.
