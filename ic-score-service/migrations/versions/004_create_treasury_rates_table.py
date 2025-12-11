"""create treasury_rates table

Revision ID: 004
Revises: 003
Create Date: 2025-11-24

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '004'
down_revision = '003'
branch_labels = None
depends_on = None


def upgrade() -> None:
    """Create treasury_rates table for storing US Treasury rates from FRED API."""

    # Create treasury_rates table
    op.execute("""
        CREATE TABLE treasury_rates (
            date DATE PRIMARY KEY,
            rate_1m DECIMAL(8,4),
            rate_3m DECIMAL(8,4),
            rate_6m DECIMAL(8,4),
            rate_1y DECIMAL(8,4),
            rate_2y DECIMAL(8,4),
            rate_10y DECIMAL(8,4),
            created_at TIMESTAMP DEFAULT NOW()
        );
    """)

    # Create index for efficient querying
    op.execute("""
        CREATE INDEX idx_treasury_date ON treasury_rates(date DESC);
    """)


def downgrade() -> None:
    """Drop treasury_rates table."""

    op.execute("DROP TABLE IF EXISTS treasury_rates CASCADE;")
