"""add TTM performance ratios to ttm_financials table

Revision ID: 003
Revises: 002
Create Date: 2025-11-22

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '003'
down_revision = '002'
branch_labels = None
depends_on = None


def upgrade() -> None:
    """Add TTM performance ratio columns to ttm_financials table.

    This adds:
    - ttm_roa: Return on Assets (%)
    - ttm_roe: Return on Equity (%)
    - ttm_roic: Return on Invested Capital (%)
    - ttm_gross_margin: Gross Margin (%)
    - ttm_operating_margin: Operating Margin (%)
    - ttm_net_margin: Net Margin (%)
    """

    # Add new columns to ttm_financials table
    op.add_column('ttm_financials', sa.Column('ttm_roa', sa.Numeric(precision=10, scale=4), nullable=True))
    op.add_column('ttm_financials', sa.Column('ttm_roe', sa.Numeric(precision=10, scale=4), nullable=True))
    op.add_column('ttm_financials', sa.Column('ttm_roic', sa.Numeric(precision=10, scale=4), nullable=True))
    op.add_column('ttm_financials', sa.Column('ttm_gross_margin', sa.Numeric(precision=10, scale=4), nullable=True))
    op.add_column('ttm_financials', sa.Column('ttm_operating_margin', sa.Numeric(precision=10, scale=4), nullable=True))
    op.add_column('ttm_financials', sa.Column('ttm_net_margin', sa.Numeric(precision=10, scale=4), nullable=True))

    # Create indexes for performance queries
    op.create_index('ix_ttm_financials_ttm_roe', 'ttm_financials', ['ttm_roe'], unique=False)
    op.create_index('ix_ttm_financials_ttm_roa', 'ttm_financials', ['ttm_roa'], unique=False)


def downgrade() -> None:
    """Remove TTM performance ratio columns from ttm_financials table."""

    # Drop indexes
    op.drop_index('ix_ttm_financials_ttm_roa', table_name='ttm_financials')
    op.drop_index('ix_ttm_financials_ttm_roe', table_name='ttm_financials')

    # Drop columns
    op.drop_column('ttm_financials', 'ttm_net_margin')
    op.drop_column('ttm_financials', 'ttm_operating_margin')
    op.drop_column('ttm_financials', 'ttm_gross_margin')
    op.drop_column('ttm_financials', 'ttm_roic')
    op.drop_column('ttm_financials', 'ttm_roe')
    op.drop_column('ttm_financials', 'ttm_roa')
