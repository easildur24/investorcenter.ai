import { FILTER_GROUPS, FILTER_DEFS, SECTORS } from '../filter-config';

describe('filter-config', () => {
  describe('SECTORS', () => {
    it('has 11 GICS sectors', () => {
      expect(SECTORS).toHaveLength(11);
    });

    it('includes Technology and Healthcare', () => {
      expect(SECTORS).toContain('Technology');
      expect(SECTORS).toContain('Healthcare');
    });
  });

  describe('FILTER_GROUPS', () => {
    it('has unique group IDs', () => {
      const ids = FILTER_GROUPS.map(g => g.id);
      expect(new Set(ids).size).toBe(ids.length);
    });

    it('general and ic_score default open', () => {
      const general = FILTER_GROUPS.find(g => g.id === 'general');
      const icScore = FILTER_GROUPS.find(g => g.id === 'ic_score');
      expect(general?.defaultOpen).toBe(true);
      expect(icScore?.defaultOpen).toBe(true);
    });
  });

  describe('FILTER_DEFS', () => {
    it('has unique filter IDs', () => {
      const ids = FILTER_DEFS.map(f => f.id);
      expect(new Set(ids).size).toBe(ids.length);
    });

    it('every filter references a valid group', () => {
      const groupIds = new Set(FILTER_GROUPS.map(g => g.id));
      for (const filter of FILTER_DEFS) {
        expect(groupIds.has(filter.group)).toBe(true);
      }
    });

    it('range filters have minKey and maxKey', () => {
      const rangeFilters = FILTER_DEFS.filter(f => f.type === 'range');
      for (const f of rangeFilters) {
        expect(f.minKey).toBeDefined();
        expect(f.maxKey).toBeDefined();
      }
    });

    it('multiselect filters have options', () => {
      const multiselects = FILTER_DEFS.filter(f => f.type === 'multiselect');
      for (const f of multiselects) {
        expect(f.options).toBeDefined();
        expect(f.options!.length).toBeGreaterThan(0);
      }
    });

    it('has filters in IC Score Factors group', () => {
      const icFactors = FILTER_DEFS.filter(f => f.group === 'ic_factors');
      expect(icFactors.length).toBeGreaterThanOrEqual(10);
    });
  });
});
