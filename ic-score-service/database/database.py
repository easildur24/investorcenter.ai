"""Database connection and session management for IC Score Service.

This module provides database connection, session management, and health check
utilities using SQLAlchemy 2.0 async patterns.
"""
import logging
import os
from contextlib import asynccontextmanager
from typing import AsyncGenerator

from sqlalchemy import text
from sqlalchemy.ext.asyncio import (
    AsyncEngine,
    AsyncSession,
    async_sessionmaker,
    create_async_engine,
)
from sqlalchemy.pool import NullPool

from models import Base

logger = logging.getLogger(__name__)


class DatabaseConfig:
    """Database configuration from environment variables."""

    def __init__(self):
        self.user = os.getenv("DB_USER", "postgres")
        self.password = os.getenv("DB_PASSWORD", "postgres")
        self.host = os.getenv("DB_HOST", "localhost")
        self.port = os.getenv("DB_PORT", "5432")
        self.database = os.getenv("DB_NAME", "investorcenter_db")
        self.sslmode = os.getenv("DB_SSLMODE", "prefer")

        # Connection pool settings
        self.pool_size = int(os.getenv("DB_POOL_SIZE", "20"))
        self.max_overflow = int(os.getenv("DB_MAX_OVERFLOW", "10"))
        self.pool_timeout = int(os.getenv("DB_POOL_TIMEOUT", "30"))
        self.pool_recycle = int(os.getenv("DB_POOL_RECYCLE", "3600"))

        # Query settings
        self.echo = os.getenv("DB_ECHO", "false").lower() == "true"
        self.echo_pool = os.getenv("DB_ECHO_POOL", "false").lower() == "true"

    @property
    def url(self) -> str:
        """Get async PostgreSQL connection URL."""
        return (
            f"postgresql+asyncpg://{self.user}:{self.password}"
            f"@{self.host}:{self.port}/{self.database}"
        )

    @property
    def sync_url(self) -> str:
        """Get synchronous PostgreSQL connection URL (for migrations)."""
        return (
            f"postgresql://{self.user}:{self.password}"
            f"@{self.host}:{self.port}/{self.database}"
        )

    def __repr__(self) -> str:
        return f"<DatabaseConfig(host='{self.host}', database='{self.database}')>"


class Database:
    """Database manager with connection pooling and session management."""

    def __init__(self, config: DatabaseConfig = None):
        """Initialize database with configuration.

        Args:
            config: Database configuration. If None, loads from environment.
        """
        self.config = config or DatabaseConfig()
        self._engine: AsyncEngine | None = None
        self._session_factory: async_sessionmaker | None = None

    def create_engine(self) -> AsyncEngine:
        """Create SQLAlchemy async engine with connection pooling.

        Returns:
            AsyncEngine instance configured with connection pool.
        """
        logger.info(f"Creating database engine for {self.config.database}")

        self._engine = create_async_engine(
            self.config.url,
            echo=self.config.echo,
            echo_pool=self.config.echo_pool,
            pool_size=self.config.pool_size,
            max_overflow=self.config.max_overflow,
            pool_timeout=self.config.pool_timeout,
            pool_recycle=self.config.pool_recycle,
            pool_pre_ping=True,  # Verify connections before using
        )

        return self._engine

    def create_session_factory(self) -> async_sessionmaker:
        """Create async session factory.

        Returns:
            async_sessionmaker for creating database sessions.
        """
        if self._engine is None:
            self.create_engine()

        self._session_factory = async_sessionmaker(
            self._engine,
            class_=AsyncSession,
            expire_on_commit=False,
            autoflush=False,
            autocommit=False,
        )

        return self._session_factory

    @property
    def engine(self) -> AsyncEngine:
        """Get database engine, creating if necessary."""
        if self._engine is None:
            self.create_engine()
        return self._engine

    @property
    def session_factory(self) -> async_sessionmaker:
        """Get session factory, creating if necessary."""
        if self._session_factory is None:
            self.create_session_factory()
        return self._session_factory

    @asynccontextmanager
    async def session(self) -> AsyncGenerator[AsyncSession, None]:
        """Provide a transactional scope around a series of operations.

        Usage:
            async with db.session() as session:
                result = await session.execute(query)
                await session.commit()

        Yields:
            AsyncSession instance.
        """
        async with self.session_factory() as session:
            try:
                yield session
                await session.commit()
            except Exception:
                await session.rollback()
                raise
            finally:
                await session.close()

    async def health_check(self) -> dict:
        """Check database connection health.

        Returns:
            Dict with health check status and details.
        """
        try:
            async with self.session() as session:
                result = await session.execute(text("SELECT 1"))
                row = result.scalar()

                # Check PostgreSQL version
                version_result = await session.execute(text("SELECT version()"))
                version = version_result.scalar()

                # Check TimescaleDB extension
                timescale_result = await session.execute(
                    text("SELECT default_version FROM pg_available_extensions WHERE name = 'timescaledb'")
                )
                timescale_version = timescale_result.scalar()

                return {
                    "status": "healthy",
                    "connected": True,
                    "database": self.config.database,
                    "host": self.config.host,
                    "postgresql_version": version.split()[1] if version else "unknown",
                    "timescaledb_version": timescale_version or "not installed",
                }
        except Exception as e:
            logger.error(f"Database health check failed: {e}")
            return {
                "status": "unhealthy",
                "connected": False,
                "error": str(e),
                "database": self.config.database,
                "host": self.config.host,
            }

    async def create_all_tables(self):
        """Create all tables defined in models (for development only)."""
        logger.warning("Creating all tables - should only be used in development")
        async with self.engine.begin() as conn:
            await conn.run_sync(Base.metadata.create_all)

    async def drop_all_tables(self):
        """Drop all tables (for development/testing only)."""
        logger.warning("Dropping all tables - should only be used in development/testing")
        async with self.engine.begin() as conn:
            await conn.run_sync(Base.metadata.drop_all)

    async def close(self):
        """Close database engine and connection pool."""
        if self._engine:
            await self._engine.dispose()
            logger.info("Database engine closed")


# Global database instance
_db_instance: Database | None = None


def get_database() -> Database:
    """Get or create global database instance.

    Returns:
        Database instance.
    """
    global _db_instance
    if _db_instance is None:
        _db_instance = Database()
    return _db_instance


async def get_session() -> AsyncGenerator[AsyncSession, None]:
    """Dependency injection for FastAPI routes.

    Usage:
        @app.get("/stocks/{ticker}")
        async def get_stock(ticker: str, session: AsyncSession = Depends(get_session)):
            result = await session.execute(select(Company).where(Company.ticker == ticker))
            return result.scalar_one_or_none()

    Yields:
        AsyncSession instance.
    """
    db = get_database()
    async with db.session() as session:
        yield session


# Synchronous database utilities (for Alembic migrations)
def get_sync_engine():
    """Create synchronous engine for Alembic migrations.

    Returns:
        Synchronous SQLAlchemy engine.
    """
    from sqlalchemy import create_engine

    config = DatabaseConfig()
    return create_engine(
        config.sync_url,
        echo=config.echo,
        pool_pre_ping=True,
    )


def run_migrations():
    """Run Alembic migrations (called from alembic env.py)."""
    from alembic import command
    from alembic.config import Config

    alembic_cfg = Config("alembic.ini")
    command.upgrade(alembic_cfg, "head")


if __name__ == "__main__":
    """Run basic database tests."""
    import asyncio

    async def test_connection():
        db = get_database()

        # Test health check
        print("Testing database connection...")
        health = await db.health_check()
        print(f"Health check: {health}")

        if health["status"] == "healthy":
            print("✓ Database connection successful")
            print(f"  PostgreSQL version: {health['postgresql_version']}")
            print(f"  TimescaleDB version: {health['timescaledb_version']}")
        else:
            print("✗ Database connection failed")
            print(f"  Error: {health.get('error')}")

        await db.close()

    asyncio.run(test_connection())
