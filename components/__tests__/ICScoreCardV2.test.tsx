/**
 * Tests for ICScoreCardV2 component.
 *
 * Verifies rendering with different score values, loading/error states,
 * color coding, missing data handling, variant modes, and interactive
 * collapsible sections.
 */

import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import type { ICScoreDataV2 } from '@/lib/types/ic-score-v2';

// Mock the IC Score API
const mockGetICScore = jest.fn();
jest.mock('@/lib/api/ic-score', () => ({
  getICScore: (...args: unknown[]) => mockGetICScore(...args),
}));

// Mock utility functions
jest.mock('@/lib/utils', () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(' '),
  formatRelativeTime: (date: string) => `2 hours ago`,
}));

// Mock child components to isolate ICScoreCardV2 behavior
jest.mock('../ic-score/ICScoreGauge', () => {
  return function MockICScoreGauge({ score, size }: { score: number; size?: string }) {
    return (
      <div data-testid="ic-score-gauge" data-score={score} data-size={size}>
        Gauge: {score}
      </div>
    );
  };
});

jest.mock('../ic-score/CategoryScoreBadge', () => ({
  CategoryScoreGrid: ({
    quality,
    valuation,
    signals,
  }: {
    quality: { score: number };
    valuation: { score: number };
    signals: { score: number };
  }) => (
    <div data-testid="category-score-grid">
      <span data-testid="quality-score">{quality.score}</span>
      <span data-testid="valuation-score">{valuation.score}</span>
      <span data-testid="signals-score">{signals.score}</span>
    </div>
  ),
}));

jest.mock('../ic-score/Badges', () => ({
  LifecycleBadge: ({ stage, size }: { stage: string; size?: string }) => (
    <span data-testid="lifecycle-badge">{stage}</span>
  ),
  ConfidenceBadge: ({ level }: { level: string }) => (
    <span data-testid="confidence-badge">{level}</span>
  ),
  RatingBadge: ({ rating, size }: { rating: string; size?: string }) => (
    <span data-testid="rating-badge">{rating}</span>
  ),
  SectorRankBadge: ({ rank, total }: { rank: number; total: number }) => (
    <span data-testid="sector-rank-badge">
      {rank}/{total}
    </span>
  ),
}));

jest.mock('../ic-score/FactorBreakdown', () => {
  return function MockFactorBreakdown() {
    return <div data-testid="factor-breakdown">FactorBreakdown</div>;
  };
});

jest.mock('../ic-score/PeerComparisonPanel', () => {
  return function MockPeerComparisonPanel() {
    return <div data-testid="peer-comparison">PeerComparison</div>;
  };
});

jest.mock('../ic-score/CatalystTimeline', () => {
  const MockCatalystTimeline = () => <div data-testid="catalyst-timeline">CatalystTimeline</div>;
  MockCatalystTimeline.CatalystCompact = () => (
    <div data-testid="catalyst-compact">CatalystCompact</div>
  );
  return {
    __esModule: true,
    default: MockCatalystTimeline,
    CatalystCompact: MockCatalystTimeline.CatalystCompact,
  };
});

jest.mock('../ic-score/ScoreChangeExplainer', () => {
  const MockScoreChangeExplainer = () => (
    <div data-testid="score-change-explainer">ScoreChangeExplainer</div>
  );
  const MockScoreChangeInline = () => (
    <span data-testid="score-change-inline">ScoreChangeInline</span>
  );
  return {
    __esModule: true,
    default: MockScoreChangeExplainer,
    ScoreChangeInline: MockScoreChangeInline,
  };
});

jest.mock('../ic-score/GranularConfidenceDisplay', () => {
  return function MockGranularConfidenceDisplay() {
    return <div data-testid="granular-confidence">GranularConfidence</div>;
  };
});

jest.mock('../ic-score/ICScoreExplainer', () => {
  const MockICScoreExplainer = ({ onClose }: { onClose: () => void }) => (
    <div data-testid="ic-score-explainer">
      <button onClick={onClose}>Close</button>
    </div>
  );
  const MockICScoreExplainerButton = ({ onClick }: { onClick: () => void }) => (
    <button data-testid="explainer-button" onClick={onClick}>
      How is this calculated?
    </button>
  );
  return {
    __esModule: true,
    default: MockICScoreExplainer,
    ICScoreExplainerButton: MockICScoreExplainerButton,
  };
});

import ICScoreCardV2 from '../ic-score/ICScoreCardV2';

// Helper to build mock IC Score data
function createMockICScore(overrides: Partial<ICScoreDataV2> = {}): ICScoreDataV2 {
  return {
    ticker: 'AAPL',
    date: '2025-12-01',
    overall_score: 75,
    previous_score: 72,
    rating: 'Buy',
    confidence_level: 'High',
    data_completeness: 0.92,
    calculated_at: '2025-12-01T10:00:00Z',
    lifecycle_stage: 'mature',
    sector: 'Technology',
    sector_rank: 5,
    sector_total: 45,
    sector_percentile: 88,
    value_score: 65,
    growth_score: 70,
    profitability_score: 80,
    financial_health_score: 75,
    momentum_score: 68,
    analyst_consensus_score: 72,
    insider_activity_score: 60,
    institutional_score: 78,
    news_sentiment_score: 55,
    technical_score: 62,
    factor_count: 10,
    available_factors: ['value', 'growth', 'profitability'],
    missing_factors: ['dividend_quality'],
    ...overrides,
  };
}

describe('ICScoreCardV2', () => {
  beforeEach(() => {
    mockGetICScore.mockReset();
  });

  describe('loading state', () => {
    it('shows loading skeleton while data is fetching', () => {
      mockGetICScore.mockReturnValue(new Promise(() => {})); // Never resolves

      const { container } = render(<ICScoreCardV2 ticker="AAPL" />);

      expect(container.querySelector('.animate-pulse')).toBeInTheDocument();
    });

    it('shows compact loading skeleton for compact variant', () => {
      mockGetICScore.mockReturnValue(new Promise(() => {}));

      const { container } = render(<ICScoreCardV2 ticker="AAPL" variant="compact" />);

      expect(container.querySelector('.animate-pulse')).toBeInTheDocument();
    });
  });

  describe('error state', () => {
    it('shows error message when API returns null', async () => {
      mockGetICScore.mockResolvedValue(null);

      render(<ICScoreCardV2 ticker="AAPL" />);

      await waitFor(() => {
        expect(screen.getByText('IC Score Not Available')).toBeInTheDocument();
      });

      expect(screen.getByText(/IC Score for AAPL/)).toBeInTheDocument();
    });

    it('shows error message when API throws', async () => {
      mockGetICScore.mockRejectedValue(new Error('API Error'));

      render(<ICScoreCardV2 ticker="TSLA" />);

      await waitFor(() => {
        expect(screen.getByText('IC Score Not Available')).toBeInTheDocument();
      });

      expect(screen.getByText(/IC Score for TSLA/)).toBeInTheDocument();
    });
  });

  describe('renders with score data (full variant)', () => {
    it('renders the gauge with overall score', async () => {
      mockGetICScore.mockResolvedValue(createMockICScore({ overall_score: 82 }));

      render(<ICScoreCardV2 ticker="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('ic-score-gauge')).toBeInTheDocument();
      });

      expect(screen.getByTestId('ic-score-gauge')).toHaveAttribute('data-score', '82');
    });

    it('renders rating, lifecycle, and confidence badges', async () => {
      mockGetICScore.mockResolvedValue(
        createMockICScore({
          rating: 'Strong Buy',
          lifecycle_stage: 'growth',
          confidence_level: 'High',
        })
      );

      render(<ICScoreCardV2 ticker="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('rating-badge')).toHaveTextContent('Strong Buy');
      });

      expect(screen.getByTestId('lifecycle-badge')).toHaveTextContent('growth');
      expect(screen.getByTestId('confidence-badge')).toHaveTextContent('High');
    });

    it('renders sector rank badge when available', async () => {
      mockGetICScore.mockResolvedValue(createMockICScore({ sector_rank: 3, sector_total: 50 }));

      render(<ICScoreCardV2 ticker="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('sector-rank-badge')).toHaveTextContent('3/50');
      });
    });

    it('does not render sector rank badge when data is missing', async () => {
      mockGetICScore.mockResolvedValue(
        createMockICScore({ sector_rank: undefined, sector_total: undefined })
      );

      render(<ICScoreCardV2 ticker="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('rating-badge')).toBeInTheDocument();
      });

      expect(screen.queryByTestId('sector-rank-badge')).not.toBeInTheDocument();
    });

    it('renders category score grid with quality, valuation, signals', async () => {
      mockGetICScore.mockResolvedValue(createMockICScore());

      render(<ICScoreCardV2 ticker="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('category-score-grid')).toBeInTheDocument();
      });
    });

    it('renders the footer with update time and factor count', async () => {
      mockGetICScore.mockResolvedValue(
        createMockICScore({ factor_count: 10, calculated_at: '2025-12-01T10:00:00Z' })
      );

      render(<ICScoreCardV2 ticker="AAPL" />);

      await waitFor(() => {
        expect(screen.getByText('10 of 13 factors available')).toBeInTheDocument();
      });

      expect(screen.getByText(/Updated/)).toBeInTheDocument();
    });
  });

  describe('score-based rendering', () => {
    it('renders Strong Buy rating for high score', async () => {
      mockGetICScore.mockResolvedValue(
        createMockICScore({ overall_score: 90, rating: 'Strong Buy' })
      );

      render(<ICScoreCardV2 ticker="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('rating-badge')).toHaveTextContent('Strong Buy');
      });
    });

    it('renders Buy rating for score in 65-79 range', async () => {
      mockGetICScore.mockResolvedValue(createMockICScore({ overall_score: 72, rating: 'Buy' }));

      render(<ICScoreCardV2 ticker="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('rating-badge')).toHaveTextContent('Buy');
      });
    });

    it('renders Hold rating for score in 50-64 range', async () => {
      mockGetICScore.mockResolvedValue(createMockICScore({ overall_score: 55, rating: 'Hold' }));

      render(<ICScoreCardV2 ticker="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('rating-badge')).toHaveTextContent('Hold');
      });
    });

    it('renders Sell rating for low score', async () => {
      mockGetICScore.mockResolvedValue(createMockICScore({ overall_score: 30, rating: 'Sell' }));

      render(<ICScoreCardV2 ticker="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('rating-badge')).toHaveTextContent('Sell');
      });
    });
  });

  describe('missing/null data handling', () => {
    it('renders without crashing when optional fields are null', async () => {
      mockGetICScore.mockResolvedValue(
        createMockICScore({
          previous_score: null,
          lifecycle_stage: null,
          sector: null,
          sector_rank: undefined,
          sector_total: undefined,
          peers: undefined,
          peer_comparison: undefined,
          catalysts: undefined,
          explanation: undefined,
        })
      );

      render(<ICScoreCardV2 ticker="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('ic-score-gauge')).toBeInTheDocument();
      });

      // Should not show peer comparison when peers not available
      expect(screen.queryByTestId('peer-comparison')).not.toBeInTheDocument();
      // Should not show catalyst timeline when catalysts not available
      expect(screen.queryByTestId('catalyst-timeline')).not.toBeInTheDocument();
    });

    it('renders without crashing when all factor scores are null', async () => {
      mockGetICScore.mockResolvedValue(
        createMockICScore({
          value_score: null,
          growth_score: null,
          profitability_score: null,
          financial_health_score: null,
          momentum_score: null,
          analyst_consensus_score: null,
          insider_activity_score: null,
          institutional_score: null,
          news_sentiment_score: null,
          technical_score: null,
        })
      );

      render(<ICScoreCardV2 ticker="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('ic-score-gauge')).toBeInTheDocument();
      });

      expect(screen.getByTestId('category-score-grid')).toBeInTheDocument();
    });
  });

  describe('collapsible sections', () => {
    it('shows Factor Breakdown when section is toggled open', async () => {
      mockGetICScore.mockResolvedValue(createMockICScore());

      render(<ICScoreCardV2 ticker="AAPL" />);

      await waitFor(() => {
        expect(screen.getByText('Factor Breakdown')).toBeInTheDocument();
      });

      // Factor breakdown should be collapsed initially
      expect(screen.queryByTestId('factor-breakdown')).not.toBeInTheDocument();

      // Toggle open
      fireEvent.click(screen.getByText('Factor Breakdown'));
      expect(screen.getByTestId('factor-breakdown')).toBeInTheDocument();

      // Toggle closed
      fireEvent.click(screen.getByText('Factor Breakdown'));
      expect(screen.queryByTestId('factor-breakdown')).not.toBeInTheDocument();
    });
  });

  describe('optional sections visibility', () => {
    it('shows peer comparison when showPeers=true and peers exist', async () => {
      mockGetICScore.mockResolvedValue(
        createMockICScore({
          peers: [
            { ticker: 'MSFT', company_name: 'Microsoft', ic_score: 80, similarity_score: 0.92 },
          ],
        })
      );

      render(<ICScoreCardV2 ticker="AAPL" showPeers={true} />);

      await waitFor(() => {
        expect(screen.getByTestId('peer-comparison')).toBeInTheDocument();
      });
    });

    it('hides peer comparison when showPeers=false', async () => {
      mockGetICScore.mockResolvedValue(
        createMockICScore({
          peers: [
            { ticker: 'MSFT', company_name: 'Microsoft', ic_score: 80, similarity_score: 0.92 },
          ],
        })
      );

      render(<ICScoreCardV2 ticker="AAPL" showPeers={false} />);

      await waitFor(() => {
        expect(screen.getByTestId('ic-score-gauge')).toBeInTheDocument();
      });

      expect(screen.queryByTestId('peer-comparison')).not.toBeInTheDocument();
    });

    it('shows catalysts when showCatalysts=true and catalysts exist', async () => {
      mockGetICScore.mockResolvedValue(
        createMockICScore({
          catalysts: [
            {
              event_type: 'earnings',
              title: 'Q4 Earnings',
              event_date: '2025-12-15',
              icon: null,
              impact: 'Positive',
              confidence: 0.8,
              days_until: 14,
            },
          ],
        })
      );

      render(<ICScoreCardV2 ticker="AAPL" showCatalysts={true} />);

      await waitFor(() => {
        expect(screen.getByTestId('catalyst-timeline')).toBeInTheDocument();
      });
    });

    it('hides catalysts when catalysts array is empty', async () => {
      mockGetICScore.mockResolvedValue(createMockICScore({ catalysts: [] }));

      render(<ICScoreCardV2 ticker="AAPL" showCatalysts={true} />);

      await waitFor(() => {
        expect(screen.getByTestId('ic-score-gauge')).toBeInTheDocument();
      });

      expect(screen.queryByTestId('catalyst-timeline')).not.toBeInTheDocument();
    });
  });

  describe('explainer modal', () => {
    it('opens and closes the explainer modal', async () => {
      mockGetICScore.mockResolvedValue(createMockICScore());

      render(<ICScoreCardV2 ticker="AAPL" />);

      await waitFor(() => {
        expect(screen.getByTestId('explainer-button')).toBeInTheDocument();
      });

      // Modal should not be visible initially
      expect(screen.queryByTestId('ic-score-explainer')).not.toBeInTheDocument();

      // Open modal
      fireEvent.click(screen.getByTestId('explainer-button'));
      expect(screen.getByTestId('ic-score-explainer')).toBeInTheDocument();

      // Close modal
      fireEvent.click(screen.getByText('Close'));
      expect(screen.queryByTestId('ic-score-explainer')).not.toBeInTheDocument();
    });
  });

  describe('compact variant', () => {
    it('renders compact card with gauge and ratings', async () => {
      mockGetICScore.mockResolvedValue(createMockICScore({ overall_score: 78, rating: 'Buy' }));

      render(<ICScoreCardV2 ticker="AAPL" variant="compact" />);

      await waitFor(() => {
        expect(screen.getByTestId('ic-score-gauge')).toBeInTheDocument();
      });

      expect(screen.getByTestId('ic-score-gauge')).toHaveAttribute('data-score', '78');
      expect(screen.getByTestId('rating-badge')).toHaveTextContent('Buy');
      expect(screen.getByTestId('lifecycle-badge')).toBeInTheDocument();
    });

    it('shows mini category badges (Q, V, S) in compact variant', async () => {
      mockGetICScore.mockResolvedValue(createMockICScore());

      render(<ICScoreCardV2 ticker="AAPL" variant="compact" />);

      await waitFor(() => {
        expect(screen.getByText(/^Q:/)).toBeInTheDocument();
      });

      expect(screen.getByText(/^V:/)).toBeInTheDocument();
      expect(screen.getByText(/^S:/)).toBeInTheDocument();
    });

    it('applies color coding to mini category badges based on score', async () => {
      mockGetICScore.mockResolvedValue(
        createMockICScore({
          // Set high quality scores to get >= 80 average
          profitability_score: 90,
          financial_health_score: 90,
          growth_score: 85,
          // Set low signal scores for red range
          momentum_score: 20,
          technical_score: 25,
        })
      );

      render(<ICScoreCardV2 ticker="AAPL" variant="compact" />);

      await waitFor(() => {
        const qualityBadge = screen.getByText(/^Q:/);
        expect(qualityBadge.className).toContain('bg-green-100');
      });
    });

    it('renders compact catalyst when catalysts are available', async () => {
      mockGetICScore.mockResolvedValue(
        createMockICScore({
          catalysts: [
            {
              event_type: 'earnings',
              title: 'Q4 Earnings',
              event_date: '2025-12-15',
              icon: null,
              impact: 'Positive',
              confidence: 0.8,
              days_until: 14,
            },
          ],
        })
      );

      render(<ICScoreCardV2 ticker="AAPL" variant="compact" />);

      await waitFor(() => {
        expect(screen.getByTestId('catalyst-compact')).toBeInTheDocument();
      });
    });
  });

  describe('re-fetches on ticker change', () => {
    it('calls getICScore with the correct ticker', async () => {
      mockGetICScore.mockResolvedValue(createMockICScore());

      render(<ICScoreCardV2 ticker="AAPL" />);

      await waitFor(() => {
        expect(mockGetICScore).toHaveBeenCalledWith('AAPL');
      });
    });
  });
});
