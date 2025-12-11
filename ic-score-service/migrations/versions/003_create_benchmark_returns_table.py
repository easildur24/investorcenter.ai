"""create benchmark_returns table

Revision ID: 003
Revises: 002
Create Date: 2025-11-24

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '003'
down_revision = '002'
branch_labels = None
depends_on = None


def upgrade() -> None:
    """Create benchmark_returns table for storing S&P 500 and other benchmark indices."""

    # Create benchmark_returns table
    op.execute("""
        CREATE TABLE benchmark_returns (
            time TIMESTAMPTZ NOT NULL,
            symbol VARCHAR(20) NOT NULL,
            close DECIMAL(12,4) NOT NULL,
            total_return DECIMAL(12,4),
            daily_return DECIMAL(10,6),
            volume BIGINT,
            PRIMARY KEY (time, symbol)
        );
    """)

    # Convert to TimescaleDB hypertable
    op.execute("""
        SELECT create_hypertable('benchmark_returns', 'time');
    """)

    # Create index for efficient querying
    op.execute("""
        CREATE INDEX idx_benchmark_symbol_time ON benchmark_returns(symbol, time DESC);
    """)


def downgrade() -> None:
    """Drop benchmark_returns table."""

    # Drop table (hypertable will be automatically removed)
    op.execute("DROP TABLE IF EXISTS benchmark_returns CASCADE;")
