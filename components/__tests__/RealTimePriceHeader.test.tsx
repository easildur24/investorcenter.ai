/**
 * Tests for RealTimePriceHeader component.
 *
 * Verifies price display, formatting, market status,
 * and crypto/stock differentiation.
 */

import React from 'react';
import { render, screen } from '@testing-library/react';

// Mock next/image
jest.mock('next/image', () => {
  return function MockImage(props: { alt: string; src: string }) {
    return <img alt={props.alt} src={props.src} />;
  };
});

// Mock useRealTimePrice hook
const mockUseRealTimePrice = jest.fn();
jest.mock('@/lib/hooks/useRealTimePrice', () => ({
  useRealTimePrice: (args: unknown) => mockUseRealTimePrice(args),
}));

import RealTimePriceHeader from '../ticker/RealTimePriceHeader';

const defaultInitialData = {
  stock: {
    symbol: 'AAPL',
    name: 'Apple Inc.',
    exchange: 'NASDAQ',
    sector: 'Technology',
    assetType: 'stock',
    isCrypto: false,
    logoUrl: '/logos/AAPL.png',
  },
  price: {
    price: '185.50',
    change: '2.30',
    changePercent: '1.25',
    volume: 50000000,
    lastUpdated: '2026-02-17T16:00:00Z',
  },
  market: {
    status: 'open',
    shouldUpdateRealtime: true,
    updateInterval: 15000,
  },
};

beforeEach(() => {
  mockUseRealTimePrice.mockReturnValue({
    priceData: null,
    error: null,
    isMarketOpen: true,
    isCrypto: false,
    updateInterval: 15000,
  });
});

describe('RealTimePriceHeader', () => {
  it('renders company name and symbol', () => {
    render(<RealTimePriceHeader symbol="AAPL" initialData={defaultInitialData} />);

    expect(screen.getByText('Apple Inc. (AAPL)')).toBeInTheDocument();
  });

  it('renders exchange and sector', () => {
    render(<RealTimePriceHeader symbol="AAPL" initialData={defaultInitialData} />);

    expect(screen.getByText('NASDAQ')).toBeInTheDocument();
    expect(screen.getByText('Technology')).toBeInTheDocument();
  });

  it('displays formatted price with $ prefix', () => {
    render(<RealTimePriceHeader symbol="AAPL" initialData={defaultInitialData} />);

    expect(screen.getByText('$185.50')).toBeInTheDocument();
  });

  it('shows positive change with + prefix', () => {
    render(<RealTimePriceHeader symbol="AAPL" initialData={defaultInitialData} />);

    expect(screen.getByText('+2.30 (+1.25%)')).toBeInTheDocument();
  });

  it('shows negative change', () => {
    const negativeData = {
      ...defaultInitialData,
      price: {
        ...defaultInitialData.price,
        change: '-3.50',
        changePercent: '-1.85',
      },
    };

    render(<RealTimePriceHeader symbol="AAPL" initialData={negativeData} />);

    expect(screen.getByText('-3.50 (-1.85%)')).toBeInTheDocument();
  });

  it('displays Market Open when market is open', () => {
    mockUseRealTimePrice.mockReturnValue({
      priceData: null,
      error: null,
      isMarketOpen: true,
      isCrypto: false,
      updateInterval: 15000,
    });

    render(<RealTimePriceHeader symbol="AAPL" initialData={defaultInitialData} />);

    expect(screen.getByText((content) => content.includes('Market Open'))).toBeInTheDocument();
  });

  it('displays Market Closed when market is closed', () => {
    mockUseRealTimePrice.mockReturnValue({
      priceData: null,
      error: null,
      isMarketOpen: false,
      isCrypto: false,
      updateInterval: 15000,
    });

    render(<RealTimePriceHeader symbol="AAPL" initialData={defaultInitialData} />);

    expect(screen.getByText((content) => content.includes('Market Closed'))).toBeInTheDocument();
  });

  it('shows Live (24/7) for crypto', () => {
    const cryptoData = {
      ...defaultInitialData,
      stock: {
        ...defaultInitialData.stock,
        symbol: 'X:BTCUSD',
        name: 'Bitcoin',
        isCrypto: true,
      },
    };

    mockUseRealTimePrice.mockReturnValue({
      priceData: null,
      error: null,
      isMarketOpen: false,
      isCrypto: true,
      updateInterval: 5000,
    });

    render(<RealTimePriceHeader symbol="X:BTCUSD" initialData={cryptoData} />);

    expect(screen.getByText((content) => content.includes('Live (24/7)'))).toBeInTheDocument();
  });

  it('renders placeholder when no logo URL', () => {
    const noLogoData = {
      ...defaultInitialData,
      stock: {
        ...defaultInitialData.stock,
        logoUrl: undefined,
      },
    };

    render(<RealTimePriceHeader symbol="AAPL" initialData={noLogoData} />);

    // Should render the letter placeholder
    expect(screen.getByText('A')).toBeInTheDocument();
  });
});
