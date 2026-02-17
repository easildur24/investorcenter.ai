import {
  formatFinancialValue,
  formatCurrencyValue,
  formatPercentValue,
  formatYoYChange,
  formatWithCommas,
  formatCompact,
  formatEPS,
  getValueColor,
  isPositiveGoodForMetric,
  formatStatementDate,
  calculateGrossMargin,
  calculateOperatingMargin,
  calculateNetMargin,
  extractTrendData,
  exportToCSV,
} from '../financial';

describe('formatFinancialValue', () => {
  it('returns "—" for null', () => {
    expect(formatFinancialValue(null, 'currency')).toBe('—');
  });

  it('returns "—" for undefined', () => {
    expect(formatFinancialValue(undefined, 'number')).toBe('—');
  });

  it('returns "—" for empty string', () => {
    expect(formatFinancialValue('', 'currency')).toBe('—');
  });

  it('returns "—" for NaN string', () => {
    expect(formatFinancialValue('not-a-number', 'currency')).toBe('—');
  });

  it('formats currency values', () => {
    expect(formatFinancialValue(1500000, 'currency')).toBe('$1.50M');
  });

  it('formats number with abbreviations', () => {
    expect(formatFinancialValue(2500000000, 'number')).toBe('2.50B');
    expect(formatFinancialValue(1500000, 'number')).toBe('1.50M');
    expect(formatFinancialValue(5000, 'number')).toBe('5.00K');
    expect(formatFinancialValue(42, 'number')).toBe('42.00');
  });

  it('formats percent values', () => {
    expect(formatFinancialValue(0.15, 'percent')).toBe('15.00%');
  });

  it('parses string values', () => {
    expect(formatFinancialValue('1000000', 'currency')).toBe('$1.00M');
  });

  it('respects custom decimals', () => {
    expect(formatFinancialValue(1234567, 'number', 1)).toBe('1.2M');
  });
});

describe('formatCurrencyValue', () => {
  it('formats trillions', () => {
    expect(formatCurrencyValue(1500000000000)).toBe('$1.50T');
  });

  it('formats billions', () => {
    expect(formatCurrencyValue(2500000000)).toBe('$2.50B');
  });

  it('formats millions', () => {
    expect(formatCurrencyValue(5000000)).toBe('$5.00M');
  });

  it('formats thousands', () => {
    expect(formatCurrencyValue(7500)).toBe('$7.50K');
  });

  it('formats small values', () => {
    expect(formatCurrencyValue(42.5)).toBe('$42.50');
  });

  it('handles negative values', () => {
    expect(formatCurrencyValue(-3000000000)).toBe('-$3.00B');
  });

  it('handles zero', () => {
    expect(formatCurrencyValue(0)).toBe('$0.00');
  });
});

describe('formatPercentValue', () => {
  it('converts decimal to percentage', () => {
    expect(formatPercentValue(0.15)).toBe('15.00%');
  });

  it('passes through values already in percentage form', () => {
    expect(formatPercentValue(25)).toBe('25.00%');
  });

  it('handles negative decimals', () => {
    expect(formatPercentValue(-0.05)).toBe('-5.00%');
  });

  it('handles zero', () => {
    expect(formatPercentValue(0)).toBe('0.00%');
  });

  it('handles exactly 1 (100%)', () => {
    expect(formatPercentValue(1)).toBe('100.00%');
  });
});

describe('formatYoYChange', () => {
  it('returns "—" for null', () => {
    const result = formatYoYChange(null);
    expect(result.text).toBe('—');
    expect(result.color).toBe('text-ic-text-muted');
  });

  it('returns "—" for undefined', () => {
    const result = formatYoYChange(undefined);
    expect(result.text).toBe('—');
  });

  it('returns "—" for NaN', () => {
    const result = formatYoYChange(NaN);
    expect(result.text).toBe('—');
  });

  it('formats positive change with green color', () => {
    const result = formatYoYChange(0.15);
    expect(result.text).toBe('+15.0%');
    expect(result.color).toBe('text-green-600');
    expect(result.bgColor).toBe('bg-green-50');
  });

  it('formats negative change with red color', () => {
    const result = formatYoYChange(-0.08);
    expect(result.text).toBe('-8.0%');
    expect(result.color).toBe('text-red-600');
    expect(result.bgColor).toBe('bg-red-50');
  });

  it('formats zero change as positive', () => {
    const result = formatYoYChange(0);
    expect(result.text).toBe('+0.0%');
    expect(result.color).toBe('text-green-600');
  });
});

describe('formatWithCommas', () => {
  it('returns "—" for null', () => {
    expect(formatWithCommas(null)).toBe('—');
  });

  it('returns "—" for undefined', () => {
    expect(formatWithCommas(undefined)).toBe('—');
  });

  it('formats large numbers with commas', () => {
    expect(formatWithCommas(1234567)).toBe('1,234,567');
  });

  it('formats string numbers', () => {
    expect(formatWithCommas('9876543')).toBe('9,876,543');
  });

  it('returns "—" for NaN string', () => {
    expect(formatWithCommas('abc')).toBe('—');
  });
});

describe('formatCompact', () => {
  it('returns "—" for null', () => {
    expect(formatCompact(null)).toBe('—');
  });

  it('returns "—" for undefined', () => {
    expect(formatCompact(undefined)).toBe('—');
  });

  it('formats large numbers in compact notation', () => {
    const result = formatCompact(1500000);
    // Intl.NumberFormat compact notation: "1.5M"
    expect(result).toMatch(/1\.5M/);
  });

  it('handles string input', () => {
    const result = formatCompact('2500000000');
    expect(result).toMatch(/2\.5B/);
  });

  it('returns "—" for NaN string', () => {
    expect(formatCompact('not-a-number')).toBe('—');
  });
});

describe('formatEPS', () => {
  it('returns "—" for null', () => {
    expect(formatEPS(null)).toBe('—');
  });

  it('returns "—" for undefined', () => {
    expect(formatEPS(undefined)).toBe('—');
  });

  it('formats positive EPS', () => {
    expect(formatEPS(3.45)).toBe('$3.45');
  });

  it('formats negative EPS', () => {
    expect(formatEPS(-1.23)).toBe('-$1.23');
  });

  it('formats zero EPS', () => {
    expect(formatEPS(0)).toBe('$0.00');
  });

  it('respects custom decimals', () => {
    expect(formatEPS(3.456, 3)).toBe('$3.456');
  });
});

describe('getValueColor', () => {
  it('returns dim for null', () => {
    expect(getValueColor(null)).toBe('text-ic-text-dim');
  });

  it('returns dim for undefined', () => {
    expect(getValueColor(undefined)).toBe('text-ic-text-dim');
  });

  it('returns muted for zero', () => {
    expect(getValueColor(0)).toBe('text-ic-text-muted');
  });

  it('returns green for positive when positiveIsGood', () => {
    expect(getValueColor(10, true)).toBe('text-green-600');
  });

  it('returns red for negative when positiveIsGood', () => {
    expect(getValueColor(-10, true)).toBe('text-red-600');
  });

  it('returns red for positive when positiveIsGood=false', () => {
    expect(getValueColor(10, false)).toBe('text-red-600');
  });

  it('returns green for negative when positiveIsGood=false', () => {
    expect(getValueColor(-10, false)).toBe('text-green-600');
  });
});

describe('isPositiveGoodForMetric', () => {
  it('returns true for revenue metrics', () => {
    expect(isPositiveGoodForMetric('revenue')).toBe(true);
    expect(isPositiveGoodForMetric('net_income')).toBe(true);
    expect(isPositiveGoodForMetric('total_assets')).toBe(true);
  });

  it('returns false for expense/debt metrics', () => {
    expect(isPositiveGoodForMetric('cost_of_revenue')).toBe(false);
    expect(isPositiveGoodForMetric('operating_expenses')).toBe(false);
    expect(isPositiveGoodForMetric('long_term_debt')).toBe(false);
    expect(isPositiveGoodForMetric('debt_to_equity')).toBe(false);
    expect(isPositiveGoodForMetric('capital_expenditure')).toBe(false);
  });
});

describe('formatStatementDate', () => {
  it('returns "—" for null', () => {
    expect(formatStatementDate(null)).toBe('—');
  });

  it('returns "—" for undefined', () => {
    expect(formatStatementDate(undefined)).toBe('—');
  });

  it('formats valid date string', () => {
    // Use a date with timezone to avoid UTC offset issues
    const result = formatStatementDate('2024-03-15T12:00:00');
    expect(result).toMatch(/Mar/);
    expect(result).toMatch(/2024/);
    // The day may vary by timezone so just check it contains a number
    expect(result).toMatch(/\d+/);
  });
});

describe('calculateGrossMargin', () => {
  it('returns null for null revenue', () => {
    const result = calculateGrossMargin(null, 100);
    expect(result.value).toBeNull();
    expect(result.formatted).toBe('—');
  });

  it('returns null for zero revenue', () => {
    const result = calculateGrossMargin(0, 50);
    expect(result.value).toBeNull();
    expect(result.formatted).toBe('—');
  });

  it('returns null for null cost', () => {
    const result = calculateGrossMargin(1000, null);
    expect(result.value).toBeNull();
  });

  it('calculates correct margin', () => {
    const result = calculateGrossMargin(1000, 600);
    expect(result.value).toBeCloseTo(40.0);
    expect(result.formatted).toBe('40.0%');
  });
});

describe('calculateOperatingMargin', () => {
  it('returns null for null revenue', () => {
    expect(calculateOperatingMargin(100, null).value).toBeNull();
  });

  it('calculates correct margin', () => {
    const result = calculateOperatingMargin(200, 1000);
    expect(result.value).toBeCloseTo(20.0);
    expect(result.formatted).toBe('20.0%');
  });
});

describe('calculateNetMargin', () => {
  it('returns null for zero revenue', () => {
    expect(calculateNetMargin(50, 0).value).toBeNull();
  });

  it('calculates correct margin', () => {
    const result = calculateNetMargin(150, 1000);
    expect(result.value).toBeCloseTo(15.0);
    expect(result.formatted).toBe('15.0%');
  });

  it('handles negative net income', () => {
    const result = calculateNetMargin(-100, 1000);
    expect(result.value).toBeCloseTo(-10.0);
    expect(result.formatted).toBe('-10.0%');
  });
});

describe('extractTrendData', () => {
  it('extracts numeric values from periods', () => {
    const periods = [
      { data: { revenue: 300 } },
      { data: { revenue: 200 } },
      { data: { revenue: 100 } },
    ];
    const result = extractTrendData(periods, 'revenue');
    // Reversed to chronological order (oldest first)
    expect(result).toEqual([100, 200, 300]);
  });

  it('filters out non-numeric values', () => {
    const periods = [
      { data: { revenue: 300 } },
      { data: { revenue: null } },
      { data: { revenue: 100 } },
    ];
    const result = extractTrendData(periods, 'revenue');
    expect(result).toEqual([100, 300]);
  });

  it('returns empty array for missing key', () => {
    const periods = [{ data: { revenue: 100 } }];
    const result = extractTrendData(periods, 'nonexistent');
    expect(result).toEqual([]);
  });
});

describe('exportToCSV', () => {
  it('generates correct CSV with annual data', () => {
    const periods = [
      { fiscal_year: 2024, data: { revenue: 1000, expenses: 500 } },
      { fiscal_year: 2023, data: { revenue: 800, expenses: 400 } },
    ];
    const rows = [
      { key: 'revenue', label: 'Revenue', format: 'currency' },
      { key: 'expenses', label: 'Expenses', format: 'currency' },
    ];

    const csv = exportToCSV(periods, rows, 'AAPL', 'Income Statement');
    const lines = csv.split('\n');

    expect(lines[0]).toContain('"Metric"');
    expect(lines[0]).toContain('"FY 2024"');
    expect(lines[0]).toContain('"FY 2023"');
    expect(lines[1]).toContain('"Revenue"');
    expect(lines[1]).toContain('"1000"');
  });

  it('generates correct CSV with quarterly data', () => {
    const periods = [
      { fiscal_year: 2024, fiscal_quarter: 2, data: { revenue: 500 } },
      { fiscal_year: 2024, fiscal_quarter: 1, data: { revenue: 450 } },
    ];
    const rows = [{ key: 'revenue', label: 'Revenue', format: 'currency' }];

    const csv = exportToCSV(periods, rows, 'MSFT', 'Income Statement');
    expect(csv).toContain('"Q2 2024"');
    expect(csv).toContain('"Q1 2024"');
  });

  it('handles null values as empty strings', () => {
    const periods = [{ fiscal_year: 2024, data: { revenue: null } }];
    const rows = [{ key: 'revenue', label: 'Revenue', format: 'currency' }];

    const csv = exportToCSV(periods, rows, 'GOOG', 'Income Statement');
    expect(csv).toContain('""');
  });
});
