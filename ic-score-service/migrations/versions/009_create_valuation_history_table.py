"""create valuation_history table for Historical Valuation factor

Revision ID: 009
Revises: 008
Create Date: 2026-02-01

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '009'
down_revision = '008'
branch_labels = None
depends_on = None


def upgrade() -> None:
    """Create valuation_history table for Historical Valuation factor."""

    # Create valuation_history table for monthly snapshots
    op.execute("""
        CREATE TABLE valuation_history (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            ticker VARCHAR(10) NOT NULL,
            snapshot_date DATE NOT NULL,

            -- Valuation metrics
            pe_ratio NUMERIC(10,2),
            ps_ratio NUMERIC(10,2),
            pb_ratio NUMERIC(10,2),
            ev_ebitda NUMERIC(10,2),
            peg_ratio NUMERIC(10,2),

            -- Price context
            stock_price NUMERIC(10,2),
            market_cap NUMERIC(20,2),

            -- Earnings data at snapshot time
            eps_ttm NUMERIC(10,4),
            revenue_ttm NUMERIC(20,2),

            created_at TIMESTAMPTZ DEFAULT NOW(),

            UNIQUE (ticker, snapshot_date)
        );
    """)

    # Create index for efficient lookups
    op.execute("""
        CREATE INDEX idx_valuation_history_ticker_date
        ON valuation_history(ticker, snapshot_date DESC);
    """)

    op.execute("""
        CREATE INDEX idx_valuation_history_date
        ON valuation_history(snapshot_date);
    """)

    # Create dividend_history table for Dividend Quality factor
    op.execute("""
        CREATE TABLE dividend_history (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            ticker VARCHAR(10) NOT NULL,
            fiscal_year INTEGER NOT NULL,

            -- Dividend metrics
            annual_dividend NUMERIC(10,4),
            dividend_yield NUMERIC(8,4),
            payout_ratio NUMERIC(8,4),
            dividend_growth_yoy NUMERIC(8,4),

            -- Ex-dividend dates (for catalyst tracking)
            ex_dividend_date DATE,
            payment_date DATE,

            -- Dividend streak tracking
            consecutive_years_paid INTEGER DEFAULT 0,
            consecutive_years_increased INTEGER DEFAULT 0,

            created_at TIMESTAMPTZ DEFAULT NOW(),
            updated_at TIMESTAMPTZ DEFAULT NOW(),

            UNIQUE (ticker, fiscal_year)
        );
    """)

    op.execute("""
        CREATE INDEX idx_dividend_history_ticker
        ON dividend_history(ticker, fiscal_year DESC);
    """)

    # Add dividend quality fields to ic_scores
    op.execute("""
        ALTER TABLE ic_scores
            ADD COLUMN IF NOT EXISTS earnings_revisions_score NUMERIC(5,2),
            ADD COLUMN IF NOT EXISTS historical_value_score NUMERIC(5,2),
            ADD COLUMN IF NOT EXISTS dividend_quality_score NUMERIC(5,2);
    """)


def downgrade() -> None:
    """Drop valuation_history and dividend_history tables."""

    op.execute("""
        ALTER TABLE ic_scores
            DROP COLUMN IF EXISTS earnings_revisions_score,
            DROP COLUMN IF EXISTS historical_value_score,
            DROP COLUMN IF EXISTS dividend_quality_score;
    """)
    op.execute("DROP TABLE IF EXISTS dividend_history CASCADE;")
    op.execute("DROP TABLE IF EXISTS valuation_history CASCADE;")
