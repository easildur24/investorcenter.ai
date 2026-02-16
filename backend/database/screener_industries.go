package database

import (
	"fmt"
	"strings"
)

// GetScreenerIndustries returns unique industry names from screener_data,
// optionally filtered by comma-separated sectors.
func GetScreenerIndustries(sectors string) ([]string, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not connected")
	}

	var industries []string

	if sectors == "" {
		// All industries
		err := DB.Select(&industries,
			`SELECT DISTINCT industry FROM screener_data
			 WHERE industry IS NOT NULL AND industry != ''
			 ORDER BY industry`)
		return industries, err
	}

	// Filter by sectors (trim whitespace and drop empty segments)
	raw := strings.Split(sectors, ",")
	sectorList := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s != "" {
			sectorList = append(sectorList, s)
		}
	}
	if len(sectorList) == 0 {
		// All segments were empty — fall back to unfiltered
		err := DB.Select(&industries,
			`SELECT DISTINCT industry FROM screener_data
			 WHERE industry IS NOT NULL AND industry != ''
			 ORDER BY industry`)
		return industries, err
	}

	placeholders := make([]string, len(sectorList))
	args := make([]interface{}, len(sectorList))
	for i, s := range sectorList {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = s
	}

	query := fmt.Sprintf(
		`SELECT DISTINCT industry FROM screener_data
		 WHERE industry IS NOT NULL AND industry != ''
		 AND sector IN (%s)
		 ORDER BY industry`,
		strings.Join(placeholders, ", "))

	err := DB.Select(&industries, query, args...)
	return industries, err
}
