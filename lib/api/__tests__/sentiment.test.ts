import {
  getSentiment,
  getSentimentHistory,
  getTrendingSentiment,
  getSentimentPosts,
} from '../sentiment';

jest.mock('@/lib/api/client', () => ({
  apiClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
  },
}));

import { apiClient } from '../client';

const mockGet = apiClient.get as jest.Mock;

beforeEach(() => {
  jest.clearAllMocks();
});

describe('getSentiment', () => {
  it('calls GET /sentiment/:ticker', async () => {
    mockGet.mockResolvedValueOnce({});
    await getSentiment('AAPL');
    expect(mockGet).toHaveBeenCalledWith('/sentiment/AAPL');
  });

  it('uppercases ticker', async () => {
    mockGet.mockResolvedValueOnce({});
    await getSentiment('aapl');
    expect(mockGet).toHaveBeenCalledWith('/sentiment/AAPL');
  });
});

describe('getSentimentHistory', () => {
  it('calls GET /sentiment/:ticker/history with default days', async () => {
    mockGet.mockResolvedValueOnce({});
    await getSentimentHistory('MSFT');
    expect(mockGet).toHaveBeenCalledWith('/sentiment/MSFT/history?days=7');
  });

  it('uses custom days parameter', async () => {
    mockGet.mockResolvedValueOnce({});
    await getSentimentHistory('MSFT', 30);
    expect(mockGet).toHaveBeenCalledWith('/sentiment/MSFT/history?days=30');
  });
});

describe('getTrendingSentiment', () => {
  it('calls GET /sentiment/trending with defaults', async () => {
    mockGet.mockResolvedValueOnce({});
    await getTrendingSentiment();
    expect(mockGet).toHaveBeenCalledWith('/sentiment/trending?period=24h&limit=20');
  });

  it('uses custom period and limit', async () => {
    mockGet.mockResolvedValueOnce({});
    await getTrendingSentiment('7d', 10);
    expect(mockGet).toHaveBeenCalledWith('/sentiment/trending?period=7d&limit=10');
  });
});

describe('getSentimentPosts', () => {
  it('calls GET /sentiment/:ticker/posts with defaults', async () => {
    mockGet.mockResolvedValueOnce({});
    await getSentimentPosts('TSLA');
    expect(mockGet).toHaveBeenCalledWith('/sentiment/TSLA/posts?sort=recent&limit=10');
  });

  it('uses custom sort and limit', async () => {
    mockGet.mockResolvedValueOnce({});
    await getSentimentPosts('TSLA', 'engagement', 5);
    expect(mockGet).toHaveBeenCalledWith('/sentiment/TSLA/posts?sort=engagement&limit=5');
  });

  it('uppercases ticker', async () => {
    mockGet.mockResolvedValueOnce({});
    await getSentimentPosts('tsla');
    expect(mockGet).toHaveBeenCalledWith('/sentiment/TSLA/posts?sort=recent&limit=10');
  });
});
