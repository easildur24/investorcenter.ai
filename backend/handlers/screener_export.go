package handlers

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"time"

	"investorcenter-api/database"

	"github.com/gin-gonic/gin"
)

const (
	// maxExportRows caps the CSV export at 10,000 rows to prevent abuse.
	maxExportRows = 10000
)

// ExportScreenerCSV handles CSV export of screener results.
// GET /api/v1/screener/stocks/export
//
// Accepts the same filter params as GetScreenerStocks but ignores
// page/limit and streams all matching rows (up to maxExportRows) as CSV.
func ExportScreenerCSV(c *gin.Context) {
	if database.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Database not available",
			"message": "Export service is temporarily unavailable",
		})
		return
	}

	// Reuse the same param parser but override pagination
	params := parseScreenerParams(c)
	params.Page = 1
	params.Limit = maxExportRows

	stocks, _, err := database.GetScreenerStocks(params)
	if err != nil {
		log.Printf("Error exporting screener stocks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to export stocks",
			"message": "An error occurred while generating the CSV export",
		})
		return
	}

	filename := fmt.Sprintf("screener-export-%s.csv", time.Now().UTC().Format("2006-01-02"))
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	w := csv.NewWriter(c.Writer)

	// Header row
	header := []string{
		"Symbol", "Name", "Sector", "Industry",
		"Market Cap", "Price",
		"P/E Ratio", "P/B Ratio", "P/S Ratio",
		"ROE (%)", "ROA (%)", "Gross Margin (%)", "Operating Margin (%)", "Net Margin (%)",
		"Debt/Equity", "Current Ratio",
		"Revenue Growth (%)", "EPS Growth (%)",
		"Dividend Yield (%)", "Payout Ratio (%)", "Consec. Div Years",
		"Beta", "DCF Upside (%)",
		"IC Score", "IC Rating",
		"Value Score", "Growth Score", "Profitability Score",
		"Financial Health Score", "Momentum Score",
		"Analyst Score", "Insider Score", "Institutional Score",
		"Sentiment Score", "Technical Score",
	}
	if err := w.Write(header); err != nil {
		log.Printf("CSV header write error: %v", err)
		return
	}

	for _, s := range stocks {
		row := []string{
			s.Symbol,
			s.Name,
			ptrStr(s.Sector),
			ptrStr(s.Industry),
			fmtFloat(s.MarketCap, 0),
			fmtFloat(s.Price, 2),
			fmtFloat(s.PERatio, 2),
			fmtFloat(s.PBRatio, 2),
			fmtFloat(s.PSRatio, 2),
			fmtFloat(s.ROE, 2),
			fmtFloat(s.ROA, 2),
			fmtFloat(s.GrossMargin, 2),
			fmtFloat(s.OperatingMargin, 2),
			fmtFloat(s.NetMargin, 2),
			fmtFloat(s.DebtToEquity, 2),
			fmtFloat(s.CurrentRatio, 2),
			fmtFloat(s.RevenueGrowth, 2),
			fmtFloat(s.EPSGrowthYoY, 2),
			fmtFloat(s.DividendYield, 2),
			fmtFloat(s.PayoutRatio, 2),
			fmtInt(s.ConsecutiveDividendYears),
			fmtFloat(s.Beta, 2),
			fmtFloat(s.DCFUpsidePercent, 2),
			fmtFloat(s.ICScore, 1),
			ptrStr(s.ICRating),
			fmtFloat(s.ValueScore, 1),
			fmtFloat(s.GrowthScore, 1),
			fmtFloat(s.ProfitabilityScore, 1),
			fmtFloat(s.FinancialHealthScore, 1),
			fmtFloat(s.MomentumScore, 1),
			fmtFloat(s.AnalystConsensusScore, 1),
			fmtFloat(s.InsiderActivityScore, 1),
			fmtFloat(s.InstitutionalScore, 1),
			fmtFloat(s.NewsSentimentScore, 1),
			fmtFloat(s.TechnicalScore, 1),
		}
		if err := w.Write(row); err != nil {
			log.Printf("CSV row write error: %v", err)
			return
		}
	}

	w.Flush()
}

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func fmtFloat(p *float64, decimals int) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%.*f", decimals, *p)
}

func fmtInt(p *int) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%d", *p)
}
