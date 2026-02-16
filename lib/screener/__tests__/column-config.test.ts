import {
  ALL_COLUMNS,
  DEFAULT_VISIBLE_KEYS,
  loadVisibleColumns,
  saveVisibleColumns,
} from '../column-config';
import type { ScreenerSortField } from '@/lib/types/screener';

describe('column-config', () => {
  describe('ALL_COLUMNS', () => {
    it('has unique keys', () => {
      const keys = ALL_COLUMNS.map(c => c.key);
      expect(new Set(keys).size).toBe(keys.length);
    });

    it('includes symbol and name as first two columns', () => {
      expect(ALL_COLUMNS[0].key).toBe('symbol');
      expect(ALL_COLUMNS[1].key).toBe('name');
    });

    it('has at least 20 columns', () => {
      expect(ALL_COLUMNS.length).toBeGreaterThanOrEqual(20);
    });

    it('every column has a format function', () => {
      const dummyStock = {
        symbol: 'TEST',
        name: 'Test Corp',
        sector: 'Technology',
        industry: null,
        market_cap: 1e9,
        price: 100,
        pe_ratio: 20,
        pb_ratio: 3,
        ps_ratio: 5,
        roe: 15,
        roa: 8,
        gross_margin: 45,
        operating_margin: 30,
        net_margin: 20,
        debt_to_equity: 0.5,
        current_ratio: 2,
        revenue_growth: 10,
        eps_growth_yoy: 12,
        dividend_yield: 2,
        payout_ratio: 40,
        consecutive_dividend_years: 5,
        beta: 1.1,
        dcf_upside_percent: 15,
        ic_score: 72,
        ic_rating: 'B+',
        value_score: 60,
        growth_score: 70,
        profitability_score: 80,
        financial_health_score: 65,
        momentum_score: 55,
        analyst_consensus_score: 75,
        insider_activity_score: 50,
        institutional_score: 60,
        news_sentiment_score: 65,
        technical_score: 70,
        ic_sector_percentile: 80,
        lifecycle_stage: 'mature',
      };

      for (const col of ALL_COLUMNS) {
        expect(typeof col.format(dummyStock)).toBe('string');
      }
    });

    it('handles null values in format functions', () => {
      const nullStock = {
        symbol: 'NULL',
        name: 'Null Corp',
        sector: null,
        industry: null,
        market_cap: null,
        price: null,
        pe_ratio: null,
        pb_ratio: null,
        ps_ratio: null,
        roe: null,
        roa: null,
        gross_margin: null,
        operating_margin: null,
        net_margin: null,
        debt_to_equity: null,
        current_ratio: null,
        revenue_growth: null,
        eps_growth_yoy: null,
        dividend_yield: null,
        payout_ratio: null,
        consecutive_dividend_years: null,
        beta: null,
        dcf_upside_percent: null,
        ic_score: null,
        ic_rating: null,
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
        ic_sector_percentile: null,
        lifecycle_stage: null,
      };

      for (const col of ALL_COLUMNS) {
        // Should not throw
        const result = col.format(nullStock);
        expect(typeof result).toBe('string');
      }
    });
  });

  describe('DEFAULT_VISIBLE_KEYS', () => {
    it('includes symbol, name, market_cap, price, ic_score', () => {
      expect(DEFAULT_VISIBLE_KEYS).toContain('symbol');
      expect(DEFAULT_VISIBLE_KEYS).toContain('name');
      expect(DEFAULT_VISIBLE_KEYS).toContain('market_cap');
      expect(DEFAULT_VISIBLE_KEYS).toContain('price');
      expect(DEFAULT_VISIBLE_KEYS).toContain('ic_score');
    });

    it('has between 5 and 15 default visible columns', () => {
      expect(DEFAULT_VISIBLE_KEYS.length).toBeGreaterThanOrEqual(5);
      expect(DEFAULT_VISIBLE_KEYS.length).toBeLessThanOrEqual(15);
    });
  });

  describe('loadVisibleColumns', () => {
    it('returns defaults when localStorage is empty', () => {
      const result = loadVisibleColumns();
      expect(result).toEqual(DEFAULT_VISIBLE_KEYS);
    });

    it('returns stored columns from localStorage', () => {
      const stored: ScreenerSortField[] = ['symbol', 'name', 'pe_ratio', 'ic_score'];
      localStorage.setItem('ic_screener_columns', JSON.stringify(stored));
      const result = loadVisibleColumns();
      expect(result).toEqual(stored);
    });

    it('filters out invalid column keys', () => {
      localStorage.setItem('ic_screener_columns', JSON.stringify(['symbol', 'invalid_key', 'name']));
      const result = loadVisibleColumns();
      expect(result).toEqual(['symbol', 'name']);
    });

    it('returns defaults when all stored keys are invalid', () => {
      localStorage.setItem('ic_screener_columns', JSON.stringify(['bad1', 'bad2']));
      const result = loadVisibleColumns();
      expect(result).toEqual(DEFAULT_VISIBLE_KEYS);
    });

    it('returns defaults on corrupted JSON', () => {
      localStorage.setItem('ic_screener_columns', 'not-json');
      const result = loadVisibleColumns();
      expect(result).toEqual(DEFAULT_VISIBLE_KEYS);
    });
  });

  describe('saveVisibleColumns', () => {
    it('saves to localStorage', () => {
      const cols: ScreenerSortField[] = ['symbol', 'pe_ratio'];
      saveVisibleColumns(cols);
      expect(localStorage.getItem('ic_screener_columns')).toBe(JSON.stringify(cols));
    });
  });
});
