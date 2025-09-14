-- Add index on parsed_data column to improve query performance
CREATE INDEX IF NOT EXISTS idx_sec_filings_parsed_data 
ON sec_filings(filing_type, ticker_id) 
WHERE s3_key IS NOT NULL 
  AND (parsed_data IS NULL OR parsed_data = '{}')
  AND filing_type IN ('10-K', '10-Q');

-- Also add an index on s3_key for faster lookups
CREATE INDEX IF NOT EXISTS idx_sec_filings_s3_key 
ON sec_filings(s3_key) 
WHERE s3_key IS NOT NULL;