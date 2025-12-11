#!/usr/bin/env python3
"""IC Score Coverage Monitoring Script.

This script checks IC Score data pipeline coverage and can be run:
- Manually for ad-hoc checks
- As a Kubernetes CronJob for periodic monitoring
- In CI/CD to validate deployment health

Usage:
    python monitor_coverage.py                    # Standard output
    python monitor_coverage.py --json             # JSON output for monitoring tools
    python monitor_coverage.py --alert-threshold 75  # Alert if below 75%
"""

import argparse
import asyncio
import json
import sys
from datetime import datetime
from pathlib import Path

# Add parent directory to path
sys.path.insert(0, str(Path(__file__).parent.parent))

from sqlalchemy import text
from database.database import get_database


async def get_coverage_metrics():
    """Get all coverage metrics from the IC Score database."""
    db = get_database()

    async with db.session() as session:
        # Total stocks
        total_stocks = (await session.execute(text("SELECT COUNT(*) FROM stocks"))).scalar()

        # Stocks with CIK
        stocks_with_cik = (await session.execute(
            text("SELECT COUNT(*) FROM stocks WHERE cik IS NOT NULL")
        )).scalar()

        # Coverage by pipeline stage
        sec_financials = (await session.execute(
            text("SELECT COUNT(DISTINCT ticker) FROM financials")
        )).scalar()

        ttm_financials = (await session.execute(
            text("SELECT COUNT(DISTINCT ticker) FROM ttm_financials")
        )).scalar()

        valuation_ratios = (await session.execute(
            text("SELECT COUNT(DISTINCT ticker) FROM valuation_ratios")
        )).scalar()

        ic_scores = (await session.execute(
            text("SELECT COUNT(DISTINCT ticker) FROM ic_scores")
        )).scalar()

        # Calculate percentages
        sec_coverage_pct = (sec_financials / stocks_with_cik * 100) if stocks_with_cik > 0 else 0
        ttm_coverage_pct = (ttm_financials / stocks_with_cik * 100) if stocks_with_cik > 0 else 0
        val_coverage_pct = (valuation_ratios / stocks_with_cik * 100) if stocks_with_cik > 0 else 0
        ic_coverage_pct = (ic_scores / stocks_with_cik * 100) if stocks_with_cik > 0 else 0

        # Gap analysis
        missing_sec = stocks_with_cik - sec_financials
        missing_ttm = stocks_with_cik - ttm_financials
        missing_val = stocks_with_cik - valuation_ratios

        return {
            "timestamp": datetime.utcnow().isoformat(),
            "inventory": {
                "total_stocks": total_stocks,
                "stocks_with_cik": stocks_with_cik,
                "stocks_without_cik": total_stocks - stocks_with_cik,
            },
            "coverage": {
                "sec_financials": {
                    "count": sec_financials,
                    "percentage": round(sec_coverage_pct, 1),
                    "missing": missing_sec,
                },
                "ttm_financials": {
                    "count": ttm_financials,
                    "percentage": round(ttm_coverage_pct, 1),
                    "missing": missing_ttm,
                },
                "valuation_ratios": {
                    "count": valuation_ratios,
                    "percentage": round(val_coverage_pct, 1),
                    "missing": missing_val,
                },
                "ic_scores": {
                    "count": ic_scores,
                    "percentage": round(ic_coverage_pct, 1),
                },
            },
        }


def print_human_readable(metrics: dict):
    """Print metrics in human-readable format."""
    print("=" * 80)
    print("IC SCORE COVERAGE REPORT")
    print(f"Generated: {metrics['timestamp']}")
    print("=" * 80)

    inv = metrics['inventory']
    print(f"\n📊 INVENTORY:")
    print(f"   Total Stocks: {inv['total_stocks']:,}")
    print(f"   With CIK: {inv['stocks_with_cik']:,} ({inv['stocks_with_cik']/inv['total_stocks']*100:.1f}%)")
    print(f"   Without CIK: {inv['stocks_without_cik']:,}")

    cov = metrics['coverage']
    print(f"\n📈 PIPELINE COVERAGE:")
    print(f"   SEC Financials: {cov['sec_financials']['count']:,} ({cov['sec_financials']['percentage']}%) - Missing: {cov['sec_financials']['missing']:,}")
    print(f"   TTM Financials: {cov['ttm_financials']['count']:,} ({cov['ttm_financials']['percentage']}%) - Missing: {cov['ttm_financials']['missing']:,}")
    print(f"   Valuation Ratios: {cov['valuation_ratios']['count']:,} ({cov['valuation_ratios']['percentage']}%) - Missing: {cov['valuation_ratios']['missing']:,}")
    print(f"   IC Scores: {cov['ic_scores']['count']:,} ({cov['ic_scores']['percentage']}%)")

    # Health status
    sec_pct = cov['sec_financials']['percentage']
    if sec_pct >= 90:
        status = "✅ HEALTHY"
    elif sec_pct >= 75:
        status = "⚠️  WARNING"
    else:
        status = "❌ CRITICAL"

    print(f"\n{status}")
    print("=" * 80)


async def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(
        description='Monitor IC Score data pipeline coverage',
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )

    parser.add_argument(
        '--json',
        action='store_true',
        help='Output metrics in JSON format',
    )
    parser.add_argument(
        '--alert-threshold',
        type=float,
        default=None,
        help='Alert threshold percentage (e.g., 75). Exit code 1 if below threshold.',
    )

    args = parser.parse_args()

    # Get metrics
    metrics = await get_coverage_metrics()

    # Output
    if args.json:
        print(json.dumps(metrics, indent=2))
    else:
        print_human_readable(metrics)

    # Alert check
    if args.alert_threshold is not None:
        sec_pct = metrics['coverage']['sec_financials']['percentage']
        if sec_pct < args.alert_threshold:
            print(f"\n⚠️  ALERT: Coverage ({sec_pct}%) is below threshold ({args.alert_threshold}%)", file=sys.stderr)
            sys.exit(1)


if __name__ == '__main__':
    asyncio.run(main())
