package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// StatementType represents the type of financial statement
type StatementType string

const (
	StatementTypeIncome       StatementType = "income"
	StatementTypeBalanceSheet StatementType = "balance_sheet"
	StatementTypeCashFlow     StatementType = "cash_flow"
	StatementTypeRatios       StatementType = "ratios"
)

// Timeframe represents the reporting period type
type Timeframe string

const (
	TimeframeQuarterly Timeframe = "quarterly"
	TimeframeAnnual    Timeframe = "annual"
	TimeframeTTM       Timeframe = "trailing_twelve_months"
)

// FinancialData represents the JSONB data stored in the financial_statements table
type FinancialData map[string]interface{}

// Value implements the driver.Valuer interface for FinancialData
func (fd FinancialData) Value() (driver.Value, error) {
	if fd == nil {
		return nil, nil
	}
	return json.Marshal(fd)
}

// Scan implements the sql.Scanner interface for FinancialData
func (fd *FinancialData) Scan(value interface{}) error {
	if value == nil {
		*fd = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, fd)
}

// FinancialStatement represents a financial statement record in the database
type FinancialStatement struct {
	ID               int           `json:"id" db:"id"`
	TickerID         int           `json:"ticker_id" db:"ticker_id"`
	CIK              *string       `json:"cik,omitempty" db:"cik"`
	StatementType    StatementType `json:"statement_type" db:"statement_type"`
	Timeframe        Timeframe     `json:"timeframe" db:"timeframe"`
	FiscalYear       int           `json:"fiscal_year" db:"fiscal_year"`
	FiscalQuarter    *int          `json:"fiscal_quarter,omitempty" db:"fiscal_quarter"`
	PeriodStart      *time.Time    `json:"period_start,omitempty" db:"period_start"`
	PeriodEnd        time.Time     `json:"period_end" db:"period_end"`
	FiledDate        *time.Time    `json:"filed_date,omitempty" db:"filed_date"`
	SourceFilingURL  *string       `json:"source_filing_url,omitempty" db:"source_filing_url"`
	SourceFilingType *string       `json:"source_filing_type,omitempty" db:"source_filing_type"`
	Data             FinancialData `json:"data" db:"data"`
	CreatedAt        time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at" db:"updated_at"`
}

// FinancialPeriod represents a single period's financial data for API response
type FinancialPeriod struct {
	FiscalYear    int                    `json:"fiscal_year"`
	FiscalQuarter *int                   `json:"fiscal_quarter,omitempty"`
	PeriodEnd     string                 `json:"period_end"`
	FiledDate     *string                `json:"filed_date,omitempty"`
	Data          map[string]interface{} `json:"data"`
	YoYChange     map[string]*float64    `json:"yoy_change,omitempty"`
}

// FinancialsMetadata contains company metadata for financial statements
type FinancialsMetadata struct {
	CompanyName string  `json:"company_name"`
	CIK         *string `json:"cik,omitempty"`
	SIC         *string `json:"sic,omitempty"`
}

// FinancialsResponse represents the API response for financial statements
type FinancialsResponse struct {
	Ticker        string             `json:"ticker"`
	StatementType StatementType      `json:"statement_type"`
	Timeframe     Timeframe          `json:"timeframe"`
	Periods       []FinancialPeriod  `json:"periods"`
	Metadata      FinancialsMetadata `json:"metadata"`
}

// FinancialsParams represents query parameters for financial statements
type FinancialsParams struct {
	Ticker     string    `json:"ticker"`
	Timeframe  Timeframe `json:"timeframe"`
	Limit      int       `json:"limit"`
	FiscalYear *int      `json:"fiscal_year,omitempty"`
	Sort       string    `json:"sort"` // "asc" or "desc"
}

// PolygonFinancialsResponse represents the Polygon.io financials API response
type PolygonFinancialsResponse struct {
	Status    string                  `json:"status"`
	RequestID string                  `json:"request_id"`
	Count     int                     `json:"count"`
	Results   []PolygonFinancialsData `json:"results"`
	NextURL   *string                 `json:"next_url,omitempty"`
}

// PolygonFinancialsData represents a single financial statement from Polygon.io
type PolygonFinancialsData struct {
	Ticker           string                 `json:"ticker"`
	CIK              string                 `json:"cik"`
	CompanyName      string                 `json:"company_name"`
	StartDate        string                 `json:"start_date"`
	EndDate          string                 `json:"end_date"`
	FilingDate       string                 `json:"filing_date"`
	FiscalPeriod     string                 `json:"fiscal_period"` // "Q1", "Q2", "Q3", "Q4", "FY", "TTM"
	FiscalYear       string                 `json:"fiscal_year"`
	SourceFilingURL  string                 `json:"source_filing_url"`
	SourceFilingType string                 `json:"source_filing_type"`
	Financials       PolygonFinancialsItems `json:"financials"`
}

// PolygonFinancialsItems contains the financial statement data organized by statement type
type PolygonFinancialsItems struct {
	IncomeStatement     map[string]PolygonFinancialValue `json:"income_statement,omitempty"`
	BalanceSheet        map[string]PolygonFinancialValue `json:"balance_sheet,omitempty"`
	CashFlowStatement   map[string]PolygonFinancialValue `json:"cash_flow_statement,omitempty"`
	ComprehensiveIncome map[string]PolygonFinancialValue `json:"comprehensive_income,omitempty"`
}

// PolygonFinancialValue represents a single financial value from Polygon.io
type PolygonFinancialValue struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
	Label string  `json:"label"`
	Order int     `json:"order"`
}

// PolygonRatiosResponse represents the Polygon.io ratios API response
type PolygonRatiosResponse struct {
	Status    string              `json:"status"`
	RequestID string              `json:"request_id"`
	Count     int                 `json:"count"`
	Results   []PolygonRatiosData `json:"results"`
	NextURL   *string             `json:"next_url,omitempty"`
}

// PolygonRatiosData represents financial ratios from Polygon.io
type PolygonRatiosData struct {
	Ticker           string                 `json:"ticker"`
	CIK              string                 `json:"cik"`
	CompanyName      string                 `json:"company_name"`
	StartDate        string                 `json:"start_date"`
	EndDate          string                 `json:"end_date"`
	FilingDate       string                 `json:"filing_date"`
	FiscalPeriod     string                 `json:"fiscal_period"`
	FiscalYear       string                 `json:"fiscal_year"`
	SourceFilingURL  string                 `json:"source_filing_url"`
	SourceFilingType string                 `json:"source_filing_type"`
	Ratios           map[string]interface{} `json:"ratios"`
}

// Common income statement field mappings
var IncomeStatementFields = map[string]string{
	"revenues":           "Total Revenue",
	"cost_of_revenue":    "Cost of Revenue",
	"gross_profit":       "Gross Profit",
	"operating_expenses": "Operating Expenses",
	"selling_general_and_administrative_expenses": "SG&A Expenses",
	"research_and_development":                    "R&D Expenses",
	"operating_income_loss":                       "Operating Income",
	"interest_expense_operating":                  "Interest Expense",
	"income_loss_before_taxes":                    "Pre-tax Income",
	"income_tax_expense_benefit":                  "Income Tax",
	"net_income_loss":                             "Net Income",
	"basic_earnings_per_share":                    "Basic EPS",
	"diluted_earnings_per_share":                  "Diluted EPS",
	"basic_average_shares":                        "Basic Shares Outstanding",
	"diluted_average_shares":                      "Diluted Shares Outstanding",
}

// Common balance sheet field mappings
var BalanceSheetFields = map[string]string{
	"assets":                        "Total Assets",
	"current_assets":                "Current Assets",
	"cash_and_cash_equivalents":     "Cash & Equivalents",
	"accounts_receivable":           "Accounts Receivable",
	"inventory":                     "Inventory",
	"prepaid_expenses":              "Prepaid Expenses",
	"noncurrent_assets":             "Non-current Assets",
	"fixed_assets":                  "Property, Plant & Equipment",
	"intangible_assets":             "Intangible Assets",
	"goodwill":                      "Goodwill",
	"liabilities":                   "Total Liabilities",
	"current_liabilities":           "Current Liabilities",
	"accounts_payable":              "Accounts Payable",
	"short_term_debt":               "Short-term Debt",
	"noncurrent_liabilities":        "Non-current Liabilities",
	"long_term_debt":                "Long-term Debt",
	"equity":                        "Total Equity",
	"equity_attributable_to_parent": "Shareholders' Equity",
	"retained_earnings":             "Retained Earnings",
	"common_stock":                  "Common Stock",
}

// Common cash flow statement field mappings
var CashFlowFields = map[string]string{
	"net_cash_flow": "Net Change in Cash",
	"net_cash_flow_from_operating_activities": "Operating Cash Flow",
	"net_cash_flow_from_investing_activities": "Investing Cash Flow",
	"net_cash_flow_from_financing_activities": "Financing Cash Flow",
	"depreciation_and_amortization":           "Depreciation & Amortization",
	"capital_expenditure":                     "Capital Expenditure",
	"purchase_of_investment_securities":       "Investment Purchases",
	"sale_of_investment_securities":           "Investment Sales",
	"payment_of_dividends":                    "Dividends Paid",
	"repurchase_of_common_stock":              "Stock Buybacks",
	"issuance_of_common_stock":                "Stock Issuance",
	"issuance_of_debt":                        "Debt Issuance",
	"repayment_of_debt":                       "Debt Repayment",
}

// Financial ratio field mappings
var RatioFields = map[string]string{
	"return_on_equity":           "Return on Equity (ROE)",
	"return_on_assets":           "Return on Assets (ROA)",
	"return_on_invested_capital": "Return on Invested Capital",
	"gross_margin":               "Gross Margin",
	"operating_margin":           "Operating Margin",
	"net_profit_margin":          "Net Profit Margin",
	"current_ratio":              "Current Ratio",
	"quick_ratio":                "Quick Ratio",
	"debt_to_equity":             "Debt to Equity",
	"debt_to_assets":             "Debt to Assets",
	"interest_coverage":          "Interest Coverage",
	"asset_turnover":             "Asset Turnover",
	"inventory_turnover":         "Inventory Turnover",
	"receivables_turnover":       "Receivables Turnover",
	"price_to_earnings":          "P/E Ratio",
	"price_to_book":              "P/B Ratio",
	"price_to_sales":             "P/S Ratio",
	"enterprise_value_to_ebitda": "EV/EBITDA",
}
