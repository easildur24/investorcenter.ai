/**
 * Tests for BacktestDashboard component.
 *
 * Verifies rendering with mock data, loading state, error state,
 * retry flow, and user interactions like toggling the config panel.
 */

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BacktestSummary } from '@/lib/types/backtest';

// Mock child chart components (canvas-based, not testable in jsdom)
jest.mock('../backtest/DecileBarChart', () => ({
  DecileBarChart: ({ data }: { data: unknown[] }) => (
    <div data-testid="decile-bar-chart">DecileBarChart ({(data as unknown[]).length} items)</div>
  ),
}));

jest.mock('../backtest/CumulativeReturnsChart', () => ({
  CumulativeReturnsChart: () => (
    <div data-testid="cumulative-returns-chart">CumulativeReturnsChart</div>
  ),
}));

jest.mock('../backtest/BacktestConfigPanel', () => ({
  BacktestConfigPanel: ({
    onRun,
    loading,
  }: {
    onRun: () => void;
    loading: boolean;
    config: unknown;
    onChange: unknown;
  }) => (
    <div data-testid="backtest-config-panel">
      <button onClick={onRun} disabled={loading}>
        Run Backtest
      </button>
    </div>
  ),
}));

jest.mock('../backtest/StatisticalSummary', () => ({
  StatisticalSummary: () => <div data-testid="statistical-summary">StatisticalSummary</div>,
}));

// Mock API routes
jest.mock('@/lib/api/routes', () => ({
  backtest: {
    latest: '/ic-scores/backtest/latest',
    run: '/ic-scores/backtest',
  },
}));

// Mock API base URL
jest.mock('@/lib/api', () => ({
  API_BASE_URL: 'http://test-api.example.com/api/v1',
}));

import BacktestDashboard from '../backtest/BacktestDashboard';

// Helper to build a valid BacktestSummary mock
function createMockSummary(overrides: Partial<BacktestSummary> = {}): BacktestSummary {
  return {
    start_date: '2021-01-01',
    end_date: '2025-12-31',
    rebalance_frequency: 'monthly',
    universe: 'sp500',
    benchmark: 'SPY',
    num_periods: 60,
    top_decile_cagr: 0.18,
    bottom_decile_cagr: -0.02,
    spread_cagr: 0.2,
    benchmark_cagr: 0.12,
    top_vs_benchmark: 0.06,
    hit_rate: 0.82,
    monotonicity_score: 0.75,
    information_ratio: 1.45,
    top_decile_sharpe: 1.2,
    top_decile_max_dd: 0.25,
    bottom_decile_sharpe: -0.1,
    decile_performance: [
      {
        decile: 1,
        total_return: 1.5,
        annualized_return: 0.18,
        volatility: 0.15,
        sharpe_ratio: 1.2,
        max_drawdown: 0.25,
        avg_score: 85.2,
        num_periods: 60,
      },
      {
        decile: 10,
        total_return: -0.1,
        annualized_return: -0.02,
        volatility: 0.28,
        sharpe_ratio: -0.1,
        max_drawdown: 0.55,
        avg_score: 22.1,
        num_periods: 60,
      },
    ],
    ...overrides,
  };
}

describe('BacktestDashboard', () => {
  const mockFetch = global.fetch as jest.Mock;

  describe('with initialData', () => {
    it('renders with initial data and does not show loading', () => {
      const summary = createMockSummary();
      render(<BacktestDashboard initialData={summary} />);

      expect(screen.getByText('IC Score Backtest Results')).toBeInTheDocument();
      expect(screen.queryByText('Loading backtest results...')).not.toBeInTheDocument();
    });

    it('displays the configuration details in the subtitle', () => {
      const summary = createMockSummary();
      render(<BacktestDashboard initialData={summary} />);

      expect(screen.getByText(/2021-01-01 to 2025-12-31/)).toBeInTheDocument();
      expect(screen.getByText(/SP500 Universe/)).toBeInTheDocument();
      expect(screen.getByText(/monthly Rebalancing/)).toBeInTheDocument();
    });

    it('renders the four key metric cards', () => {
      const summary = createMockSummary();
      render(<BacktestDashboard initialData={summary} />);

      expect(screen.getByText('Top Decile CAGR')).toBeInTheDocument();
      expect(screen.getByText('Bottom Decile CAGR')).toBeInTheDocument();
      expect(screen.getByText('D1-D10 Spread')).toBeInTheDocument();
      expect(screen.getByText('vs Benchmark')).toBeInTheDocument();
    });

    it('renders the statistical validity section', () => {
      const summary = createMockSummary();
      render(<BacktestDashboard initialData={summary} />);

      expect(screen.getByText('Statistical Validity')).toBeInTheDocument();
      expect(screen.getByText('Hit Rate')).toBeInTheDocument();
      expect(screen.getByText('Monotonicity')).toBeInTheDocument();
      expect(screen.getByText('Information Ratio')).toBeInTheDocument();
    });

    it('renders the risk metrics section', () => {
      const summary = createMockSummary();
      render(<BacktestDashboard initialData={summary} />);

      expect(screen.getByText('Risk Metrics')).toBeInTheDocument();
      expect(screen.getByText('Top Decile Sharpe')).toBeInTheDocument();
      expect(screen.getByText('Top Decile Max DD')).toBeInTheDocument();
      expect(screen.getByText('Bottom Decile Sharpe')).toBeInTheDocument();
    });

    it('renders chart components', () => {
      const summary = createMockSummary();
      render(<BacktestDashboard initialData={summary} />);

      expect(screen.getByTestId('decile-bar-chart')).toBeInTheDocument();
      expect(screen.getByTestId('cumulative-returns-chart')).toBeInTheDocument();
    });

    it('renders decile performance table with rows', () => {
      const summary = createMockSummary();
      render(<BacktestDashboard initialData={summary} />);

      expect(screen.getByText('D1')).toBeInTheDocument();
      expect(screen.getByText('D10')).toBeInTheDocument();
    });

    it('renders the statistical summary component', () => {
      const summary = createMockSummary();
      render(<BacktestDashboard initialData={summary} />);

      expect(screen.getByTestId('statistical-summary')).toBeInTheDocument();
    });

    it('renders the footer disclaimer', () => {
      const summary = createMockSummary();
      render(<BacktestDashboard initialData={summary} />);

      expect(
        screen.getByText(/Past performance does not guarantee future results/)
      ).toBeInTheDocument();
    });

    it('toggles the config panel when Configure button is clicked', () => {
      const summary = createMockSummary();
      render(<BacktestDashboard initialData={summary} />);

      // Config panel should not be visible initially
      expect(screen.queryByTestId('backtest-config-panel')).not.toBeInTheDocument();

      // Click Configure to show it
      fireEvent.click(screen.getByText('Configure'));
      expect(screen.getByTestId('backtest-config-panel')).toBeInTheDocument();
      expect(screen.getByText('Hide Config')).toBeInTheDocument();

      // Click Hide Config to hide it
      fireEvent.click(screen.getByText('Hide Config'));
      expect(screen.queryByTestId('backtest-config-panel')).not.toBeInTheDocument();
    });
  });

  describe('loading state', () => {
    it('shows loading spinner when no initialData is provided', () => {
      // Fetch never resolves during this test
      mockFetch.mockReturnValue(new Promise(() => {}));

      render(<BacktestDashboard />);

      expect(screen.getByText('Loading backtest results...')).toBeInTheDocument();
    });
  });

  describe('error state', () => {
    it('shows error message when fetch fails', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network failure'));

      render(<BacktestDashboard />);

      await waitFor(() => {
        expect(screen.getByText('Network failure')).toBeInTheDocument();
      });

      expect(screen.getByText('Retry')).toBeInTheDocument();
    });

    it('shows error when response is not ok', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
      });

      render(<BacktestDashboard />);

      await waitFor(() => {
        expect(screen.getByText('Failed to fetch backtest results')).toBeInTheDocument();
      });
    });

    it('retries fetch when Retry button is clicked', async () => {
      // First call fails, second call also fails but with different message
      // Note: fetchLatestBacktest does not clear the error state before retrying,
      // so each failure overwrites the previous error message.
      mockFetch
        .mockRejectedValueOnce(new Error('Temporary error'))
        .mockRejectedValueOnce(new Error('Second attempt failed'));

      render(<BacktestDashboard />);

      await waitFor(() => {
        expect(screen.getByText('Temporary error')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText('Retry'));

      // Verify that the component triggers a new fetch on retry
      await waitFor(() => {
        expect(mockFetch).toHaveBeenCalledTimes(2);
      });

      // The second error replaces the first
      await waitFor(() => {
        expect(screen.getByText('Second attempt failed')).toBeInTheDocument();
      });
    });
  });

  describe('no data state', () => {
    it('shows "No backtest results available" when fetch returns null-like', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => null,
      });

      render(<BacktestDashboard />);

      await waitFor(() => {
        expect(screen.getByText('No backtest results available')).toBeInTheDocument();
      });

      expect(screen.getByText('Run New Backtest')).toBeInTheDocument();
    });
  });

  describe('run backtest flow', () => {
    it('runs a new backtest via the config panel', async () => {
      const summary = createMockSummary();

      // Initial data shows the dashboard
      render(<BacktestDashboard initialData={summary} />);

      // Open config panel
      fireEvent.click(screen.getByText('Configure'));
      expect(screen.getByTestId('backtest-config-panel')).toBeInTheDocument();

      // Mock the run POST request
      const newSummary = createMockSummary({ top_decile_cagr: 0.22 });
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => newSummary,
      });

      // Click Run Backtest
      fireEvent.click(screen.getByText('Run Backtest'));

      await waitFor(() => {
        // The config panel should be hidden after success
        expect(screen.queryByTestId('backtest-config-panel')).not.toBeInTheDocument();
      });

      // Verify the POST was called
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/ic-scores/backtest'),
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
        })
      );
    });

    it('shows error when run backtest fails', async () => {
      const summary = createMockSummary();
      render(<BacktestDashboard initialData={summary} />);

      // Open config panel
      fireEvent.click(screen.getByText('Configure'));

      // Mock a failure
      mockFetch.mockRejectedValueOnce(new Error('Backtest failed'));

      fireEvent.click(screen.getByText('Run Backtest'));

      await waitFor(() => {
        expect(screen.getByText('Backtest failed')).toBeInTheDocument();
      });
    });
  });
});
