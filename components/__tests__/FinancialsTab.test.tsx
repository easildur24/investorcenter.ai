/**
 * Tests for FinancialsTab component.
 *
 * Verifies rendering with financial data, loading/error states,
 * switching between statement types (income, balance_sheet, cash_flow, ratios),
 * and switching between timeframes (quarterly, annual, TTM).
 */

import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';

// Mock the getAllFinancials API call
const mockGetAllFinancials = jest.fn();
jest.mock('@/lib/api/financials', () => ({
  getAllFinancials: (...args: unknown[]) => mockGetAllFinancials(...args),
}));

// Mock cn utility used by child components
jest.mock('@/lib/utils', () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(' '),
}));

// Mock StatementTabs as a lightweight interactive component
jest.mock('@/components/financials/StatementTabs', () => {
  return function MockStatementTabs({
    activeTab,
    onChange,
  }: {
    activeTab: string;
    onChange: (tab: string) => void;
  }) {
    return (
      <div data-testid="statement-tabs">
        <button
          data-testid="tab-income"
          onClick={() => onChange('income')}
          className={activeTab === 'income' ? 'active' : ''}
        >
          Income Statement
        </button>
        <button
          data-testid="tab-balance"
          onClick={() => onChange('balance_sheet')}
          className={activeTab === 'balance_sheet' ? 'active' : ''}
        >
          Balance Sheet
        </button>
        <button
          data-testid="tab-cashflow"
          onClick={() => onChange('cash_flow')}
          className={activeTab === 'cash_flow' ? 'active' : ''}
        >
          Cash Flow
        </button>
        <button
          data-testid="tab-ratios"
          onClick={() => onChange('ratios')}
          className={activeTab === 'ratios' ? 'active' : ''}
        >
          Ratios
        </button>
      </div>
    );
  };
});

// Mock TimeframePicker as a lightweight interactive component
jest.mock('@/components/financials/TimeframePicker', () => {
  return function MockTimeframePicker({
    value,
    onChange,
  }: {
    value: string;
    onChange: (tf: string) => void;
  }) {
    return (
      <div data-testid="timeframe-picker">
        <button
          data-testid="tf-quarterly"
          onClick={() => onChange('quarterly')}
          className={value === 'quarterly' ? 'active' : ''}
        >
          Quarterly
        </button>
        <button
          data-testid="tf-annual"
          onClick={() => onChange('annual')}
          className={value === 'annual' ? 'active' : ''}
        >
          Annual
        </button>
        <button
          data-testid="tf-ttm"
          onClick={() => onChange('trailing_twelve_months')}
          className={value === 'trailing_twelve_months' ? 'active' : ''}
        >
          TTM
        </button>
      </div>
    );
  };
});

// Mock FinancialTable to capture props passed to it
jest.mock('@/components/financials/FinancialTable', () => {
  return function MockFinancialTable({
    ticker,
    statementType,
    timeframe,
    showYoY,
    limit,
  }: {
    ticker: string;
    statementType: string;
    timeframe: string;
    showYoY: boolean;
    limit: number;
  }) {
    return (
      <div data-testid="financial-table">
        <span data-testid="table-ticker">{ticker}</span>
        <span data-testid="table-statement-type">{statementType}</span>
        <span data-testid="table-timeframe">{timeframe}</span>
        <span data-testid="table-show-yoy">{String(showYoY)}</span>
        <span data-testid="table-limit">{limit}</span>
      </div>
    );
  };
});

import FinancialsTab from '../ticker/tabs/FinancialsTab';

describe('FinancialsTab', () => {
  beforeEach(() => {
    mockGetAllFinancials.mockReset();
  });

  describe('loading state', () => {
    it('shows loading skeleton while metadata is being fetched', () => {
      mockGetAllFinancials.mockReturnValue(new Promise(() => {})); // Never resolves

      const { container } = render(<FinancialsTab symbol="AAPL" />);

      expect(container.querySelector('.animate-pulse')).toBeInTheDocument();
    });
  });

  describe('successful data loading', () => {
    it('renders the header and financial table after loading', async () => {
      mockGetAllFinancials.mockResolvedValue({
        metadata: {
          company_name: 'Apple Inc.',
        },
      });

      render(<FinancialsTab symbol="AAPL" />);

      await waitFor(() => {
        expect(screen.getByText('Financial Statements')).toBeInTheDocument();
      });

      expect(screen.getByText('Apple Inc.')).toBeInTheDocument();
      expect(screen.getByTestId('financial-table')).toBeInTheDocument();
    });

    it('renders statement tabs and timeframe picker', async () => {
      mockGetAllFinancials.mockResolvedValue({
        metadata: { company_name: 'Apple Inc.' },
      });

      render(<FinancialsTab symbol="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('statement-tabs')).toBeInTheDocument();
      });

      expect(screen.getByTestId('timeframe-picker')).toBeInTheDocument();
    });

    it('starts with income statement and quarterly timeframe by default', async () => {
      mockGetAllFinancials.mockResolvedValue({
        metadata: { company_name: 'Apple Inc.' },
      });

      render(<FinancialsTab symbol="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('table-statement-type')).toHaveTextContent('income');
      });

      expect(screen.getByTestId('table-timeframe')).toHaveTextContent('quarterly');
    });

    it('shows statement description for income statement', async () => {
      mockGetAllFinancials.mockResolvedValue({
        metadata: { company_name: 'Apple Inc.' },
      });

      render(<FinancialsTab symbol="AAPL" />);

      await waitFor(() => {
        // The heading "Income Statement" appears both in the tab button and the h4
        // Check for the description to confirm the right section is rendered
        expect(
          screen.getByText('Revenue, expenses, and profitability over the reporting period')
        ).toBeInTheDocument();
      });
    });

    it('passes the correct symbol to the FinancialTable', async () => {
      mockGetAllFinancials.mockResolvedValue({
        metadata: { company_name: 'Apple Inc.' },
      });

      render(<FinancialsTab symbol="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('table-ticker')).toHaveTextContent('AAPL');
      });
    });

    it('renders the About This Data footer', async () => {
      mockGetAllFinancials.mockResolvedValue({
        metadata: { company_name: 'Apple Inc.' },
      });

      render(<FinancialsTab symbol="AAPL" />);

      await waitFor(() => {
        expect(screen.getByText('About This Data')).toBeInTheDocument();
      });

      expect(screen.getByText(/SEC EDGAR filings via Polygon.io/)).toBeInTheDocument();
    });
  });

  describe('switching statement types', () => {
    it('switches to Balance Sheet when tab is clicked', async () => {
      mockGetAllFinancials.mockResolvedValue({
        metadata: { company_name: 'Apple Inc.' },
      });

      render(<FinancialsTab symbol="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('financial-table')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByTestId('tab-balance'));

      expect(screen.getByTestId('table-statement-type')).toHaveTextContent('balance_sheet');
      expect(
        screen.getByText('Assets, liabilities, and equity at a point in time')
      ).toBeInTheDocument();
    });

    it('switches to Cash Flow when tab is clicked', async () => {
      mockGetAllFinancials.mockResolvedValue({
        metadata: { company_name: 'Apple Inc.' },
      });

      render(<FinancialsTab symbol="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('financial-table')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByTestId('tab-cashflow'));

      expect(screen.getByTestId('table-statement-type')).toHaveTextContent('cash_flow');
      expect(
        screen.getByText(
          'Cash generated and used in operating, investing, and financing activities'
        )
      ).toBeInTheDocument();
    });

    it('switches to Ratios when tab is clicked', async () => {
      mockGetAllFinancials.mockResolvedValue({
        metadata: { company_name: 'Apple Inc.' },
      });

      render(<FinancialsTab symbol="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('financial-table')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByTestId('tab-ratios'));

      expect(screen.getByTestId('table-statement-type')).toHaveTextContent('ratios');
      expect(
        screen.getByText('Key financial ratios for valuation, profitability, and efficiency')
      ).toBeInTheDocument();
    });

    it('switches back to income after switching to another tab', async () => {
      mockGetAllFinancials.mockResolvedValue({
        metadata: { company_name: 'Apple Inc.' },
      });

      render(<FinancialsTab symbol="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('financial-table')).toBeInTheDocument();
      });

      // Switch to balance sheet
      fireEvent.click(screen.getByTestId('tab-balance'));
      expect(screen.getByTestId('table-statement-type')).toHaveTextContent('balance_sheet');

      // Switch back to income
      fireEvent.click(screen.getByTestId('tab-income'));
      expect(screen.getByTestId('table-statement-type')).toHaveTextContent('income');
    });
  });

  describe('switching timeframes', () => {
    it('switches to annual timeframe', async () => {
      mockGetAllFinancials.mockResolvedValue({
        metadata: { company_name: 'Apple Inc.' },
      });

      render(<FinancialsTab symbol="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('financial-table')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByTestId('tf-annual'));

      expect(screen.getByTestId('table-timeframe')).toHaveTextContent('annual');
    });

    it('switches to TTM timeframe', async () => {
      mockGetAllFinancials.mockResolvedValue({
        metadata: { company_name: 'Apple Inc.' },
      });

      render(<FinancialsTab symbol="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('financial-table')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByTestId('tf-ttm'));

      expect(screen.getByTestId('table-timeframe')).toHaveTextContent('trailing_twelve_months');
    });

    it('switches back to quarterly after changing timeframe', async () => {
      mockGetAllFinancials.mockResolvedValue({
        metadata: { company_name: 'Apple Inc.' },
      });

      render(<FinancialsTab symbol="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('financial-table')).toBeInTheDocument();
      });

      // Switch to annual
      fireEvent.click(screen.getByTestId('tf-annual'));
      expect(screen.getByTestId('table-timeframe')).toHaveTextContent('annual');

      // Switch back to quarterly
      fireEvent.click(screen.getByTestId('tf-quarterly'));
      expect(screen.getByTestId('table-timeframe')).toHaveTextContent('quarterly');
    });
  });

  describe('combined statement type and timeframe changes', () => {
    it('passes both statement type and timeframe to FinancialTable', async () => {
      mockGetAllFinancials.mockResolvedValue({
        metadata: { company_name: 'Apple Inc.' },
      });

      render(<FinancialsTab symbol="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('financial-table')).toBeInTheDocument();
      });

      // Switch to balance sheet and annual
      fireEvent.click(screen.getByTestId('tab-balance'));
      fireEvent.click(screen.getByTestId('tf-annual'));

      expect(screen.getByTestId('table-statement-type')).toHaveTextContent('balance_sheet');
      expect(screen.getByTestId('table-timeframe')).toHaveTextContent('annual');
    });
  });

  describe('error handling', () => {
    it('renders the table even when metadata fetch fails', async () => {
      // The component sets error but still renders (no error UI shown for failed metadata)
      mockGetAllFinancials.mockRejectedValue(new Error('Network error'));

      render(<FinancialsTab symbol="AAPL" />);

      await waitFor(() => {
        expect(screen.getByText('Financial Statements')).toBeInTheDocument();
      });

      // Table still renders even after metadata error
      expect(screen.getByTestId('financial-table')).toBeInTheDocument();
    });

    it('does not show company name when metadata is null', async () => {
      mockGetAllFinancials.mockResolvedValue(null);

      render(<FinancialsTab symbol="AAPL" />);

      await waitFor(() => {
        expect(screen.getByText('Financial Statements')).toBeInTheDocument();
      });

      // Company name should not appear
      expect(screen.queryByText('Apple Inc.')).not.toBeInTheDocument();
    });
  });

  describe('table props', () => {
    it('passes showYoY=true and limit=8 to FinancialTable', async () => {
      mockGetAllFinancials.mockResolvedValue({
        metadata: { company_name: 'Apple Inc.' },
      });

      render(<FinancialsTab symbol="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('table-show-yoy')).toHaveTextContent('true');
      });

      expect(screen.getByTestId('table-limit')).toHaveTextContent('8');
    });
  });
});
