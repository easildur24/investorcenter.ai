/**
 * Shared TypeScript interfaces for all fundamentals API responses.
 * Used across sector percentiles, health summary, peers, fair value, and metric history.
 */

// ─── Sector Percentiles ───────────────────────────────────────────────────────

export interface SectorPercentilesResponse {
  ticker: string;
  sector: string;
  calculated_at: string;
  sample_count: number;
  metrics: Record<string, MetricPercentileData>;
  meta: {
    source: string;
    metric_count: number;
    timestamp: string;
  };
}

export interface MetricPercentileData {
  value: number;
  percentile: number;
  lower_is_better: boolean;
  distribution: PercentileDistribution;
  sample_count: number;
}

export interface PercentileDistribution {
  min: number;
  p10: number;
  p25: number;
  p50: number;
  p75: number;
  p90: number;
  max: number;
}

// ─── Health Summary ───────────────────────────────────────────────────────────

export interface HealthSummaryResponse {
  ticker: string;
  health: {
    badge: 'Strong' | 'Healthy' | 'Fair' | 'Weak' | 'Distressed';
    score: number;
    components: {
      piotroski_f_score: {
        value: number;
        max: number;
        interpretation: string;
      };
      altman_z_score: {
        value: number;
        zone: 'safe' | 'grey' | 'distress';
        interpretation: string;
      };
      ic_financial_health: {
        value: number;
        max: number;
      };
      debt_percentile: {
        value: number;
        interpretation: string;
      };
    };
  };
  lifecycle: {
    stage: 'hypergrowth' | 'growth' | 'mature' | 'value' | 'turnaround';
    description: string;
    classified_at: string;
  };
  strengths: Array<{
    metric: string;
    value: number;
    percentile: number;
    message: string;
  }>;
  concerns: Array<{
    metric: string;
    value: number;
    percentile: number;
    message: string;
  }>;
  red_flags: Array<{
    id: string;
    severity: 'high' | 'medium' | 'low';
    title: string;
    description: string;
    related_metrics: string[];
  }>;
}

// ─── Peers ────────────────────────────────────────────────────────────────────

export interface PeersResponse {
  ticker: string;
  ic_score: number;
  peers: Array<{
    ticker: string;
    company_name: string;
    ic_score: number;
    similarity_score: number;
    metrics: {
      pe_ratio: number;
      roe: number;
      revenue_growth_yoy: number;
      net_margin: number;
      debt_to_equity: number;
      market_cap: number;
    };
  }>;
  stock_metrics: Record<string, number>;
  avg_peer_score: number;
  vs_peers_delta: number;
}

// ─── Fair Value ───────────────────────────────────────────────────────────────

export interface FairValueResponse {
  ticker: string;
  current_price: number;
  models: {
    dcf: {
      fair_value: number;
      upside_percent: number;
      confidence: string;
      inputs: Record<string, number>;
    };
    graham_number: {
      fair_value: number;
      upside_percent: number;
      confidence: string;
    };
    epv: {
      fair_value: number;
      upside_percent: number;
      confidence: string;
    };
  };
  analyst_consensus: {
    target_price: number;
    upside_percent: number;
    num_analysts: number;
    consensus: string;
  } | null;
  margin_of_safety: {
    avg_fair_value: number;
    zone: 'undervalued' | 'fairly_valued' | 'overvalued';
    description: string;
  };
  meta: {
    suppressed: boolean;
    suppression_reason: string | null;
  };
}

// ─── Metric History ───────────────────────────────────────────────────────────

export interface MetricHistoryResponse {
  ticker: string;
  metric: string;
  timeframe: 'quarterly' | 'annual';
  unit: 'USD' | 'percent' | 'ratio';
  data_points: Array<{
    period_end: string;
    fiscal_year: number;
    fiscal_quarter: number;
    value: number;
    yoy_change: number | null;
  }>;
  trend: {
    direction: 'up' | 'down' | 'flat';
    slope: number;
    consecutive_growth_quarters: number;
  };
}
