#!/usr/bin/env python3
"""Backfill 3-Year Historical Data Script.

This script orchestrates the backfill of 3 years of historical data for:
1. Stock prices (from Polygon.io)
2. S&P 500 benchmark data (from Polygon.io)
3. Treasury rates (from FRED API)

This should be run once during initial setup to populate historical data.

Usage:
    python backfill_3year_data.py --all              # Backfill everything
    python backfill_3year_data.py --prices           # Only stock prices
    python backfill_3year_data.py --benchmark        # Only benchmark data
    python backfill_3year_data.py --treasury         # Only treasury rates
    python backfill_3year_data.py --limit 100        # Test on 100 stocks
"""

import argparse
import asyncio
import logging
import subprocess
import sys
from datetime import datetime
from pathlib import Path
from typing import List, Optional

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler('/app/logs/backfill_3year_data.log')
    ]
)
logger = logging.getLogger(__name__)


class BackfillOrchestrator:
    """Orchestrates the backfill of 3 years of historical data."""

    def __init__(self, pipelines_dir: Path):
        """Initialize the orchestrator.

        Args:
            pipelines_dir: Path to pipelines directory.
        """
        self.pipelines_dir = pipelines_dir
        self.start_time = datetime.now()

    def run_pipeline(self, script_name: str, args: List[str] = None) -> bool:
        """Run a pipeline script.

        Args:
            script_name: Name of pipeline script.
            args: Additional command-line arguments.

        Returns:
            True if successful.
        """
        script_path = self.pipelines_dir / script_name
        command = ["python", str(script_path)]

        if args:
            command.extend(args)

        logger.info("=" * 80)
        logger.info(f"Running: {' '.join(command)}")
        logger.info("=" * 80)

        try:
            result = subprocess.run(
                command,
                check=True,
                capture_output=True,
                text=True
            )

            logger.info(f"✓ {script_name} completed successfully")
            logger.debug(f"Output:\n{result.stdout}")

            return True

        except subprocess.CalledProcessError as e:
            logger.error(f"✗ {script_name} failed with exit code {e.returncode}")
            logger.error(f"Error output:\n{e.stderr}")
            return False

        except Exception as e:
            logger.error(f"✗ {script_name} failed: {e}", exc_info=True)
            return False

    def backfill_stock_prices(self, limit: Optional[int] = None) -> bool:
        """Backfill stock prices using technical indicators pipeline.

        Args:
            limit: Limit number of stocks.

        Returns:
            True if successful.
        """
        logger.info("Step 1: Backfilling stock prices (3 years)")

        args = []
        if limit:
            args.extend(["--limit", str(limit)])
        else:
            args.append("--all")

        return self.run_pipeline("technical_indicators_calculator.py", args)

    def backfill_benchmark_data(self) -> bool:
        """Backfill S&P 500 benchmark data.

        Returns:
            True if successful.
        """
        logger.info("Step 2: Backfilling S&P 500 benchmark data (3 years)")

        return self.run_pipeline("benchmark_data_ingestion.py", ["--backfill"])

    def backfill_treasury_rates(self) -> bool:
        """Backfill Treasury rates from FRED.

        Returns:
            True if successful.
        """
        logger.info("Step 3: Backfilling Treasury rates (3 years)")

        return self.run_pipeline("treasury_rates_ingestion.py", ["--backfill"])

    def run(
        self,
        prices: bool = False,
        benchmark: bool = False,
        treasury: bool = False,
        all_data: bool = False,
        limit: Optional[int] = None
    ):
        """Run the backfill process.

        Args:
            prices: Backfill stock prices only.
            benchmark: Backfill benchmark data only.
            treasury: Backfill treasury rates only.
            all_data: Backfill all data.
            limit: Limit number of stocks for price backfill.
        """
        logger.info("=" * 80)
        logger.info("3-Year Historical Data Backfill")
        logger.info("=" * 80)

        if limit:
            logger.info(f"Testing mode: Limited to {limit} stocks")

        # Track results
        results = {}

        # Determine what to backfill
        if all_data or (not prices and not benchmark and not treasury):
            # Backfill everything in order
            steps = [
                ("Benchmark Data", lambda: self.backfill_benchmark_data()),
                ("Treasury Rates", lambda: self.backfill_treasury_rates()),
                ("Stock Prices", lambda: self.backfill_stock_prices(limit))
            ]
        else:
            # Backfill specific components
            steps = []
            if benchmark:
                steps.append(("Benchmark Data", lambda: self.backfill_benchmark_data()))
            if treasury:
                steps.append(("Treasury Rates", lambda: self.backfill_treasury_rates()))
            if prices:
                steps.append(("Stock Prices", lambda: self.backfill_stock_prices(limit)))

        # Run each step
        for step_name, step_func in steps:
            logger.info("")
            logger.info(f"Starting: {step_name}")
            success = step_func()
            results[step_name] = success

            if not success:
                logger.warning(f"⚠ {step_name} failed, but continuing...")

        # Print summary
        duration = datetime.now() - self.start_time
        logger.info("")
        logger.info("=" * 80)
        logger.info("Backfill Complete")
        logger.info("=" * 80)
        logger.info(f"Duration: {duration}")
        logger.info("")
        logger.info("Results:")

        success_count = 0
        for step_name, success in results.items():
            status = "✓ SUCCESS" if success else "✗ FAILED"
            logger.info(f"  {step_name}: {status}")
            if success:
                success_count += 1

        logger.info("")
        logger.info(f"Overall: {success_count}/{len(results)} steps successful")

        if success_count == len(results):
            logger.info("✓ All backfill steps completed successfully!")
        else:
            logger.warning("⚠ Some backfill steps failed. Check logs for details.")


def main():
    """Main entry point for CLI."""
    parser = argparse.ArgumentParser(
        description='Backfill 3 years of historical data',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Backfill everything (recommended)
  python backfill_3year_data.py --all

  # Test on 100 stocks first
  python backfill_3year_data.py --all --limit 100

  # Backfill specific components
  python backfill_3year_data.py --prices --benchmark --treasury

  # Backfill only stock prices
  python backfill_3year_data.py --prices --limit 500

  # Backfill only benchmark and treasury (faster)
  python backfill_3year_data.py --benchmark --treasury

Recommended Order:
  1. Benchmark data (fast, ~1 minute)
  2. Treasury rates (fast, ~1 minute)
  3. Stock prices (slow, ~2-4 hours for all stocks)

Notes:
  - Requires POLYGON_API_KEY environment variable
  - Requires FRED_API_KEY environment variable
  - Run this script ONCE during initial setup
  - Daily CronJobs will handle incremental updates thereafter
        """
    )

    parser.add_argument(
        '--all',
        action='store_true',
        help='Backfill all data (prices, benchmark, treasury)'
    )
    parser.add_argument(
        '--prices',
        action='store_true',
        help='Backfill stock prices only'
    )
    parser.add_argument(
        '--benchmark',
        action='store_true',
        help='Backfill benchmark data only'
    )
    parser.add_argument(
        '--treasury',
        action='store_true',
        help='Backfill treasury rates only'
    )
    parser.add_argument(
        '--limit',
        type=int,
        help='Limit number of stocks for price backfill (for testing)'
    )

    args = parser.parse_args()

    # Determine pipelines directory
    pipelines_dir = Path(__file__).parent
    logger.info(f"Pipelines directory: {pipelines_dir}")

    # Run orchestrator
    orchestrator = BackfillOrchestrator(pipelines_dir)
    orchestrator.run(
        prices=args.prices,
        benchmark=args.benchmark,
        treasury=args.treasury,
        all_data=args.all,
        limit=args.limit
    )


if __name__ == '__main__':
    main()
