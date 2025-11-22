"""add EBITDA columns to financials, ttm_financials, and valuation_ratios tables

Revision ID: 004
Revises: 003
Create Date: 2025-11-22

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '004'
down_revision = '003'
branch_labels = None
depends_on = None


def upgrade() -> None:
    """Add EBITDA-related columns to multiple tables.

    This adds:
    - financials: depreciation_and_amortization, ebitda
    - ttm_financials: ttm_ebitda
    - valuation_ratios: ttm_ev_ebitda_ratio, enterprise_value
    """

    # Add columns to financials table
    op.add_column('financials', sa.Column('depreciation_and_amortization', sa.BigInteger, nullable=True))
    op.add_column('financials', sa.Column('ebitda', sa.BigInteger, nullable=True))

    # Add column to ttm_financials table
    op.add_column('ttm_financials', sa.Column('ttm_ebitda', sa.BigInteger, nullable=True))

    # Add EV/EBITDA columns to financials table (valuation metrics stored here)
    op.add_column('financials', sa.Column('enterprise_value', sa.BigInteger, nullable=True))
    op.add_column('financials', sa.Column('ttm_ev_ebitda_ratio', sa.Numeric(precision=10, scale=2), nullable=True))

    # Create indexes for performance queries
    op.create_index('ix_financials_ebitda', 'financials', ['ebitda'], unique=False)
    op.create_index('ix_ttm_financials_ttm_ebitda', 'ttm_financials', ['ttm_ebitda'], unique=False)
    op.create_index('ix_financials_ev_ebitda', 'financials', ['ttm_ev_ebitda_ratio'], unique=False)


def downgrade() -> None:
    """Remove EBITDA-related columns from tables."""

    # Drop indexes
    op.drop_index('ix_financials_ev_ebitda', table_name='financials')
    op.drop_index('ix_ttm_financials_ttm_ebitda', table_name='ttm_financials')
    op.drop_index('ix_financials_ebitda', table_name='financials')

    # Drop EV/EBITDA columns from financials
    op.drop_column('financials', 'ttm_ev_ebitda_ratio')
    op.drop_column('financials', 'enterprise_value')

    # Drop column from ttm_financials
    op.drop_column('ttm_financials', 'ttm_ebitda')

    # Drop EBITDA columns from financials
    op.drop_column('financials', 'ebitda')
    op.drop_column('financials', 'depreciation_and_amortization')
