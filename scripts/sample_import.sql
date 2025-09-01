-- Ticker data import
-- Generated from demo_tickers.csv

-- Batch 1/1
INSERT INTO stocks (symbol, name, exchange, sector, industry, country, currency, market_cap, description, website)
VALUES
('AAPL', 'APPLE INC.', 'Nasdaq', NULL, NULL, 'US', 'USD', NULL, NULL, NULL),
  ('MSFT', 'MICROSOFT CORPORATION', 'Nasdaq', NULL, NULL, 'US', 'USD', NULL, NULL, NULL),
  ('GOOGL', 'ALPHABET INC.', 'Nasdaq', NULL, NULL, 'US', 'USD', NULL, NULL, NULL),
  ('TSLA', 'TESLA, INC.', 'Nasdaq', NULL, NULL, 'US', 'USD', NULL, NULL, NULL),
  ('AMZN', 'AMAZON.COM, INC.', 'Nasdaq', NULL, NULL, 'US', 'USD', NULL, NULL, NULL),
  ('META', 'META PLATFORMS, INC.', 'Nasdaq', NULL, NULL, 'US', 'USD', NULL, NULL, NULL),
  ('BAC', 'BANK OF AMERICA CORPORATION', 'NYSE', NULL, NULL, 'US', 'USD', NULL, NULL, NULL),
  ('JPM', 'JP MORGAN CHASE & CO', 'NYSE', NULL, NULL, 'US', 'USD', NULL, NULL, NULL),
  ('WFC', 'WELLS FARGO & COMPANY', 'NYSE', NULL, NULL, 'US', 'USD', NULL, NULL, NULL),
  ('C', 'CITIGROUP, INC.', 'NYSE', NULL, NULL, 'US', 'USD', NULL, NULL, NULL),
  ('GS', 'GOLDMAN SACHS GROUP, INC. (THE)', 'NYSE', NULL, NULL, 'US', 'USD', NULL, NULL, NULL),
  ('O'REILLY', 'O''REILLY AUTOMOTIVE, INC.', 'Nasdaq', NULL, NULL, 'US', 'USD', NULL, NULL, NULL)
ON CONFLICT (symbol) DO NOTHING;
