"""create lifecycle_classifications table

Revision ID: 007
Revises: 006
Create Date: 2026-02-01

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '007'
down_revision = '006'
branch_labels = None
depends_on = None


def upgrade() -> None:
    """Create lifecycle_classifications table for company lifecycle stage tracking."""

    # Create lifecycle_classifications table
    op.execute("""
        CREATE TABLE lifecycle_classifications (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            ticker VARCHAR(10) NOT NULL,
            classified_at DATE NOT NULL DEFAULT CURRENT_DATE,

            -- Classification result
            lifecycle_stage VARCHAR(20) NOT NULL,  -- hypergrowth, growth, mature, value, turnaround

            -- Input metrics used for classification
            revenue_growth_yoy NUMERIC(10,4),
            net_margin NUMERIC(10,4),
            pe_ratio NUMERIC(10,4),
            market_cap BIGINT,

            -- Adjusted weights applied
            weights_applied JSONB,

            created_at TIMESTAMPTZ DEFAULT NOW(),

            UNIQUE (ticker, classified_at)
        );
    """)

    # Create index for efficient lookups
    op.execute("""
        CREATE INDEX idx_lifecycle_ticker_date
        ON lifecycle_classifications(ticker, classified_at DESC);
    """)

    # Add new columns to ic_scores table for v2.1 features
    op.execute("""
        ALTER TABLE ic_scores
            ADD COLUMN IF NOT EXISTS lifecycle_stage VARCHAR(20),
            ADD COLUMN IF NOT EXISTS raw_score NUMERIC(5,2),
            ADD COLUMN IF NOT EXISTS smoothing_applied BOOLEAN DEFAULT FALSE,
            ADD COLUMN IF NOT EXISTS weights_used JSONB,
            ADD COLUMN IF NOT EXISTS sector_rank INTEGER,
            ADD COLUMN IF NOT EXISTS sector_total INTEGER;
    """)


def downgrade() -> None:
    """Drop lifecycle_classifications table and remove ic_scores columns."""

    op.execute("""
        ALTER TABLE ic_scores
            DROP COLUMN IF EXISTS lifecycle_stage,
            DROP COLUMN IF EXISTS raw_score,
            DROP COLUMN IF EXISTS smoothing_applied,
            DROP COLUMN IF EXISTS weights_used,
            DROP COLUMN IF EXISTS sector_rank,
            DROP COLUMN IF EXISTS sector_total;
    """)

    op.execute("DROP TABLE IF EXISTS lifecycle_classifications CASCADE;")
