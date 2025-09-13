-- Add column for storing parsed filing data
ALTER TABLE sec_filings
ADD COLUMN IF NOT EXISTS parsed_data JSONB,
ADD COLUMN IF NOT EXISTS parsed_at TIMESTAMP;