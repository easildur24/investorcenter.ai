"""create eps_estimates table for Earnings Revisions factor

Revision ID: 008
Revises: 007
Create Date: 2026-02-01

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '008'
down_revision = '007'
branch_labels = None
depends_on = None


def upgrade() -> None:
    """Create eps_estimates table for Earnings Revisions factor."""

    # Create eps_estimates table
    op.execute("""
        CREATE TABLE eps_estimates (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            ticker VARCHAR(10) NOT NULL,
            fiscal_year INTEGER NOT NULL,
            fiscal_quarter INTEGER,  -- NULL for annual estimates

            -- Estimate data
            consensus_eps NUMERIC(10,4),
            num_analysts INTEGER,
            high_estimate NUMERIC(10,4),
            low_estimate NUMERIC(10,4),

            -- Historical tracking for revision calculations
            estimate_30d_ago NUMERIC(10,4),
            estimate_60d_ago NUMERIC(10,4),
            estimate_90d_ago NUMERIC(10,4),

            -- Revision counts
            upgrades_30d INTEGER DEFAULT 0,
            downgrades_30d INTEGER DEFAULT 0,
            upgrades_60d INTEGER DEFAULT 0,
            downgrades_60d INTEGER DEFAULT 0,
            upgrades_90d INTEGER DEFAULT 0,
            downgrades_90d INTEGER DEFAULT 0,

            -- Calculated revision metrics
            revision_pct_30d NUMERIC(10,4),
            revision_pct_60d NUMERIC(10,4),
            revision_pct_90d NUMERIC(10,4),

            fetched_at TIMESTAMPTZ DEFAULT NOW(),
            created_at TIMESTAMPTZ DEFAULT NOW(),
            updated_at TIMESTAMPTZ DEFAULT NOW(),

            UNIQUE (ticker, fiscal_year, fiscal_quarter)
        );
    """)

    # Create indexes for efficient lookups
    op.execute("""
        CREATE INDEX idx_eps_estimates_ticker
        ON eps_estimates(ticker);
    """)

    op.execute("""
        CREATE INDEX idx_eps_estimates_ticker_year
        ON eps_estimates(ticker, fiscal_year DESC);
    """)

    # Create eps_estimate_history table to track daily snapshots
    op.execute("""
        CREATE TABLE eps_estimate_history (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            ticker VARCHAR(10) NOT NULL,
            fiscal_year INTEGER NOT NULL,
            fiscal_quarter INTEGER,
            snapshot_date DATE NOT NULL DEFAULT CURRENT_DATE,

            consensus_eps NUMERIC(10,4),
            num_analysts INTEGER,
            high_estimate NUMERIC(10,4),
            low_estimate NUMERIC(10,4),

            created_at TIMESTAMPTZ DEFAULT NOW(),

            UNIQUE (ticker, fiscal_year, fiscal_quarter, snapshot_date)
        );
    """)

    op.execute("""
        CREATE INDEX idx_eps_estimate_history_ticker_date
        ON eps_estimate_history(ticker, snapshot_date DESC);
    """)


def downgrade() -> None:
    """Drop eps_estimates and related tables."""

    op.execute("DROP TABLE IF EXISTS eps_estimate_history CASCADE;")
    op.execute("DROP TABLE IF EXISTS eps_estimates CASCADE;")
