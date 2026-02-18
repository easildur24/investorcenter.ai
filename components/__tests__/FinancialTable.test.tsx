/**
 * Tests for FinancialTable component.
 *
 * Verifies loading states, data rendering, error handling,
 * and export functionality.
 */

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';

// Mock the API call
const mockGetFinancialStatements = jest.fn();
jest.mock('@/lib/api/financials', () => ({
  getFinancialStatements: (...args: unknown[]) => mockGetFinancialStatements(...args),
}));

// Mock formatters
jest.mock('@/lib/formatters/financial', () => ({
  formatFinancialValue: (value: number | null, format: string, decimals?: number) => {
    if (value === null || value === undefined) return '-';
    if (format === 'currency') return `$${(value / 1e9).toFixed(1)}B`;
    if (format === 'percent') return `${value.toFixed(1)}%`;
    return value.toFixed(decimals ?? 0);
  },
  formatYoYChange: (change: number | undefined) => {
    if (change === undefined || change === null) {
      return { text: '', color: '' };
    }
    const color = change >= 0 ? 'text-green-600' : 'text-red-600';
    return {
      text: `${change >= 0 ? '+' : ''}${change.toFixed(1)}%`,
      color,
    };
  },
  exportToCSV: jest.fn(() => 'csv-content'),
  downloadFile: jest.fn(),
}));

// Mock cn utility
jest.mock('@/lib/utils', () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(' '),
}));

import FinancialTable from '../financials/FinancialTable';

const makePeriod = (year: number, quarter: number | null, data: Record<string, number | null>) => ({
  fiscal_year: year,
  fiscal_quarter: quarter,
  period_end: `${year}-${quarter ? quarter * 3 : 12}-31`,
  data,
  yoy_change: null,
});

describe('FinancialTable', () => {
  beforeEach(() => {
    mockGetFinancialStatements.mockReset();
  });

  it('shows loading skeleton initially', () => {
    mockGetFinancialStatements.mockReturnValue(new Promise(() => {})); // Never resolves

    const { container } = render(
      <FinancialTable ticker="AAPL" statementType="income" timeframe="annual" />
    );

    // Skeleton has animate-pulse class
    expect(container.querySelector('.animate-pulse')).toBeInTheDocument();
  });

  it('renders metric names and period headers after data loads', async () => {
    mockGetFinancialStatements.mockResolvedValue({
      periods: [
        makePeriod(2025, null, {
          revenues: 394000000000,
          net_income_loss: 97000000000,
        }),
        makePeriod(2024, null, {
          revenues: 383000000000,
          net_income_loss: 93000000000,
        }),
      ],
    });

    render(<FinancialTable ticker="AAPL" statementType="income" timeframe="annual" />);

    await waitFor(() => {
      expect(screen.getByText('Total Revenue')).toBeInTheDocument();
    });

    expect(screen.getByText('Net Income')).toBeInTheDocument();

    // Period headers
    expect(screen.getByText('FY 2025')).toBeInTheDocument();
    expect(screen.getByText('FY 2024')).toBeInTheDocument();
  });

  it('shows error message when API fails', async () => {
    mockGetFinancialStatements.mockRejectedValue(new Error('Network error'));

    render(<FinancialTable ticker="AAPL" statementType="income" timeframe="annual" />);

    await waitFor(() => {
      expect(screen.getByText('Failed to load financial data')).toBeInTheDocument();
    });
  });

  it('shows no data message for null response', async () => {
    mockGetFinancialStatements.mockResolvedValue(null);

    render(<FinancialTable ticker="AAPL" statementType="income" timeframe="annual" />);

    await waitFor(() => {
      expect(screen.getByText('No financial data available')).toBeInTheDocument();
    });
  });

  it('shows no data message for empty periods', async () => {
    mockGetFinancialStatements.mockResolvedValue({
      periods: [],
    });

    render(<FinancialTable ticker="AAPL" statementType="income" timeframe="annual" />);

    await waitFor(() => {
      expect(screen.getByText('No financial data available')).toBeInTheDocument();
    });
  });

  it('filters out rows with no data', async () => {
    // Only revenues has data; cost_of_revenue is undefined
    mockGetFinancialStatements.mockResolvedValue({
      periods: [makePeriod(2025, null, { revenues: 394000000000 })],
    });

    render(<FinancialTable ticker="AAPL" statementType="income" timeframe="annual" />);

    await waitFor(() => {
      expect(screen.getByText('Total Revenue')).toBeInTheDocument();
    });

    // Cost of Revenue should NOT appear since it has no data
    expect(screen.queryByText('Cost of Revenue')).not.toBeInTheDocument();
  });

  it('renders Export CSV button', async () => {
    mockGetFinancialStatements.mockResolvedValue({
      periods: [makePeriod(2025, null, { revenues: 394000000000 })],
    });

    render(<FinancialTable ticker="AAPL" statementType="income" timeframe="annual" />);

    await waitFor(() => {
      expect(screen.getByText('Export CSV')).toBeInTheDocument();
    });
  });
});
