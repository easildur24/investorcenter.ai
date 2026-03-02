/**
 * Centralized API route paths — SINGLE SOURCE OF TRUTH.
 *
 * All paths are relative to the API base URL (e.g., /api/v1).
 * Cross-referenced against backend/main.go for correctness.
 *
 * Usage with apiClient:  apiClient.get(routes.watchlists.list)
 * Usage with raw fetch:  fetch(`${API_BASE_URL}${routes.tickers.price('AAPL')}`)
 *
 * When adding a new backend endpoint, add the route here FIRST,
 * then use it in the calling code.
 */

// ─── Auth ──────────────────────────────────────────────────────────────────────
export const auth = {
  login: '/auth/login',
  signup: '/auth/signup',
  refresh: '/auth/refresh',
  logout: '/auth/logout',
  verifyEmail: '/auth/verify-email',
  forgotPassword: '/auth/forgot-password',
  resetPassword: '/auth/reset-password',
} as const;

// ─── User ──────────────────────────────────────────────────────────────────────
export const user = {
  me: '/user/me',
  password: '/user/password',
} as const;

// ─── Markets ───────────────────────────────────────────────────────────────────
export const markets = {
  indices: '/markets/indices',
  movers: '/markets/movers',
  search: '/markets/search',
  summary: '/markets/summary',
} as const;

// ─── Tickers ───────────────────────────────────────────────────────────────────
export const tickers = {
  bySymbol: (symbol: string) => `/tickers/${symbol}`,
  chart: (symbol: string) => `/tickers/${symbol}/chart`,
  price: (symbol: string) => `/tickers/${symbol}/price`,
  volume: (symbol: string) => `/tickers/${symbol}/volume`,
  volumeAggregates: (symbol: string) => `/tickers/${symbol}/volume/aggregates`,
  news: (symbol: string) => `/tickers/${symbol}/news`,
  keyStats: (symbol: string) => `/tickers/${symbol}/keystats`,
  xPosts: (symbol: string) => `/tickers/${symbol}/x-posts`,
} as const;

// ─── Stocks (IC Score, Financials, Risk, Technical, Earnings) ───────────────────
export const stocks = {
  icScore: (ticker: string) => `/stocks/${ticker}/ic-score`,
  icScoreHistory: (ticker: string) => `/stocks/${ticker}/ic-score/history`,
  financials: (ticker: string) => `/stocks/${ticker}/financials`,
  metrics: (ticker: string) => `/stocks/${ticker}/metrics`,
  risk: (ticker: string) => `/stocks/${ticker}/risk`,
  technical: (ticker: string) => `/stocks/${ticker}/technical`,
  earnings: (ticker: string) => `/stocks/${ticker}/earnings`,
  financialsAll: (ticker: string) => `/stocks/${ticker}/financials/all`,
  financialsIncome: (ticker: string) => `/stocks/${ticker}/financials/income`,
  financialsBalance: (ticker: string) => `/stocks/${ticker}/financials/balance`,
  financialsCashflow: (ticker: string) => `/stocks/${ticker}/financials/cashflow`,
  financialsRatios: (ticker: string) => `/stocks/${ticker}/financials/ratios`,
  financialsRefresh: (ticker: string) => `/stocks/${ticker}/financials/refresh`,

  // Fundamentals enhancement endpoints (Project 1)
  sectorPercentiles: (ticker: string) => `/stocks/${ticker}/sector-percentiles`,
  peers: (ticker: string) => `/stocks/${ticker}/peers`,
  fairValue: (ticker: string) => `/stocks/${ticker}/fair-value`,
  healthSummary: (ticker: string) => `/stocks/${ticker}/health-summary`,
  metricHistory: (ticker: string, metric: string) => `/stocks/${ticker}/metric-history/${metric}`,
} as const;

// ─── Backtest ──────────────────────────────────────────────────────────────────
export const backtest = {
  latest: '/ic-scores/backtest/latest',
  defaultConfig: '/ic-scores/backtest/config/default',
  quick: '/ic-scores/backtest/quick',
  run: '/ic-scores/backtest',
  charts: '/ic-scores/backtest/charts',
  submitJob: '/ic-scores/backtest/jobs',
  jobStatus: (jobId: string) => `/ic-scores/backtest/jobs/${jobId}`,
  jobResult: (jobId: string) => `/ic-scores/backtest/jobs/${jobId}/result`,
} as const;

// ─── Crypto ────────────────────────────────────────────────────────────────────
export const crypto = {
  list: '/crypto/',
} as const;

// ─── Reddit ────────────────────────────────────────────────────────────────────
export const reddit = {
  heatmap: '/reddit/heatmap',
  tickerHistory: (symbol: string) => `/reddit/ticker/${symbol}/history`,
  health: '/reddit/health', // BUG-004: Pipeline freshness endpoint
} as const;

// ─── Sentiment ─────────────────────────────────────────────────────────────────
export const sentiment = {
  byTicker: (ticker: string) => `/sentiment/${ticker}`,
  history: (ticker: string) => `/sentiment/${ticker}/history`,
  trending: '/sentiment/trending',
  posts: (ticker: string) => `/sentiment/${ticker}/posts`,
} as const;

// ─── Screener ──────────────────────────────────────────────────────────────────
export const screener = {
  stocks: '/screener/stocks',
} as const;

// ─── Earnings Calendar ────────────────────────────────────────────────────────
export const earningsCalendar = {
  /** Query params: ?from=YYYY-MM-DD&to=YYYY-MM-DD (max 14-day window) */
  list: '/earnings-calendar',
} as const;

// ─── Logos ─────────────────────────────────────────────────────────────────────
export const logos = {
  bySymbol: (symbol: string) => `/logos/${symbol}`,
} as const;

// ─── Watchlists ────────────────────────────────────────────────────────────────
export const watchlists = {
  list: '/watchlists',
  create: '/watchlists',
  tags: '/watchlists/tags',
  byId: (id: string) => `/watchlists/${id}`,
  items: {
    add: (id: string) => `/watchlists/${id}/items`,
    remove: (id: string, symbol: string) => `/watchlists/${id}/items/${symbol}`,
    update: (id: string, symbol: string) => `/watchlists/${id}/items/${symbol}`,
  },
  bulk: (id: string) => `/watchlists/${id}/bulk`,
  reorder: (id: string) => `/watchlists/${id}/reorder`,
  heatmap: {
    data: (id: string) => `/watchlists/${id}/heatmap`,
    configs: (id: string) => `/watchlists/${id}/heatmap/configs`,
    config: (id: string, configId: string) => `/watchlists/${id}/heatmap/configs/${configId}`,
  },
} as const;

// ─── Alerts ────────────────────────────────────────────────────────────────────
export const alerts = {
  list: '/alerts',
  create: '/alerts',
  bulk: '/alerts/bulk',
  byId: (id: string) => `/alerts/${id}`,
  logs: {
    list: '/alerts/logs',
    read: (id: string) => `/alerts/logs/${id}/read`,
    dismiss: (id: string) => `/alerts/logs/${id}/dismiss`,
  },
} as const;

// ─── Notifications ─────────────────────────────────────────────────────────────
export const notifications = {
  list: '/notifications',
  unreadCount: '/notifications/unread-count',
  read: (id: string) => `/notifications/${id}/read`,
  readAll: '/notifications/read-all',
  dismiss: (id: string) => `/notifications/${id}/dismiss`,
  preferences: '/notifications/preferences',
} as const;

// ─── Subscriptions ─────────────────────────────────────────────────────────────
export const subscriptions = {
  create: '/subscriptions',
  plans: '/subscriptions/plans',
  planById: (id: string) => `/subscriptions/plans/${id}`,
  me: '/subscriptions/me',
  cancel: '/subscriptions/me/cancel',
  limits: '/subscriptions/limits',
  payments: '/subscriptions/payments',
} as const;

// ─── Admin ─────────────────────────────────────────────────────────────────────
export const admin = {
  stats: '/admin/stats',
  stocks: '/admin/stocks',
  users: '/admin/users',
  news: '/admin/news',
  fundamentals: '/admin/fundamentals',
  secFinancials: '/admin/sec-financials',
  ttmFinancials: '/admin/ttm-financials',
  valuationRatios: '/admin/valuation-ratios',
  alerts: '/admin/alerts',
  watchlists: '/admin/watchlists',
  analystRatings: '/admin/analyst-ratings',
  insiderTrades: '/admin/insider-trades',
  institutionalHoldings: '/admin/institutional-holdings',
  technicalIndicators: '/admin/technical-indicators',
  companies: '/admin/companies',
  riskMetrics: '/admin/risk-metrics',
  icScores: '/admin/ic-scores',
  cronjobs: {
    overview: '/admin/cronjobs/overview',
    schedules: '/admin/cronjobs/schedules',
    metrics: '/admin/cronjobs/metrics',
    jobHistory: (jobName: string) => `/admin/cronjobs/${jobName}/history`,
    details: (executionId: string) => `/admin/cronjobs/details/${executionId}`,
  },
  notes: {
    tree: '/admin/notes/tree',
    groups: {
      list: '/admin/notes/groups',
      byId: (id: string) => `/admin/notes/groups/${id}`,
    },
    features: {
      byGroup: (groupId: string) => `/admin/notes/groups/${groupId}/features`,
      byId: (id: string) => `/admin/notes/features/${id}`,
    },
    noteEntries: {
      byFeature: (featureId: string) => `/admin/notes/features/${featureId}/notes`,
      byId: (id: string) => `/admin/notes/notes/${id}`,
    },
  },
  workers: {
    list: '/admin/workers',
    byId: (id: string) => `/admin/workers/${id}`,
    taskTypes: {
      list: '/admin/workers/task-types',
      byId: (id: number) => `/admin/workers/task-types/${id}`,
    },
    tasks: {
      list: '/admin/workers/tasks',
      byId: (id: string) => `/admin/workers/tasks/${id}`,
      updates: (taskId: string) => `/admin/workers/tasks/${taskId}/updates`,
      data: (taskId: string) => `/admin/workers/tasks/${taskId}/data`,
      files: (taskId: string) => `/admin/workers/tasks/${taskId}/files`,
      fileDownload: (taskId: string, fileId: number) =>
        `/admin/workers/tasks/${taskId}/files/${fileId}/download`,
    },
  },
} as const;

// ─── Worker Portal ─────────────────────────────────────────────────────────────
export const worker = {
  tasks: '/worker/tasks',
  taskById: (id: string) => `/worker/tasks/${id}`,
  taskStatus: (id: string) => `/worker/tasks/${id}/status`,
  taskUpdates: (taskId: string) => `/worker/tasks/${taskId}/updates`,
  heartbeat: '/worker/heartbeat',
} as const;

// ─── IC Score Service (SEPARATE BASE URL: IC_SCORE_API_BASE) ───────────────────
// These routes target the IC Score Python service, NOT the Go backend.
// Used only by lib/api.ts icScoreApi.
export const icScoreService = {
  score: (ticker: string) => `/api/scores/${ticker}`,
  history: (ticker: string) => `/api/scores/${ticker}/history`,
  top: '/api/scores/top',
  screener: '/api/scores/screener',
  statistics: '/api/scores/statistics',
  health: '/health',
} as const;
