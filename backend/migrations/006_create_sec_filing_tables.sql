-- Migration to create SEC filing tables
-- Stores SEC filing metadata, content, and sync status

-- CIK mapping table (not needed - using tickers.cik instead)
-- Kept for reference only
CREATE TABLE IF NOT EXISTS cik_mapping (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) UNIQUE NOT NULL,
    cik VARCHAR(10) NOT NULL,
    company_name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cik_symbol ON cik_mapping(symbol);
CREATE INDEX IF NOT EXISTS idx_cik_number ON cik_mapping(cik);

-- SEC Filings metadata table
CREATE TABLE IF NOT EXISTS sec_filings (
    id SERIAL PRIMARY KEY,
    ticker_id INTEGER REFERENCES tickers(id),
    symbol VARCHAR(20) NOT NULL,
    cik VARCHAR(10) NOT NULL,
    filing_type VARCHAR(20) NOT NULL, -- '10-K', '10-Q', '8-K', etc.
    filing_date DATE NOT NULL,
    report_date DATE, -- Period end date
    accession_number VARCHAR(50) UNIQUE NOT NULL,
    file_number VARCHAR(50),
    form VARCHAR(20),
    size_bytes INTEGER,
    
    -- URLs for accessing the filing
    primary_document VARCHAR(255),
    primary_doc_description VARCHAR(255),
    filing_detail_url TEXT,
    interactive_data_url TEXT,
    
    -- Processing status
    is_processed BOOLEAN DEFAULT false,
    processing_started_at TIMESTAMP WITH TIME ZONE,
    processing_completed_at TIMESTAMP WITH TIME ZONE,
    processing_error TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_filings_symbol ON sec_filings(symbol);
CREATE INDEX IF NOT EXISTS idx_filings_cik ON sec_filings(cik);
CREATE INDEX IF NOT EXISTS idx_filings_type ON sec_filings(filing_type);
CREATE INDEX IF NOT EXISTS idx_filings_date ON sec_filings(filing_date DESC);
CREATE INDEX IF NOT EXISTS idx_filings_report_date ON sec_filings(report_date DESC);
CREATE INDEX IF NOT EXISTS idx_filings_processed ON sec_filings(is_processed);

-- Filing content and extracted data
CREATE TABLE IF NOT EXISTS filing_content (
    id SERIAL PRIMARY KEY,
    filing_id INTEGER UNIQUE REFERENCES sec_filings(id) ON DELETE CASCADE,
    
    -- Key sections from 10-K/10-Q
    business_description TEXT,
    risk_factors TEXT,
    md_and_a TEXT, -- Management Discussion & Analysis
    financial_statements TEXT,
    notes_to_financials TEXT,
    
    -- Extracted financial metrics (from XBRL or parsed)
    revenue DECIMAL(20,2),
    revenue_yoy_change DECIMAL(10,4), -- Year-over-year change percentage
    net_income DECIMAL(20,2),
    net_income_yoy_change DECIMAL(10,4),
    earnings_per_share DECIMAL(10,4),
    eps_yoy_change DECIMAL(10,4),
    
    total_assets DECIMAL(20,2),
    total_liabilities DECIMAL(20,2),
    shareholders_equity DECIMAL(20,2),
    cash_and_equivalents DECIMAL(20,2),
    
    operating_cash_flow DECIMAL(20,2),
    free_cash_flow DECIMAL(20,2),
    
    -- Ratios
    gross_margin DECIMAL(10,4),
    operating_margin DECIMAL(10,4),
    net_margin DECIMAL(10,4),
    return_on_equity DECIMAL(10,4),
    return_on_assets DECIMAL(10,4),
    debt_to_equity DECIMAL(10,4),
    current_ratio DECIMAL(10,4),
    
    -- Full text for search
    full_text TEXT, -- Complete filing text for full-text search
    
    -- Metadata
    word_count INTEGER,
    extracted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_content_filing ON filing_content(filing_id);
CREATE INDEX IF NOT EXISTS idx_content_revenue ON filing_content(revenue);
CREATE INDEX IF NOT EXISTS idx_content_net_income ON filing_content(net_income);

-- XBRL facts storage (structured data from company facts API)
CREATE TABLE IF NOT EXISTS xbrl_facts (
    id SERIAL PRIMARY KEY,
    filing_id INTEGER REFERENCES sec_filings(id) ON DELETE CASCADE,
    cik VARCHAR(10) NOT NULL,
    
    -- Fact identification
    taxonomy VARCHAR(50), -- 'us-gaap', 'dei', etc.
    fact_name VARCHAR(255), -- 'Revenues', 'NetIncomeLoss', etc.
    
    -- Fact details
    value DECIMAL(30,4),
    unit VARCHAR(50), -- 'USD', 'shares', 'USD/shares', etc.
    
    -- Period information
    fiscal_year INTEGER,
    fiscal_period VARCHAR(10), -- 'FY', 'Q1', 'Q2', 'Q3', 'Q4'
    start_date DATE,
    end_date DATE,
    instant_date DATE, -- For point-in-time facts
    
    -- Additional metadata
    segment JSON, -- Segment information if applicable
    decimals INTEGER,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(filing_id, taxonomy, fact_name, fiscal_period)
);

CREATE INDEX IF NOT EXISTS idx_xbrl_filing ON xbrl_facts(filing_id);
CREATE INDEX IF NOT EXISTS idx_xbrl_cik ON xbrl_facts(cik);
CREATE INDEX IF NOT EXISTS idx_xbrl_fact ON xbrl_facts(fact_name);
CREATE INDEX IF NOT EXISTS idx_xbrl_period ON xbrl_facts(fiscal_year, fiscal_period);

-- Track sync status for each company
CREATE TABLE IF NOT EXISTS sec_sync_status (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) UNIQUE NOT NULL,
    cik VARCHAR(10),
    
    -- Last successful sync timestamps
    last_10k_date DATE,
    last_10q_date DATE,
    last_8k_date DATE,
    last_filing_check TIMESTAMP WITH TIME ZONE,
    last_successful_sync TIMESTAMP WITH TIME ZONE,
    
    -- Sync statistics
    total_filings_count INTEGER DEFAULT 0,
    processed_filings_count INTEGER DEFAULT 0,
    failed_filings_count INTEGER DEFAULT 0,
    
    -- Sync configuration
    sync_enabled BOOLEAN DEFAULT true,
    sync_priority INTEGER DEFAULT 0, -- Higher = more important
    
    -- Error tracking
    last_error TEXT,
    error_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sync_symbol ON sec_sync_status(symbol);
CREATE INDEX IF NOT EXISTS idx_sync_cik ON sec_sync_status(cik);
CREATE INDEX IF NOT EXISTS idx_sync_enabled ON sec_sync_status(sync_enabled);
CREATE INDEX IF NOT EXISTS idx_sync_priority ON sec_sync_status(sync_priority DESC);

-- Create full-text search index for filing content
CREATE INDEX IF NOT EXISTS idx_filing_content_fulltext 
ON filing_content 
USING gin(to_tsvector('english', 
    COALESCE(business_description, '') || ' ' || 
    COALESCE(risk_factors, '') || ' ' || 
    COALESCE(md_and_a, '')
));

-- Add comments for documentation
COMMENT ON TABLE cik_mapping IS 'Maps stock symbols to SEC CIK numbers';
COMMENT ON TABLE sec_filings IS 'SEC filing metadata and processing status';
COMMENT ON TABLE filing_content IS 'Extracted content and financial metrics from filings';
COMMENT ON TABLE xbrl_facts IS 'Structured XBRL data from company facts API';
COMMENT ON TABLE sec_sync_status IS 'Tracks synchronization status for each company';

COMMENT ON COLUMN sec_filings.accession_number IS 'Unique SEC accession number for the filing';
COMMENT ON COLUMN filing_content.md_and_a IS 'Management Discussion and Analysis section';
COMMENT ON COLUMN xbrl_facts.taxonomy IS 'XBRL taxonomy (us-gaap, dei, etc.)';
COMMENT ON COLUMN sec_sync_status.sync_priority IS 'Higher priority companies are synced more frequently';