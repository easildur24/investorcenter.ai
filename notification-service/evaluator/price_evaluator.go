package evaluator

import (
	"encoding/json"
	"fmt"
	"math"

	"notification-service/models"
)

// evaluatePriceAbove returns true if the current price >= threshold.
func evaluatePriceAbove(alert *models.AlertRule, quote *models.SymbolQuote) (bool, error) {
	var cond models.ThresholdCondition
	if err := json.Unmarshal(alert.Conditions, &cond); err != nil {
		return false, fmt.Errorf("parse price_above conditions: %w", err)
	}
	if cond.Threshold <= 0 {
		return false, fmt.Errorf("invalid threshold: %f", cond.Threshold)
	}
	return quote.Price >= cond.Threshold, nil
}

// evaluatePriceBelow returns true if the current price <= threshold.
func evaluatePriceBelow(alert *models.AlertRule, quote *models.SymbolQuote) (bool, error) {
	var cond models.ThresholdCondition
	if err := json.Unmarshal(alert.Conditions, &cond); err != nil {
		return false, fmt.Errorf("parse price_below conditions: %w", err)
	}
	if cond.Threshold <= 0 {
		return false, fmt.Errorf("invalid threshold: %f", cond.Threshold)
	}
	return quote.Price <= cond.Threshold, nil
}

// evaluatePriceChangePct returns true if the absolute change percentage
// meets or exceeds the configured threshold.
func evaluatePriceChangePct(alert *models.AlertRule, quote *models.SymbolQuote) (bool, error) {
	var cond models.PriceChangeCondition
	if err := json.Unmarshal(alert.Conditions, &cond); err != nil {
		return false, fmt.Errorf("parse price_change_pct conditions: %w", err)
	}
	if cond.PercentChange <= 0 {
		return false, fmt.Errorf("invalid percent_change: %f", cond.PercentChange)
	}

	absPct := math.Abs(quote.ChangePct)

	switch cond.Direction {
	case "up":
		return quote.ChangePct >= cond.PercentChange, nil
	case "down":
		return quote.ChangePct <= -cond.PercentChange, nil
	default: // "either" or empty
		return absPct >= cond.PercentChange, nil
	}
}
