-- Create ingestion_log table for tracking raw data uploads to S3
-- This table stores pointers to S3 objects, not the raw data itself

CREATE TABLE IF NOT EXISTS ingestion_log (
    id           BIGSERIAL PRIMARY KEY,
    source       VARCHAR(50) NOT NULL,       -- ycharts, seekingalpha, sec_edgar, etc.
    ticker       VARCHAR(20),                -- optional ticker symbol
    data_type    VARCHAR(100) NOT NULL,      -- financials, valuation, ratings, dividends, etc.
    source_url   TEXT,                       -- original URL that was scraped
    s3_key       TEXT NOT NULL UNIQUE,       -- S3 object key (raw/{source}/{ticker}/{type}/{date}/{ts}.json)
    s3_bucket    VARCHAR(100) NOT NULL,      -- S3 bucket name
    file_size    BIGINT,                     -- size in bytes of the S3 object
    collected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- when the source data was scraped
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()  -- when the record was created
);

-- Index for querying by source + ticker (most common query pattern)
CREATE INDEX idx_il_source_ticker ON ingestion_log(source, ticker, collected_at DESC)
    WHERE ticker IS NOT NULL;

-- Index for querying by data type
CREATE INDEX idx_il_data_type ON ingestion_log(data_type, collected_at DESC);

-- Index for time-range queries and cleanup
CREATE INDEX idx_il_created_at ON ingestion_log(created_at);

-- Index for source-only queries
CREATE INDEX idx_il_source ON ingestion_log(source, created_at DESC);
