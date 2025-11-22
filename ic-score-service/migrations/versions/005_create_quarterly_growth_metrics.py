"""create quarterly_growth_metrics table for QoQ growth tracking

Revision ID: 005
Revises: 004
Create Date: 2025-11-22

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '005'
down_revision = '004'
branch_labels = None
depends_on = None


def upgrade() -> None:
    """Create quarterly_growth_metrics table for tracking quarter-over-quarter growth.

    This table stores growth metrics comparing current quarter to previous quarter for:
    - Revenue growth
    - EPS growth
    - EBITDA growth
    - Net income growth
    """

    op.create_table(
        'quarterly_growth_metrics',
        sa.Column('id', sa.BigInteger, primary_key=True, autoincrement=True),
        sa.Column('ticker', sa.String(10), nullable=False),
        sa.Column('financial_id', sa.BigInteger, sa.ForeignKey('financials.id'), nullable=False),
        sa.Column('period_end_date', sa.Date, nullable=False),

        # QoQ Revenue Growth
        sa.Column('qoq_revenue_growth', sa.Numeric(precision=10, scale=2), nullable=True),
        sa.Column('revenue_current', sa.BigInteger, nullable=True),
        sa.Column('revenue_previous', sa.BigInteger, nullable=True),

        # QoQ EPS Growth
        sa.Column('qoq_eps_growth', sa.Numeric(precision=10, scale=2), nullable=True),
        sa.Column('eps_current', sa.Numeric(precision=10, scale=4), nullable=True),
        sa.Column('eps_previous', sa.Numeric(precision=10, scale=4), nullable=True),

        # QoQ EBITDA Growth
        sa.Column('qoq_ebitda_growth', sa.Numeric(precision=10, scale=2), nullable=True),
        sa.Column('ebitda_current', sa.BigInteger, nullable=True),
        sa.Column('ebitda_previous', sa.BigInteger, nullable=True),

        # QoQ Net Income Growth
        sa.Column('qoq_net_income_growth', sa.Numeric(precision=10, scale=2), nullable=True),
        sa.Column('net_income_current', sa.BigInteger, nullable=True),
        sa.Column('net_income_previous', sa.BigInteger, nullable=True),

        # Metadata
        sa.Column('calculation_date', sa.Date, nullable=False, server_default=sa.text('CURRENT_DATE')),
        sa.Column('created_at', sa.TIMESTAMP, nullable=False, server_default=sa.text('NOW()')),

        # Unique constraint
        sa.UniqueConstraint('ticker', 'period_end_date', name='uq_qgm_ticker_period'),
    )

    # Create indexes for performance
    op.create_index('ix_qgm_ticker', 'quarterly_growth_metrics', ['ticker'])
    op.create_index('ix_qgm_period', 'quarterly_growth_metrics', ['period_end_date'])
    op.create_index('ix_qgm_revenue_growth', 'quarterly_growth_metrics', ['qoq_revenue_growth'])
    op.create_index('ix_qgm_eps_growth', 'quarterly_growth_metrics', ['qoq_eps_growth'])


def downgrade() -> None:
    """Drop quarterly_growth_metrics table."""

    # Drop indexes
    op.drop_index('ix_qgm_eps_growth', table_name='quarterly_growth_metrics')
    op.drop_index('ix_qgm_revenue_growth', table_name='quarterly_growth_metrics')
    op.drop_index('ix_qgm_period', table_name='quarterly_growth_metrics')
    op.drop_index('ix_qgm_ticker', table_name='quarterly_growth_metrics')

    # Drop table
    op.drop_table('quarterly_growth_metrics')
