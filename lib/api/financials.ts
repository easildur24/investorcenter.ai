/**
 * Financial Statements API Client
 *
 * Handles all API requests related to financial statements (Income Statement,
 * Balance Sheet, Cash Flow Statement) from SEC EDGAR filings.
 */

import {
  FinancialsResponse,
  AllFinancialsResponse,
  Timeframe,
  StatementType,
} from '@/types/financials';

import { stocks } from './routes';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

/**
 * API Response wrapper
 */
interface APIResponse<T> {
  data: T;
  meta?: {
    timestamp: string;
  };
}

/**
 * Error response from API
 */
interface APIError {
  error: string;
  message: string;
  ticker?: string;
}

/**
 * Query parameters for financial statements
 */
export interface FinancialsQueryParams {
  timeframe?: Timeframe;
  limit?: number;
  fiscal_year?: number;
  sort?: 'asc' | 'desc';
}

/**
 * Build query string from parameters
 */
function buildQueryString(params: FinancialsQueryParams): string {
  const queryParams = new URLSearchParams();

  if (params.timeframe) {
    queryParams.set('timeframe', params.timeframe);
  }
  if (params.limit) {
    queryParams.set('limit', params.limit.toString());
  }
  if (params.fiscal_year) {
    queryParams.set('fiscal_year', params.fiscal_year.toString());
  }
  if (params.sort) {
    queryParams.set('sort', params.sort);
  }

  const queryString = queryParams.toString();
  return queryString ? `?${queryString}` : '';
}

/**
 * Fetch income statements for a ticker
 *
 * @param ticker Stock symbol (e.g., "AAPL")
 * @param params Query parameters
 * @returns Income statement data or null if not available
 */
export async function getIncomeStatements(
  ticker: string,
  params: FinancialsQueryParams = {}
): Promise<FinancialsResponse | null> {
  try {
    const queryString = buildQueryString(params);
    const response = await fetch(
      `${API_BASE_URL}${stocks.financialsIncome(ticker.toUpperCase())}${queryString}`,
      {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        cache: 'no-store',
      }
    );

    if (response.status === 404) {
      return null;
    }

    if (!response.ok) {
      console.error(`Failed to fetch income statements for ${ticker}: ${response.status}`);
      return null;
    }

    const result: APIResponse<FinancialsResponse> = await response.json();
    return result.data;
  } catch (error) {
    console.error(`Error fetching income statements for ${ticker}:`, error);
    return null;
  }
}

/**
 * Fetch balance sheets for a ticker
 *
 * @param ticker Stock symbol (e.g., "AAPL")
 * @param params Query parameters
 * @returns Balance sheet data or null if not available
 */
export async function getBalanceSheets(
  ticker: string,
  params: FinancialsQueryParams = {}
): Promise<FinancialsResponse | null> {
  try {
    const queryString = buildQueryString(params);
    const response = await fetch(
      `${API_BASE_URL}${stocks.financialsBalance(ticker.toUpperCase())}${queryString}`,
      {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        cache: 'no-store',
      }
    );

    if (response.status === 404) {
      return null;
    }

    if (!response.ok) {
      console.error(`Failed to fetch balance sheets for ${ticker}: ${response.status}`);
      return null;
    }

    const result: APIResponse<FinancialsResponse> = await response.json();
    return result.data;
  } catch (error) {
    console.error(`Error fetching balance sheets for ${ticker}:`, error);
    return null;
  }
}

/**
 * Fetch cash flow statements for a ticker
 *
 * @param ticker Stock symbol (e.g., "AAPL")
 * @param params Query parameters
 * @returns Cash flow statement data or null if not available
 */
export async function getCashFlowStatements(
  ticker: string,
  params: FinancialsQueryParams = {}
): Promise<FinancialsResponse | null> {
  try {
    const queryString = buildQueryString(params);
    const response = await fetch(
      `${API_BASE_URL}${stocks.financialsCashflow(ticker.toUpperCase())}${queryString}`,
      {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        cache: 'no-store',
      }
    );

    if (response.status === 404) {
      return null;
    }

    if (!response.ok) {
      console.error(`Failed to fetch cash flow statements for ${ticker}: ${response.status}`);
      return null;
    }

    const result: APIResponse<FinancialsResponse> = await response.json();
    return result.data;
  } catch (error) {
    console.error(`Error fetching cash flow statements for ${ticker}:`, error);
    return null;
  }
}

/**
 * Fetch financial ratios for a ticker
 *
 * @param ticker Stock symbol (e.g., "AAPL")
 * @param params Query parameters
 * @returns Financial ratios data or null if not available
 */
export async function getFinancialRatios(
  ticker: string,
  params: FinancialsQueryParams = {}
): Promise<FinancialsResponse | null> {
  try {
    const queryString = buildQueryString(params);
    const response = await fetch(
      `${API_BASE_URL}${stocks.financialsRatios(ticker.toUpperCase())}${queryString}`,
      {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        cache: 'no-store',
      }
    );

    if (response.status === 404) {
      return null;
    }

    if (!response.ok) {
      console.error(`Failed to fetch financial ratios for ${ticker}: ${response.status}`);
      return null;
    }

    const result: APIResponse<FinancialsResponse> = await response.json();
    return result.data;
  } catch (error) {
    console.error(`Error fetching financial ratios for ${ticker}:`, error);
    return null;
  }
}

/**
 * Fetch all financial statements for a ticker
 *
 * @param ticker Stock symbol (e.g., "AAPL")
 * @param params Query parameters
 * @returns All financial statements data or null if not available
 */
export async function getAllFinancials(
  ticker: string,
  params: FinancialsQueryParams = {}
): Promise<AllFinancialsResponse | null> {
  try {
    const queryString = buildQueryString(params);
    const response = await fetch(
      `${API_BASE_URL}${stocks.financialsAll(ticker.toUpperCase())}${queryString}`,
      {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        cache: 'no-store',
      }
    );

    if (response.status === 404) {
      return null;
    }

    if (!response.ok) {
      console.error(`Failed to fetch all financials for ${ticker}: ${response.status}`);
      return null;
    }

    const result: APIResponse<AllFinancialsResponse> = await response.json();
    return result.data;
  } catch (error) {
    console.error(`Error fetching all financials for ${ticker}:`, error);
    return null;
  }
}

/**
 * Fetch financial statements by type
 *
 * @param ticker Stock symbol (e.g., "AAPL")
 * @param statementType Type of statement to fetch
 * @param params Query parameters
 * @returns Financial statement data or null if not available
 */
export async function getFinancialStatements(
  ticker: string,
  statementType: StatementType,
  params: FinancialsQueryParams = {}
): Promise<FinancialsResponse | null> {
  switch (statementType) {
    case 'income':
      return getIncomeStatements(ticker, params);
    case 'balance_sheet':
      return getBalanceSheets(ticker, params);
    case 'cash_flow':
      return getCashFlowStatements(ticker, params);
    case 'ratios':
      return getFinancialRatios(ticker, params);
    default:
      return null;
  }
}

/**
 * Refresh financial data for a ticker (forces re-fetch from Polygon.io)
 *
 * @param ticker Stock symbol (e.g., "AAPL")
 * @returns Success status
 */
export async function refreshFinancials(ticker: string): Promise<boolean> {
  try {
    const response = await fetch(
      `${API_BASE_URL}${stocks.financialsRefresh(ticker.toUpperCase())}`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      }
    );

    return response.ok;
  } catch (error) {
    console.error(`Error refreshing financials for ${ticker}:`, error);
    return false;
  }
}

/**
 * Check if financial data is available for a ticker
 *
 * @param ticker Stock symbol
 * @returns True if data is available
 */
export async function hasFinancialData(ticker: string): Promise<boolean> {
  try {
    const response = await fetch(
      `${API_BASE_URL}${stocks.financialsIncome(ticker.toUpperCase())}?limit=1`,
      {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      }
    );

    if (!response.ok) {
      return false;
    }

    const result: APIResponse<FinancialsResponse> = await response.json();
    return result.data.periods && result.data.periods.length > 0;
  } catch (error) {
    return false;
  }
}
