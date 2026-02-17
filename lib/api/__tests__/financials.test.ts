/**
 * Tests for lib/api/financials.ts
 *
 * This module uses fetch directly (not apiClient), so we mock global.fetch.
 */

import {
  getIncomeStatements,
  getBalanceSheets,
  getCashFlowStatements,
  getFinancialRatios,
  getAllFinancials,
  getFinancialStatements,
  refreshFinancials,
  hasFinancialData,
} from '../financials';

const mockFetch = global.fetch as jest.Mock;

beforeEach(() => {
  jest.clearAllMocks();
  mockFetch.mockReset();
});

// Helper: build a successful response
function mockOkResponse(data: any) {
  return {
    ok: true,
    status: 200,
    json: async () => ({ data }),
  };
}

function mock404Response() {
  return {
    ok: false,
    status: 404,
    json: async () => ({ error: 'Not found' }),
  };
}

function mock500Response() {
  return {
    ok: false,
    status: 500,
    json: async () => ({ error: 'Server error' }),
  };
}

describe('getIncomeStatements', () => {
  it('calls correct endpoint and returns data', async () => {
    const mockData = { periods: [{ revenue: 100 }] };
    mockFetch.mockResolvedValueOnce(mockOkResponse(mockData));

    const result = await getIncomeStatements('AAPL');
    expect(result).toEqual(mockData);
    expect(mockFetch.mock.calls[0][0]).toContain('/stocks/AAPL/financials/income');
  });

  it('uppercases ticker', async () => {
    mockFetch.mockResolvedValueOnce(mockOkResponse({}));
    await getIncomeStatements('aapl');
    expect(mockFetch.mock.calls[0][0]).toContain('/stocks/AAPL/');
  });

  it('appends query params', async () => {
    mockFetch.mockResolvedValueOnce(mockOkResponse({}));
    await getIncomeStatements('AAPL', { timeframe: 'annual', limit: 5 });
    const url = mockFetch.mock.calls[0][0];
    expect(url).toContain('timeframe=annual');
    expect(url).toContain('limit=5');
  });

  it('returns null on 404', async () => {
    mockFetch.mockResolvedValueOnce(mock404Response());
    const result = await getIncomeStatements('UNKNOWN');
    expect(result).toBeNull();
  });

  it('returns null on non-ok response', async () => {
    mockFetch.mockResolvedValueOnce(mock500Response());
    const result = await getIncomeStatements('AAPL');
    expect(result).toBeNull();
  });

  it('returns null on fetch error', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));
    const result = await getIncomeStatements('AAPL');
    expect(result).toBeNull();
  });
});

describe('getBalanceSheets', () => {
  it('calls correct endpoint', async () => {
    mockFetch.mockResolvedValueOnce(mockOkResponse({}));
    await getBalanceSheets('MSFT');
    expect(mockFetch.mock.calls[0][0]).toContain('/stocks/MSFT/financials/balance');
  });

  it('returns null on 404', async () => {
    mockFetch.mockResolvedValueOnce(mock404Response());
    expect(await getBalanceSheets('X')).toBeNull();
  });
});

describe('getCashFlowStatements', () => {
  it('calls correct endpoint', async () => {
    mockFetch.mockResolvedValueOnce(mockOkResponse({}));
    await getCashFlowStatements('GOOG');
    expect(mockFetch.mock.calls[0][0]).toContain('/stocks/GOOG/financials/cashflow');
  });

  it('returns null on error', async () => {
    mockFetch.mockRejectedValueOnce(new Error('fail'));
    expect(await getCashFlowStatements('GOOG')).toBeNull();
  });
});

describe('getFinancialRatios', () => {
  it('calls correct endpoint', async () => {
    mockFetch.mockResolvedValueOnce(mockOkResponse({}));
    await getFinancialRatios('TSLA');
    expect(mockFetch.mock.calls[0][0]).toContain('/stocks/TSLA/financials/ratios');
  });
});

describe('getAllFinancials', () => {
  it('calls correct endpoint', async () => {
    mockFetch.mockResolvedValueOnce(mockOkResponse({}));
    await getAllFinancials('NVDA');
    expect(mockFetch.mock.calls[0][0]).toContain('/stocks/NVDA/financials/all');
  });

  it('returns null on 404', async () => {
    mockFetch.mockResolvedValueOnce(mock404Response());
    expect(await getAllFinancials('X')).toBeNull();
  });
});

describe('getFinancialStatements', () => {
  it('dispatches to getIncomeStatements for income', async () => {
    mockFetch.mockResolvedValueOnce(mockOkResponse({}));
    await getFinancialStatements('AAPL', 'income');
    expect(mockFetch.mock.calls[0][0]).toContain('/financials/income');
  });

  it('dispatches to getBalanceSheets for balance_sheet', async () => {
    mockFetch.mockResolvedValueOnce(mockOkResponse({}));
    await getFinancialStatements('AAPL', 'balance_sheet');
    expect(mockFetch.mock.calls[0][0]).toContain('/financials/balance');
  });

  it('dispatches to getCashFlowStatements for cash_flow', async () => {
    mockFetch.mockResolvedValueOnce(mockOkResponse({}));
    await getFinancialStatements('AAPL', 'cash_flow');
    expect(mockFetch.mock.calls[0][0]).toContain('/financials/cashflow');
  });

  it('dispatches to getFinancialRatios for ratios', async () => {
    mockFetch.mockResolvedValueOnce(mockOkResponse({}));
    await getFinancialStatements('AAPL', 'ratios');
    expect(mockFetch.mock.calls[0][0]).toContain('/financials/ratios');
  });

  it('returns null for unknown statement type', async () => {
    const result = await getFinancialStatements('AAPL', 'unknown' as any);
    expect(result).toBeNull();
    expect(mockFetch).not.toHaveBeenCalled();
  });
});

describe('refreshFinancials', () => {
  it('calls POST to correct endpoint', async () => {
    mockFetch.mockResolvedValueOnce({ ok: true, status: 200 });
    const result = await refreshFinancials('AAPL');
    expect(result).toBe(true);
    const [url, options] = mockFetch.mock.calls[0];
    expect(url).toContain('/stocks/AAPL/financials/refresh');
    expect(options.method).toBe('POST');
  });

  it('returns false on failure', async () => {
    mockFetch.mockResolvedValueOnce({ ok: false, status: 500 });
    const result = await refreshFinancials('AAPL');
    expect(result).toBe(false);
  });

  it('returns false on network error', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));
    const result = await refreshFinancials('AAPL');
    expect(result).toBe(false);
  });
});

describe('hasFinancialData', () => {
  it('returns true when periods exist', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ data: { periods: [{ revenue: 100 }] } }),
    });
    const result = await hasFinancialData('AAPL');
    expect(result).toBe(true);
  });

  it('returns false when periods are empty', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ data: { periods: [] } }),
    });
    const result = await hasFinancialData('AAPL');
    expect(result).toBe(false);
  });

  it('returns false on non-ok response', async () => {
    mockFetch.mockResolvedValueOnce({ ok: false, status: 404 });
    const result = await hasFinancialData('X');
    expect(result).toBe(false);
  });

  it('returns false on fetch error', async () => {
    mockFetch.mockRejectedValueOnce(new Error('fail'));
    const result = await hasFinancialData('X');
    expect(result).toBe(false);
  });
});
