"""
IC Score v2.1 Backtesting Infrastructure

This module provides tools for backtesting the IC Score methodology
against historical stock returns.
"""

from .backtester import ICScoreBacktester
from .portfolio import DecilePortfolioBuilder, Portfolio
from .metrics import BacktestMetrics, PerformanceCalculator
from .report import BacktestReportGenerator

__all__ = [
    'ICScoreBacktester',
    'DecilePortfolioBuilder',
    'Portfolio',
    'BacktestMetrics',
    'PerformanceCalculator',
    'BacktestReportGenerator',
]
