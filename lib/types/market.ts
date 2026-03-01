/**
 * Shared market types used across home page widgets.
 * These interfaces define the data contracts between backend and frontend.
 */

// ─── Market Asset (Index / ETF / Stock) ──────────────────────────────────────

export interface MarketAsset {
  symbol: string;
  name: string;
  price: number;
  change: number;
  changePercent: number;
  lastUpdated: string;
  displayFormat: 'points' | 'usd';
  dataType: 'index' | 'etf_proxy' | 'etf' | 'stock';
  sparkline?: SparklinePoint[];
}

// ─── Sparkline ───────────────────────────────────────────────────────────────

export interface SparklinePoint {
  timestamp: number; // Unix timestamp (ms)
  value: number;
}

// ─── Market Status ───────────────────────────────────────────────────────────

export type MarketState = 'pre_market' | 'regular' | 'after_hours' | 'closed';

export interface MarketStatus {
  state: MarketState;
  label: string;
  color: 'green' | 'amber' | 'grey';
  nextEvent: string; // e.g., "Opens in 2h 15m" or "Closes in 1h 30m"
  nextEventTime: Date;
}

// ─── News Article ────────────────────────────────────────────────────────────

export interface NewsArticle {
  id: string;
  title: string;
  summary: string;
  source: string;
  sourceLogoUrl?: string;
  url: string;
  publishedAt: string; // ISO 8601
  category?: 'earnings' | 'macro' | 'sector' | 'company' | 'general';
  tickers?: string[];
  sentiment?: 'positive' | 'negative' | 'neutral';
  imageUrl?: string;
}

// ─── Earnings Entry ──────────────────────────────────────────────────────────

export interface EarningsEntry {
  symbol: string;
  companyName: string;
  date: string; // YYYY-MM-DD
  time: 'bmo' | 'amc' | 'dmh' | '--'; // before market open / after market close / during market hours
  epsEstimate: number | null;
  epsActual: number | null;
  epsSurprise: number | null;
  revenueEstimate: number | null;
  revenueActual: number | null;
  marketCap?: number;
  sector?: string;
  logoUrl?: string;
}

// ─── Market Summary ──────────────────────────────────────────────────────────

export interface MarketSummary {
  text: string;
  timestamp: string; // ISO 8601
  method: 'template' | 'llm';
  scope: string; // e.g., "us_equities"
}

// ─── GICS Sector ─────────────────────────────────────────────────────────────

export interface GICSSector {
  id: string;
  label: string;
  color: string; // Light mode color
  darkColor: string; // Dark mode color
}

// ─── Market Overview Tab Config ──────────────────────────────────────────────

export interface MarketOverviewTab {
  id: string;
  label: string;
  category: string; // Query param value for API
  description?: string;
}

// ─── Mover Stock (enhanced) ──────────────────────────────────────────────────

export interface MoverStock {
  symbol: string;
  name?: string;
  price: number;
  change: number;
  changePercent: number;
  volume: number;
  sector?: string;
  avgVolume20d?: number;
  sparkline?: SparklinePoint[];
}
