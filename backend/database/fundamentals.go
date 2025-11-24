package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// FundamentalMetrics represents all calculated fundamental metrics
type FundamentalMetrics struct {
	Symbol    string    `json:"symbol"`
	UpdatedAt time.Time `json:"updated_at"`

	// Income Statement - TTM
	RevenueTTM   *float64 `json:"revenue_ttm"`
	NetIncomeTTM *float64 `json:"net_income_ttm"`
	EBITTTM      *float64 `json:"ebit_ttm"`
	EBITDATTM    *float64 `json:"ebitda_ttm"`

	// Income Statement - Quarterly
	RevenueQuarterly   *float64 `json:"revenue_quarterly"`
	NetIncomeQuarterly *float64 `json:"net_income_quarterly"`
	EBITQuarterly      *float64 `json:"ebit_quarterly"`
	EBITDAQuarterly    *float64 `json:"ebitda_quarterly"`

	// Growth Rates
	RevenueQoQGrowth *float64 `json:"revenue_qoq_growth"`
	EPSQoQGrowth     *float64 `json:"eps_qoq_growth"`
	EBITDAQoQGrowth  *float64 `json:"ebitda_qoq_growth"`

	// Balance Sheet
	TotalAssets                 *float64 `json:"total_assets"`
	TotalLiabilities            *float64 `json:"total_liabilities"`
	ShareholdersEquity          *float64 `json:"shareholders_equity"`
	CashAndShortTermInvestments *float64 `json:"cash_short_term_investments"`
	TotalLongTermAssets         *float64 `json:"total_long_term_assets"`
	TotalLongTermDebt           *float64 `json:"total_long_term_debt"`
	BookValue                   *float64 `json:"book_value"`

	// Cash Flow
	CashFromOperations      *float64 `json:"cash_from_operations"`
	CashFromInvesting       *float64 `json:"cash_from_investing"`
	CashFromFinancing       *float64 `json:"cash_from_financing"`
	ChangeInReceivables     *float64 `json:"change_in_receivables"`
	ChangesInWorkingCapital *float64 `json:"changes_in_working_capital"`
	CapitalExpenditures     *float64 `json:"capital_expenditures"`
	EndingCash              *float64 `json:"ending_cash"`
	FreeCashFlow            *float64 `json:"free_cash_flow"`

	// Calculated Ratios
	ReturnOnAssets          *float64 `json:"return_on_assets"`
	ReturnOnEquity          *float64 `json:"return_on_equity"`
	ReturnOnInvestedCapital *float64 `json:"return_on_invested_capital"`
	OperatingMargin         *float64 `json:"operating_margin"`
	GrossProfitMargin       *float64 `json:"gross_profit_margin"`

	// Common Size
	EPSDiluted        *float64 `json:"eps_diluted"`
	EPSBasic          *float64 `json:"eps_basic"`
	SharesOutstanding *float64 `json:"shares_outstanding"`

	// Market-based metrics (will be "needs data")
	MarketCap         string `json:"market_cap"`
	PERatio           string `json:"pe_ratio"`
	PriceToBook       string `json:"price_to_book"`
	DividendYield     string `json:"dividend_yield"`
	OneMonthReturns   string `json:"one_month_returns"`
	ThreeMonthReturns string `json:"three_month_returns"`
	SixMonthReturns   string `json:"six_month_returns"`
	YearToDateReturns string `json:"year_to_date_returns"`
	OneYearReturns    string `json:"one_year_returns"`
	ThreeYearReturns  string `json:"three_year_returns"`
	FiveYearReturns   string `json:"five_year_returns"`
	FiftyTwoWeekHigh  string `json:"fifty_two_week_high"`
	FiftyTwoWeekLow   string `json:"fifty_two_week_low"`
	Alpha5Y           string `json:"alpha_5y"`
	Beta5Y            string `json:"beta_5y"`

	// Employee metrics (usually in 10-K only)
	TotalEmployees       *float64 `json:"total_employees"`
	RevenuePerEmployee   *float64 `json:"revenue_per_employee"`
	NetIncomePerEmployee *float64 `json:"net_income_per_employee"`
}

// StoreFundamentalMetrics stores calculated fundamental metrics in the database
func StoreFundamentalMetrics(metrics *FundamentalMetrics) error {
	log.Printf("💾 Storing fundamental metrics for %s", metrics.Symbol)

	// Convert metrics to JSON
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %v", err)
	}

	// Upsert the metrics
	query := `
		INSERT INTO fundamental_metrics (symbol, metrics_data, updated_at) 
		VALUES ($1, $2, $3)
		ON CONFLICT (symbol) 
		DO UPDATE SET 
			metrics_data = EXCLUDED.metrics_data,
			updated_at = EXCLUDED.updated_at
	`

	_, err = DB.Exec(query, metrics.Symbol, metricsJSON, metrics.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to store metrics: %v", err)
	}

	log.Printf("✅ Successfully stored fundamental metrics for %s", metrics.Symbol)
	return nil
}

// GetFundamentalMetrics retrieves fundamental metrics from the database
func GetFundamentalMetrics(symbol string) (*FundamentalMetrics, error) {
	var metricsJSON []byte
	var updatedAt time.Time

	query := `SELECT metrics_data, updated_at FROM fundamental_metrics WHERE symbol = $1`
	err := DB.QueryRow(query, symbol).Scan(&metricsJSON, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, nil // No metrics found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %v", err)
	}

	var metrics FundamentalMetrics
	err = json.Unmarshal(metricsJSON, &metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics: %v", err)
	}

	return &metrics, nil
}

// ListAllFundamentalMetrics returns all symbols that have fundamental metrics
func ListAllFundamentalMetrics() ([]string, error) {
	query := `SELECT symbol FROM fundamental_metrics ORDER BY symbol`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list metrics: %v", err)
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			continue
		}
		symbols = append(symbols, symbol)
	}

	return symbols, nil
}

// GetMetricsAge returns how old the metrics are for a symbol
func GetMetricsAge(symbol string) (*time.Duration, error) {
	var updatedAt time.Time

	query := `SELECT updated_at FROM fundamental_metrics WHERE symbol = $1`
	err := DB.QueryRow(query, symbol).Scan(&updatedAt)

	if err == sql.ErrNoRows {
		return nil, nil // No metrics found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics age: %v", err)
	}

	age := time.Since(updatedAt)
	return &age, nil
}
