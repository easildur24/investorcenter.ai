"""Initial schema for IC Score Service

Revision ID: 001
Revises:
Create Date: 2025-11-12 00:00:00.000000

"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import postgresql

# revision identifiers, used by Alembic.
revision = '001'
down_revision = None
branch_labels = None
depends_on = None


def upgrade() -> None:
    """Create all tables for IC Score Service."""

    # Enable extensions
    op.execute('CREATE EXTENSION IF NOT EXISTS "uuid-ossp"')
    op.execute('CREATE EXTENSION IF NOT EXISTS "timescaledb"')

    # Create users table
    op.create_table(
        'users',
        sa.Column('id', postgresql.UUID(as_uuid=True), server_default=sa.text('gen_random_uuid()'), primary_key=True),
        sa.Column('email', sa.String(255), unique=True, nullable=False),
        sa.Column('password_hash', sa.String(255), nullable=False),
        sa.Column('first_name', sa.String(100)),
        sa.Column('last_name', sa.String(100)),
        sa.Column('subscription_tier', sa.String(50), server_default='free'),
        sa.Column('stripe_customer_id', sa.String(255)),
        sa.Column('created_at', sa.TIMESTAMP, server_default=sa.text('NOW()')),
        sa.Column('updated_at', sa.TIMESTAMP, server_default=sa.text('NOW()')),
        sa.Column('last_login_at', sa.TIMESTAMP),
        sa.Column('is_active', sa.Boolean, server_default='true'),
        sa.Column('email_verified', sa.Boolean, server_default='false'),
        sa.Column('preferences', postgresql.JSONB, server_default=sa.text("'{}'::jsonb"))
    )
    op.create_index('idx_users_email', 'users', ['email'])
    op.create_index('idx_users_subscription', 'users', ['subscription_tier'])

    # Create companies table
    op.create_table(
        'companies',
        sa.Column('id', sa.Integer, primary_key=True, autoincrement=True),
        sa.Column('ticker', sa.String(10), unique=True, nullable=False),
        sa.Column('name', sa.String(255), nullable=False),
        sa.Column('sector', sa.String(100)),
        sa.Column('industry', sa.String(100)),
        sa.Column('market_cap', sa.BigInteger),
        sa.Column('country', sa.String(50)),
        sa.Column('exchange', sa.String(50)),
        sa.Column('currency', sa.String(10)),
        sa.Column('website', sa.String(255)),
        sa.Column('description', sa.Text),
        sa.Column('employees', sa.Integer),
        sa.Column('founded_year', sa.Integer),
        sa.Column('hq_location', sa.String(255)),
        sa.Column('logo_url', sa.String(255)),
        sa.Column('is_active', sa.Boolean, server_default='true'),
        sa.Column('last_updated', sa.TIMESTAMP, server_default=sa.text('NOW()')),
        sa.Column('created_at', sa.TIMESTAMP, server_default=sa.text('NOW()'))
    )
    op.create_index('idx_companies_ticker', 'companies', ['ticker'])
    op.create_index('idx_companies_sector', 'companies', ['sector'])
    op.create_index('idx_companies_market_cap', 'companies', ['market_cap'])
    op.execute('CREATE INDEX idx_companies_active ON companies(is_active) WHERE is_active = true')

    # Create ic_scores table
    op.create_table(
        'ic_scores',
        sa.Column('id', sa.BigInteger, primary_key=True, autoincrement=True),
        sa.Column('ticker', sa.String(10), nullable=False),
        sa.Column('date', sa.Date, nullable=False),
        sa.Column('overall_score', sa.DECIMAL(5, 2), nullable=False),
        sa.Column('value_score', sa.DECIMAL(5, 2)),
        sa.Column('growth_score', sa.DECIMAL(5, 2)),
        sa.Column('profitability_score', sa.DECIMAL(5, 2)),
        sa.Column('financial_health_score', sa.DECIMAL(5, 2)),
        sa.Column('momentum_score', sa.DECIMAL(5, 2)),
        sa.Column('analyst_consensus_score', sa.DECIMAL(5, 2)),
        sa.Column('insider_activity_score', sa.DECIMAL(5, 2)),
        sa.Column('institutional_score', sa.DECIMAL(5, 2)),
        sa.Column('news_sentiment_score', sa.DECIMAL(5, 2)),
        sa.Column('technical_score', sa.DECIMAL(5, 2)),
        sa.Column('rating', sa.String(20)),
        sa.Column('sector_percentile', sa.DECIMAL(5, 2)),
        sa.Column('confidence_level', sa.String(20)),
        sa.Column('data_completeness', sa.DECIMAL(5, 2)),
        sa.Column('calculation_metadata', postgresql.JSONB),
        sa.Column('created_at', sa.TIMESTAMP, server_default=sa.text('NOW()')),
        sa.UniqueConstraint('ticker', 'date')
    )
    op.create_index('idx_ic_scores_ticker', 'ic_scores', ['ticker'])
    op.create_index('idx_ic_scores_date', 'ic_scores', ['date'])
    op.create_index('idx_ic_scores_overall', 'ic_scores', [sa.text('overall_score DESC')])
    op.create_index('idx_ic_scores_ticker_date', 'ic_scores', ['ticker', sa.text('date DESC')])
    op.create_index('idx_ic_scores_rating', 'ic_scores', ['rating'])

    # Create financials table
    op.create_table(
        'financials',
        sa.Column('id', sa.BigInteger, primary_key=True, autoincrement=True),
        sa.Column('ticker', sa.String(10), nullable=False),
        sa.Column('filing_date', sa.Date, nullable=False),
        sa.Column('period_end_date', sa.Date, nullable=False),
        sa.Column('fiscal_year', sa.Integer, nullable=False),
        sa.Column('fiscal_quarter', sa.Integer),
        sa.Column('statement_type', sa.String(20)),
        # Income Statement
        sa.Column('revenue', sa.BigInteger),
        sa.Column('cost_of_revenue', sa.BigInteger),
        sa.Column('gross_profit', sa.BigInteger),
        sa.Column('operating_expenses', sa.BigInteger),
        sa.Column('operating_income', sa.BigInteger),
        sa.Column('net_income', sa.BigInteger),
        sa.Column('eps_basic', sa.DECIMAL(10, 4)),
        sa.Column('eps_diluted', sa.DECIMAL(10, 4)),
        sa.Column('shares_outstanding', sa.BigInteger),
        # Balance Sheet
        sa.Column('total_assets', sa.BigInteger),
        sa.Column('total_liabilities', sa.BigInteger),
        sa.Column('shareholders_equity', sa.BigInteger),
        sa.Column('cash_and_equivalents', sa.BigInteger),
        sa.Column('short_term_debt', sa.BigInteger),
        sa.Column('long_term_debt', sa.BigInteger),
        # Cash Flow
        sa.Column('operating_cash_flow', sa.BigInteger),
        sa.Column('investing_cash_flow', sa.BigInteger),
        sa.Column('financing_cash_flow', sa.BigInteger),
        sa.Column('free_cash_flow', sa.BigInteger),
        sa.Column('capex', sa.BigInteger),
        # Calculated Metrics
        sa.Column('pe_ratio', sa.DECIMAL(10, 2)),
        sa.Column('pb_ratio', sa.DECIMAL(10, 2)),
        sa.Column('ps_ratio', sa.DECIMAL(10, 2)),
        sa.Column('debt_to_equity', sa.DECIMAL(10, 2)),
        sa.Column('current_ratio', sa.DECIMAL(10, 2)),
        sa.Column('quick_ratio', sa.DECIMAL(10, 2)),
        sa.Column('roe', sa.DECIMAL(10, 2)),
        sa.Column('roa', sa.DECIMAL(10, 2)),
        sa.Column('roic', sa.DECIMAL(10, 2)),
        sa.Column('gross_margin', sa.DECIMAL(10, 2)),
        sa.Column('operating_margin', sa.DECIMAL(10, 2)),
        sa.Column('net_margin', sa.DECIMAL(10, 2)),
        # Metadata
        sa.Column('sec_filing_url', sa.String(500)),
        sa.Column('raw_data', postgresql.JSONB),
        sa.Column('created_at', sa.TIMESTAMP, server_default=sa.text('NOW()')),
        sa.UniqueConstraint('ticker', 'period_end_date', 'fiscal_quarter')
    )
    op.create_index('idx_financials_ticker', 'financials', ['ticker'])
    op.create_index('idx_financials_period', 'financials', [sa.text('period_end_date DESC')])
    op.create_index('idx_financials_ticker_period', 'financials', ['ticker', sa.text('period_end_date DESC')])
    op.create_index('idx_financials_fiscal_year', 'financials', [sa.text('fiscal_year DESC')])

    # Create insider_trades table
    op.create_table(
        'insider_trades',
        sa.Column('id', sa.BigInteger, primary_key=True, autoincrement=True),
        sa.Column('ticker', sa.String(10), nullable=False),
        sa.Column('filing_date', sa.Date, nullable=False),
        sa.Column('transaction_date', sa.Date, nullable=False),
        sa.Column('insider_name', sa.String(255), nullable=False),
        sa.Column('insider_title', sa.String(255)),
        sa.Column('transaction_type', sa.String(50)),
        sa.Column('shares', sa.BigInteger, nullable=False),
        sa.Column('price_per_share', sa.DECIMAL(10, 2)),
        sa.Column('total_value', sa.BigInteger),
        sa.Column('shares_owned_after', sa.BigInteger),
        sa.Column('is_derivative', sa.Boolean, server_default='false'),
        sa.Column('form_type', sa.String(10)),
        sa.Column('sec_filing_url', sa.String(500)),
        sa.Column('created_at', sa.TIMESTAMP, server_default=sa.text('NOW()'))
    )
    op.create_index('idx_insider_ticker', 'insider_trades', ['ticker'])
    op.create_index('idx_insider_date', 'insider_trades', [sa.text('transaction_date DESC')])
    op.create_index('idx_insider_ticker_date', 'insider_trades', ['ticker', sa.text('transaction_date DESC')])
    op.create_index('idx_insider_type', 'insider_trades', ['transaction_type'])

    # Create institutional_holdings table
    op.create_table(
        'institutional_holdings',
        sa.Column('id', sa.BigInteger, primary_key=True, autoincrement=True),
        sa.Column('ticker', sa.String(10), nullable=False),
        sa.Column('filing_date', sa.Date, nullable=False),
        sa.Column('quarter_end_date', sa.Date, nullable=False),
        sa.Column('institution_name', sa.String(255), nullable=False),
        sa.Column('institution_cik', sa.String(20)),
        sa.Column('shares', sa.BigInteger, nullable=False),
        sa.Column('market_value', sa.BigInteger, nullable=False),
        sa.Column('percent_of_portfolio', sa.DECIMAL(10, 4)),
        sa.Column('position_change', sa.String(50)),
        sa.Column('shares_change', sa.BigInteger),
        sa.Column('percent_change', sa.DECIMAL(10, 2)),
        sa.Column('sec_filing_url', sa.String(500)),
        sa.Column('created_at', sa.TIMESTAMP, server_default=sa.text('NOW()')),
        sa.UniqueConstraint('ticker', 'quarter_end_date', 'institution_cik')
    )
    op.create_index('idx_institutional_ticker', 'institutional_holdings', ['ticker'])
    op.create_index('idx_institutional_date', 'institutional_holdings', [sa.text('quarter_end_date DESC')])
    op.create_index('idx_institutional_ticker_date', 'institutional_holdings', ['ticker', sa.text('quarter_end_date DESC')])
    op.create_index('idx_institutional_institution', 'institutional_holdings', ['institution_cik'])

    # Create analyst_ratings table
    op.create_table(
        'analyst_ratings',
        sa.Column('id', sa.BigInteger, primary_key=True, autoincrement=True),
        sa.Column('ticker', sa.String(10), nullable=False),
        sa.Column('rating_date', sa.Date, nullable=False),
        sa.Column('analyst_name', sa.String(255), nullable=False),
        sa.Column('analyst_firm', sa.String(255)),
        sa.Column('rating', sa.String(50), nullable=False),
        sa.Column('rating_numeric', sa.DECIMAL(3, 1)),
        sa.Column('price_target', sa.DECIMAL(10, 2)),
        sa.Column('prior_rating', sa.String(50)),
        sa.Column('prior_price_target', sa.DECIMAL(10, 2)),
        sa.Column('action', sa.String(50)),
        sa.Column('notes', sa.Text),
        sa.Column('source', sa.String(100)),
        sa.Column('created_at', sa.TIMESTAMP, server_default=sa.text('NOW()'))
    )
    op.create_index('idx_analyst_ticker', 'analyst_ratings', ['ticker'])
    op.create_index('idx_analyst_date', 'analyst_ratings', [sa.text('rating_date DESC')])
    op.create_index('idx_analyst_ticker_date', 'analyst_ratings', ['ticker', sa.text('rating_date DESC')])
    op.create_index('idx_analyst_firm', 'analyst_ratings', ['analyst_firm'])

    # Create news_articles table
    op.create_table(
        'news_articles',
        sa.Column('id', sa.BigInteger, primary_key=True, autoincrement=True),
        sa.Column('title', sa.String(500), nullable=False),
        sa.Column('url', sa.String(1000), unique=True, nullable=False),
        sa.Column('source', sa.String(255), nullable=False),
        sa.Column('published_at', sa.TIMESTAMP, nullable=False),
        sa.Column('summary', sa.Text),
        sa.Column('content', sa.Text),
        sa.Column('author', sa.String(255)),
        sa.Column('tickers', postgresql.ARRAY(sa.String(50))),
        sa.Column('sentiment_score', sa.DECIMAL(5, 2)),
        sa.Column('sentiment_label', sa.String(20)),
        sa.Column('relevance_score', sa.DECIMAL(5, 2)),
        sa.Column('categories', postgresql.ARRAY(sa.String(50))),
        sa.Column('image_url', sa.String(500)),
        sa.Column('created_at', sa.TIMESTAMP, server_default=sa.text('NOW()'))
    )
    op.create_index('idx_news_published', 'news_articles', [sa.text('published_at DESC')])
    op.execute('CREATE INDEX idx_news_tickers ON news_articles USING GIN(tickers)')
    op.create_index('idx_news_sentiment', 'news_articles', ['sentiment_score'])
    op.create_index('idx_news_source', 'news_articles', ['source'])

    # Create watchlists table
    op.create_table(
        'watchlists',
        sa.Column('id', postgresql.UUID(as_uuid=True), server_default=sa.text('gen_random_uuid()'), primary_key=True),
        sa.Column('user_id', postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column('name', sa.String(255), nullable=False),
        sa.Column('description', sa.Text),
        sa.Column('is_default', sa.Boolean, server_default='false'),
        sa.Column('color', sa.String(20)),
        sa.Column('sort_order', sa.Integer),
        sa.Column('created_at', sa.TIMESTAMP, server_default=sa.text('NOW()')),
        sa.Column('updated_at', sa.TIMESTAMP, server_default=sa.text('NOW()')),
        sa.ForeignKeyConstraint(['user_id'], ['users.id'], ondelete='CASCADE')
    )
    op.create_index('idx_watchlists_user', 'watchlists', ['user_id'])

    # Create watchlist_stocks table
    op.create_table(
        'watchlist_stocks',
        sa.Column('id', sa.BigInteger, primary_key=True, autoincrement=True),
        sa.Column('watchlist_id', postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column('ticker', sa.String(10), nullable=False),
        sa.Column('notes', sa.Text),
        sa.Column('position', sa.Integer),
        sa.Column('added_at', sa.TIMESTAMP, server_default=sa.text('NOW()')),
        sa.ForeignKeyConstraint(['watchlist_id'], ['watchlists.id'], ondelete='CASCADE'),
        sa.UniqueConstraint('watchlist_id', 'ticker')
    )
    op.create_index('idx_watchlist_stocks_watchlist', 'watchlist_stocks', ['watchlist_id'])
    op.create_index('idx_watchlist_stocks_ticker', 'watchlist_stocks', ['ticker'])

    # Create portfolios table
    op.create_table(
        'portfolios',
        sa.Column('id', postgresql.UUID(as_uuid=True), server_default=sa.text('gen_random_uuid()'), primary_key=True),
        sa.Column('user_id', postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column('name', sa.String(255), nullable=False),
        sa.Column('description', sa.Text),
        sa.Column('currency', sa.String(10), server_default='USD'),
        sa.Column('is_default', sa.Boolean, server_default='false'),
        sa.Column('created_at', sa.TIMESTAMP, server_default=sa.text('NOW()')),
        sa.Column('updated_at', sa.TIMESTAMP, server_default=sa.text('NOW()')),
        sa.ForeignKeyConstraint(['user_id'], ['users.id'], ondelete='CASCADE')
    )
    op.create_index('idx_portfolios_user', 'portfolios', ['user_id'])

    # Create portfolio_positions table
    op.create_table(
        'portfolio_positions',
        sa.Column('id', postgresql.UUID(as_uuid=True), server_default=sa.text('gen_random_uuid()'), primary_key=True),
        sa.Column('portfolio_id', postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column('ticker', sa.String(10), nullable=False),
        sa.Column('shares', sa.DECIMAL(18, 6), nullable=False),
        sa.Column('average_cost', sa.DECIMAL(10, 2), nullable=False),
        sa.Column('first_purchased_at', sa.TIMESTAMP),
        sa.Column('last_updated_at', sa.TIMESTAMP, server_default=sa.text('NOW()')),
        sa.Column('notes', sa.Text),
        sa.ForeignKeyConstraint(['portfolio_id'], ['portfolios.id'], ondelete='CASCADE'),
        sa.UniqueConstraint('portfolio_id', 'ticker')
    )
    op.create_index('idx_positions_portfolio', 'portfolio_positions', ['portfolio_id'])
    op.create_index('idx_positions_ticker', 'portfolio_positions', ['ticker'])

    # Create portfolio_transactions table
    op.create_table(
        'portfolio_transactions',
        sa.Column('id', postgresql.UUID(as_uuid=True), server_default=sa.text('gen_random_uuid()'), primary_key=True),
        sa.Column('portfolio_id', postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column('ticker', sa.String(10), nullable=False),
        sa.Column('transaction_type', sa.String(20), nullable=False),
        sa.Column('shares', sa.DECIMAL(18, 6), nullable=False),
        sa.Column('price', sa.DECIMAL(10, 2), nullable=False),
        sa.Column('total_amount', sa.DECIMAL(18, 2), nullable=False),
        sa.Column('fees', sa.DECIMAL(10, 2), server_default='0'),
        sa.Column('transaction_date', sa.Date, nullable=False),
        sa.Column('notes', sa.Text),
        sa.Column('created_at', sa.TIMESTAMP, server_default=sa.text('NOW()')),
        sa.ForeignKeyConstraint(['portfolio_id'], ['portfolios.id'], ondelete='CASCADE')
    )
    op.create_index('idx_transactions_portfolio', 'portfolio_transactions', ['portfolio_id'])
    op.create_index('idx_transactions_ticker', 'portfolio_transactions', ['ticker'])
    op.create_index('idx_transactions_date', 'portfolio_transactions', [sa.text('transaction_date DESC')])
    op.create_index('idx_transactions_type', 'portfolio_transactions', ['transaction_type'])

    # Create alerts table
    op.create_table(
        'alerts',
        sa.Column('id', postgresql.UUID(as_uuid=True), server_default=sa.text('gen_random_uuid()'), primary_key=True),
        sa.Column('user_id', postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column('ticker', sa.String(10), nullable=False),
        sa.Column('alert_type', sa.String(50), nullable=False),
        sa.Column('condition', postgresql.JSONB, nullable=False),
        sa.Column('delivery_method', postgresql.ARRAY(sa.String(50)), server_default=sa.text("ARRAY['email']")),
        sa.Column('is_active', sa.Boolean, server_default='true'),
        sa.Column('triggered_count', sa.Integer, server_default='0'),
        sa.Column('last_triggered_at', sa.TIMESTAMP),
        sa.Column('created_at', sa.TIMESTAMP, server_default=sa.text('NOW()')),
        sa.Column('expires_at', sa.TIMESTAMP),
        sa.ForeignKeyConstraint(['user_id'], ['users.id'], ondelete='CASCADE')
    )
    op.create_index('idx_alerts_user', 'alerts', ['user_id'])
    op.create_index('idx_alerts_ticker', 'alerts', ['ticker'])
    op.execute('CREATE INDEX idx_alerts_active ON alerts(is_active) WHERE is_active = true')
    op.create_index('idx_alerts_type', 'alerts', ['alert_type'])

    # Create stock_prices table (TimescaleDB hypertable)
    op.create_table(
        'stock_prices',
        sa.Column('time', sa.TIMESTAMP(timezone=True), nullable=False),
        sa.Column('ticker', sa.String(10), nullable=False),
        sa.Column('open', sa.DECIMAL(10, 2)),
        sa.Column('high', sa.DECIMAL(10, 2)),
        sa.Column('low', sa.DECIMAL(10, 2)),
        sa.Column('close', sa.DECIMAL(10, 2)),
        sa.Column('volume', sa.BigInteger),
        sa.Column('vwap', sa.DECIMAL(10, 2)),
        sa.Column('interval', sa.String(10), server_default='1day')
    )
    # Convert to hypertable
    op.execute("SELECT create_hypertable('stock_prices', 'time')")
    op.create_index('idx_prices_ticker_time', 'stock_prices', ['ticker', sa.text('time DESC')])
    op.create_index('idx_prices_time', 'stock_prices', [sa.text('time DESC')])
    op.create_index('idx_prices_interval', 'stock_prices', ['interval'])

    # Create technical_indicators table (TimescaleDB hypertable)
    op.create_table(
        'technical_indicators',
        sa.Column('time', sa.TIMESTAMP(timezone=True), nullable=False),
        sa.Column('ticker', sa.String(10), nullable=False),
        sa.Column('indicator_name', sa.String(50), nullable=False),
        sa.Column('value', sa.DECIMAL(18, 6)),
        sa.Column('metadata', postgresql.JSONB)
    )
    # Convert to hypertable
    op.execute("SELECT create_hypertable('technical_indicators', 'time')")
    op.create_index('idx_indicators_ticker_time', 'technical_indicators', ['ticker', sa.text('time DESC')])
    op.create_index('idx_indicators_name', 'technical_indicators', ['indicator_name'])
    op.create_index('idx_indicators_ticker_name', 'technical_indicators', ['ticker', 'indicator_name'])

    # Create triggers for updated_at columns
    op.execute("""
        CREATE OR REPLACE FUNCTION update_updated_at_column()
        RETURNS TRIGGER AS $$
        BEGIN
            NEW.updated_at = NOW();
            RETURN NEW;
        END;
        $$ language 'plpgsql';
    """)

    op.execute("""
        CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    """)

    op.execute("""
        CREATE TRIGGER update_watchlists_updated_at BEFORE UPDATE ON watchlists
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    """)

    op.execute("""
        CREATE TRIGGER update_portfolios_updated_at BEFORE UPDATE ON portfolios
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    """)

    # Create views
    op.execute("""
        CREATE VIEW latest_ic_scores AS
        SELECT DISTINCT ON (ticker)
            ticker,
            date,
            overall_score,
            rating,
            value_score,
            growth_score,
            profitability_score,
            financial_health_score,
            momentum_score,
            analyst_consensus_score,
            insider_activity_score,
            institutional_score,
            news_sentiment_score,
            technical_score,
            sector_percentile,
            confidence_level,
            data_completeness
        FROM ic_scores
        ORDER BY ticker, date DESC
    """)

    op.execute("""
        CREATE VIEW latest_financials AS
        SELECT DISTINCT ON (ticker)
            ticker,
            period_end_date,
            fiscal_year,
            fiscal_quarter,
            revenue,
            net_income,
            eps_diluted,
            pe_ratio,
            pb_ratio,
            debt_to_equity,
            roe,
            free_cash_flow
        FROM financials
        ORDER BY ticker, period_end_date DESC
    """)


def downgrade() -> None:
    """Drop all tables and extensions."""

    # Drop views
    op.execute('DROP VIEW IF EXISTS latest_financials')
    op.execute('DROP VIEW IF EXISTS latest_ic_scores')

    # Drop triggers
    op.execute('DROP TRIGGER IF EXISTS update_portfolios_updated_at ON portfolios')
    op.execute('DROP TRIGGER IF EXISTS update_watchlists_updated_at ON watchlists')
    op.execute('DROP TRIGGER IF EXISTS update_users_updated_at ON users')
    op.execute('DROP FUNCTION IF EXISTS update_updated_at_column()')

    # Drop tables (in reverse order due to foreign keys)
    op.drop_table('technical_indicators')
    op.drop_table('stock_prices')
    op.drop_table('alerts')
    op.drop_table('portfolio_transactions')
    op.drop_table('portfolio_positions')
    op.drop_table('portfolios')
    op.drop_table('watchlist_stocks')
    op.drop_table('watchlists')
    op.drop_table('news_articles')
    op.drop_table('analyst_ratings')
    op.drop_table('institutional_holdings')
    op.drop_table('insider_trades')
    op.drop_table('financials')
    op.drop_table('ic_scores')
    op.drop_table('companies')
    op.drop_table('users')

    # Drop extensions
    op.execute('DROP EXTENSION IF EXISTS timescaledb')
    op.execute('DROP EXTENSION IF EXISTS "uuid-ossp"')
