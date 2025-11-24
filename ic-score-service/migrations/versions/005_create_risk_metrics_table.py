"""create risk_metrics table

Revision ID: 005
Revises: 004
Create Date: 2025-11-24

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '005'
down_revision = '004'
branch_labels = None
depends_on = None


def upgrade() -> None:
    """Create risk_metrics table for storing calculated risk metrics (Alpha, Beta, Sharpe, etc.)."""

    # Create risk_metrics table
    op.execute("""
        CREATE TABLE risk_metrics (
            time TIMESTAMPTZ NOT NULL,
            ticker VARCHAR(10) NOT NULL,
            period VARCHAR(10) NOT NULL,

            alpha DECIMAL(10,4),
            beta DECIMAL(10,4),
            sharpe_ratio DECIMAL(10,4),
            sortino_ratio DECIMAL(10,4),
            std_dev DECIMAL(10,4),
            max_drawdown DECIMAL(10,4),
            var_5 DECIMAL(10,4),

            annualized_return DECIMAL(10,4),
            downside_deviation DECIMAL(10,4),

            data_points INT,
            calculation_date TIMESTAMP DEFAULT NOW(),

            PRIMARY KEY (time, ticker, period)
        );
    """)

    # Convert to TimescaleDB hypertable
    op.execute("""
        SELECT create_hypertable('risk_metrics', 'time');
    """)

    # Create indexes for efficient querying
    op.execute("""
        CREATE INDEX idx_risk_metrics_ticker ON risk_metrics(ticker, time DESC);
    """)

    op.execute("""
        CREATE INDEX idx_risk_metrics_period ON risk_metrics(period);
    """)


def downgrade() -> None:
    """Drop risk_metrics table."""

    # Drop table (hypertable will be automatically removed)
    op.execute("DROP TABLE IF EXISTS risk_metrics CASCADE;")
