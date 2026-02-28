/**
 * Shared TypeScript interfaces for Peer Comparison & Fair Value features.
 *
 * Used by:
 *   - usePeerComparison hook
 *   - useFairValue hook
 *   - PeerComparisonPanel component
 *   - FairValueGauge component
 */

export interface PeerMetrics {
  pe_ratio: number;
  roe: number;
  revenue_growth_yoy: number;
  net_margin: number;
  debt_to_equity: number;
  market_cap: number;
}

export interface Peer {
  ticker: string;
  company_name: string;
  ic_score: number;
  similarity_score: number;
  metrics: PeerMetrics;
}

export interface PeersResponse {
  ticker: string;
  ic_score: number;
  peers: Peer[];
  stock_metrics: Record<string, number>;
  avg_peer_score: number;
  vs_peers_delta: number;
}

export interface FairValueModel {
  fair_value: number;
  upside_percent: number;
  confidence: string;
  inputs?: Record<string, number>;
}

export interface AnalystConsensus {
  target_price: number;
  upside_percent: number;
  num_analysts: number;
  consensus: string;
}

export interface MarginOfSafety {
  avg_fair_value: number;
  zone: 'undervalued' | 'fairly_valued' | 'overvalued';
  description: string;
}

export interface FairValueResponse {
  ticker: string;
  current_price: number;
  models: {
    dcf: FairValueModel;
    graham_number: FairValueModel;
    epv: FairValueModel;
  };
  analyst_consensus: AnalystConsensus | null;
  margin_of_safety: MarginOfSafety;
  meta: {
    suppressed: boolean;
    suppression_reason: string | null;
  };
}
