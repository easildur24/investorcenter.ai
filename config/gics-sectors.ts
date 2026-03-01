/**
 * 11 GICS (Global Industry Classification Standard) sectors.
 * Used for sector tags, heatmap coloring, and sector filtering.
 */

import type { GICSSector } from '@/lib/types/market';

export const GICS_SECTORS: GICSSector[] = [
  { id: 'technology',            label: 'Technology',            color: '#3B82F6', darkColor: '#60A5FA' },
  { id: 'healthcare',            label: 'Healthcare',            color: '#10B981', darkColor: '#34D399' },
  { id: 'financials',            label: 'Financials',            color: '#F59E0B', darkColor: '#FBBF24' },
  { id: 'consumer_discretionary', label: 'Consumer Disc.',       color: '#EC4899', darkColor: '#F472B6' },
  { id: 'consumer_staples',      label: 'Consumer Staples',     color: '#8B5CF6', darkColor: '#A78BFA' },
  { id: 'energy',                label: 'Energy',                color: '#EF4444', darkColor: '#F87171' },
  { id: 'industrials',           label: 'Industrials',           color: '#6366F1', darkColor: '#818CF8' },
  { id: 'materials',             label: 'Materials',             color: '#14B8A6', darkColor: '#2DD4BF' },
  { id: 'real_estate',           label: 'Real Estate',           color: '#F97316', darkColor: '#FB923C' },
  { id: 'utilities',             label: 'Utilities',             color: '#06B6D4', darkColor: '#22D3EE' },
  { id: 'communication_services', label: 'Communication',        color: '#D946EF', darkColor: '#E879F9' },
];

/** Map sector name (from DB/API) to GICS config — case-insensitive match */
export function getSectorConfig(sectorName: string): GICSSector | undefined {
  if (!sectorName) return undefined;
  const normalized = sectorName.toLowerCase().replace(/[\s-]+/g, '_');
  return GICS_SECTORS.find(
    (s) =>
      s.id === normalized ||
      s.label.toLowerCase().replace(/[\s.-]+/g, '_') === normalized ||
      sectorName.toLowerCase().includes(s.label.toLowerCase().split(' ')[0].toLowerCase())
  );
}
