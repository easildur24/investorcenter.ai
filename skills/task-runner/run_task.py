#!/usr/bin/env python3
"""
Task Runner - Grab next task, execute it, exit.

Called by OpenClaw agent when skill is invoked.
No loops, no scheduling - just one task per run.
"""
import json
import sys
import os
import requests
from datetime import datetime
from pathlib import Path

# Add parse helpers from scrape-ycharts-keystats skill
SKILL_DIR = Path(__file__).parent.parent / "scrape-ycharts-keystats"
sys.path.insert(0, str(SKILL_DIR))
from parse_helpers import parse_dollar_amount, parse_percentage, parse_float, parse_integer, parse_ycharts_date

# Configuration
API_BASE = os.getenv("API_BASE_URL", "https://investorcenter.ai/api/v1")
WORKER_EMAIL = os.getenv("WORKER_EMAIL", "nikola@investorcenter.ai")
WORKER_PASSWORD = os.getenv("WORKER_PASSWORD", "ziyj9VNdHH5tjqB2m3lup3MG")


def log(msg):
    """Print timestamped log message."""
    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    print(f"[{timestamp}] {msg}", flush=True)


def get_auth_token():
    """Get fresh auth token."""
    try:
        resp = requests.post(
            f"{API_BASE}/auth/login",
            json={"email": WORKER_EMAIL, "password": WORKER_PASSWORD},
            timeout=10
        )
        resp.raise_for_status()
        return resp.json()["access_token"]
    except Exception as e:
        log(f"❌ Auth failed: {e}")
        sys.exit(1)


def claim_next_task(token):
    """Claim next pending task from queue."""
    try:
        resp = requests.post(
            f"{API_BASE}/tasks/next",
            headers={"Authorization": f"Bearer {token}"},
            timeout=10
        )
        if resp.status_code == 204:
            return None  # Queue empty
        resp.raise_for_status()
        data = resp.json()
        if not data.get("success"):
            return None
        return data["data"]
    except Exception as e:
        log(f"❌ Failed to claim task: {e}")
        sys.exit(1)


def mark_task_status(task_id, token, status, error=None):
    """Mark task as completed or failed."""
    payload = {"status": status}
    if error:
        payload["error"] = str(error)[:500]  # Limit error message length
    
    try:
        resp = requests.put(
            f"{API_BASE}/tasks/{task_id}",
            headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"},
            json=payload,
            timeout=10
        )
        resp.raise_for_status()
        return True
    except Exception as e:
        log(f"⚠️  Failed to mark task {status}: {e}")
        return False


def extract_metrics_from_snapshot(snapshot_text):
    """
    Extract YCharts key stats metrics from browser snapshot.
    Returns dict mapping JSON field names to parsed values.
    """
    import re
    
    def extract_row_value(metric_name):
        """Extract value from snapshot row like: row 'Revenue (TTM) 403.06B'"""
        pattern = rf'row "[^"]*{re.escape(metric_name)}\s+([^"]+)"'
        match = re.search(pattern, snapshot_text)
        if match:
            return match.group(1).strip()
        return None
    
    metrics = {}
    
    # Income Statement
    metrics["revenue"] = parse_dollar_amount(extract_row_value("Revenue (TTM)") or "")
    metrics["net_income"] = parse_dollar_amount(extract_row_value("Net Income (TTM)") or "")
    metrics["ebit"] = parse_dollar_amount(extract_row_value("EBIT (TTM)") or "")
    metrics["ebitda"] = parse_dollar_amount(extract_row_value("EBITDA (TTM)") or "")
    metrics["revenue_quarterly"] = parse_dollar_amount(extract_row_value("Revenue (Quarterly)") or "")
    metrics["net_income_quarterly"] = parse_dollar_amount(extract_row_value("Net Income (Quarterly)") or "")
    metrics["ebit_quarterly"] = parse_dollar_amount(extract_row_value("EBIT (Quarterly)") or "")
    metrics["ebitda_quarterly"] = parse_dollar_amount(extract_row_value("EBITDA (Quarterly)") or "")
    metrics["revenue_growth_quarterly_yoy"] = parse_percentage(extract_row_value("Revenue (Quarterly YoY Growth)") or "")
    metrics["eps_diluted_growth_quarterly_yoy"] = parse_percentage(extract_row_value("EPS Diluted (Quarterly YoY Growth)") or "")
    metrics["ebitda_growth_quarterly_yoy"] = parse_percentage(extract_row_value("EBITDA (Quarterly YoY Growth)") or "")
    
    # Common Size
    metrics["eps_diluted"] = parse_float(extract_row_value("EPS Diluted (TTM)") or "")
    metrics["eps_basic"] = parse_float(extract_row_value("EPS Basic (TTM)") or "")
    metrics["shares_outstanding"] = parse_dollar_amount(extract_row_value("Shares Outstanding") or "")
    
    # Balance Sheet
    metrics["total_assets"] = parse_dollar_amount(extract_row_value("Total Assets (Quarterly)") or "")
    metrics["total_liabilities"] = parse_dollar_amount(extract_row_value("Total Liabilities (Quarterly)") or "")
    metrics["shareholders_equity"] = parse_dollar_amount(extract_row_value("Shareholders Equity (Quarterly)") or "")
    metrics["cash_and_short_term_investments"] = parse_dollar_amount(extract_row_value("Cash and Short Term Investments (Quarterly)") or "")
    metrics["total_long_term_assets"] = parse_dollar_amount(extract_row_value("Total Long Term Assets (Quarterly)") or "")
    metrics["total_long_term_debt"] = parse_dollar_amount(extract_row_value("Total Long Term Debt (Quarterly)") or "")
    metrics["book_value"] = parse_dollar_amount(extract_row_value("Book Value (Quarterly)") or "")
    
    # Earnings Quality
    metrics["return_on_assets"] = parse_percentage(extract_row_value("Return on Assets") or "")
    metrics["return_on_equity"] = parse_percentage(extract_row_value("Return on Equity") or "")
    metrics["return_on_invested_capital"] = parse_percentage(extract_row_value("Return on Invested Capital") or "")
    
    # Cash Flow
    metrics["cash_from_operations"] = parse_dollar_amount(extract_row_value("Cash from Operations (TTM)") or "")
    metrics["cash_from_investing"] = parse_dollar_amount(extract_row_value("Cash from Investing (TTM)") or "")
    metrics["cash_from_financing"] = parse_dollar_amount(extract_row_value("Cash from Financing (TTM)") or "")
    metrics["change_in_receivables"] = parse_dollar_amount(extract_row_value("Change in Receivables (TTM)") or "")
    metrics["changes_in_working_capital"] = parse_dollar_amount(extract_row_value("Changes in Working Capital (TTM)") or "")
    metrics["capital_expenditures"] = parse_dollar_amount(extract_row_value("Capital Expenditures (TTM)") or "")
    metrics["ending_cash"] = parse_dollar_amount(extract_row_value("Ending Cash (Quarterly)") or "")
    metrics["free_cash_flow"] = parse_dollar_amount(extract_row_value("Free Cash Flow") or "")
    
    # Profitability
    metrics["operating_margin"] = parse_percentage(extract_row_value("Operating Margin (TTM)") or "")
    metrics["gross_profit_margin"] = parse_percentage(extract_row_value("Gross Profit Margin") or "")
    
    # Stock Performance
    metrics["one_month_total_return"] = parse_percentage(extract_row_value("1 Month Total Returns (Daily)") or "")
    metrics["three_month_total_return"] = parse_percentage(extract_row_value("3 Month Total Returns (Daily)") or "")
    metrics["six_month_total_return"] = parse_percentage(extract_row_value("6 Month Total Returns (Daily)") or "")
    metrics["ytd_total_return"] = parse_percentage(extract_row_value("Year to Date Total Returns (Daily)") or "")
    metrics["one_year_total_return"] = parse_percentage(extract_row_value("1 Year Total Returns (Daily)") or "")
    metrics["three_year_total_return_annualized"] = parse_percentage(extract_row_value("Annualized 3 Year Total Returns (Daily)") or "")
    metrics["five_year_total_return_annualized"] = parse_percentage(extract_row_value("Annualized 5 Year Total Returns (Daily)") or "")
    metrics["ten_year_total_return_annualized"] = parse_percentage(extract_row_value("Annualized 10 Year Total Returns (Daily)") or "")
    metrics["fifteen_year_total_return_annualized"] = parse_percentage(extract_row_value("Annualized 15 Year Total Returns (Daily)") or "")
    metrics["since_inception_total_return_annualized"] = parse_percentage(extract_row_value("Annualized Total Returns Since Inception (Daily)") or "")
    metrics["fifty_two_week_high"] = parse_float(extract_row_value("52 Week High (Daily)") or "")
    metrics["fifty_two_week_low"] = parse_float(extract_row_value("52 Week Low (Daily)") or "")
    
    # Dates
    high_date = extract_row_value("52-Week High Date")
    low_date = extract_row_value("52-Week Low Date")
    metrics["fifty_two_week_high_date"] = parse_ycharts_date(high_date) if high_date else None
    metrics["fifty_two_week_low_date"] = parse_ycharts_date(low_date) if low_date else None
    
    # Estimates
    metrics["revenue_estimates_current_quarter"] = parse_dollar_amount(extract_row_value("Revenue Estimates for Current Quarter") or "")
    metrics["revenue_estimates_next_quarter"] = parse_dollar_amount(extract_row_value("Revenue Estimates for Next Quarter") or "")
    metrics["revenue_estimates_current_year"] = parse_dollar_amount(extract_row_value("Revenue Estimates for Current Fiscal Year") or "")
    metrics["revenue_estimates_next_year"] = parse_dollar_amount(extract_row_value("Revenue Estimates for Next Fiscal Year") or "")
    metrics["eps_estimates_current_quarter"] = parse_float(extract_row_value("EPS Estimates for Current Quarter") or "")
    metrics["eps_estimates_next_quarter"] = parse_float(extract_row_value("EPS Estimates for Next Quarter") or "")
    metrics["eps_estimates_current_year"] = parse_float(extract_row_value("EPS Estimates for Current Fiscal Year") or "")
    metrics["eps_estimates_next_year"] = parse_float(extract_row_value("EPS Estimates for Next Fiscal Year") or "")
    
    # Dividends
    metrics["dividend_yield"] = parse_percentage(extract_row_value("Dividend Yield") or "")
    metrics["dividend_yield_forward"] = parse_percentage(extract_row_value("Dividend Yield (Forward)") or "")
    metrics["payout_ratio"] = parse_percentage(extract_row_value("Payout Ratio (TTM)") or "")
    metrics["cash_dividend_payout_ratio"] = parse_percentage(extract_row_value("Cash Dividend Payout Ratio") or "")
    metrics["last_dividend_amount"] = parse_float(extract_row_value("Last Dividend Amount") or "")
    ex_div_date = extract_row_value("Last Ex-Dividend Date")
    metrics["last_ex_dividend_date"] = parse_ycharts_date(ex_div_date) if ex_div_date else None
    
    # Management
    metrics["asset_utilization"] = parse_float(extract_row_value("Asset Utilization (TTM)") or "")
    metrics["days_sales_outstanding"] = parse_float(extract_row_value("Days Sales Outstanding (Quarterly)") or "")
    metrics["days_inventory_outstanding"] = parse_float(extract_row_value("Days Inventory Outstanding (Quarterly)") or "")
    metrics["days_payable_outstanding"] = parse_float(extract_row_value("Days Payable Outstanding (Quarterly)") or "")
    metrics["total_receivables"] = parse_dollar_amount(extract_row_value("Total Receivables (Quarterly)") or "")
    
    # Valuation
    metrics["market_cap"] = parse_dollar_amount(extract_row_value("Market Cap") or "")
    metrics["enterprise_value"] = parse_dollar_amount(extract_row_value("Enterprise Value") or "")
    metrics["price"] = parse_float(extract_row_value("Price") or "")
    metrics["pe_ratio"] = parse_float(extract_row_value("PE Ratio") or "")
    metrics["pe_ratio_forward"] = parse_float(extract_row_value("PE Ratio (Forward)") or "")
    metrics["pe_ratio_forward_1y"] = parse_float(extract_row_value("PE Ratio (Forward 1y)") or "")
    metrics["ps_ratio"] = parse_float(extract_row_value("PS Ratio") or "")
    metrics["ps_ratio_forward"] = parse_float(extract_row_value("PS Ratio (Forward)") or "")
    metrics["ps_ratio_forward_1y"] = parse_float(extract_row_value("PS Ratio (Forward 1y)") or "")
    metrics["price_to_book_value"] = parse_float(extract_row_value("Price to Book Value") or "")
    metrics["price_to_free_cash_flow"] = parse_float(extract_row_value("Price to Free Cash Flow") or "")
    metrics["peg_ratio"] = parse_float(extract_row_value("PEG Ratio") or "")
    metrics["ev_to_ebitda"] = parse_float(extract_row_value("EV to EBITDA") or "")
    metrics["ev_to_ebitda_forward"] = parse_float(extract_row_value("EV to EBITDA (Forward)") or "")
    metrics["ev_to_ebit"] = parse_float(extract_row_value("EV to EBIT") or "")
    metrics["ebit_margin"] = parse_percentage(extract_row_value("EBIT Margin (TTM)") or "")
    
    # Risk
    metrics["alpha_5y"] = parse_float(extract_row_value("Alpha (5Y)") or "")
    metrics["beta_5y"] = parse_float(extract_row_value("Beta (5Y)") or "")
    metrics["standard_deviation_monthly_5y"] = parse_percentage(extract_row_value("Annualized Standard Deviation of Monthly Returns (5Y Lookback)") or "")
    metrics["sharpe_ratio_5y"] = parse_float(extract_row_value("Historical Sharpe Ratio (5Y)") or "")
    metrics["sortino_ratio_5y"] = parse_float(extract_row_value("Historical Sortino (5Y)") or "")
    metrics["max_drawdown_5y"] = parse_percentage(extract_row_value("Max Drawdown (5Y)") or "")
    metrics["value_at_risk_monthly_5y"] = parse_percentage(extract_row_value("Monthly Value at Risk (VaR) 5% (5Y Lookback)") or "")
    
    # Advanced
    metrics["piotroski_f_score"] = parse_float(extract_row_value("Piotroski F Score (TTM)") or "")
    metrics["sustainable_growth_rate"] = parse_percentage(extract_row_value("Sustainable Growth Rate (TTM)") or "")
    metrics["tobin_q"] = parse_float(extract_row_value("Tobin's Q (Approximate) (Quarterly)") or "")
    metrics["momentum_score"] = parse_float(extract_row_value("Momentum Score") or "")
    metrics["market_cap_score"] = parse_float(extract_row_value("Market Cap Score") or "")
    metrics["quality_ratio_score"] = parse_float(extract_row_value("Quality Ratio Score") or "")
    
    # Liquidity
    metrics["debt_to_equity_ratio"] = parse_float(extract_row_value("Debt to Equity Ratio") or "")
    metrics["free_cash_flow_quarterly"] = parse_dollar_amount(extract_row_value("Free Cash Flow (Quarterly)") or "")
    metrics["current_ratio"] = parse_float(extract_row_value("Current Ratio") or "")
    metrics["quick_ratio"] = parse_float(extract_row_value("Quick Ratio (Quarterly)") or "")
    metrics["altman_z_score"] = parse_float(extract_row_value("Altman Z-Score (TTM)") or "")
    metrics["times_interest_earned"] = parse_float(extract_row_value("Times Interest Earned (TTM)") or "")
    
    # Employees
    metrics["total_employees"] = parse_integer(extract_row_value("Total Employees (Annual)") or "")
    metrics["revenue_per_employee"] = parse_float(extract_row_value("Revenue Per Employee (Annual)") or "")
    metrics["net_income_per_employee"] = parse_float(extract_row_value("Net Income Per Employee (Annual)") or "")
    
    return metrics


def execute_ycharts_task(task, token):
    """
    Execute a YCharts key stats scraping task.
    NOTE: Browser automation requires OpenClaw environment.
    """
    ticker = task.get("params", {}).get("ticker")
    if not ticker:
        raise ValueError("Task missing ticker parameter")
    
    log(f"🏀 Processing {ticker}...")
    
    # This is a placeholder - actual implementation needs OpenClaw browser tool
    # The agent invoking this skill will handle browser automation
    raise NotImplementedError(
        "This script is a template. Actual execution requires OpenClaw agent context with browser access."
    )


def main():
    """Main execution: claim task, run it, exit."""
    log("🏀 Task Runner starting")
    
    # Get auth token
    token = get_auth_token()
    log("✅ Authenticated")
    
    # Claim next task
    task = claim_next_task(token)
    if not task:
        log("ℹ️  No tasks in queue")
        sys.exit(0)
    
    task_id = task["id"]
    task_type = task.get("type", "unknown")
    ticker = task.get("params", {}).get("ticker", "UNKNOWN")
    
    log(f"📋 Claimed task {task_id}: {task_type} ({ticker})")
    
    # Execute task
    try:
        if task_type == "ycharts_key_stats":
            execute_ycharts_task(task, token)
            mark_task_status(task_id, token, "completed")
            log(f"✅ Task {task_id} completed: {ticker}")
        else:
            raise ValueError(f"Unknown task type: {task_type}")
    
    except Exception as e:
        error_msg = str(e)
        log(f"❌ Task {task_id} failed: {error_msg}")
        mark_task_status(task_id, token, "failed", error=error_msg)
        sys.exit(1)
    
    sys.exit(0)


if __name__ == "__main__":
    main()
