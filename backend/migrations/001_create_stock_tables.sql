-- Create extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Stocks table
CREATE TABLE stocks (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    exchange VARCHAR(50),
    sector VARCHAR(100),
    industry VARCHAR(100),
    country VARCHAR(50) DEFAULT 'US',
    currency VARCHAR(3) DEFAULT 'USD',
    market_cap DECIMAL(20,2),
    description TEXT,
    website VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Stock prices table (for historical and current prices)
CREATE TABLE stock_prices (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    price DECIMAL(15,4) NOT NULL,
    open DECIMAL(15,4),
    high DECIMAL(15,4),
    low DECIMAL(15,4),
    close DECIMAL(15,4),
    volume BIGINT,
    change DECIMAL(15,4),
    change_percent DECIMAL(8,4),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    FOREIGN KEY (symbol) REFERENCES stocks(symbol) ON DELETE CASCADE
);

-- Fundamentals table
CREATE TABLE fundamentals (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    period VARCHAR(10) NOT NULL, -- Q1, Q2, Q3, Q4, FY
    year INTEGER NOT NULL,
    
    -- Valuation Metrics
    pe DECIMAL(10,2),
    peg DECIMAL(10,2),
    pb DECIMAL(10,2),
    ps DECIMAL(10,2),
    ev DECIMAL(20,2),
    ev_revenue DECIMAL(10,2),
    ev_ebitda DECIMAL(10,2),
    
    -- Per Share Metrics
    eps DECIMAL(10,4),
    eps_diluted DECIMAL(10,4),
    book_value_per_share DECIMAL(10,2),
    tangible_book_value DECIMAL(10,2),
    
    -- Income Statement
    revenue DECIMAL(20,2),
    gross_profit DECIMAL(20,2),
    operating_income DECIMAL(20,2),
    ebitda DECIMAL(20,2),
    net_income DECIMAL(20,2),
    
    -- Margins (as percentages)
    gross_margin DECIMAL(8,4),
    operating_margin DECIMAL(8,4),
    net_margin DECIMAL(8,4),
    
    -- Returns (as percentages)
    roe DECIMAL(8,4),
    roa DECIMAL(8,4),
    roic DECIMAL(8,4),
    
    -- Balance Sheet
    total_assets DECIMAL(20,2),
    total_liabilities DECIMAL(20,2),
    total_equity DECIMAL(20,2),
    total_debt DECIMAL(20,2),
    cash DECIMAL(20,2),
    
    -- Ratios
    debt_to_equity DECIMAL(10,2),
    current_ratio DECIMAL(10,2),
    quick_ratio DECIMAL(10,2),
    
    -- Cash Flow
    operating_cash_flow DECIMAL(20,2),
    free_cash_flow DECIMAL(20,2),
    capex DECIMAL(20,2),
    
    -- Share Information
    shares_outstanding BIGINT,
    shares_float BIGINT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    FOREIGN KEY (symbol) REFERENCES stocks(symbol) ON DELETE CASCADE,
    UNIQUE(symbol, period, year)
);

-- Dividends table
CREATE TABLE dividends (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    ex_date DATE NOT NULL,
    pay_date DATE,
    amount DECIMAL(10,4) NOT NULL,
    frequency VARCHAR(20), -- Monthly, Quarterly, Annual
    type VARCHAR(20) DEFAULT 'Regular', -- Regular, Special
    yield_percent DECIMAL(8,4),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    FOREIGN KEY (symbol) REFERENCES stocks(symbol) ON DELETE CASCADE
);

-- Earnings table
CREATE TABLE earnings (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    quarter VARCHAR(10) NOT NULL,
    year INTEGER NOT NULL,
    report_date DATE NOT NULL,
    
    -- Earnings Data
    eps_actual DECIMAL(10,4),
    eps_estimate DECIMAL(10,4),
    eps_surprise DECIMAL(10,4),
    eps_surprise_percent DECIMAL(8,4),
    
    -- Revenue Data
    revenue_actual DECIMAL(20,2),
    revenue_estimate DECIMAL(20,2),
    revenue_surprise DECIMAL(20,2),
    
    -- Guidance
    eps_guidance_low DECIMAL(10,4),
    eps_guidance_high DECIMAL(10,4),
    revenue_guidance_low DECIMAL(20,2),
    revenue_guidance_high DECIMAL(20,2),
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    FOREIGN KEY (symbol) REFERENCES stocks(symbol) ON DELETE CASCADE,
    UNIQUE(symbol, quarter, year)
);

-- Analyst ratings table
CREATE TABLE analyst_ratings (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    firm VARCHAR(100) NOT NULL,
    analyst VARCHAR(100),
    rating VARCHAR(20) NOT NULL, -- Strong Buy, Buy, Hold, Sell, Strong Sell
    price_target DECIMAL(10,2),
    previous_rating VARCHAR(20),
    rating_date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    FOREIGN KEY (symbol) REFERENCES stocks(symbol) ON DELETE CASCADE
);

-- News articles table
CREATE TABLE news_articles (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    title VARCHAR(500) NOT NULL,
    summary TEXT,
    content TEXT,
    author VARCHAR(100),
    source VARCHAR(100),
    url VARCHAR(500),
    sentiment VARCHAR(20), -- Positive, Negative, Neutral
    published_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    FOREIGN KEY (symbol) REFERENCES stocks(symbol) ON DELETE CASCADE
);

-- Insider trading table
CREATE TABLE insider_trading (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    insider_name VARCHAR(100) NOT NULL,
    title VARCHAR(100),
    transaction_type VARCHAR(20) NOT NULL, -- Buy, Sell
    shares BIGINT NOT NULL,
    price DECIMAL(10,4) NOT NULL,
    value DECIMAL(20,2) NOT NULL,
    shares_owned BIGINT,
    transaction_date DATE NOT NULL,
    filing_date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    FOREIGN KEY (symbol) REFERENCES stocks(symbol) ON DELETE CASCADE
);

-- Technical indicators table (for storing calculated indicators)
CREATE TABLE technical_indicators (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    rsi DECIMAL(8,4),
    macd DECIMAL(10,6),
    macd_signal DECIMAL(10,6),
    macd_histogram DECIMAL(10,6),
    sma_20 DECIMAL(15,4),
    sma_50 DECIMAL(15,4),
    sma_200 DECIMAL(15,4),
    ema_12 DECIMAL(15,4),
    ema_26 DECIMAL(15,4),
    bollinger_upper DECIMAL(15,4),
    bollinger_lower DECIMAL(15,4),
    support DECIMAL(15,4),
    resistance DECIMAL(15,4),
    volume_20_day_avg BIGINT,
    beta DECIMAL(8,4),
    volatility DECIMAL(8,4),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    FOREIGN KEY (symbol) REFERENCES stocks(symbol) ON DELETE CASCADE
);

-- Create indexes for better performance
CREATE INDEX idx_stocks_symbol ON stocks(symbol);
CREATE INDEX idx_stock_prices_symbol_timestamp ON stock_prices(symbol, timestamp DESC);
CREATE INDEX idx_fundamentals_symbol_year ON fundamentals(symbol, year DESC);
CREATE INDEX idx_earnings_symbol_year ON earnings(symbol, year DESC, quarter);
CREATE INDEX idx_analyst_ratings_symbol_date ON analyst_ratings(symbol, rating_date DESC);
CREATE INDEX idx_news_symbol_published ON news_articles(symbol, published_at DESC);
CREATE INDEX idx_insider_symbol_date ON insider_trading(symbol, transaction_date DESC);
CREATE INDEX idx_technical_symbol_timestamp ON technical_indicators(symbol, timestamp DESC);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Add triggers for updated_at
CREATE TRIGGER update_stocks_updated_at BEFORE UPDATE ON stocks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_fundamentals_updated_at BEFORE UPDATE ON fundamentals
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
