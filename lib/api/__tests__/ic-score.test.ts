import {
  getICScore,
  getICScoreHistory,
  getICScores,
  getLetterGrade,
  getScoreColor,
  getScoreBgColor,
  getFactorDetails,
  getStarRating,
  ICScoreData,
} from '../ic-score';

// Mock apiClient for getICScores (which uses apiClient.get)
jest.mock('@/lib/api/client', () => ({
  apiClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
  },
}));

import { apiClient } from '../client';

const mockApiGet = apiClient.get as jest.Mock;
const mockFetch = global.fetch as jest.Mock;

beforeEach(() => {
  jest.clearAllMocks();
  mockFetch.mockReset();
});

// ──────────────────────────────────────────────────────────────
// Helper: build a full ICScoreData object
// ──────────────────────────────────────────────────────────────
function buildICScoreData(overrides: Partial<ICScoreData> = {}): ICScoreData {
  return {
    ticker: 'AAPL',
    date: '2025-01-15',
    overall_score: 82,
    value_score: 75,
    growth_score: 88,
    profitability_score: 90,
    financial_health_score: 70,
    momentum_score: 65,
    analyst_consensus_score: 80,
    insider_activity_score: 55,
    institutional_score: 72,
    news_sentiment_score: 60,
    technical_score: 78,
    rating: 'Buy',
    sector_percentile: 85,
    confidence_level: 'High',
    data_completeness: 95,
    calculated_at: '2025-01-15T10:00:00Z',
    factor_count: 10,
    available_factors: ['value', 'growth'],
    missing_factors: [],
    ...overrides,
  };
}

// ──────────────────────────────────────────────────────────────
// getICScore
// ──────────────────────────────────────────────────────────────
describe('getICScore', () => {
  it('fetches IC Score for a ticker and returns data', async () => {
    const scoreData = buildICScoreData();
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ data: scoreData }),
    });

    const result = await getICScore('AAPL');

    expect(result).toEqual(scoreData);
    expect(mockFetch).toHaveBeenCalledTimes(1);
    const [url, options] = mockFetch.mock.calls[0];
    expect(url).toContain('/stocks/AAPL/ic-score');
    expect(options.method).toBe('GET');
  });

  it('uppercases ticker symbol', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ data: buildICScoreData({ ticker: 'MSFT' }) }),
    });

    await getICScore('msft');

    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain('/stocks/MSFT/ic-score');
  });

  it('returns null on 404', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 404,
      json: async () => ({ error: 'Not found' }),
    });

    const result = await getICScore('UNKNOWN');

    expect(result).toBeNull();
  });

  it('returns null on non-ok, non-404 response', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      json: async () => ({ error: 'Server error' }),
    });

    const result = await getICScore('AAPL');

    expect(result).toBeNull();
  });

  it('returns null on network error', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));

    const result = await getICScore('AAPL');

    expect(result).toBeNull();
  });

  it('sets cache to no-store', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ data: buildICScoreData() }),
    });

    await getICScore('AAPL');

    const [, options] = mockFetch.mock.calls[0];
    expect(options.cache).toBe('no-store');
  });
});

// ──────────────────────────────────────────────────────────────
// getICScoreHistory
// ──────────────────────────────────────────────────────────────
describe('getICScoreHistory', () => {
  it('fetches IC Score history with default days', async () => {
    const historyData = [buildICScoreData(), buildICScoreData({ date: '2025-01-14' })];
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ data: historyData }),
    });

    const result = await getICScoreHistory('AAPL');

    expect(result).toEqual(historyData);
    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain('/stocks/AAPL/ic-score/history?days=90');
  });

  it('passes custom days parameter', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ data: [] }),
    });

    await getICScoreHistory('AAPL', 30);

    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain('days=30');
  });

  it('uppercases ticker symbol', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ data: [] }),
    });

    await getICScoreHistory('aapl');

    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain('/stocks/AAPL/');
  });

  it('returns empty array on error response', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      json: async () => ({ error: 'Server error' }),
    });

    const result = await getICScoreHistory('AAPL');

    expect(result).toEqual([]);
  });

  it('returns empty array on network error', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));

    const result = await getICScoreHistory('AAPL');

    expect(result).toEqual([]);
  });
});

// ──────────────────────────────────────────────────────────────
// getICScores (admin, uses apiClient)
// ──────────────────────────────────────────────────────────────
describe('getICScores', () => {
  it('calls apiClient.get with no params', async () => {
    const response = {
      data: [],
      meta: { total: 0, limit: 20, offset: 0, total_stocks: 100, coverage_percent: 50 },
    };
    mockApiGet.mockResolvedValueOnce(response);

    const result = await getICScores();

    expect(mockApiGet).toHaveBeenCalledWith('/admin/ic-scores');
    expect(result).toEqual(response);
  });

  it('appends limit and offset params', async () => {
    const response = {
      data: [],
      meta: { total: 0, limit: 10, offset: 5, total_stocks: 100, coverage_percent: 50 },
    };
    mockApiGet.mockResolvedValueOnce(response);

    await getICScores({ limit: 10, offset: 5 });

    expect(mockApiGet).toHaveBeenCalledWith('/admin/ic-scores?limit=10&offset=5');
  });

  it('appends search param', async () => {
    mockApiGet.mockResolvedValueOnce({
      data: [],
      meta: { total: 0, limit: 20, offset: 0, total_stocks: 0, coverage_percent: 0 },
    });

    await getICScores({ search: 'AAPL' });

    expect(mockApiGet).toHaveBeenCalledWith('/admin/ic-scores?search=AAPL');
  });

  it('appends sort and order params', async () => {
    mockApiGet.mockResolvedValueOnce({
      data: [],
      meta: { total: 0, limit: 20, offset: 0, total_stocks: 0, coverage_percent: 0 },
    });

    await getICScores({ sort: 'overall_score', order: 'desc' });

    expect(mockApiGet).toHaveBeenCalledWith('/admin/ic-scores?sort=overall_score&order=desc');
  });

  it('appends all params together', async () => {
    mockApiGet.mockResolvedValueOnce({
      data: [],
      meta: { total: 0, limit: 10, offset: 20, total_stocks: 0, coverage_percent: 0 },
    });

    await getICScores({ limit: 10, offset: 20, search: 'TSLA', sort: 'ticker', order: 'asc' });

    expect(mockApiGet).toHaveBeenCalledWith(
      '/admin/ic-scores?limit=10&offset=20&search=TSLA&sort=ticker&order=asc'
    );
  });

  it('returns fallback on error', async () => {
    mockApiGet.mockRejectedValueOnce(new Error('Unauthorized'));

    const result = await getICScores({ limit: 10, offset: 5 });

    expect(result).toEqual({
      data: [],
      meta: { total: 0, limit: 10, offset: 5, total_stocks: 0, coverage_percent: 0 },
    });
  });

  it('returns default limit/offset in fallback when no params', async () => {
    mockApiGet.mockRejectedValueOnce(new Error('Server error'));

    const result = await getICScores();

    expect(result.meta.limit).toBe(20);
    expect(result.meta.offset).toBe(0);
  });
});

// ──────────────────────────────────────────────────────────────
// getLetterGrade
// ──────────────────────────────────────────────────────────────
describe('getLetterGrade', () => {
  it('returns N/A for null', () => {
    expect(getLetterGrade(null)).toBe('N/A');
  });

  it('returns A+ for score >= 97', () => {
    expect(getLetterGrade(97)).toBe('A+');
    expect(getLetterGrade(100)).toBe('A+');
  });

  it('returns A for score >= 93 and < 97', () => {
    expect(getLetterGrade(93)).toBe('A');
    expect(getLetterGrade(96)).toBe('A');
  });

  it('returns A- for score >= 90 and < 93', () => {
    expect(getLetterGrade(90)).toBe('A-');
    expect(getLetterGrade(92)).toBe('A-');
  });

  it('returns B+ for score >= 87 and < 90', () => {
    expect(getLetterGrade(87)).toBe('B+');
    expect(getLetterGrade(89)).toBe('B+');
  });

  it('returns B for score >= 83 and < 87', () => {
    expect(getLetterGrade(83)).toBe('B');
    expect(getLetterGrade(86)).toBe('B');
  });

  it('returns B- for score >= 80 and < 83', () => {
    expect(getLetterGrade(80)).toBe('B-');
    expect(getLetterGrade(82)).toBe('B-');
  });

  it('returns C+ for score >= 77 and < 80', () => {
    expect(getLetterGrade(77)).toBe('C+');
    expect(getLetterGrade(79)).toBe('C+');
  });

  it('returns C for score >= 73 and < 77', () => {
    expect(getLetterGrade(73)).toBe('C');
    expect(getLetterGrade(76)).toBe('C');
  });

  it('returns C- for score >= 70 and < 73', () => {
    expect(getLetterGrade(70)).toBe('C-');
    expect(getLetterGrade(72)).toBe('C-');
  });

  it('returns D+ for score >= 67 and < 70', () => {
    expect(getLetterGrade(67)).toBe('D+');
    expect(getLetterGrade(69)).toBe('D+');
  });

  it('returns D for score >= 63 and < 67', () => {
    expect(getLetterGrade(63)).toBe('D');
    expect(getLetterGrade(66)).toBe('D');
  });

  it('returns D- for score >= 60 and < 63', () => {
    expect(getLetterGrade(60)).toBe('D-');
    expect(getLetterGrade(62)).toBe('D-');
  });

  it('returns F for score < 60', () => {
    expect(getLetterGrade(59)).toBe('F');
    expect(getLetterGrade(0)).toBe('F');
  });
});

// ──────────────────────────────────────────────────────────────
// getScoreColor
// ──────────────────────────────────────────────────────────────
describe('getScoreColor', () => {
  it('returns muted for null', () => {
    expect(getScoreColor(null)).toBe('text-ic-text-muted');
  });

  it('returns green-600 for >= 80', () => {
    expect(getScoreColor(80)).toBe('text-green-600');
    expect(getScoreColor(100)).toBe('text-green-600');
  });

  it('returns green-500 for >= 65 and < 80', () => {
    expect(getScoreColor(65)).toBe('text-green-500');
    expect(getScoreColor(79)).toBe('text-green-500');
  });

  it('returns yellow-500 for >= 50 and < 65', () => {
    expect(getScoreColor(50)).toBe('text-yellow-500');
    expect(getScoreColor(64)).toBe('text-yellow-500');
  });

  it('returns orange-500 for >= 35 and < 50', () => {
    expect(getScoreColor(35)).toBe('text-orange-500');
    expect(getScoreColor(49)).toBe('text-orange-500');
  });

  it('returns red-500 for < 35', () => {
    expect(getScoreColor(34)).toBe('text-red-500');
    expect(getScoreColor(0)).toBe('text-red-500');
  });
});

// ──────────────────────────────────────────────────────────────
// getScoreBgColor
// ──────────────────────────────────────────────────────────────
describe('getScoreBgColor', () => {
  it('returns bg-ic-surface for null', () => {
    expect(getScoreBgColor(null)).toBe('bg-ic-surface');
  });

  it('returns bg-green-100 for >= 80', () => {
    expect(getScoreBgColor(80)).toBe('bg-green-100');
    expect(getScoreBgColor(100)).toBe('bg-green-100');
  });

  it('returns bg-green-50 for >= 65 and < 80', () => {
    expect(getScoreBgColor(65)).toBe('bg-green-50');
    expect(getScoreBgColor(79)).toBe('bg-green-50');
  });

  it('returns bg-yellow-50 for >= 50 and < 65', () => {
    expect(getScoreBgColor(50)).toBe('bg-yellow-50');
    expect(getScoreBgColor(64)).toBe('bg-yellow-50');
  });

  it('returns bg-orange-50 for >= 35 and < 50', () => {
    expect(getScoreBgColor(35)).toBe('bg-orange-50');
    expect(getScoreBgColor(49)).toBe('bg-orange-50');
  });

  it('returns bg-red-50 for < 35', () => {
    expect(getScoreBgColor(34)).toBe('bg-red-50');
    expect(getScoreBgColor(0)).toBe('bg-red-50');
  });
});

// ──────────────────────────────────────────────────────────────
// getFactorDetails
// ──────────────────────────────────────────────────────────────
describe('getFactorDetails', () => {
  it('returns 10 factors', () => {
    const factors = getFactorDetails(buildICScoreData());
    expect(factors).toHaveLength(10);
  });

  it('maps factor names correctly', () => {
    const factors = getFactorDetails(buildICScoreData());
    const names = factors.map((f) => f.name);
    expect(names).toEqual([
      'value',
      'growth',
      'profitability',
      'financial_health',
      'momentum',
      'analyst_consensus',
      'insider_activity',
      'institutional',
      'news_sentiment',
      'technical',
    ]);
  });

  it('maps scores from ICScoreData to factors', () => {
    const data = buildICScoreData({ value_score: 75, growth_score: 88 });
    const factors = getFactorDetails(data);
    expect(factors[0].score).toBe(75); // value
    expect(factors[1].score).toBe(88); // growth
  });

  it('computes letter grades for each factor', () => {
    const data = buildICScoreData({ value_score: 95, growth_score: null });
    const factors = getFactorDetails(data);
    expect(factors[0].grade).toBe('A'); // value_score 95
    expect(factors[1].grade).toBe('N/A'); // growth_score null
  });

  it('marks factors as available/unavailable based on null', () => {
    const data = buildICScoreData({ value_score: 75, momentum_score: null });
    const factors = getFactorDetails(data);
    const value = factors.find((f) => f.name === 'value');
    const momentum = factors.find((f) => f.name === 'momentum');
    expect(value!.available).toBe(true);
    expect(momentum!.available).toBe(false);
  });

  it('includes weights and descriptions for each factor', () => {
    const factors = getFactorDetails(buildICScoreData());
    // Growth factor should have weight 15
    const growth = factors.find((f) => f.name === 'growth');
    expect(growth!.weight).toBe(15);
    expect(growth!.description).toBe('Revenue, EPS growth trends');
  });
});

// ──────────────────────────────────────────────────────────────
// getStarRating
// ──────────────────────────────────────────────────────────────
describe('getStarRating', () => {
  it('returns 5 for score 100', () => {
    expect(getStarRating(100)).toBe(5);
  });

  it('returns 0 for score 0', () => {
    expect(getStarRating(0)).toBe(0);
  });

  it('returns 3 for score 50', () => {
    expect(getStarRating(50)).toBe(3);
  });

  it('rounds to nearest integer', () => {
    // 70/100 * 5 = 3.5 -> rounds to 4
    expect(getStarRating(70)).toBe(4);
    // 30/100 * 5 = 1.5 -> rounds to 2
    expect(getStarRating(30)).toBe(2);
  });
});
