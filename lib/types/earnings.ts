// Earnings data types matching backend response shapes

export interface EarningsResult {
  symbol: string;
  date: string;
  fiscalQuarter: string;
  epsEstimated: number | null;
  epsActual: number | null;
  epsSurprisePercent: number | null;
  epsBeat: boolean | null;
  revenueEstimated: number | null;
  revenueActual: number | null;
  revenueSurprisePercent: number | null;
  revenueBeat: boolean | null;
  isUpcoming: boolean;
}

export interface BeatRate {
  epsBeats: number;
  revenueBeats: number;
  totalQuarters: number;
}

export interface EarningsResponse {
  earnings: EarningsResult[];
  nextEarnings: EarningsResult | null;
  beatRate: BeatRate | null;
}

export interface EarningsCalendarResponse {
  earnings: EarningsResult[];
  earningsCounts: Record<string, number>;
}
