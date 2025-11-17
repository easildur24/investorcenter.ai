"""SQLAlchemy ORM models for InvestorCenter.ai IC Score Service.

This module defines all database models using SQLAlchemy 2.0 syntax with
type annotations and modern ORM patterns.
"""
from datetime import date, datetime
from decimal import Decimal
from typing import List, Optional
from uuid import UUID, uuid4

from sqlalchemy import (
    ARRAY, BigInteger, Boolean, Date, ForeignKey, Integer, Numeric, String,
    Text, TIMESTAMP, UniqueConstraint, text
)
from sqlalchemy.dialects.postgresql import JSONB, UUID as PGUUID
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column, relationship


class Base(DeclarativeBase):
    """Base class for all ORM models."""
    pass


# ============================================================================
# CORE TABLES
# ============================================================================

class User(Base):
    """User accounts with authentication and subscription information."""
    __tablename__ = 'users'

    id: Mapped[UUID] = mapped_column(
        PGUUID(as_uuid=True),
        primary_key=True,
        server_default=text('gen_random_uuid()')
    )
    email: Mapped[str] = mapped_column(String(255), unique=True, nullable=False)
    password_hash: Mapped[str] = mapped_column(String(255), nullable=False)
    first_name: Mapped[Optional[str]] = mapped_column(String(100))
    last_name: Mapped[Optional[str]] = mapped_column(String(100))
    subscription_tier: Mapped[str] = mapped_column(String(50), server_default='free')
    stripe_customer_id: Mapped[Optional[str]] = mapped_column(String(255))
    created_at: Mapped[datetime] = mapped_column(TIMESTAMP, server_default=text('NOW()'))
    updated_at: Mapped[datetime] = mapped_column(TIMESTAMP, server_default=text('NOW()'))
    last_login_at: Mapped[Optional[datetime]] = mapped_column(TIMESTAMP)
    is_active: Mapped[bool] = mapped_column(Boolean, server_default='true')
    email_verified: Mapped[bool] = mapped_column(Boolean, server_default='false')
    preferences: Mapped[dict] = mapped_column(JSONB, server_default=text("'{}'::jsonb"))

    # Relationships
    watchlists: Mapped[List["Watchlist"]] = relationship(back_populates="user", cascade="all, delete-orphan")
    portfolios: Mapped[List["Portfolio"]] = relationship(back_populates="user", cascade="all, delete-orphan")
    alerts: Mapped[List["Alert"]] = relationship(back_populates="user", cascade="all, delete-orphan")

    def __repr__(self) -> str:
        return f"<User(id={self.id}, email='{self.email}', tier='{self.subscription_tier}')>"


class Company(Base):
    """Company master data for all tracked stocks."""
    __tablename__ = 'companies'

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    ticker: Mapped[str] = mapped_column(String(10), unique=True, nullable=False)
    name: Mapped[str] = mapped_column(String(255), nullable=False)
    sector: Mapped[Optional[str]] = mapped_column(String(100))
    industry: Mapped[Optional[str]] = mapped_column(String(100))
    market_cap: Mapped[Optional[int]] = mapped_column(BigInteger)
    country: Mapped[Optional[str]] = mapped_column(String(50))
    exchange: Mapped[Optional[str]] = mapped_column(String(50))
    currency: Mapped[Optional[str]] = mapped_column(String(10))
    website: Mapped[Optional[str]] = mapped_column(String(255))
    description: Mapped[Optional[str]] = mapped_column(Text)
    employees: Mapped[Optional[int]] = mapped_column(Integer)
    founded_year: Mapped[Optional[int]] = mapped_column(Integer)
    hq_location: Mapped[Optional[str]] = mapped_column(String(255))
    logo_url: Mapped[Optional[str]] = mapped_column(String(255))
    is_active: Mapped[bool] = mapped_column(Boolean, server_default='true')
    last_updated: Mapped[datetime] = mapped_column(TIMESTAMP, server_default=text('NOW()'))
    created_at: Mapped[datetime] = mapped_column(TIMESTAMP, server_default=text('NOW()'))

    def __repr__(self) -> str:
        return f"<Company(ticker='{self.ticker}', name='{self.name}', sector='{self.sector}')>"


# ============================================================================
# IC SCORE SYSTEM
# ============================================================================

class ICScore(Base):
    """InvestorCenter proprietary 10-factor stock scores (1-100)."""
    __tablename__ = 'ic_scores'
    __table_args__ = (
        UniqueConstraint('ticker', 'date', name='uq_ic_scores_ticker_date'),
    )

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    ticker: Mapped[str] = mapped_column(String(10), nullable=False)
    date: Mapped[date] = mapped_column(Date, nullable=False)
    overall_score: Mapped[Decimal] = mapped_column(Numeric(5, 2), nullable=False)
    value_score: Mapped[Optional[Decimal]] = mapped_column(Numeric(5, 2))
    growth_score: Mapped[Optional[Decimal]] = mapped_column(Numeric(5, 2))
    profitability_score: Mapped[Optional[Decimal]] = mapped_column(Numeric(5, 2))
    financial_health_score: Mapped[Optional[Decimal]] = mapped_column(Numeric(5, 2))
    momentum_score: Mapped[Optional[Decimal]] = mapped_column(Numeric(5, 2))
    analyst_consensus_score: Mapped[Optional[Decimal]] = mapped_column(Numeric(5, 2))
    insider_activity_score: Mapped[Optional[Decimal]] = mapped_column(Numeric(5, 2))
    institutional_score: Mapped[Optional[Decimal]] = mapped_column(Numeric(5, 2))
    news_sentiment_score: Mapped[Optional[Decimal]] = mapped_column(Numeric(5, 2))
    technical_score: Mapped[Optional[Decimal]] = mapped_column(Numeric(5, 2))
    rating: Mapped[Optional[str]] = mapped_column(String(20))
    sector_percentile: Mapped[Optional[Decimal]] = mapped_column(Numeric(5, 2))
    confidence_level: Mapped[Optional[str]] = mapped_column(String(20))
    data_completeness: Mapped[Optional[Decimal]] = mapped_column(Numeric(5, 2))
    calculation_metadata: Mapped[Optional[dict]] = mapped_column(JSONB)
    created_at: Mapped[datetime] = mapped_column(TIMESTAMP, server_default=text('NOW()'))

    def __repr__(self) -> str:
        return f"<ICScore(ticker='{self.ticker}', date={self.date}, score={self.overall_score}, rating='{self.rating}')>"


# ============================================================================
# FINANCIAL DATA
# ============================================================================

class Financial(Base):
    """Quarterly and annual financial statements from SEC filings."""
    __tablename__ = 'financials'
    __table_args__ = (
        UniqueConstraint('ticker', 'period_end_date', 'fiscal_quarter', name='uq_financials_ticker_period_quarter'),
    )

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    ticker: Mapped[str] = mapped_column(String(10), nullable=False)
    filing_date: Mapped[date] = mapped_column(Date, nullable=False)
    period_end_date: Mapped[date] = mapped_column(Date, nullable=False)
    fiscal_year: Mapped[int] = mapped_column(Integer, nullable=False)
    fiscal_quarter: Mapped[Optional[int]] = mapped_column(Integer)
    statement_type: Mapped[Optional[str]] = mapped_column(String(20))

    # Income Statement
    revenue: Mapped[Optional[int]] = mapped_column(BigInteger)
    cost_of_revenue: Mapped[Optional[int]] = mapped_column(BigInteger)
    gross_profit: Mapped[Optional[int]] = mapped_column(BigInteger)
    operating_expenses: Mapped[Optional[int]] = mapped_column(BigInteger)
    operating_income: Mapped[Optional[int]] = mapped_column(BigInteger)
    net_income: Mapped[Optional[int]] = mapped_column(BigInteger)
    eps_basic: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 4))
    eps_diluted: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 4))
    shares_outstanding: Mapped[Optional[int]] = mapped_column(BigInteger)

    # Balance Sheet
    total_assets: Mapped[Optional[int]] = mapped_column(BigInteger)
    total_liabilities: Mapped[Optional[int]] = mapped_column(BigInteger)
    shareholders_equity: Mapped[Optional[int]] = mapped_column(BigInteger)
    cash_and_equivalents: Mapped[Optional[int]] = mapped_column(BigInteger)
    short_term_debt: Mapped[Optional[int]] = mapped_column(BigInteger)
    long_term_debt: Mapped[Optional[int]] = mapped_column(BigInteger)

    # Cash Flow
    operating_cash_flow: Mapped[Optional[int]] = mapped_column(BigInteger)
    investing_cash_flow: Mapped[Optional[int]] = mapped_column(BigInteger)
    financing_cash_flow: Mapped[Optional[int]] = mapped_column(BigInteger)
    free_cash_flow: Mapped[Optional[int]] = mapped_column(BigInteger)
    capex: Mapped[Optional[int]] = mapped_column(BigInteger)

    # Calculated Metrics
    pe_ratio: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    pb_ratio: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    ps_ratio: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    debt_to_equity: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    current_ratio: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    quick_ratio: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    roe: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    roa: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    roic: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    gross_margin: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    operating_margin: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    net_margin: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))

    # Metadata
    sec_filing_url: Mapped[Optional[str]] = mapped_column(String(500))
    raw_data: Mapped[Optional[dict]] = mapped_column(JSONB)
    created_at: Mapped[datetime] = mapped_column(TIMESTAMP, server_default=text('NOW()'))

    def __repr__(self) -> str:
        return f"<Financial(ticker='{self.ticker}', period={self.period_end_date}, FY{self.fiscal_year}Q{self.fiscal_quarter})>"


# ============================================================================
# INSIDER & INSTITUTIONAL DATA
# ============================================================================

class InsiderTrade(Base):
    """Insider trading activity from SEC Form 4 filings."""
    __tablename__ = 'insider_trades'

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    ticker: Mapped[str] = mapped_column(String(10), nullable=False)
    filing_date: Mapped[date] = mapped_column(Date, nullable=False)
    transaction_date: Mapped[date] = mapped_column(Date, nullable=False)
    insider_name: Mapped[str] = mapped_column(String(255), nullable=False)
    insider_title: Mapped[Optional[str]] = mapped_column(String(255))
    transaction_type: Mapped[Optional[str]] = mapped_column(String(50))
    shares: Mapped[int] = mapped_column(BigInteger, nullable=False)
    price_per_share: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    total_value: Mapped[Optional[int]] = mapped_column(BigInteger)
    shares_owned_after: Mapped[Optional[int]] = mapped_column(BigInteger)
    is_derivative: Mapped[bool] = mapped_column(Boolean, server_default='false')
    form_type: Mapped[Optional[str]] = mapped_column(String(10))
    sec_filing_url: Mapped[Optional[str]] = mapped_column(String(500))
    created_at: Mapped[datetime] = mapped_column(TIMESTAMP, server_default=text('NOW()'))

    def __repr__(self) -> str:
        return f"<InsiderTrade(ticker='{self.ticker}', date={self.transaction_date}, type='{self.transaction_type}', shares={self.shares})>"


class InstitutionalHolding(Base):
    """Institutional ownership data from SEC Form 13F filings."""
    __tablename__ = 'institutional_holdings'
    __table_args__ = (
        UniqueConstraint('ticker', 'quarter_end_date', 'institution_cik', name='uq_institutional_ticker_quarter_cik'),
    )

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    ticker: Mapped[str] = mapped_column(String(10), nullable=False)
    filing_date: Mapped[date] = mapped_column(Date, nullable=False)
    quarter_end_date: Mapped[date] = mapped_column(Date, nullable=False)
    institution_name: Mapped[str] = mapped_column(String(255), nullable=False)
    institution_cik: Mapped[Optional[str]] = mapped_column(String(20))
    shares: Mapped[int] = mapped_column(BigInteger, nullable=False)
    market_value: Mapped[int] = mapped_column(BigInteger, nullable=False)
    percent_of_portfolio: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 4))
    position_change: Mapped[Optional[str]] = mapped_column(String(50))
    shares_change: Mapped[Optional[int]] = mapped_column(BigInteger)
    percent_change: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    sec_filing_url: Mapped[Optional[str]] = mapped_column(String(500))
    created_at: Mapped[datetime] = mapped_column(TIMESTAMP, server_default=text('NOW()'))

    def __repr__(self) -> str:
        return f"<InstitutionalHolding(ticker='{self.ticker}', institution='{self.institution_name}', quarter={self.quarter_end_date})>"


# ============================================================================
# ANALYST & NEWS DATA
# ============================================================================

class AnalystRating(Base):
    """Wall Street analyst ratings and price targets."""
    __tablename__ = 'analyst_ratings'

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    ticker: Mapped[str] = mapped_column(String(10), nullable=False)
    rating_date: Mapped[date] = mapped_column(Date, nullable=False)
    analyst_name: Mapped[str] = mapped_column(String(255), nullable=False)
    analyst_firm: Mapped[Optional[str]] = mapped_column(String(255))
    rating: Mapped[str] = mapped_column(String(50), nullable=False)
    rating_numeric: Mapped[Optional[Decimal]] = mapped_column(Numeric(3, 1))
    price_target: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    prior_rating: Mapped[Optional[str]] = mapped_column(String(50))
    prior_price_target: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    action: Mapped[Optional[str]] = mapped_column(String(50))
    notes: Mapped[Optional[str]] = mapped_column(Text)
    source: Mapped[Optional[str]] = mapped_column(String(100))
    created_at: Mapped[datetime] = mapped_column(TIMESTAMP, server_default=text('NOW()'))

    def __repr__(self) -> str:
        return f"<AnalystRating(ticker='{self.ticker}', date={self.rating_date}, rating='{self.rating}', target=${self.price_target})>"


class NewsArticle(Base):
    """News articles with AI-powered sentiment analysis."""
    __tablename__ = 'news_articles'

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    title: Mapped[str] = mapped_column(String(500), nullable=False)
    url: Mapped[str] = mapped_column(String(1000), unique=True, nullable=False)
    source: Mapped[str] = mapped_column(String(255), nullable=False)
    published_at: Mapped[datetime] = mapped_column(TIMESTAMP, nullable=False)
    summary: Mapped[Optional[str]] = mapped_column(Text)
    content: Mapped[Optional[str]] = mapped_column(Text)
    author: Mapped[Optional[str]] = mapped_column(String(255))
    tickers: Mapped[Optional[List[str]]] = mapped_column(ARRAY(String(50)))
    sentiment_score: Mapped[Optional[Decimal]] = mapped_column(Numeric(5, 2))
    sentiment_label: Mapped[Optional[str]] = mapped_column(String(20))
    relevance_score: Mapped[Optional[Decimal]] = mapped_column(Numeric(5, 2))
    categories: Mapped[Optional[List[str]]] = mapped_column(ARRAY(String(50)))
    image_url: Mapped[Optional[str]] = mapped_column(String(500))
    created_at: Mapped[datetime] = mapped_column(TIMESTAMP, server_default=text('NOW()'))

    def __repr__(self) -> str:
        return f"<NewsArticle(title='{self.title[:50]}...', source='{self.source}', sentiment={self.sentiment_score})>"


# ============================================================================
# WATCHLISTS & PORTFOLIOS
# ============================================================================

class Watchlist(Base):
    """User-created stock watchlists."""
    __tablename__ = 'watchlists'

    id: Mapped[UUID] = mapped_column(
        PGUUID(as_uuid=True),
        primary_key=True,
        server_default=text('gen_random_uuid()')
    )
    user_id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), ForeignKey('users.id', ondelete='CASCADE'), nullable=False)
    name: Mapped[str] = mapped_column(String(255), nullable=False)
    description: Mapped[Optional[str]] = mapped_column(Text)
    is_default: Mapped[bool] = mapped_column(Boolean, server_default='false')
    color: Mapped[Optional[str]] = mapped_column(String(20))
    sort_order: Mapped[Optional[int]] = mapped_column(Integer)
    created_at: Mapped[datetime] = mapped_column(TIMESTAMP, server_default=text('NOW()'))
    updated_at: Mapped[datetime] = mapped_column(TIMESTAMP, server_default=text('NOW()'))

    # Relationships
    user: Mapped["User"] = relationship(back_populates="watchlists")
    stocks: Mapped[List["WatchlistStock"]] = relationship(back_populates="watchlist", cascade="all, delete-orphan")

    def __repr__(self) -> str:
        return f"<Watchlist(id={self.id}, name='{self.name}', user_id={self.user_id})>"


class WatchlistStock(Base):
    """Stocks added to user watchlists."""
    __tablename__ = 'watchlist_stocks'
    __table_args__ = (
        UniqueConstraint('watchlist_id', 'ticker', name='uq_watchlist_stocks_watchlist_ticker'),
    )

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True, autoincrement=True)
    watchlist_id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), ForeignKey('watchlists.id', ondelete='CASCADE'), nullable=False)
    ticker: Mapped[str] = mapped_column(String(10), nullable=False)
    notes: Mapped[Optional[str]] = mapped_column(Text)
    position: Mapped[Optional[int]] = mapped_column(Integer)
    added_at: Mapped[datetime] = mapped_column(TIMESTAMP, server_default=text('NOW()'))

    # Relationships
    watchlist: Mapped["Watchlist"] = relationship(back_populates="stocks")

    def __repr__(self) -> str:
        return f"<WatchlistStock(watchlist_id={self.watchlist_id}, ticker='{self.ticker}')>"


class Portfolio(Base):
    """User investment portfolios."""
    __tablename__ = 'portfolios'

    id: Mapped[UUID] = mapped_column(
        PGUUID(as_uuid=True),
        primary_key=True,
        server_default=text('gen_random_uuid()')
    )
    user_id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), ForeignKey('users.id', ondelete='CASCADE'), nullable=False)
    name: Mapped[str] = mapped_column(String(255), nullable=False)
    description: Mapped[Optional[str]] = mapped_column(Text)
    currency: Mapped[str] = mapped_column(String(10), server_default='USD')
    is_default: Mapped[bool] = mapped_column(Boolean, server_default='false')
    created_at: Mapped[datetime] = mapped_column(TIMESTAMP, server_default=text('NOW()'))
    updated_at: Mapped[datetime] = mapped_column(TIMESTAMP, server_default=text('NOW()'))

    # Relationships
    user: Mapped["User"] = relationship(back_populates="portfolios")
    positions: Mapped[List["PortfolioPosition"]] = relationship(back_populates="portfolio", cascade="all, delete-orphan")
    transactions: Mapped[List["PortfolioTransaction"]] = relationship(back_populates="portfolio", cascade="all, delete-orphan")

    def __repr__(self) -> str:
        return f"<Portfolio(id={self.id}, name='{self.name}', user_id={self.user_id})>"


class PortfolioPosition(Base):
    """Current stock positions in portfolios."""
    __tablename__ = 'portfolio_positions'
    __table_args__ = (
        UniqueConstraint('portfolio_id', 'ticker', name='uq_positions_portfolio_ticker'),
    )

    id: Mapped[UUID] = mapped_column(
        PGUUID(as_uuid=True),
        primary_key=True,
        server_default=text('gen_random_uuid()')
    )
    portfolio_id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), ForeignKey('portfolios.id', ondelete='CASCADE'), nullable=False)
    ticker: Mapped[str] = mapped_column(String(10), nullable=False)
    shares: Mapped[Decimal] = mapped_column(Numeric(18, 6), nullable=False)
    average_cost: Mapped[Decimal] = mapped_column(Numeric(10, 2), nullable=False)
    first_purchased_at: Mapped[Optional[datetime]] = mapped_column(TIMESTAMP)
    last_updated_at: Mapped[datetime] = mapped_column(TIMESTAMP, server_default=text('NOW()'))
    notes: Mapped[Optional[str]] = mapped_column(Text)

    # Relationships
    portfolio: Mapped["Portfolio"] = relationship(back_populates="positions")

    def __repr__(self) -> str:
        return f"<PortfolioPosition(portfolio_id={self.portfolio_id}, ticker='{self.ticker}', shares={self.shares})>"


class PortfolioTransaction(Base):
    """Transaction history for portfolio tracking."""
    __tablename__ = 'portfolio_transactions'

    id: Mapped[UUID] = mapped_column(
        PGUUID(as_uuid=True),
        primary_key=True,
        server_default=text('gen_random_uuid()')
    )
    portfolio_id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), ForeignKey('portfolios.id', ondelete='CASCADE'), nullable=False)
    ticker: Mapped[str] = mapped_column(String(10), nullable=False)
    transaction_type: Mapped[str] = mapped_column(String(20), nullable=False)
    shares: Mapped[Decimal] = mapped_column(Numeric(18, 6), nullable=False)
    price: Mapped[Decimal] = mapped_column(Numeric(10, 2), nullable=False)
    total_amount: Mapped[Decimal] = mapped_column(Numeric(18, 2), nullable=False)
    fees: Mapped[Decimal] = mapped_column(Numeric(10, 2), server_default='0')
    transaction_date: Mapped[date] = mapped_column(Date, nullable=False)
    notes: Mapped[Optional[str]] = mapped_column(Text)
    created_at: Mapped[datetime] = mapped_column(TIMESTAMP, server_default=text('NOW()'))

    # Relationships
    portfolio: Mapped["Portfolio"] = relationship(back_populates="transactions")

    def __repr__(self) -> str:
        return f"<PortfolioTransaction(ticker='{self.ticker}', type='{self.transaction_type}', date={self.transaction_date})>"


# ============================================================================
# ALERTS
# ============================================================================

class Alert(Base):
    """User-configured alerts for price, score, and event notifications."""
    __tablename__ = 'alerts'

    id: Mapped[UUID] = mapped_column(
        PGUUID(as_uuid=True),
        primary_key=True,
        server_default=text('gen_random_uuid()')
    )
    user_id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), ForeignKey('users.id', ondelete='CASCADE'), nullable=False)
    ticker: Mapped[str] = mapped_column(String(10), nullable=False)
    alert_type: Mapped[str] = mapped_column(String(50), nullable=False)
    condition: Mapped[dict] = mapped_column(JSONB, nullable=False)
    delivery_method: Mapped[List[str]] = mapped_column(ARRAY(String(50)), server_default=text("ARRAY['email']"))
    is_active: Mapped[bool] = mapped_column(Boolean, server_default='true')
    triggered_count: Mapped[int] = mapped_column(Integer, server_default='0')
    last_triggered_at: Mapped[Optional[datetime]] = mapped_column(TIMESTAMP)
    created_at: Mapped[datetime] = mapped_column(TIMESTAMP, server_default=text('NOW()'))
    expires_at: Mapped[Optional[datetime]] = mapped_column(TIMESTAMP)

    # Relationships
    user: Mapped["User"] = relationship(back_populates="alerts")

    def __repr__(self) -> str:
        return f"<Alert(ticker='{self.ticker}', type='{self.alert_type}', active={self.is_active})>"


# ============================================================================
# TIME-SERIES DATA (TimescaleDB Hypertables)
# ============================================================================

class StockPrice(Base):
    """TimescaleDB hypertable for efficient time-series price storage."""
    __tablename__ = 'stock_prices'

    time: Mapped[datetime] = mapped_column(TIMESTAMP(timezone=True), primary_key=True, nullable=False)
    ticker: Mapped[str] = mapped_column(String(10), primary_key=True, nullable=False)
    open: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    high: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    low: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    close: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    volume: Mapped[Optional[int]] = mapped_column(BigInteger)
    vwap: Mapped[Optional[Decimal]] = mapped_column(Numeric(10, 2))
    interval: Mapped[str] = mapped_column(String(10), server_default='1day')

    def __repr__(self) -> str:
        return f"<StockPrice(ticker='{self.ticker}', time={self.time}, close={self.close})>"


class TechnicalIndicator(Base):
    """TimescaleDB hypertable for technical indicators (RSI, MACD, etc.)."""
    __tablename__ = 'technical_indicators'

    time: Mapped[datetime] = mapped_column(TIMESTAMP(timezone=True), primary_key=True, nullable=False)
    ticker: Mapped[str] = mapped_column(String(10), primary_key=True, nullable=False)
    indicator_name: Mapped[str] = mapped_column(String(50), primary_key=True, nullable=False)
    value: Mapped[Optional[Decimal]] = mapped_column(Numeric(18, 6))
    indicator_metadata: Mapped[Optional[dict]] = mapped_column('metadata', JSONB)

    def __repr__(self) -> str:
        return f"<TechnicalIndicator(ticker='{self.ticker}', indicator='{self.indicator_name}', value={self.value})>"
