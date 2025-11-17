"""add cusip column to companies table

Revision ID: 002
Revises: 001
Create Date: 2025-11-17

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '002'
down_revision = '001'
branch_labels = None
depends_on = None


def upgrade() -> None:
    """Add CUSIP column to companies table for 13F institutional holdings mapping."""

    # Add cusip column to companies table
    op.add_column('companies', sa.Column('cusip', sa.String(length=9), nullable=True))

    # Recreate stocks view to include cusip column
    op.execute("""
        DROP VIEW IF EXISTS stocks;
        CREATE VIEW stocks AS
        SELECT
            id, ticker, name, sector, industry, market_cap, country, exchange,
            currency, website, description, employees, founded_year, hq_location,
            logo_url, is_active, last_updated, created_at, cik, cusip
        FROM companies;
    """)

    # Create index on cusip for faster lookups
    op.create_index('ix_companies_cusip', 'companies', ['cusip'], unique=False)


def downgrade() -> None:
    """Remove CUSIP column from companies table."""

    # Drop index
    op.drop_index('ix_companies_cusip', table_name='companies')

    # Recreate view without cusip
    op.execute("""
        DROP VIEW IF EXISTS stocks;
        CREATE VIEW stocks AS
        SELECT
            id, ticker, name, sector, industry, market_cap, country, exchange,
            currency, website, description, employees, founded_year, hq_location,
            logo_url, is_active, last_updated, created_at, cik
        FROM companies;
    """)

    # Drop column
    op.drop_column('companies', 'cusip')
