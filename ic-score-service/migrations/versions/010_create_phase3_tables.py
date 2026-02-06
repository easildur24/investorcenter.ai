"""create ic_score_events table for Score Stability

Revision ID: 010
Revises: 009
Create Date: 2026-02-01

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '010'
down_revision = '009'
branch_labels = None
depends_on = None


def upgrade() -> None:
    """Create ic_score_events table and related structures for score stability."""

    # Create ic_score_events table
    op.execute("""
        CREATE TABLE ic_score_events (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            ticker VARCHAR(10) NOT NULL,
            event_type VARCHAR(50) NOT NULL,
            event_date DATE NOT NULL,
            description TEXT,
            impact_direction VARCHAR(10),  -- 'positive', 'negative', 'neutral'
            impact_magnitude NUMERIC(5,2),  -- Estimated score impact
            source VARCHAR(50),  -- Where the event was detected from
            metadata JSONB,  -- Additional event-specific data
            created_at TIMESTAMPTZ DEFAULT NOW()
        );
    """)

    # Create indexes
    op.execute("""
        CREATE INDEX idx_score_events_ticker_date
        ON ic_score_events(ticker, event_date DESC);
    """)

    op.execute("""
        CREATE INDEX idx_score_events_type
        ON ic_score_events(event_type, event_date DESC);
    """)

    # Create stock_peers table for peer comparison
    op.execute("""
        CREATE TABLE stock_peers (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            ticker VARCHAR(10) NOT NULL,
            peer_ticker VARCHAR(10) NOT NULL,
            similarity_score NUMERIC(5,4),  -- 0-1 similarity measure
            similarity_factors JSONB,  -- Breakdown of similarity components
            calculated_at DATE NOT NULL DEFAULT CURRENT_DATE,
            created_at TIMESTAMPTZ DEFAULT NOW(),

            UNIQUE (ticker, peer_ticker, calculated_at)
        );
    """)

    op.execute("""
        CREATE INDEX idx_stock_peers_ticker
        ON stock_peers(ticker, calculated_at DESC);
    """)

    # Create catalyst_events table
    op.execute("""
        CREATE TABLE catalyst_events (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            ticker VARCHAR(10) NOT NULL,
            event_type VARCHAR(50) NOT NULL,
            title VARCHAR(200) NOT NULL,
            description TEXT,
            event_date DATE,
            icon VARCHAR(10),  -- Emoji or icon identifier
            impact VARCHAR(20),  -- 'bullish', 'bearish', 'neutral'
            confidence NUMERIC(3,2),  -- 0-1 confidence in the catalyst
            days_until INTEGER,  -- Days until the event (negative = past)
            source VARCHAR(50),
            metadata JSONB,
            created_at TIMESTAMPTZ DEFAULT NOW(),
            expires_at TIMESTAMPTZ  -- When this catalyst is no longer relevant
        );
    """)

    op.execute("""
        CREATE INDEX idx_catalysts_ticker_date
        ON catalyst_events(ticker, event_date);
    """)

    op.execute("""
        CREATE INDEX idx_catalysts_type
        ON catalyst_events(event_type, event_date);
    """)

    # Create ic_score_changes table for tracking score changes
    op.execute("""
        CREATE TABLE ic_score_changes (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            ticker VARCHAR(10) NOT NULL,
            calculated_at DATE NOT NULL,

            previous_score NUMERIC(5,2),
            current_score NUMERIC(5,2),
            delta NUMERIC(5,2),

            -- Factor-level changes
            factor_changes JSONB,  -- [{"factor": "momentum", "delta": -15, "reason": "..."}]

            -- Trigger information
            trigger_events JSONB,  -- Events that triggered the change
            smoothing_applied BOOLEAN DEFAULT FALSE,

            created_at TIMESTAMPTZ DEFAULT NOW()
        );
    """)

    op.execute("""
        CREATE INDEX idx_score_changes_ticker
        ON ic_score_changes(ticker, calculated_at DESC);
    """)

    # Add previous_score column to ic_scores if not exists
    op.execute("""
        ALTER TABLE ic_scores
            ADD COLUMN IF NOT EXISTS previous_score NUMERIC(5,2);
    """)


def downgrade() -> None:
    """Drop Phase 3 tables."""

    op.execute("ALTER TABLE ic_scores DROP COLUMN IF EXISTS previous_score;")
    op.execute("DROP TABLE IF EXISTS ic_score_changes CASCADE;")
    op.execute("DROP TABLE IF EXISTS catalyst_events CASCADE;")
    op.execute("DROP TABLE IF EXISTS stock_peers CASCADE;")
    op.execute("DROP TABLE IF EXISTS ic_score_events CASCADE;")
