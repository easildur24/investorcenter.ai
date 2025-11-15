"""Seed database with sample data for development and testing.

This script creates:
- Sample users
- 5 major tech stocks (AAPL, MSFT, GOOGL, AMZN, TSLA)
- Sample financial data
- Sample IC scores
- Sample watchlists and portfolios
"""
import asyncio
import sys
from datetime import date, datetime, timedelta
from decimal import Decimal
from pathlib import Path

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

from sqlalchemy import select

from database import get_database
from models import (
    Alert, AnalystRating, Company, Financial, ICScore, InstitutionalHolding,
    InsiderTrade, NewsArticle, Portfolio, PortfolioPosition,
    PortfolioTransaction, User, Watchlist, WatchlistStock
)


async def create_sample_users(session):
    """Create sample users for testing."""
    print("Creating sample users...")

    users = [
        User(
            email="demo@investorcenter.ai",
            password_hash="$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5cqJ8T6t8J7ti",  # password: demo123
            first_name="Demo",
            last_name="User",
            subscription_tier="professional",
            email_verified=True,
            is_active=True,
        ),
        User(
            email="free@investorcenter.ai",
            password_hash="$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5cqJ8T6t8J7ti",
            first_name="Free",
            last_name="User",
            subscription_tier="free",
            email_verified=True,
            is_active=True,
        ),
    ]

    for user in users:
        session.add(user)

    await session.flush()
    print(f"✓ Created {len(users)} users")
    return users


async def create_sample_companies(session):
    """Create 5 major tech companies."""
    print("Creating sample companies...")

    companies = [
        Company(
            ticker="AAPL",
            name="Apple Inc.",
            sector="Technology",
            industry="Consumer Electronics",
            market_cap=3000000000000,  # $3T
            country="United States",
            exchange="NASDAQ",
            currency="USD",
            website="https://www.apple.com",
            description="Apple Inc. designs, manufactures, and markets smartphones, personal computers, tablets, wearables, and accessories worldwide.",
            employees=161000,
            founded_year=1976,
            hq_location="Cupertino, California",
            is_active=True,
        ),
        Company(
            ticker="MSFT",
            name="Microsoft Corporation",
            sector="Technology",
            industry="Software - Infrastructure",
            market_cap=2800000000000,  # $2.8T
            country="United States",
            exchange="NASDAQ",
            currency="USD",
            website="https://www.microsoft.com",
            description="Microsoft Corporation develops, licenses, and supports software, services, devices, and solutions worldwide.",
            employees=221000,
            founded_year=1975,
            hq_location="Redmond, Washington",
            is_active=True,
        ),
        Company(
            ticker="GOOGL",
            name="Alphabet Inc.",
            sector="Communication Services",
            industry="Internet Content & Information",
            market_cap=1800000000000,  # $1.8T
            country="United States",
            exchange="NASDAQ",
            currency="USD",
            website="https://www.abc.xyz",
            description="Alphabet Inc. offers various products and platforms worldwide, including Search, ads, Android, Chrome, hardware, and YouTube.",
            employees=182000,
            founded_year=1998,
            hq_location="Mountain View, California",
            is_active=True,
        ),
        Company(
            ticker="AMZN",
            name="Amazon.com, Inc.",
            sector="Consumer Cyclical",
            industry="Internet Retail",
            market_cap=1600000000000,  # $1.6T
            country="United States",
            exchange="NASDAQ",
            currency="USD",
            website="https://www.amazon.com",
            description="Amazon.com, Inc. engages in the retail sale of consumer products and subscriptions in North America and internationally.",
            employees=1541000,
            founded_year=1994,
            hq_location="Seattle, Washington",
            is_active=True,
        ),
        Company(
            ticker="TSLA",
            name="Tesla, Inc.",
            sector="Consumer Cyclical",
            industry="Auto Manufacturers",
            market_cap=800000000000,  # $800B
            country="United States",
            exchange="NASDAQ",
            currency="USD",
            website="https://www.tesla.com",
            description="Tesla, Inc. designs, develops, manufactures, leases, and sells electric vehicles, and energy generation and storage systems.",
            employees=127855,
            founded_year=2003,
            hq_location="Austin, Texas",
            is_active=True,
        ),
    ]

    for company in companies:
        session.add(company)

    await session.flush()
    print(f"✓ Created {len(companies)} companies")
    return companies


async def create_sample_financials(session, companies):
    """Create sample financial data for the past 4 quarters."""
    print("Creating sample financial data...")

    financials = []
    today = date.today()

    # Sample financial data (simplified for demo)
    company_data = {
        "AAPL": {
            "revenue": 95000000000,
            "net_income": 23000000000,
            "eps_diluted": Decimal("1.52"),
            "pe_ratio": Decimal("29.5"),
        },
        "MSFT": {
            "revenue": 60000000000,
            "net_income": 21000000000,
            "eps_diluted": Decimal("2.93"),
            "pe_ratio": Decimal("35.2"),
        },
        "GOOGL": {
            "revenue": 80000000000,
            "net_income": 18000000000,
            "eps_diluted": Decimal("1.44"),
            "pe_ratio": Decimal("25.8"),
        },
        "AMZN": {
            "revenue": 143000000000,
            "net_income": 9800000000,
            "eps_diluted": Decimal("0.94"),
            "pe_ratio": Decimal("45.3"),
        },
        "TSLA": {
            "revenue": 25000000000,
            "net_income": 2700000000,
            "eps_diluted": Decimal("0.85"),
            "pe_ratio": Decimal("60.5"),
        },
    }

    for company in companies:
        ticker = company.ticker
        data = company_data.get(ticker, {})

        # Create 4 quarters of financial data
        for quarter in range(1, 5):
            period_end = today - timedelta(days=90 * quarter)
            fiscal_year = period_end.year
            fiscal_quarter = ((period_end.month - 1) // 3) + 1

            financial = Financial(
                ticker=ticker,
                filing_date=period_end + timedelta(days=45),
                period_end_date=period_end,
                fiscal_year=fiscal_year,
                fiscal_quarter=fiscal_quarter,
                statement_type="10-Q",
                revenue=data.get("revenue", 50000000000) + (quarter * 1000000000),
                net_income=data.get("net_income", 10000000000) + (quarter * 200000000),
                eps_diluted=data.get("eps_diluted", Decimal("1.00")) + Decimal(str(quarter * 0.05)),
                pe_ratio=data.get("pe_ratio", Decimal("30.0")),
                pb_ratio=Decimal("8.5"),
                ps_ratio=Decimal("6.2"),
                debt_to_equity=Decimal("1.5"),
                current_ratio=Decimal("1.2"),
                roe=Decimal("42.5"),
                roa=Decimal("18.3"),
                gross_margin=Decimal("43.2"),
                operating_margin=Decimal("30.5"),
                net_margin=Decimal("25.8"),
            )
            financials.append(financial)
            session.add(financial)

    await session.flush()
    print(f"✓ Created {len(financials)} financial records")
    return financials


async def create_sample_ic_scores(session, companies):
    """Create sample IC scores for the past 30 days."""
    print("Creating sample IC scores...")

    ic_scores = []
    today = date.today()

    # Sample IC score data
    company_scores = {
        "AAPL": {"overall": 82, "rating": "Strong Buy"},
        "MSFT": {"overall": 78, "rating": "Buy"},
        "GOOGL": {"overall": 72, "rating": "Buy"},
        "AMZN": {"overall": 68, "rating": "Buy"},
        "TSLA": {"overall": 58, "rating": "Hold"},
    }

    for company in companies:
        ticker = company.ticker
        score_data = company_scores.get(ticker, {"overall": 65, "rating": "Buy"})

        # Create scores for past 30 days
        for days_ago in range(0, 31):
            score_date = today - timedelta(days=days_ago)
            overall = score_data["overall"] + ((-1) ** days_ago) * (days_ago % 3)  # Add variation

            ic_score = ICScore(
                ticker=ticker,
                date=score_date,
                overall_score=Decimal(str(overall)),
                value_score=Decimal(str(overall - 5)),
                growth_score=Decimal(str(overall + 3)),
                profitability_score=Decimal(str(overall)),
                financial_health_score=Decimal(str(overall - 2)),
                momentum_score=Decimal(str(overall + 5)),
                analyst_consensus_score=Decimal(str(overall - 3)),
                insider_activity_score=Decimal(str(overall)),
                institutional_score=Decimal(str(overall + 2)),
                news_sentiment_score=Decimal(str(overall - 1)),
                technical_score=Decimal(str(overall + 4)),
                rating=score_data["rating"],
                sector_percentile=Decimal("75.5"),
                confidence_level="High",
                data_completeness=Decimal("95.0"),
                calculation_metadata={
                    "calculated_at": datetime.now().isoformat(),
                    "factors_used": 10,
                    "data_sources": ["financials", "insider_trades", "analyst_ratings"],
                },
            )
            ic_scores.append(ic_score)
            session.add(ic_score)

    await session.flush()
    print(f"✓ Created {len(ic_scores)} IC scores")
    return ic_scores


async def create_sample_analyst_ratings(session, companies):
    """Create sample analyst ratings."""
    print("Creating sample analyst ratings...")

    ratings = []
    today = date.today()

    analysts = [
        ("Goldman Sachs", "Strong Buy", Decimal("4.5")),
        ("Morgan Stanley", "Buy", Decimal("4.0")),
        ("JPMorgan", "Hold", Decimal("3.0")),
    ]

    for company in companies:
        for firm, rating, rating_numeric in analysts:
            analyst_rating = AnalystRating(
                ticker=company.ticker,
                rating_date=today - timedelta(days=7),
                analyst_name="John Analyst",
                analyst_firm=firm,
                rating=rating,
                rating_numeric=rating_numeric,
                price_target=Decimal("200.00"),
                action="Reiterated",
                source="Wall Street Journal",
            )
            ratings.append(analyst_rating)
            session.add(analyst_rating)

    await session.flush()
    print(f"✓ Created {len(ratings)} analyst ratings")
    return ratings


async def create_sample_insider_trades(session, companies):
    """Create sample insider trades."""
    print("Creating sample insider trades...")

    trades = []
    today = date.today()

    for company in companies:
        # Create a buy and a sell trade
        for transaction_type, shares in [("Buy", 10000), ("Sell", 5000)]:
            trade = InsiderTrade(
                ticker=company.ticker,
                filing_date=today - timedelta(days=3),
                transaction_date=today - timedelta(days=5),
                insider_name="John Insider",
                insider_title="CEO",
                transaction_type=transaction_type,
                shares=shares,
                price_per_share=Decimal("180.50"),
                total_value=shares * 180,
                shares_owned_after=100000,
                is_derivative=False,
                form_type="Form 4",
            )
            trades.append(trade)
            session.add(trade)

    await session.flush()
    print(f"✓ Created {len(trades)} insider trades")
    return trades


async def create_sample_institutional_holdings(session, companies):
    """Create sample institutional holdings."""
    print("Creating sample institutional holdings...")

    holdings = []
    today = date.today()
    quarter_end = date(today.year, ((today.month - 1) // 3) * 3 + 1, 1) - timedelta(days=1)

    institutions = [
        ("Vanguard Group Inc", "0000102909"),
        ("BlackRock Inc", "0001086364"),
        ("State Street Corp", "0000093751"),
    ]

    for company in companies:
        for institution_name, cik in institutions:
            holding = InstitutionalHolding(
                ticker=company.ticker,
                filing_date=today - timedelta(days=45),
                quarter_end_date=quarter_end,
                institution_name=institution_name,
                institution_cik=cik,
                shares=50000000,
                market_value=9000000000,
                percent_of_portfolio=Decimal("2.5"),
                position_change="Increased",
                shares_change=5000000,
                percent_change=Decimal("11.1"),
            )
            holdings.append(holding)
            session.add(holding)

    await session.flush()
    print(f"✓ Created {len(holdings)} institutional holdings")
    return holdings


async def create_sample_news(session, companies):
    """Create sample news articles."""
    print("Creating sample news articles...")

    articles = []
    today = datetime.now()

    news_titles = [
        "{company} Reports Strong Quarterly Earnings",
        "{company} Announces New Product Launch",
        "Analysts Bullish on {company} Stock",
    ]

    for i, company in enumerate(companies):
        for j, title_template in enumerate(news_titles):
            article = NewsArticle(
                title=title_template.format(company=company.name),
                url=f"https://news.example.com/{company.ticker.lower()}-article-{i}-{j}",
                source="Financial Times",
                published_at=today - timedelta(hours=j * 8),
                summary=f"Breaking news about {company.name}...",
                tickers=[company.ticker],
                sentiment_score=Decimal("75.5"),
                sentiment_label="Positive",
                relevance_score=Decimal("85.0"),
                categories=["earnings", "technology"],
            )
            articles.append(article)
            session.add(article)

    await session.flush()
    print(f"✓ Created {len(articles)} news articles")
    return articles


async def create_sample_watchlists(session, users, companies):
    """Create sample watchlists for users."""
    print("Creating sample watchlists...")

    watchlists = []

    for user in users:
        # Create default watchlist
        watchlist = Watchlist(
            user_id=user.id,
            name="My Watchlist",
            description="Tracking major tech stocks",
            is_default=True,
            color="#3B82F6",
            sort_order=1,
        )
        session.add(watchlist)
        await session.flush()
        watchlists.append(watchlist)

        # Add stocks to watchlist
        for i, company in enumerate(companies):
            stock = WatchlistStock(
                watchlist_id=watchlist.id,
                ticker=company.ticker,
                notes=f"Monitoring {company.name}",
                position=i + 1,
            )
            session.add(stock)

    await session.flush()
    print(f"✓ Created {len(watchlists)} watchlists with {len(companies) * len(users)} stocks")
    return watchlists


async def create_sample_portfolios(session, users, companies):
    """Create sample portfolios for users."""
    print("Creating sample portfolios...")

    portfolios = []

    for user in users:
        # Create default portfolio
        portfolio = Portfolio(
            user_id=user.id,
            name="My Portfolio",
            description="Long-term growth portfolio",
            currency="USD",
            is_default=True,
        )
        session.add(portfolio)
        await session.flush()
        portfolios.append(portfolio)

        # Add positions
        for i, company in enumerate(companies):
            position = PortfolioPosition(
                portfolio_id=portfolio.id,
                ticker=company.ticker,
                shares=Decimal("100.0"),
                average_cost=Decimal("150.00"),
                first_purchased_at=datetime.now() - timedelta(days=365),
            )
            session.add(position)

            # Add a buy transaction
            transaction = PortfolioTransaction(
                portfolio_id=portfolio.id,
                ticker=company.ticker,
                transaction_type="BUY",
                shares=Decimal("100.0"),
                price=Decimal("150.00"),
                total_amount=Decimal("15000.00"),
                fees=Decimal("9.99"),
                transaction_date=date.today() - timedelta(days=365),
            )
            session.add(transaction)

    await session.flush()
    print(f"✓ Created {len(portfolios)} portfolios")
    return portfolios


async def create_sample_alerts(session, users, companies):
    """Create sample alerts for users."""
    print("Creating sample alerts...")

    alerts = []

    for user in users:
        for company in companies[:2]:  # Create alerts for first 2 companies
            alert = Alert(
                user_id=user.id,
                ticker=company.ticker,
                alert_type="PRICE",
                condition={"threshold": 200.00, "direction": "above"},
                delivery_method=["email", "push"],
                is_active=True,
            )
            alerts.append(alert)
            session.add(alert)

    await session.flush()
    print(f"✓ Created {len(alerts)} alerts")
    return alerts


async def seed_database():
    """Main function to seed the database."""
    print("\n" + "=" * 60)
    print("  InvestorCenter.ai - Database Seeding Script")
    print("=" * 60 + "\n")

    db = get_database()

    try:
        async with db.session() as session:
            # Create all sample data
            users = await create_sample_users(session)
            companies = await create_sample_companies(session)
            await create_sample_financials(session, companies)
            await create_sample_ic_scores(session, companies)
            await create_sample_analyst_ratings(session, companies)
            await create_sample_insider_trades(session, companies)
            await create_sample_institutional_holdings(session, companies)
            await create_sample_news(session, companies)
            await create_sample_watchlists(session, users, companies)
            await create_sample_portfolios(session, users, companies)
            await create_sample_alerts(session, users, companies)

            await session.commit()

        print("\n" + "=" * 60)
        print("  ✓ Database seeding completed successfully!")
        print("=" * 60 + "\n")
        print("Sample users:")
        print("  - demo@investorcenter.ai (password: demo123) - Professional tier")
        print("  - free@investorcenter.ai (password: demo123) - Free tier")
        print("\nSample stocks: AAPL, MSFT, GOOGL, AMZN, TSLA\n")

    except Exception as e:
        print(f"\n✗ Error seeding database: {e}")
        raise
    finally:
        await db.close()


if __name__ == "__main__":
    asyncio.run(seed_database())
