/**
 * Financial Formatting Utilities
 *
 * Utilities for formatting financial values, percentages, and changes
 * in financial statements and reports.
 */

/**
 * Format a financial value with appropriate abbreviation
 *
 * @param value The numeric value to format
 * @param format The format type: 'currency', 'number', or 'percent'
 * @param decimals Number of decimal places (default: 2)
 * @returns Formatted string
 */
export function formatFinancialValue(
  value: number | string | null | undefined,
  format: 'currency' | 'number' | 'percent',
  decimals: number = 2
): string {
  if (value === null || value === undefined || value === '') {
    return '—';
  }

  const numValue = typeof value === 'string' ? parseFloat(value) : value;

  if (isNaN(numValue)) {
    return '—';
  }

  if (format === 'currency') {
    return formatCurrencyValue(numValue, decimals);
  }

  if (format === 'percent') {
    return formatPercentValue(numValue, decimals);
  }

  // Number format
  if (Math.abs(numValue) >= 1e9) {
    return (numValue / 1e9).toFixed(decimals) + 'B';
  }
  if (Math.abs(numValue) >= 1e6) {
    return (numValue / 1e6).toFixed(decimals) + 'M';
  }
  if (Math.abs(numValue) >= 1e3) {
    return (numValue / 1e3).toFixed(decimals) + 'K';
  }

  return numValue.toFixed(decimals);
}

/**
 * Format a currency value with appropriate abbreviation
 */
export function formatCurrencyValue(value: number, decimals: number = 2): string {
  const absValue = Math.abs(value);
  const sign = value < 0 ? '-' : '';

  if (absValue >= 1e12) {
    return `${sign}$${(absValue / 1e12).toFixed(decimals)}T`;
  }
  if (absValue >= 1e9) {
    return `${sign}$${(absValue / 1e9).toFixed(decimals)}B`;
  }
  if (absValue >= 1e6) {
    return `${sign}$${(absValue / 1e6).toFixed(decimals)}M`;
  }
  if (absValue >= 1e3) {
    return `${sign}$${(absValue / 1e3).toFixed(decimals)}K`;
  }

  return `${sign}$${absValue.toFixed(decimals)}`;
}

/**
 * Format a percentage value
 */
export function formatPercentValue(value: number, decimals: number = 2): string {
  // Check if value is already in percentage form (0-100) or decimal form (0-1)
  const percentValue = Math.abs(value) <= 1 ? value * 100 : value;
  return `${percentValue.toFixed(decimals)}%`;
}

/**
 * Format a year-over-year change value
 *
 * @param change The YoY change as a decimal (e.g., 0.1 for 10%)
 * @returns Object with formatted text and color class
 */
export function formatYoYChange(change: number | null | undefined): {
  text: string;
  color: string;
  bgColor: string;
} {
  if (change === null || change === undefined || isNaN(change)) {
    return {
      text: '—',
      color: 'text-ic-text-muted',
      bgColor: 'bg-ic-surface',
    };
  }

  const percentage = (change * 100).toFixed(1);
  const sign = change >= 0 ? '+' : '';

  return {
    text: `${sign}${percentage}%`,
    color: change >= 0 ? 'text-green-600' : 'text-red-600',
    bgColor: change >= 0 ? 'bg-green-50' : 'bg-red-50',
  };
}

/**
 * Format a large number with commas
 */
export function formatWithCommas(value: number | string | null | undefined): string {
  if (value === null || value === undefined) {
    return '—';
  }

  const numValue = typeof value === 'string' ? parseFloat(value) : value;

  if (isNaN(numValue)) {
    return '—';
  }

  return new Intl.NumberFormat('en-US').format(numValue);
}

/**
 * Format a number in compact notation (e.g., 1.2M, 3.5B)
 */
export function formatCompact(value: number | string | null | undefined): string {
  if (value === null || value === undefined) {
    return '—';
  }

  const numValue = typeof value === 'string' ? parseFloat(value) : value;

  if (isNaN(numValue)) {
    return '—';
  }

  return new Intl.NumberFormat('en-US', {
    notation: 'compact',
    maximumFractionDigits: 2,
  }).format(numValue);
}

/**
 * Format EPS or other per-share values
 */
export function formatEPS(value: number | null | undefined, decimals: number = 2): string {
  if (value === null || value === undefined) {
    return '—';
  }

  const sign = value >= 0 ? '' : '-';
  return `${sign}$${Math.abs(value).toFixed(decimals)}`;
}

/**
 * Get the appropriate color class for a value based on whether positive is good
 *
 * @param value The numeric value
 * @param positiveIsGood Whether positive values should be green (true) or red (false)
 */
export function getValueColor(
  value: number | null | undefined,
  positiveIsGood: boolean = true
): string {
  if (value === null || value === undefined) {
    return 'text-ic-text-dim';
  }

  if (value === 0) {
    return 'text-ic-text-muted';
  }

  const isPositive = value > 0;

  if (positiveIsGood) {
    return isPositive ? 'text-green-600' : 'text-red-600';
  } else {
    return isPositive ? 'text-red-600' : 'text-green-600';
  }
}

/**
 * Determine if a financial metric should show green for positive values
 */
export function isPositiveGoodForMetric(metricKey: string): boolean {
  // Metrics where positive is BAD (expenses, debt, etc.)
  const negativeIsGood = [
    'cost_of_revenue',
    'operating_expenses',
    'selling_general_and_administrative_expenses',
    'research_and_development',
    'interest_expense_operating',
    'income_tax_expense_benefit',
    'total_liabilities',
    'current_liabilities',
    'noncurrent_liabilities',
    'long_term_debt',
    'short_term_debt',
    'capital_expenditure',
    'purchase_of_investment_securities',
    'payment_of_dividends',
    'repurchase_of_common_stock',
    'repayment_of_debt',
    'debt_to_equity',
    'debt_to_assets',
  ];

  return !negativeIsGood.includes(metricKey);
}

/**
 * Format a date string for financial statements
 */
export function formatStatementDate(dateString: string | null | undefined): string {
  if (!dateString) {
    return '—';
  }

  try {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    });
  } catch {
    return dateString;
  }
}

/**
 * Calculate and format gross margin from revenue and cost of revenue
 */
export function calculateGrossMargin(
  revenue: number | null | undefined,
  costOfRevenue: number | null | undefined
): { value: number | null; formatted: string } {
  if (revenue === null || revenue === undefined || revenue === 0) {
    return { value: null, formatted: '—' };
  }

  if (costOfRevenue === null || costOfRevenue === undefined) {
    return { value: null, formatted: '—' };
  }

  const margin = ((revenue - costOfRevenue) / revenue) * 100;
  return {
    value: margin,
    formatted: `${margin.toFixed(1)}%`,
  };
}

/**
 * Calculate and format operating margin
 */
export function calculateOperatingMargin(
  operatingIncome: number | null | undefined,
  revenue: number | null | undefined
): { value: number | null; formatted: string } {
  if (revenue === null || revenue === undefined || revenue === 0) {
    return { value: null, formatted: '—' };
  }

  if (operatingIncome === null || operatingIncome === undefined) {
    return { value: null, formatted: '—' };
  }

  const margin = (operatingIncome / revenue) * 100;
  return {
    value: margin,
    formatted: `${margin.toFixed(1)}%`,
  };
}

/**
 * Calculate and format net margin
 */
export function calculateNetMargin(
  netIncome: number | null | undefined,
  revenue: number | null | undefined
): { value: number | null; formatted: string } {
  if (revenue === null || revenue === undefined || revenue === 0) {
    return { value: null, formatted: '—' };
  }

  if (netIncome === null || netIncome === undefined) {
    return { value: null, formatted: '—' };
  }

  const margin = (netIncome / revenue) * 100;
  return {
    value: margin,
    formatted: `${margin.toFixed(1)}%`,
  };
}

/**
 * Generate trend data for sparklines
 *
 * @param periods Array of financial periods
 * @param metricKey The metric to extract
 * @returns Array of numbers for sparkline (oldest to newest)
 */
export function extractTrendData(
  periods: { data: Record<string, unknown> }[],
  metricKey: string
): number[] {
  const data = periods
    .map((period) => {
      const value = period.data[metricKey];
      return typeof value === 'number' ? value : null;
    })
    .filter((v): v is number => v !== null)
    .reverse(); // Reverse to get chronological order (oldest first)

  return data;
}

/**
 * Export financial data to CSV format
 *
 * @param periods Array of financial periods
 * @param rows Row configuration
 * @param ticker Stock ticker
 * @param statementType Statement type name
 * @returns CSV string
 */
export function exportToCSV(
  periods: { fiscal_year: number; fiscal_quarter?: number | null; data: Record<string, unknown> }[],
  rows: { key: string; label: string; format: string }[],
  ticker: string,
  statementType: string
): string {
  // Create headers
  const headers = [
    'Metric',
    ...periods.map((p) =>
      p.fiscal_quarter ? `Q${p.fiscal_quarter} ${p.fiscal_year}` : `FY ${p.fiscal_year}`
    ),
  ];

  // Create rows
  const csvRows = rows.map((row) => {
    const values = periods.map((p) => {
      const value = p.data[row.key];
      if (value === null || value === undefined) return '';
      if (typeof value === 'number') return value.toString();
      return String(value);
    });
    return [row.label, ...values];
  });

  // Combine headers and rows
  const allRows = [headers, ...csvRows];

  // Convert to CSV string
  return allRows.map((row) => row.map((cell) => `"${cell}"`).join(',')).join('\n');
}

/**
 * Download a string as a file
 */
export function downloadFile(content: string, filename: string, mimeType: string): void {
  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}
