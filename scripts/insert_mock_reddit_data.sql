-- Insert mock tickers for Reddit trending stocks
INSERT INTO tickers (symbol, name, exchange, asset_type, active) VALUES
('BYND', 'Beyond Meat Inc.', 'NASDAQ', 'stock', true),
('ASST', 'Asset Entities Inc.', 'NASDAQ', 'stock', true),
('SPY', 'SPDR S&P 500 ETF Trust', 'NYSE', 'etf', true),
('DTE', 'DTE Energy Co.', 'NYSE', 'stock', true),
('DFLI', 'Dragonfly Energy Holdings Corp.', 'NASDAQ', 'stock', true),
('GME', 'GameStop Corp.', 'NYSE', 'stock', true),
('TSLA', 'Tesla Inc.', 'NASDAQ', 'stock', true),
('RR', 'Rolls-Royce Holdings plc', 'OTC', 'stock', true),
('AMZN', 'Amazon.com Inc.', 'NASDAQ', 'stock', true),
('NVDA', 'NVIDIA Corporation', 'NASDAQ', 'stock', true),
('AAPL', 'Apple Inc.', 'NASDAQ', 'stock', true),
('MSFT', 'Microsoft Corporation', 'NASDAQ', 'stock', true),
('META', 'Meta Platforms Inc.', 'NASDAQ', 'stock', true),
('GOOGL', 'Alphabet Inc.', 'NASDAQ', 'stock', true),
('AMD', 'Advanced Micro Devices Inc.', 'NASDAQ', 'stock', true),
('PLTR', 'Palantir Technologies Inc.', 'NYSE', 'stock', true),
('SOFI', 'SoFi Technologies Inc.', 'NASDAQ', 'stock', true),
('NIO', 'NIO Inc.', 'NYSE', 'stock', true),
('RIVN', 'Rivian Automotive Inc.', 'NASDAQ', 'stock', true),
('LCID', 'Lucid Group Inc.', 'NASDAQ', 'stock', true)
ON CONFLICT (symbol) DO NOTHING;

-- Insert mock Reddit heatmap data for the last 7 days
INSERT INTO reddit_heatmap_daily (ticker_symbol, date, avg_rank, min_rank, max_rank, total_mentions, total_upvotes, rank_volatility, trend_direction, popularity_score, data_source) VALUES
-- BYND - Top trending
('BYND', CURRENT_DATE, 1.0, 1, 1, 363, 4299, 0.0, 'rising', 100.0, 'apewisdom'),
('BYND', CURRENT_DATE - 1, 4.0, 3, 5, 285, 3200, 1.5, 'rising', 85.0, 'apewisdom'),
('BYND', CURRENT_DATE - 2, 8.0, 6, 10, 220, 2500, 2.0, 'rising', 75.0, 'apewisdom'),
('BYND', CURRENT_DATE - 3, 12.0, 10, 15, 180, 2000, 2.5, 'stable', 65.0, 'apewisdom'),
('BYND', CURRENT_DATE - 4, 15.0, 12, 18, 150, 1800, 3.0, 'stable', 60.0, 'apewisdom'),
('BYND', CURRENT_DATE - 5, 18.0, 15, 22, 130, 1500, 3.5, 'falling', 55.0, 'apewisdom'),
('BYND', CURRENT_DATE - 6, 20.0, 17, 25, 110, 1200, 4.0, 'falling', 50.0, 'apewisdom'),

-- ASST
('ASST', CURRENT_DATE, 2.0, 2, 2, 185, 2100, 0.0, 'rising', 100.0, 'apewisdom'),
('ASST', CURRENT_DATE - 1, 7.0, 5, 9, 150, 1800, 2.0, 'rising', 80.0, 'apewisdom'),
('ASST', CURRENT_DATE - 2, 12.0, 10, 15, 120, 1500, 2.5, 'rising', 70.0, 'apewisdom'),

-- SPY
('SPY', CURRENT_DATE, 3.0, 3, 3, 69, 820, 0.0, 'stable', 57.8, 'apewisdom'),
('SPY', CURRENT_DATE - 1, 3.0, 2, 4, 72, 850, 1.0, 'stable', 58.0, 'apewisdom'),
('SPY', CURRENT_DATE - 2, 4.0, 3, 5, 68, 800, 1.0, 'stable', 56.5, 'apewisdom'),
('SPY', CURRENT_DATE - 3, 4.0, 3, 6, 65, 780, 1.5, 'stable', 55.0, 'apewisdom'),

-- DTE
('DTE', CURRENT_DATE, 4.0, 4, 4, 47, 550, 0.0, 'rising', 48.3, 'apewisdom'),
('DTE', CURRENT_DATE - 1, 5.0, 4, 6, 45, 520, 1.0, 'rising', 47.0, 'apewisdom'),
('DTE', CURRENT_DATE - 2, 8.0, 6, 10, 40, 480, 2.0, 'stable', 45.0, 'apewisdom'),

-- DFLI
('DFLI', CURRENT_DATE, 5.0, 5, 5, 46, 540, 0.0, 'falling', 48.3, 'apewisdom'),
('DFLI', CURRENT_DATE - 1, 3.0, 2, 4, 52, 600, 1.0, 'falling', 52.0, 'apewisdom'),
('DFLI', CURRENT_DATE - 2, 2.0, 1, 3, 58, 650, 1.0, 'falling', 55.0, 'apewisdom'),

-- GME
('GME', CURRENT_DATE, 6.0, 6, 6, 45, 520, 0.0, 'rising', 47.1, 'apewisdom'),
('GME', CURRENT_DATE - 1, 8.0, 7, 10, 42, 480, 1.5, 'rising', 44.0, 'apewisdom'),
('GME', CURRENT_DATE - 2, 12.0, 10, 15, 38, 430, 2.5, 'stable', 40.0, 'apewisdom'),
('GME', CURRENT_DATE - 3, 10.0, 8, 12, 40, 450, 2.0, 'stable', 42.0, 'apewisdom'),

-- TSLA
('TSLA', CURRENT_DATE, 7.0, 7, 7, 40, 470, 0.0, 'falling', 44.6, 'apewisdom'),
('TSLA', CURRENT_DATE - 1, 6.0, 5, 7, 43, 500, 1.0, 'stable', 46.0, 'apewisdom'),
('TSLA', CURRENT_DATE - 2, 5.0, 4, 6, 46, 530, 1.0, 'stable', 48.0, 'apewisdom'),
('TSLA', CURRENT_DATE - 3, 4.0, 3, 5, 50, 570, 1.0, 'rising', 50.0, 'apewisdom'),

-- RR
('RR', CURRENT_DATE, 8.0, 8, 8, 37, 430, 0.0, 'stable', 42.9, 'apewisdom'),
('RR', CURRENT_DATE - 1, 8.0, 7, 9, 38, 440, 1.0, 'stable', 43.0, 'apewisdom'),
('RR', CURRENT_DATE - 2, 9.0, 8, 11, 36, 420, 1.5, 'stable', 42.0, 'apewisdom'),

-- AMZN
('AMZN', CURRENT_DATE, 9.0, 9, 9, 35, 410, 0.0, 'rising', 42.1, 'apewisdom'),
('AMZN', CURRENT_DATE - 1, 13.0, 11, 15, 30, 350, 2.0, 'rising', 38.0, 'apewisdom'),
('AMZN', CURRENT_DATE - 2, 18.0, 15, 22, 25, 290, 3.5, 'stable', 33.0, 'apewisdom'),

-- NVDA
('NVDA', CURRENT_DATE, 10.0, 10, 10, 33, 390, 0.0, 'falling', 40.8, 'apewisdom'),
('NVDA', CURRENT_DATE - 1, 7.0, 6, 8, 38, 450, 1.0, 'falling', 44.0, 'apewisdom'),
('NVDA', CURRENT_DATE - 2, 5.0, 4, 6, 42, 490, 1.0, 'falling', 47.0, 'apewisdom'),
('NVDA', CURRENT_DATE - 3, 3.0, 2, 4, 48, 550, 1.0, 'falling', 51.0, 'apewisdom'),

-- AAPL
('AAPL', CURRENT_DATE, 11.0, 11, 11, 32, 375, 0.0, 'stable', 39.5, 'apewisdom'),
('AAPL', CURRENT_DATE - 1, 10.0, 9, 12, 33, 385, 1.5, 'stable', 40.0, 'apewisdom'),
('AAPL', CURRENT_DATE - 2, 11.0, 10, 13, 31, 365, 1.5, 'stable', 38.5, 'apewisdom'),

-- MSFT
('MSFT', CURRENT_DATE, 12.0, 12, 12, 30, 350, 0.0, 'stable', 38.2, 'apewisdom'),
('MSFT', CURRENT_DATE - 1, 11.0, 10, 13, 31, 360, 1.5, 'stable', 39.0, 'apewisdom'),

-- META
('META', CURRENT_DATE, 13.0, 13, 13, 28, 330, 0.0, 'rising', 36.8, 'apewisdom'),
('META', CURRENT_DATE - 1, 15.0, 13, 17, 26, 310, 2.0, 'rising', 34.5, 'apewisdom'),

-- GOOGL
('GOOGL', CURRENT_DATE, 14.0, 14, 14, 27, 320, 0.0, 'stable', 35.5, 'apewisdom'),
('GOOGL', CURRENT_DATE - 1, 14.0, 12, 16, 27, 315, 2.0, 'stable', 35.0, 'apewisdom'),

-- AMD
('AMD', CURRENT_DATE, 15.0, 15, 15, 25, 295, 0.0, 'falling', 33.9, 'apewisdom'),
('AMD', CURRENT_DATE - 1, 12.0, 10, 14, 29, 340, 2.0, 'falling', 37.0, 'apewisdom'),

-- PLTR
('PLTR', CURRENT_DATE, 16.0, 16, 16, 24, 285, 0.0, 'rising', 32.6, 'apewisdom'),
('PLTR', CURRENT_DATE - 1, 20.0, 18, 23, 20, 240, 2.5, 'rising', 28.0, 'apewisdom'),

-- SOFI
('SOFI', CURRENT_DATE, 17.0, 17, 17, 23, 270, 0.0, 'stable', 31.3, 'apewisdom'),
('SOFI', CURRENT_DATE - 1, 17.0, 15, 19, 23, 265, 2.0, 'stable', 31.0, 'apewisdom'),

-- NIO
('NIO', CURRENT_DATE, 18.0, 18, 18, 22, 260, 0.0, 'falling', 30.0, 'apewisdom'),
('NIO', CURRENT_DATE - 1, 16.0, 14, 18, 24, 280, 2.0, 'falling', 32.5, 'apewisdom'),

-- RIVN
('RIVN', CURRENT_DATE, 19.0, 19, 19, 21, 245, 0.0, 'rising', 28.7, 'apewisdom'),
('RIVN', CURRENT_DATE - 1, 22.0, 20, 25, 18, 210, 2.5, 'rising', 25.0, 'apewisdom'),

-- LCID
('LCID', CURRENT_DATE, 20.0, 20, 20, 20, 235, 0.0, 'stable', 27.4, 'apewisdom'),
('LCID', CURRENT_DATE - 1, 21.0, 19, 23, 19, 225, 2.0, 'stable', 26.5, 'apewisdom');

-- Verify data
SELECT COUNT(*) as "Total Heatmap Records" FROM reddit_heatmap_daily;
SELECT COUNT(DISTINCT ticker_symbol) as "Unique Tickers" FROM reddit_heatmap_daily;
SELECT ticker_symbol, COUNT(*) as days_of_data
FROM reddit_heatmap_daily
GROUP BY ticker_symbol
ORDER BY ticker_symbol;
