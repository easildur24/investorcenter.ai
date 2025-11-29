/**
 * Financial Statements Types
 *
 * Type definitions for financial statement data from SEC EDGAR filings
 */

// Statement types
export type StatementType = 'income' | 'balance_sheet' | 'cash_flow' | 'ratios';

// Timeframe options
export type Timeframe = 'quarterly' | 'annual' | 'trailing_twelve_months';

// Year-over-year change data
export type YoYChange = Record<string, number | null>;

// Financial data (key-value pairs of metric names and values)
export type FinancialData = Record<string, number | string | null>;

/**
 * Company metadata for financial statements
 */
export interface FinancialsMetadata {
  company_name: string;
  cik?: string | null;
  sic?: string | null;
}

/**
 * A single financial period (quarter or year)
 */
export interface FinancialPeriod {
  fiscal_year: number;
  fiscal_quarter?: number | null;
  period_end: string;
  filed_date?: string | null;
  data: FinancialData;
  yoy_change?: YoYChange | null;
}

/**
 * Response from financial statements API
 */
export interface FinancialsResponse {
  ticker: string;
  statement_type: StatementType;
  timeframe: Timeframe;
  periods: FinancialPeriod[];
  metadata: FinancialsMetadata;
}

/**
 * Combined response with all statement types
 */
export interface AllFinancialsResponse {
  ticker: string;
  timeframe: Timeframe;
  metadata: FinancialsMetadata;
  income: FinancialPeriod[] | null;
  balance: FinancialPeriod[] | null;
  cashflow: FinancialPeriod[] | null;
}

/**
 * API wrapper response
 */
export interface FinancialsAPIResponse {
  data: FinancialsResponse;
  meta: {
    timestamp: string;
  };
}

export interface AllFinancialsAPIResponse {
  data: AllFinancialsResponse;
  meta: {
    timestamp: string;
  };
}

/**
 * Row configuration for displaying financial data in tables
 */
export interface FinancialRowConfig {
  key: string;
  label: string;
  format: 'currency' | 'number' | 'percent';
  decimals?: number;
  bold?: boolean;
  indent?: number;
  calculated?: boolean;
  tooltip?: string;
}

/**
 * Income Statement row configuration
 */
export const incomeStatementRows: FinancialRowConfig[] = [
  { key: 'revenues', label: 'Total Revenue', format: 'currency', bold: true },
  { key: 'cost_of_revenue', label: 'Cost of Revenue', format: 'currency', indent: 1 },
  { key: 'gross_profit', label: 'Gross Profit', format: 'currency', bold: true },
  { key: 'operating_expenses', label: 'Operating Expenses', format: 'currency' },
  { key: 'selling_general_and_administrative_expenses', label: 'SG&A Expenses', format: 'currency', indent: 1 },
  { key: 'research_and_development', label: 'R&D Expenses', format: 'currency', indent: 1 },
  { key: 'operating_income_loss', label: 'Operating Income', format: 'currency', bold: true },
  { key: 'interest_expense_operating', label: 'Interest Expense', format: 'currency', indent: 1 },
  { key: 'income_loss_before_taxes', label: 'Pre-tax Income', format: 'currency' },
  { key: 'income_tax_expense_benefit', label: 'Income Tax', format: 'currency', indent: 1 },
  { key: 'net_income_loss', label: 'Net Income', format: 'currency', bold: true },
  { key: 'basic_earnings_per_share', label: 'Basic EPS', format: 'number', decimals: 2 },
  { key: 'diluted_earnings_per_share', label: 'Diluted EPS', format: 'number', decimals: 2 },
  { key: 'basic_average_shares', label: 'Basic Shares Outstanding', format: 'number', decimals: 0 },
  { key: 'diluted_average_shares', label: 'Diluted Shares Outstanding', format: 'number', decimals: 0 },
];

/**
 * Balance Sheet row configuration
 */
export const balanceSheetRows: FinancialRowConfig[] = [
  // Assets
  { key: 'assets', label: 'Total Assets', format: 'currency', bold: true },
  { key: 'current_assets', label: 'Current Assets', format: 'currency', indent: 1 },
  { key: 'cash_and_cash_equivalents', label: 'Cash & Equivalents', format: 'currency', indent: 2 },
  { key: 'accounts_receivable', label: 'Accounts Receivable', format: 'currency', indent: 2 },
  { key: 'inventory', label: 'Inventory', format: 'currency', indent: 2 },
  { key: 'prepaid_expenses', label: 'Prepaid Expenses', format: 'currency', indent: 2 },
  { key: 'noncurrent_assets', label: 'Non-current Assets', format: 'currency', indent: 1 },
  { key: 'fixed_assets', label: 'Property, Plant & Equipment', format: 'currency', indent: 2 },
  { key: 'intangible_assets', label: 'Intangible Assets', format: 'currency', indent: 2 },
  { key: 'goodwill', label: 'Goodwill', format: 'currency', indent: 2 },
  // Liabilities
  { key: 'liabilities', label: 'Total Liabilities', format: 'currency', bold: true },
  { key: 'current_liabilities', label: 'Current Liabilities', format: 'currency', indent: 1 },
  { key: 'accounts_payable', label: 'Accounts Payable', format: 'currency', indent: 2 },
  { key: 'short_term_debt', label: 'Short-term Debt', format: 'currency', indent: 2 },
  { key: 'noncurrent_liabilities', label: 'Non-current Liabilities', format: 'currency', indent: 1 },
  { key: 'long_term_debt', label: 'Long-term Debt', format: 'currency', indent: 2 },
  // Equity
  { key: 'equity', label: 'Total Equity', format: 'currency', bold: true },
  { key: 'equity_attributable_to_parent', label: "Shareholders' Equity", format: 'currency', indent: 1 },
  { key: 'retained_earnings', label: 'Retained Earnings', format: 'currency', indent: 2 },
  { key: 'common_stock', label: 'Common Stock', format: 'currency', indent: 2 },
];

/**
 * Cash Flow Statement row configuration
 */
export const cashFlowRows: FinancialRowConfig[] = [
  { key: 'net_cash_flow_from_operating_activities', label: 'Operating Cash Flow', format: 'currency', bold: true },
  { key: 'depreciation_and_amortization', label: 'Depreciation & Amortization', format: 'currency', indent: 1 },
  { key: 'net_cash_flow_from_investing_activities', label: 'Investing Cash Flow', format: 'currency', bold: true },
  { key: 'capital_expenditure', label: 'Capital Expenditure', format: 'currency', indent: 1 },
  { key: 'purchase_of_investment_securities', label: 'Investment Purchases', format: 'currency', indent: 1 },
  { key: 'sale_of_investment_securities', label: 'Investment Sales', format: 'currency', indent: 1 },
  { key: 'net_cash_flow_from_financing_activities', label: 'Financing Cash Flow', format: 'currency', bold: true },
  { key: 'payment_of_dividends', label: 'Dividends Paid', format: 'currency', indent: 1 },
  { key: 'repurchase_of_common_stock', label: 'Stock Buybacks', format: 'currency', indent: 1 },
  { key: 'issuance_of_common_stock', label: 'Stock Issuance', format: 'currency', indent: 1 },
  { key: 'issuance_of_debt', label: 'Debt Issuance', format: 'currency', indent: 1 },
  { key: 'repayment_of_debt', label: 'Debt Repayment', format: 'currency', indent: 1 },
  { key: 'net_cash_flow', label: 'Net Change in Cash', format: 'currency', bold: true },
  { key: 'free_cash_flow', label: 'Free Cash Flow', format: 'currency', bold: true, calculated: true, tooltip: 'Operating Cash Flow - Capital Expenditure' },
];

/**
 * Financial Ratios row configuration
 */
export const ratioRows: FinancialRowConfig[] = [
  // Profitability
  { key: 'return_on_equity', label: 'Return on Equity (ROE)', format: 'percent', bold: true },
  { key: 'return_on_assets', label: 'Return on Assets (ROA)', format: 'percent' },
  { key: 'return_on_invested_capital', label: 'Return on Invested Capital', format: 'percent' },
  { key: 'gross_margin', label: 'Gross Margin', format: 'percent' },
  { key: 'operating_margin', label: 'Operating Margin', format: 'percent' },
  { key: 'net_profit_margin', label: 'Net Profit Margin', format: 'percent' },
  // Liquidity
  { key: 'current_ratio', label: 'Current Ratio', format: 'number', decimals: 2, bold: true },
  { key: 'quick_ratio', label: 'Quick Ratio', format: 'number', decimals: 2 },
  // Leverage
  { key: 'debt_to_equity', label: 'Debt to Equity', format: 'number', decimals: 2, bold: true },
  { key: 'debt_to_assets', label: 'Debt to Assets', format: 'percent' },
  { key: 'interest_coverage', label: 'Interest Coverage', format: 'number', decimals: 2 },
  // Efficiency
  { key: 'asset_turnover', label: 'Asset Turnover', format: 'number', decimals: 2, bold: true },
  { key: 'inventory_turnover', label: 'Inventory Turnover', format: 'number', decimals: 2 },
  { key: 'receivables_turnover', label: 'Receivables Turnover', format: 'number', decimals: 2 },
  // Valuation
  { key: 'price_to_earnings', label: 'P/E Ratio', format: 'number', decimals: 2, bold: true },
  { key: 'price_to_book', label: 'P/B Ratio', format: 'number', decimals: 2 },
  { key: 'price_to_sales', label: 'P/S Ratio', format: 'number', decimals: 2 },
  { key: 'enterprise_value_to_ebitda', label: 'EV/EBITDA', format: 'number', decimals: 2 },
];

/**
 * Get row configuration for a specific statement type
 */
export function getRowConfigForStatementType(statementType: StatementType): FinancialRowConfig[] {
  switch (statementType) {
    case 'income':
      return incomeStatementRows;
    case 'balance_sheet':
      return balanceSheetRows;
    case 'cash_flow':
      return cashFlowRows;
    case 'ratios':
      return ratioRows;
    default:
      return [];
  }
}

/**
 * Statement type display labels
 */
export const statementTypeLabels: Record<StatementType, string> = {
  income: 'Income Statement',
  balance_sheet: 'Balance Sheet',
  cash_flow: 'Cash Flow',
  ratios: 'Ratios',
};

/**
 * Timeframe display labels
 */
export const timeframeLabels: Record<Timeframe, string> = {
  quarterly: 'Quarterly',
  annual: 'Annual',
  trailing_twelve_months: 'TTM',
};

/**
 * Format a period label (e.g., "Q4 2024" or "FY 2024")
 */
export function formatPeriodLabel(period: FinancialPeriod, timeframe: Timeframe): string {
  if (timeframe === 'quarterly' && period.fiscal_quarter) {
    return `Q${period.fiscal_quarter} ${period.fiscal_year}`;
  }
  if (timeframe === 'annual') {
    return `FY ${period.fiscal_year}`;
  }
  if (timeframe === 'trailing_twelve_months') {
    return `TTM ${period.fiscal_year}`;
  }
  return period.period_end;
}
