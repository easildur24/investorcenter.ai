#!/usr/bin/env python3
"""Data freshness and cronjob health report generator.

Connects to PostgreSQL and produces a Markdown report covering:
- CronJob execution health (from cronjob_execution_logs / schedules)
- Data freshness per pipeline output table
- Overall system health status

Usage:
    DB_HOST=localhost DB_PORT=5432 DB_USER=investorcenter \
        DB_PASSWORD=secret DB_NAME=investorcenter_db \
        python scripts/freshness_report.py

Exit codes:
    0 - healthy
    1 - critical issues detected
    2 - warnings only (no critical)
"""

import os
import re
import smtplib
import sys
from datetime import datetime, timedelta, timezone
from email.mime.multipart import MIMEMultipart
from email.mime.text import MIMEText
from typing import Any, Dict, List, Optional, Tuple

import psycopg2
from psycopg2 import sql
from psycopg2.extras import RealDictCursor

# ============================================================
# Pipeline output table mappings
# ============================================================

PIPELINE_OUTPUT_TABLE: Dict[str, str] = {
    "benchmark_data": "benchmark_returns",
    "treasury_rates": "treasury_rates",
    "sec_financials": "financials",
    "sec_13f_holdings": "institutional_holdings",
    "analyst_ratings": "analyst_ratings",
    "insider_trades": "insider_trades",
    "daily_price_update": "stock_prices",
    "news_sentiment": "news_articles",
    "reddit_sentiment": "reddit_posts",
    "ttm_financials": "ttm_financials",
    "fundamental_metrics": "fundamental_metrics_extended",
    "sector_percentiles": "fundamental_metrics_extended",
    "fair_value": "fundamental_metrics_extended",
    "technical_indicators": "technical_indicators",
    "valuation_ratios": "valuation_ratios",
    "risk_metrics": "risk_metrics",
    "ic_score_calculator": "ic_scores",
}

FRESHNESS_COLUMN: Dict[str, str] = {
    "benchmark_returns": "time",
    "treasury_rates": "date",
    "financials": "created_at",
    "institutional_holdings": "created_at",
    "analyst_ratings": "created_at",
    "insider_trades": "created_at",
    "stock_prices": "time",
    "news_articles": "created_at",
    "reddit_posts": "created_at",
    "ttm_financials": "created_at",
    "fundamental_metrics_extended": "updated_at",
    "technical_indicators": "time",
    "valuation_ratios": "created_at",
    "risk_metrics": "time",
    "ic_scores": "created_at",
}

# ============================================================
# Staleness thresholds (hours) -- override via env
# THRESHOLD_<TABLE_NAME_UPPER>=<hours>
# ============================================================

DEFAULT_THRESHOLDS: Dict[str, int] = {
    "stock_prices": 36,
    "technical_indicators": 36,
    "ic_scores": 48,
    "valuation_ratios": 48,
    "risk_metrics": 72,
    "fundamental_metrics_extended": 168,
    "treasury_rates": 48,
    "benchmark_returns": 48,
    "analyst_ratings": 48,
    "news_articles": 12,
    "insider_trades": 48,
    "financials": 336,
    "institutional_holdings": 2400,
    "ttm_financials": 168,
    "reddit_posts": 12,
}

CRITICAL_TABLES = {
    "stock_prices",
    "ic_scores",
    "technical_indicators",
    "valuation_ratios",
}


def _get_threshold_hours(table: str) -> int:
    """Return staleness threshold for a table.

    Checks for an environment variable override named
    ``THRESHOLD_<TABLE_UPPER>`` before falling back to the
    built-in default.
    """
    env_key = f"THRESHOLD_{table.upper()}"
    env_val = os.environ.get(env_key)
    if env_val is not None:
        try:
            return int(env_val)
        except ValueError:
            pass
    return DEFAULT_THRESHOLDS.get(table, 48)


# ============================================================
# Database connection
# ============================================================


def get_db_connection() -> psycopg2.extensions.connection:
    """Open a psycopg2 connection using environment variables.

    Reads ``DB_HOST``, ``DB_PORT``, ``DB_USER``, ``DB_PASSWORD``,
    ``DB_NAME``, and ``DB_SSLMODE`` from the environment.

    Returns:
        A psycopg2 connection object.
    """
    return psycopg2.connect(
        host=os.environ.get("DB_HOST", "localhost"),
        port=int(os.environ.get("DB_PORT", "5432")),
        user=os.environ.get("DB_USER", "postgres"),
        password=os.environ.get("DB_PASSWORD", ""),
        dbname=os.environ.get("DB_NAME", "investorcenter_db"),
        sslmode=os.environ.get("DB_SSLMODE", "disable"),
        cursor_factory=RealDictCursor,
    )


# ============================================================
# CronJob health
# ============================================================


def check_cronjob_health(
    conn: psycopg2.extensions.connection,
) -> List[Dict[str, Any]]:
    """Query cronjob schedules and execution logs.

    For every *active* job in ``cronjob_schedules``, returns a dict
    with keys:

    - ``job_name``
    - ``last_status``   (most recent execution status or ``"unknown"``)
    - ``last_run``      (datetime or ``None``)
    - ``consecutive_failures``
    - ``success_rate_7d`` (float 0-100 or ``None``)
    - ``schedule_cron``
    - ``last_success_at``
    - ``expected_duration_seconds``
    - ``health``        (``"healthy"`` / ``"warning"`` / ``"critical"``)
    """
    results: List[Dict[str, Any]] = []
    now = datetime.now(timezone.utc)

    with conn.cursor() as cur:
        # Fetch active schedules
        cur.execute(
            "SELECT job_name, schedule_cron, "
            "       schedule_description, "
            "       last_success_at, last_failure_at, "
            "       consecutive_failures, "
            "       expected_duration_seconds "
            "FROM cronjob_schedules "
            "WHERE is_active = true "
            "ORDER BY job_name"
        )
        schedules = cur.fetchall()

        for sched in schedules:
            job_name = sched["job_name"]
            consecutive_failures = sched["consecutive_failures"] or 0

            # Most recent execution
            cur.execute(
                "SELECT status, "
                "       COALESCE(completed_at, started_at) "
                "           AS last_run "
                "FROM cronjob_execution_logs "
                "WHERE job_name = %s "
                "ORDER BY started_at DESC LIMIT 1",
                (job_name,),
            )
            latest = cur.fetchone()
            last_status = latest["status"] if latest else "unknown"
            last_run = latest["last_run"] if latest else None

            # 7-day success rate
            cur.execute(
                "SELECT "
                "  COUNT(*) AS total, "
                "  COUNT(*) FILTER "
                "    (WHERE status = 'success') AS ok "
                "FROM cronjob_execution_logs "
                "WHERE job_name = %s "
                "  AND started_at >= %s",
                (job_name, now - timedelta(days=7)),
            )
            stats = cur.fetchone()
            total = stats["total"] if stats else 0
            ok = stats["ok"] if stats else 0
            success_rate: Optional[float] = None
            if total > 0:
                success_rate = round((ok / total) * 100, 1)

            # Determine health
            health = _job_health(
                consecutive_failures,
                success_rate,
                last_status,
                sched["last_success_at"],
                sched["schedule_cron"],
                now,
            )

            results.append(
                {
                    "job_name": job_name,
                    "last_status": last_status,
                    "last_run": last_run,
                    "consecutive_failures": consecutive_failures,
                    "success_rate_7d": success_rate,
                    "schedule_cron": sched["schedule_cron"],
                    "last_success_at": sched["last_success_at"],
                    "expected_duration_seconds": sched[
                        "expected_duration_seconds"
                    ],
                    "health": health,
                }
            )

    return results


def _job_health(
    consecutive_failures: int,
    success_rate: Optional[float],
    last_status: str,
    last_success_at: Optional[datetime],
    schedule_cron: Optional[str],
    now: datetime,
) -> str:
    """Classify a single job's health.

    Returns one of ``"healthy"``, ``"warning"``, ``"critical"``.
    """
    # Critical: 3+ consecutive failures or very low success rate
    if consecutive_failures >= 3:
        return "critical"
    if success_rate is not None and success_rate < 50:
        return "critical"

    # Detect missed schedules: if last_success_at is much older
    # than the schedule frequency implies, flag it.
    if last_success_at is not None and schedule_cron:
        max_gap = _max_expected_gap_hours(schedule_cron)
        age_hours = (now - last_success_at).total_seconds() / 3600
        if age_hours > max_gap * 3:
            return "critical"
        if age_hours > max_gap * 2:
            return "warning"

    # Warning: 1-2 consecutive failures or moderate success rate
    if consecutive_failures >= 1:
        return "warning"
    if success_rate is not None and success_rate < 80:
        return "warning"
    if last_status in ("failed", "timeout"):
        return "warning"

    return "healthy"


def _max_expected_gap_hours(cron_expr: str) -> float:
    """Estimate the maximum expected gap between runs (hours).

    A rough heuristic based on common cron patterns.  Not a full
    cron parser -- just good enough for alerting.
    """
    if not cron_expr:
        return 48.0

    parts = cron_expr.split()
    if len(parts) < 5:
        return 48.0

    minute, hour, dom, month, dow = parts[:5]

    # Hourly (e.g. "0 14-21 * * 1-5")
    if hour != "*" and "," in hour:
        hours_list = []
        for h in hour.split(","):
            try:
                hours_list.append(int(h))
            except ValueError:
                pass
        if len(hours_list) >= 2:
            return 24.0
    if hour != "*" and "-" in hour:
        return 24.0

    # Every N hours (e.g. "0 */4 * * *")
    if "/" in hour:
        try:
            interval = int(hour.split("/")[1])
            return float(interval * 2)
        except (ValueError, IndexError):
            pass

    # Weekly (dow has specific day, not *)
    if dow != "*" and dom == "*":
        return 7 * 24.0

    # Quarterly check (specific months)
    if month != "*":
        return 100 * 24.0

    # Daily
    return 24.0


# ============================================================
# Data freshness
# ============================================================


def check_data_freshness(
    conn: psycopg2.extensions.connection,
) -> List[Dict[str, Any]]:
    """Check the most recent timestamp in each pipeline output table.

    Returns a list of dicts with keys:

    - ``pipeline``
    - ``table``
    - ``latest_data``   (datetime or ``None``)
    - ``age_hours``     (float or ``None``)
    - ``threshold_hours``
    - ``is_stale``      (bool)
    - ``is_critical``   (bool -- stale AND table is in CRITICAL_TABLES)
    - ``status``        (display string)
    """
    results: List[Dict[str, Any]] = []
    now = datetime.now(timezone.utc)

    # De-duplicate: multiple pipelines can write to the same table.
    # We query each table only once and map back to pipelines.
    seen_tables: Dict[str, Optional[datetime]] = {}

    with conn.cursor() as cur:
        for pipeline, table in PIPELINE_OUTPUT_TABLE.items():
            if table not in seen_tables:
                col = FRESHNESS_COLUMN.get(table, "created_at")
                latest = _query_max_timestamp(cur, table, col)
                seen_tables[table] = latest

            latest = seen_tables[table]
            threshold = _get_threshold_hours(table)

            age_hours: Optional[float] = None
            is_stale = False

            if latest is not None:
                # Normalise to offset-aware UTC
                if latest.tzinfo is None:
                    latest = latest.replace(tzinfo=timezone.utc)
                age_hours = round((now - latest).total_seconds() / 3600, 1)
                is_stale = age_hours > threshold
            else:
                is_stale = True  # no data at all counts as stale

            is_critical = is_stale and table in CRITICAL_TABLES

            if is_critical:
                status = "CRITICAL"
            elif is_stale:
                status = "WARNING"
            else:
                status = "OK"

            results.append(
                {
                    "pipeline": pipeline,
                    "table": table,
                    "latest_data": latest,
                    "age_hours": age_hours,
                    "threshold_hours": threshold,
                    "is_stale": is_stale,
                    "is_critical": is_critical,
                    "status": status,
                }
            )

    return results


def _query_max_timestamp(
    cur: Any, table: str, column: str
) -> Optional[datetime]:
    """Run ``SELECT MAX(column) FROM table`` safely."""
    try:
        cur.execute(
            sql.SQL("SELECT MAX({}) AS latest FROM {}").format(
                sql.Identifier(column), sql.Identifier(table)
            )
        )
        row = cur.fetchone()
        if row and row["latest"] is not None:
            val = row["latest"]
            # Handle date-only columns (e.g. treasury_rates.date)
            if not isinstance(val, datetime):
                val = datetime.combine(
                    val, datetime.min.time(), tzinfo=timezone.utc
                )
            return val
    except Exception:
        # Table might not exist yet; roll back the failed txn
        cur.connection.rollback()
    return None


# ============================================================
# Severity determination
# ============================================================


def determine_severity(
    job_health: List[Dict[str, Any]],
    freshness: List[Dict[str, Any]],
) -> Tuple[int, str]:
    """Derive the overall system status.

    Args:
        job_health: Output of :func:`check_cronjob_health`.
        freshness: Output of :func:`check_data_freshness`.

    Returns:
        ``(exit_code, status_label)`` where exit_code is 0, 1, or 2
        and status_label is a human-readable badge string.
    """
    has_critical = False
    has_warning = False

    for j in job_health:
        if j["health"] == "critical":
            has_critical = True
        elif j["health"] == "warning":
            has_warning = True

    for f in freshness:
        if f["is_critical"]:
            has_critical = True
        elif f["is_stale"]:
            has_warning = True

    if has_critical:
        return 1, "CRITICAL"
    if has_warning:
        return 2, "WARNING"
    return 0, "HEALTHY"


# ============================================================
# Report formatting
# ============================================================

_HEALTH_ICON = {
    "healthy": "OK",
    "warning": "WARN",
    "critical": "FAIL",
}

_FRESHNESS_ICON = {
    "OK": "OK",
    "WARNING": "WARN",
    "CRITICAL": "CRIT",
}


def format_report(
    job_health: List[Dict[str, Any]],
    freshness: List[Dict[str, Any]],
    severity: Tuple[int, str],
) -> str:
    """Render the full Markdown report.

    Args:
        job_health: Output of :func:`check_cronjob_health`.
        freshness: Output of :func:`check_data_freshness`.
        severity: ``(exit_code, status_label)`` from
            :func:`determine_severity`.

    Returns:
        A Markdown-formatted string suitable for stdout.
    """
    now = datetime.now(timezone.utc)
    exit_code, status_label = severity
    lines: List[str] = []

    # Header
    lines.append("# Data Freshness & CronJob Health Report")
    lines.append("")
    lines.append(f"**Generated:** {now.strftime('%Y-%m-%d %H:%M:%S UTC')}")
    lines.append("")
    lines.append(f"**Overall Status:** {status_label}")
    lines.append("")

    # ----------------------------------------------------------
    # CronJob Health
    # ----------------------------------------------------------
    lines.append("## CronJob Health")
    lines.append("")
    lines.append(
        "| Job | Last Status | Last Run | "
        "Consecutive Failures | 7d Success Rate | Health |"
    )
    lines.append("| --- | --- | --- | ---: | ---: | --- |")

    for j in job_health:
        last_run_str = (
            j["last_run"].strftime("%Y-%m-%d %H:%M UTC")
            if j["last_run"]
            else "never"
        )
        rate_str = (
            f"{j['success_rate_7d']:.1f}%"
            if j["success_rate_7d"] is not None
            else "n/a"
        )
        health_str = _HEALTH_ICON.get(j["health"], j["health"])
        lines.append(
            f"| {j['job_name']} "
            f"| {j['last_status']} "
            f"| {last_run_str} "
            f"| {j['consecutive_failures']} "
            f"| {rate_str} "
            f"| {health_str} |"
        )

    lines.append("")

    # ----------------------------------------------------------
    # Data Freshness
    # ----------------------------------------------------------
    lines.append("## Data Freshness")
    lines.append("")
    lines.append(
        "| Pipeline | Table | Latest Data | "
        "Age (h) | Threshold (h) | Status |"
    )
    lines.append("| --- | --- | --- | ---: | ---: | --- |")

    for f in freshness:
        latest_str = (
            f["latest_data"].strftime("%Y-%m-%d %H:%M UTC")
            if f["latest_data"]
            else "no data"
        )
        age_str = (
            f"{f['age_hours']:.1f}" if f["age_hours"] is not None else "-"
        )
        status_icon = _FRESHNESS_ICON.get(f["status"], f["status"])
        lines.append(
            f"| {f['pipeline']} "
            f"| {f['table']} "
            f"| {latest_str} "
            f"| {age_str} "
            f"| {f['threshold_hours']} "
            f"| {status_icon} |"
        )

    lines.append("")

    # ----------------------------------------------------------
    # Issue summary
    # ----------------------------------------------------------
    issues: List[str] = []

    for j in job_health:
        if j["health"] == "critical":
            issues.append(
                f"CRITICAL: CronJob **{j['job_name']}** -- "
                f"{j['consecutive_failures']} consecutive failures, "
                f"7d success rate {j['success_rate_7d']}%"
            )
        elif j["health"] == "warning":
            issues.append(
                f"WARNING: CronJob **{j['job_name']}** -- "
                f"last status {j['last_status']}, "
                f"{j['consecutive_failures']} consecutive failure(s)"
            )

    for f in freshness:
        if f["is_critical"]:
            issues.append(
                f"CRITICAL: **{f['table']}** is stale "
                f"(age {f['age_hours']}h, "
                f"threshold {f['threshold_hours']}h)"
            )
        elif f["is_stale"]:
            issues.append(
                f"WARNING: **{f['table']}** is stale "
                f"(age {f['age_hours']}h, "
                f"threshold {f['threshold_hours']}h)"
            )

    if issues:
        lines.append("## Issues")
        lines.append("")
        for issue in issues:
            lines.append(f"- {issue}")
        lines.append("")
    else:
        lines.append("## Issues")
        lines.append("")
        lines.append("No issues detected.")
        lines.append("")

    return "\n".join(lines)


# ============================================================
# Email delivery
# ============================================================


def markdown_to_html(md: str) -> str:
    """Convert a simple Markdown report to HTML.

    Handles headings (``#``/``##``), bold (``**``), bullet lists
    (``- ``), and pipe-delimited tables.  Not a full parser —
    just enough for the report format produced by
    :func:`format_report`.
    """
    lines = md.split("\n")
    html_lines: List[str] = []
    in_table = False
    in_list = False

    for line in lines:
        stripped = line.strip()

        # Close table if we leave pipe-delimited rows
        if in_table and not stripped.startswith("|"):
            html_lines.append("</table>")
            in_table = False

        # Close list if we leave bullet items
        if in_list and not stripped.startswith("- "):
            html_lines.append("</ul>")
            in_list = False

        if stripped.startswith("## "):
            html_lines.append(f"<h2>{stripped[3:]}</h2>")
        elif stripped.startswith("# "):
            html_lines.append(f"<h1>{stripped[2:]}</h1>")
        elif stripped.startswith("| ---"):
            # Separator row — skip (already handled by <th>)
            continue
        elif stripped.startswith("|"):
            cells = [c.strip() for c in stripped.strip("|").split("|")]
            if not in_table:
                html_lines.append(
                    '<table border="1" cellpadding="6" '
                    'cellspacing="0" '
                    'style="border-collapse:collapse;'
                    'font-size:14px;">'
                )
                # First row is header
                row = "".join(f"<th>{c}</th>" for c in cells)
                html_lines.append(f"<tr>{row}</tr>")
                in_table = True
            else:
                row = "".join(f"<td>{c}</td>" for c in cells)
                html_lines.append(f"<tr>{row}</tr>")
        elif stripped.startswith("- "):
            if not in_list:
                html_lines.append("<ul>")
                in_list = True
            html_lines.append(f"<li>{stripped[2:]}</li>")
        elif stripped == "":
            html_lines.append("<br>")
        else:
            html_lines.append(f"<p>{stripped}</p>")

    if in_table:
        html_lines.append("</table>")
    if in_list:
        html_lines.append("</ul>")

    body = "\n".join(html_lines)
    # Convert **bold** to <strong>
    body = re.sub(r"\*\*(.+?)\*\*", r"<strong>\1</strong>", body)
    return body


def send_email_report(
    report_md: str,
    severity_label: str,
    to_email: str,
) -> bool:
    """Send the report via SMTP.

    Reads ``SMTP_HOST``, ``SMTP_PORT``, ``SMTP_USERNAME``,
    ``SMTP_PASSWORD``, and ``SMTP_FROM_EMAIL`` from the
    environment.  Returns ``True`` on success, ``False`` on
    failure (never raises).

    Args:
        report_md: Markdown report string.
        severity_label: One of ``HEALTHY``, ``WARNING``,
            ``CRITICAL``.
        to_email: Comma-separated recipient email addresses.
    """
    smtp_host = os.environ.get("SMTP_HOST", "")
    smtp_port = int(os.environ.get("SMTP_PORT", "587"))
    smtp_user = os.environ.get("SMTP_USERNAME", "")
    smtp_pass = os.environ.get("SMTP_PASSWORD", "")
    from_email = os.environ.get("SMTP_FROM_EMAIL", "noreply@investorcenter.ai")
    from_name = os.environ.get("SMTP_FROM_NAME", "InvestorCenter.ai")

    if not smtp_host or not smtp_pass:
        print(
            "SMTP not configured — skipping email.",
            file=sys.stderr,
        )
        return False

    # Support comma-separated recipients
    recipients = [e.strip() for e in to_email.split(",") if e.strip()]
    if not recipients:
        print("No recipients specified — skipping email.", file=sys.stderr)
        return False

    subject = f"[InvestorCenter] Data Freshness Report" f" — {severity_label}"

    html_body = markdown_to_html(report_md)
    html_body = (
        '<div style="font-family:Arial,sans-serif;">' f"{html_body}</div>"
    )

    msg = MIMEMultipart("alternative")
    msg["Subject"] = subject
    msg["From"] = f"{from_name} <{from_email}>"
    msg["To"] = ", ".join(recipients)

    # Attach both plain text and HTML
    msg.attach(MIMEText(report_md, "plain"))
    msg.attach(MIMEText(html_body, "html"))

    try:
        with smtplib.SMTP(smtp_host, smtp_port) as server:
            server.ehlo()
            server.starttls()
            server.ehlo()
            server.login(smtp_user, smtp_pass)
            server.sendmail(from_email, recipients, msg.as_string())
        print(f"Report emailed to {to_email}")
        return True
    except Exception as exc:
        print(
            f"Failed to send email: {exc}",
            file=sys.stderr,
        )
        return False


# ============================================================
# Main
# ============================================================


def main() -> int:
    """Run the freshness report and print to stdout.

    If ``SMTP_HOST`` is set, also emails the report to the
    address in ``REPORT_EMAIL_TO``.

    Returns:
        Exit code (0 = healthy, 1 = critical, 2 = warnings).
    """
    conn = get_db_connection()
    try:
        job_health = check_cronjob_health(conn)
        freshness = check_data_freshness(conn)
        severity = determine_severity(job_health, freshness)
        report = format_report(job_health, freshness, severity)
        print(report)

        # Email if configured
        to_email = os.environ.get("REPORT_EMAIL_TO", "")
        if to_email and os.environ.get("SMTP_HOST"):
            send_email_report(report, severity[1], to_email)

        return severity[0]
    finally:
        conn.close()


if __name__ == "__main__":
    sys.exit(main())
