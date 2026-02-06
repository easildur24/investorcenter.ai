"""create sector_percentiles table

Revision ID: 006
Revises: 005
Create Date: 2026-02-01

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '006'
down_revision = '005'
branch_labels = None
depends_on = None


def upgrade() -> None:
    """Create sector_percentiles table for sector-relative IC Score calculations."""

    # Create sector_percentiles table
    op.execute("""
        CREATE TABLE sector_percentiles (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            sector VARCHAR(50) NOT NULL,
            metric_name VARCHAR(50) NOT NULL,
            calculated_at DATE NOT NULL DEFAULT CURRENT_DATE,

            -- Distribution statistics
            min_value NUMERIC(20,4),
            p10_value NUMERIC(20,4),
            p25_value NUMERIC(20,4),
            p50_value NUMERIC(20,4),  -- median
            p75_value NUMERIC(20,4),
            p90_value NUMERIC(20,4),
            max_value NUMERIC(20,4),
            mean_value NUMERIC(20,4),
            std_dev NUMERIC(20,4),
            sample_count INTEGER,

            created_at TIMESTAMPTZ DEFAULT NOW(),

            UNIQUE (sector, metric_name, calculated_at)
        );
    """)

    # Create index for efficient lookups
    op.execute("""
        CREATE INDEX idx_sector_percentiles_lookup
        ON sector_percentiles(sector, metric_name, calculated_at DESC);
    """)

    # Create materialized view for quick sector lookups (latest values only)
    op.execute("""
        CREATE MATERIALIZED VIEW mv_latest_sector_percentiles AS
        SELECT DISTINCT ON (sector, metric_name)
            id,
            sector,
            metric_name,
            calculated_at,
            min_value,
            p10_value,
            p25_value,
            p50_value,
            p75_value,
            p90_value,
            max_value,
            mean_value,
            std_dev,
            sample_count
        FROM sector_percentiles
        ORDER BY sector, metric_name, calculated_at DESC;
    """)

    # Create unique index on materialized view for concurrent refresh
    op.execute("""
        CREATE UNIQUE INDEX idx_mv_sector_percentiles
        ON mv_latest_sector_percentiles(sector, metric_name);
    """)


def downgrade() -> None:
    """Drop sector_percentiles table and materialized view."""

    op.execute("DROP MATERIALIZED VIEW IF EXISTS mv_latest_sector_percentiles CASCADE;")
    op.execute("DROP TABLE IF EXISTS sector_percentiles CASCADE;")
