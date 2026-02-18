/**
 * Tests for ICScoreGauge component.
 *
 * Verifies score display, rating badges, size variants,
 * and show/hide options for label and rating.
 */

import React from 'react';
import { render, screen } from '@testing-library/react';

// Mock react-gauge-chart (canvas-based, not testable in jsdom)
jest.mock('react-gauge-chart', () => {
  return function MockGaugeChart(props: { percent: number; id: string }) {
    return (
      <div data-testid="gauge-chart" data-percent={props.percent}>
        Gauge
      </div>
    );
  };
});

// Mock ThemeContext
jest.mock('@/lib/contexts/ThemeContext', () => ({
  useTheme: () => ({
    resolvedTheme: 'light',
    theme: 'light',
    setTheme: jest.fn(),
    toggleTheme: jest.fn(),
  }),
}));

// Mock theme module
jest.mock('@/lib/theme', () => ({
  themeColors: {
    light: { textPrimary: '#000' },
    dark: { textPrimary: '#fff' },
    accent: {
      negative: '#ef4444',
      orange: '#f97316',
      warning: '#eab308',
      positive: '#10b981',
    },
  },
}));

import ICScoreGauge from '../ic-score/ICScoreGauge';

describe('ICScoreGauge', () => {
  it('renders score number and IC Score label', () => {
    render(<ICScoreGauge score={78} />);

    expect(screen.getByText('78')).toBeInTheDocument();
    expect(screen.getByText('IC Score')).toBeInTheDocument();
  });

  it('displays Strong Buy rating for score 85', () => {
    render(<ICScoreGauge score={85} />);

    expect(screen.getByText('Strong Buy')).toBeInTheDocument();
  });

  it('displays Sell rating for score 20', () => {
    render(<ICScoreGauge score={20} />);

    expect(screen.getByText('Sell')).toBeInTheDocument();
  });

  it('normalizes score to 0-1 range for gauge', () => {
    render(<ICScoreGauge score={75} />);

    const gauge = screen.getByTestId('gauge-chart');
    expect(gauge).toHaveAttribute('data-percent', '0.75');
  });

  it('clamps score over 100', () => {
    render(<ICScoreGauge score={150} />);

    const gauge = screen.getByTestId('gauge-chart');
    expect(gauge).toHaveAttribute('data-percent', '1');
  });

  it('hides label when showLabel is false', () => {
    render(<ICScoreGauge score={78} showLabel={false} />);

    expect(screen.queryByText('IC Score')).not.toBeInTheDocument();
  });

  it('hides rating when showRating is false', () => {
    render(<ICScoreGauge score={85} showRating={false} />);

    expect(screen.queryByText('Strong Buy')).not.toBeInTheDocument();
  });

  it('renders with sm size variant', () => {
    const { container } = render(<ICScoreGauge score={50} size="sm" />);

    // sm size uses width: 200
    const gaugeWrapper = container.querySelector('[style*="width"]');
    expect(gaugeWrapper).toBeInTheDocument();
  });
});
