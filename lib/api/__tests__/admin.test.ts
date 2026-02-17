import {
  getAdminStocks,
  getAdminUsers,
  getAdminNews,
  getAdminFundamentals,
  getAdminAlerts,
  getAdminWatchLists,
  getAdminDatabaseStats,
  getAdminSECFinancials,
  getAdminTTMFinancials,
  getAdminValuationRatios,
  getAdminAnalystRatings,
  getAdminInsiderTrades,
  getAdminInstitutionalHoldings,
  getAdminTechnicalIndicators,
  getAdminCompanies,
  getAdminRiskMetrics,
} from '../admin';

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

const mockResponse = { data: [], meta: { total: 0, limit: 20, offset: 0 } };

// Helper to test admin endpoints that follow the same pattern
function testAdminEndpoint(
  name: string,
  fn: (params?: any) => Promise<any>,
  endpoint: string,
  supportsSearch: boolean = true
) {
  describe(name, () => {
    it(`calls GET ${endpoint} with no params`, async () => {
      mockGet.mockResolvedValueOnce(mockResponse);
      await fn();
      expect(mockGet).toHaveBeenCalledWith(endpoint);
    });

    it('appends limit and offset params', async () => {
      mockGet.mockResolvedValueOnce(mockResponse);
      await fn({ limit: 10, offset: 5 });
      const calledUrl = mockGet.mock.calls[0][0];
      expect(calledUrl).toContain('limit=10');
      expect(calledUrl).toContain('offset=5');
    });

    if (supportsSearch) {
      it('appends search param', async () => {
        mockGet.mockResolvedValueOnce(mockResponse);
        await fn({ search: 'AAPL' });
        expect(mockGet.mock.calls[0][0]).toContain('search=AAPL');
      });
    }
  });
}

testAdminEndpoint('getAdminStocks', getAdminStocks, '/admin/stocks');
testAdminEndpoint('getAdminUsers', getAdminUsers, '/admin/users');
testAdminEndpoint('getAdminNews', getAdminNews, '/admin/news');
testAdminEndpoint('getAdminFundamentals', getAdminFundamentals, '/admin/fundamentals');
testAdminEndpoint('getAdminAlerts', getAdminAlerts, '/admin/alerts', false);
testAdminEndpoint('getAdminWatchLists', getAdminWatchLists, '/admin/watchlists', false);
testAdminEndpoint('getAdminSECFinancials', getAdminSECFinancials, '/admin/sec-financials');
testAdminEndpoint('getAdminTTMFinancials', getAdminTTMFinancials, '/admin/ttm-financials');
testAdminEndpoint('getAdminValuationRatios', getAdminValuationRatios, '/admin/valuation-ratios');
testAdminEndpoint('getAdminAnalystRatings', getAdminAnalystRatings, '/admin/analyst-ratings');
testAdminEndpoint('getAdminInsiderTrades', getAdminInsiderTrades, '/admin/insider-trades');
testAdminEndpoint('getAdminInstitutionalHoldings', getAdminInstitutionalHoldings, '/admin/institutional-holdings');
testAdminEndpoint('getAdminTechnicalIndicators', getAdminTechnicalIndicators, '/admin/technical-indicators');
testAdminEndpoint('getAdminCompanies', getAdminCompanies, '/admin/companies');
testAdminEndpoint('getAdminRiskMetrics', getAdminRiskMetrics, '/admin/risk-metrics');

describe('getAdminDatabaseStats', () => {
  it('calls GET /admin/stats', async () => {
    mockGet.mockResolvedValueOnce({ stats: {} });
    await getAdminDatabaseStats();
    expect(mockGet).toHaveBeenCalledWith('/admin/stats');
  });
});

describe('getAdminStocks sorting', () => {
  it('appends sort and order params', async () => {
    mockGet.mockResolvedValueOnce(mockResponse);
    await getAdminStocks({ sort: 'symbol', order: 'asc' });
    const calledUrl = mockGet.mock.calls[0][0];
    expect(calledUrl).toContain('sort=symbol');
    expect(calledUrl).toContain('order=asc');
  });
});
