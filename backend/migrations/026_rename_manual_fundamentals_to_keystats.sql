-- Rename manual_fundamentals table to keystats
-- and rename the 'data' column to 'key_stats'

ALTER TABLE manual_fundamentals RENAME TO keystats;
ALTER TABLE keystats RENAME COLUMN data TO key_stats;

-- Rename indexes
ALTER INDEX IF EXISTS idx_manual_fundamentals_ticker RENAME TO idx_keystats_ticker;
ALTER INDEX IF EXISTS idx_manual_fundamentals_data RENAME TO idx_keystats_key_stats;

-- Recreate trigger with new name
DROP TRIGGER IF EXISTS trigger_update_manual_fundamentals_updated_at ON keystats;
DROP FUNCTION IF EXISTS update_manual_fundamentals_updated_at();

CREATE OR REPLACE FUNCTION update_keystats_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_keystats_updated_at
    BEFORE UPDATE ON keystats
    FOR EACH ROW
    EXECUTE FUNCTION update_keystats_updated_at();
