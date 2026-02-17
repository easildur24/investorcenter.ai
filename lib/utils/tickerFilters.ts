/**
 * Ticker filtering utilities for identifying and filtering derivative securities
 * that don't file independent financial statements.
 *
 * @module tickerFilters
 */

/**
 * Identifies if a ticker is a derivative security that shouldn't have independent financials.
 *
 * Derivative securities include:
 * - Warrants (W, WW suffix)
 * - Rights (R suffix)
 * - Units (U suffix) - typically SPAC units combining common stock + warrant
 * - Preferred shares (N, O, P suffix) - report under parent company
 *
 * @param ticker - Stock ticker symbol
 * @returns true if ticker is a derivative security, false otherwise
 *
 * @example
 * ```typescript
 * isDerivativeSecurity('ABPWW')  // true - warrant
 * isDerivativeSecurity('AACBR')  // true - rights
 * isDerivativeSecurity('AACBU')  // true - units
 * isDerivativeSecurity('ACGLN')  // true - preferred series N
 * isDerivativeSecurity('AAPL')   // false - common stock
 * ```
 */
export function isDerivativeSecurity(ticker: string): boolean {
  if (!ticker || ticker.length < 1) {
    return false;
  }

  const upperTicker = ticker.toUpperCase();

  // Warrants - W or WW suffix
  if (upperTicker.endsWith('WW') || upperTicker.endsWith('W')) {
    return true;
  }

  // Rights - R suffix (typically short tickers like AACBR)
  if (upperTicker.endsWith('R') && upperTicker.length <= 6) {
    return true;
  }

  // Units - U suffix (typically short tickers like AACBU)
  if (upperTicker.endsWith('U') && upperTicker.length <= 6) {
    return true;
  }

  // Preferred shares - Series letters (N, O, P, etc.)
  // Usually appear as TICKERN, TICKERO, TICKERP for different preferred series
  if (upperTicker.length > 4) {
    const lastChar = upperTicker[upperTicker.length - 1];
    if (['N', 'O', 'P', 'Q'].includes(lastChar)) {
      return true;
    }
  }

  return false;
}

/**
 * Gets a human-readable description of the security type.
 *
 * @param ticker - Stock ticker symbol
 * @returns Description of the security type
 *
 * @example
 * ```typescript
 * getSecurityType('ABPWW')  // "Warrant"
 * getSecurityType('AACBR')  // "Rights"
 * getSecurityType('AACBU')  // "Unit (Stock + Warrant)"
 * getSecurityType('ACGLN')  // "Preferred Stock (Series N)"
 * getSecurityType('AAPL')   // "Common Stock"
 * ```
 */
export function getSecurityType(ticker: string): string {
  if (!ticker) return 'Unknown';

  const upperTicker = ticker.toUpperCase();

  if (upperTicker.endsWith('WW')) {
    return 'Warrant';
  }
  if (upperTicker.endsWith('W')) {
    return 'Warrant';
  }
  if (upperTicker.endsWith('R') && upperTicker.length <= 6) {
    return 'Rights';
  }
  if (upperTicker.endsWith('U') && upperTicker.length <= 6) {
    return 'Unit (Stock + Warrant)';
  }
  if (upperTicker.length > 4) {
    const lastChar = upperTicker[upperTicker.length - 1];
    if (['N', 'O', 'P', 'Q'].includes(lastChar)) {
      return `Preferred Stock (Series ${lastChar})`;
    }
  }

  return 'Common Stock';
}

/**
 * Filters an array of data to exclude derivative securities.
 *
 * @param data - Array of objects containing a ticker field
 * @param tickerField - Name of the field containing the ticker symbol (default: 'ticker')
 * @returns Filtered array excluding derivative securities
 *
 * @example
 * ```typescript
 * const valuationData = [
 *   { ticker: 'AAPL', pe_ratio: 25.5 },
 *   { ticker: 'ABPWW', pe_ratio: null },  // Will be filtered out
 *   { ticker: 'MSFT', pe_ratio: 32.1 }
 * ];
 *
 * const filtered = filterDerivatives(valuationData);
 * // Result: [{ ticker: 'AAPL', ... }, { ticker: 'MSFT', ... }]
 * ```
 */
export function filterDerivatives<T extends Record<string, any>>(
  data: T[],
  tickerField: keyof T = 'ticker' as keyof T
): T[] {
  return data.filter((item) => {
    const ticker = item[tickerField];
    if (typeof ticker !== 'string') {
      return true; // Keep items without valid ticker field
    }
    return !isDerivativeSecurity(ticker);
  });
}

/**
 * Categorizes tickers into common stocks and derivatives.
 *
 * @param data - Array of objects containing a ticker field
 * @param tickerField - Name of the field containing the ticker symbol
 * @returns Object with commonStocks and derivatives arrays
 *
 * @example
 * ```typescript
 * const data = [
 *   { ticker: 'AAPL', pe_ratio: 25.5 },
 *   { ticker: 'ABPWW', pe_ratio: null },
 *   { ticker: 'MSFT', pe_ratio: 32.1 }
 * ];
 *
 * const { commonStocks, derivatives } = categorizeTickers(data);
 * // commonStocks: [AAPL, MSFT]
 * // derivatives: [ABPWW]
 * ```
 */
export function categorizeTickers<T extends Record<string, any>>(
  data: T[],
  tickerField: keyof T = 'ticker' as keyof T
): { commonStocks: T[]; derivatives: T[] } {
  const commonStocks: T[] = [];
  const derivatives: T[] = [];

  data.forEach((item) => {
    const ticker = item[tickerField];
    if (typeof ticker !== 'string') {
      commonStocks.push(item);
      return;
    }

    if (isDerivativeSecurity(ticker)) {
      derivatives.push(item);
    } else {
      commonStocks.push(item);
    }
  });

  return { commonStocks, derivatives };
}
