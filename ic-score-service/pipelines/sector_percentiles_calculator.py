"""Daily sector percentiles calculation pipeline.

This pipeline calculates distribution statistics for all financial metrics
within each GICS sector. These percentiles are used by the IC Score calculator
for sector-relative scoring.

Usage:
    python -m pipelines.sector_percentiles_calculator

Scheduling:
    Run daily before IC Score calculation (recommended: 4 AM EST)
"""
import asyncio
import argparse
import logging
import sys
from datetime import datetime
from typing import Optional

from database.database import Database, get_database
from pipelines.utils.sector_percentile import SectorPercentileAggregator

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout)
    ]
)
logger = logging.getLogger(__name__)


async def calculate_sector_percentiles(
    sector: Optional[str] = None,
    dry_run: bool = False
) -> dict:
    """Calculate sector percentile statistics.

    Args:
        sector: Optional specific sector to calculate (default: all sectors)
        dry_run: If True, only log what would be calculated without persisting

    Returns:
        Dict with calculation results
    """
    start_time = datetime.now()
    logger.info(f"Starting sector percentile calculation at {start_time}")

    db = get_database()

    try:
        async with db.session() as session:
            aggregator = SectorPercentileAggregator(session)

            if sector:
                logger.info(f"Calculating percentiles for sector: {sector}")
                if dry_run:
                    logger.info("[DRY RUN] Would calculate percentiles")
                    return {"sector": sector, "dry_run": True}

                count = await aggregator.calculate_sector(sector)
                results = {sector: count}
            else:
                logger.info("Calculating percentiles for all sectors")
                if dry_run:
                    sectors = await aggregator._get_active_sectors()
                    logger.info(f"[DRY RUN] Would calculate for {len(sectors)} sectors: {sectors}")
                    return {"sectors": sectors, "dry_run": True}

                results = await aggregator.calculate_all_sectors()

        end_time = datetime.now()
        duration = (end_time - start_time).total_seconds()

        # Log summary
        total_metrics = sum(results.values())
        logger.info(f"Completed sector percentile calculation in {duration:.2f}s")
        logger.info(f"Sectors processed: {len(results)}")
        logger.info(f"Total metrics calculated: {total_metrics}")

        for sector_name, metric_count in sorted(results.items()):
            logger.info(f"  {sector_name}: {metric_count} metrics")

        return {
            "status": "success",
            "sectors_processed": len(results),
            "total_metrics": total_metrics,
            "duration_seconds": duration,
            "results": results,
        }

    except Exception as e:
        logger.exception(f"Error calculating sector percentiles: {e}")
        return {
            "status": "error",
            "error": str(e),
        }


async def run_lifecycle_classification(dry_run: bool = False) -> dict:
    """Run lifecycle classification for all active companies.

    This should be run after sector percentiles are calculated.

    Args:
        dry_run: If True, only log what would be calculated

    Returns:
        Dict with classification results
    """
    from pipelines.utils.lifecycle import LifecycleClassifier, LifecycleStage
    from sqlalchemy import text

    start_time = datetime.now()
    logger.info(f"Starting lifecycle classification at {start_time}")

    db = get_database()
    results = {stage.value: 0 for stage in LifecycleStage}

    try:
        async with db.session() as session:
            classifier = LifecycleClassifier(session)

            # Get all active companies with their metrics
            query = text("""
                SELECT
                    c.ticker,
                    fme.revenue_growth_yoy,
                    fme.net_margin,
                    vr.ttm_pe_ratio as pe_ratio,
                    c.market_cap
                FROM companies c
                LEFT JOIN (
                    SELECT ticker, revenue_growth_yoy, net_margin
                    FROM fundamental_metrics_extended
                    WHERE (ticker, calculation_date) IN (
                        SELECT ticker, MAX(calculation_date)
                        FROM fundamental_metrics_extended
                        GROUP BY ticker
                    )
                ) fme ON c.ticker = fme.ticker
                LEFT JOIN (
                    SELECT ticker, ttm_pe_ratio
                    FROM valuation_ratios
                    WHERE (ticker, calculation_date) IN (
                        SELECT ticker, MAX(calculation_date)
                        FROM valuation_ratios
                        GROUP BY ticker
                    )
                ) vr ON c.ticker = vr.ticker
                WHERE c.is_active = true
                ORDER BY c.ticker
            """)

            result = await session.execute(query)
            companies = result.fetchall()

            logger.info(f"Classifying {len(companies)} companies")

            if dry_run:
                # Sample classification for logging
                sample_count = min(10, len(companies))
                logger.info(f"[DRY RUN] Sample classifications (first {sample_count}):")
                for row in companies[:sample_count]:
                    data = {
                        'revenue_growth_yoy': row.revenue_growth_yoy,
                        'net_margin': row.net_margin,
                        'pe_ratio': row.pe_ratio,
                        'market_cap': row.market_cap,
                    }
                    classification = classifier.classify(data)
                    logger.info(f"  {row.ticker}: {classification.stage.value} "
                               f"(confidence: {classification.confidence:.2f})")
                return {"dry_run": True, "companies": len(companies)}

            # Classify and store all companies
            for row in companies:
                data = {
                    'revenue_growth_yoy': row.revenue_growth_yoy,
                    'net_margin': row.net_margin,
                    'pe_ratio': row.pe_ratio,
                    'market_cap': row.market_cap,
                }
                classification = await classifier.classify_and_store(row.ticker, data)
                results[classification.stage.value] += 1

        end_time = datetime.now()
        duration = (end_time - start_time).total_seconds()

        logger.info(f"Completed lifecycle classification in {duration:.2f}s")
        logger.info(f"Classification distribution:")
        for stage, count in sorted(results.items()):
            pct = count / len(companies) * 100 if companies else 0
            logger.info(f"  {stage}: {count} ({pct:.1f}%)")

        return {
            "status": "success",
            "companies_processed": len(companies),
            "duration_seconds": duration,
            "distribution": results,
        }

    except Exception as e:
        logger.exception(f"Error classifying companies: {e}")
        return {
            "status": "error",
            "error": str(e),
        }


async def main(args: argparse.Namespace):
    """Main entry point for the pipeline."""
    logger.info("=" * 60)
    logger.info("IC Score v2.1 - Sector Percentiles & Lifecycle Pipeline")
    logger.info("=" * 60)

    # Get database (no async initialization needed - engine created on first use)
    db = get_database()

    results = {}

    # Step 1: Calculate sector percentiles
    if not args.skip_percentiles:
        percentile_results = await calculate_sector_percentiles(
            sector=args.sector,
            dry_run=args.dry_run
        )
        results['sector_percentiles'] = percentile_results

        if percentile_results.get('status') == 'error':
            logger.error("Sector percentile calculation failed, aborting")
            return 1

    # Step 2: Run lifecycle classification
    if not args.skip_lifecycle:
        lifecycle_results = await run_lifecycle_classification(
            dry_run=args.dry_run
        )
        results['lifecycle_classification'] = lifecycle_results

        if lifecycle_results.get('status') == 'error':
            logger.error("Lifecycle classification failed")
            return 1

    logger.info("=" * 60)
    logger.info("Pipeline completed successfully")
    logger.info("=" * 60)

    return 0


def parse_args():
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(
        description="Calculate sector percentiles and lifecycle classifications"
    )
    parser.add_argument(
        '--sector',
        type=str,
        default=None,
        help='Specific sector to calculate (default: all sectors)'
    )
    parser.add_argument(
        '--dry-run',
        action='store_true',
        help='Log what would be calculated without persisting'
    )
    parser.add_argument(
        '--skip-percentiles',
        action='store_true',
        help='Skip sector percentile calculation'
    )
    parser.add_argument(
        '--skip-lifecycle',
        action='store_true',
        help='Skip lifecycle classification'
    )
    return parser.parse_args()


if __name__ == "__main__":
    args = parse_args()
    exit_code = asyncio.run(main(args))
    sys.exit(exit_code)
