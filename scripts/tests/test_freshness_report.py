"""Unit tests for the data freshness report script.

Mocks psycopg2 connections so no real database is needed.
"""

import os
from datetime import datetime, timedelta, timezone
from unittest.mock import MagicMock, patch

import pytest

# Import the module under test
from scripts.freshness_report import (
    CRITICAL_TABLES,
    DEFAULT_THRESHOLDS,
    _get_threshold_hours,
    _job_health,
    _max_expected_gap_hours,
    check_cronjob_health,
    check_data_freshness,
    determine_severity,
    format_report,
    main,
    markdown_to_html,
    send_email_report,
)

# ==================================================================
# Helpers
# ==================================================================

NOW = datetime(2026, 2, 18, 12, 0, 0, tzinfo=timezone.utc)


def _fresh_dt(hours_ago: float) -> datetime:
    """Return a UTC datetime *hours_ago* before NOW."""
    return NOW - timedelta(hours=hours_ago)


def _make_freshness_row(
    pipeline="daily_price_update",
    table="stock_prices",
    age_hours=10.0,
    threshold_hours=36,
    is_stale=False,
    is_critical=False,
    status="OK",
):
    return {
        "pipeline": pipeline,
        "table": table,
        "latest_data": _fresh_dt(age_hours),
        "age_hours": age_hours,
        "threshold_hours": threshold_hours,
        "is_stale": is_stale,
        "is_critical": is_critical,
        "status": status,
    }


def _make_job_row(
    job_name="ic-score-calculator",
    last_status="success",
    consecutive_failures=0,
    success_rate_7d=100.0,
    health="healthy",
):
    return {
        "job_name": job_name,
        "last_status": last_status,
        "last_run": NOW - timedelta(hours=2),
        "consecutive_failures": consecutive_failures,
        "success_rate_7d": success_rate_7d,
        "schedule_cron": "0 0 * * *",
        "last_success_at": NOW - timedelta(hours=2),
        "expected_duration_seconds": 7200,
        "health": health,
    }


# ==================================================================
# _get_threshold_hours
# ==================================================================


class TestGetThresholdHours:
    def test_default(self):
        assert _get_threshold_hours("stock_prices") == 36

    def test_default_unknown_table(self):
        assert _get_threshold_hours("nonexistent") == 48

    def test_env_override(self):
        with patch.dict(os.environ, {"THRESHOLD_STOCK_PRICES": "72"}):
            assert _get_threshold_hours("stock_prices") == 72

    def test_invalid_env_falls_back(self):
        with patch.dict(os.environ, {"THRESHOLD_IC_SCORES": "not_a_num"}):
            assert _get_threshold_hours("ic_scores") == 48


# ==================================================================
# _max_expected_gap_hours
# ==================================================================


class TestMaxExpectedGapHours:
    def test_daily(self):
        assert _max_expected_gap_hours("0 2 * * *") == 24.0

    def test_every_4h(self):
        assert _max_expected_gap_hours("0 */4 * * *") == 8.0

    def test_weekly(self):
        assert _max_expected_gap_hours("0 2 * * 0") == 168.0

    def test_hourly_range(self):
        assert _max_expected_gap_hours("0 14-21 * * 1-5") == 24.0

    def test_multiple_hours(self):
        assert _max_expected_gap_hours("0 14,18,22 * * *") == 24.0

    def test_quarterly(self):
        result = _max_expected_gap_hours("0 3 15 1,4,7,10 *")
        assert result == 2400.0

    def test_empty(self):
        assert _max_expected_gap_hours("") == 48.0

    def test_malformed(self):
        assert _max_expected_gap_hours("bad") == 48.0


# ==================================================================
# _job_health
# ==================================================================


class TestJobHealth:
    def test_healthy(self):
        result = _job_health(
            consecutive_failures=0,
            success_rate=100.0,
            last_status="success",
            last_success_at=_fresh_dt(2),
            schedule_cron="0 0 * * *",
            now=NOW,
        )
        assert result == "healthy"

    def test_critical_3_failures(self):
        result = _job_health(
            consecutive_failures=3,
            success_rate=50.0,
            last_status="failed",
            last_success_at=_fresh_dt(48),
            schedule_cron="0 0 * * *",
            now=NOW,
        )
        assert result == "critical"

    def test_critical_low_success_rate(self):
        result = _job_health(
            consecutive_failures=0,
            success_rate=30.0,
            last_status="success",
            last_success_at=_fresh_dt(2),
            schedule_cron="0 0 * * *",
            now=NOW,
        )
        assert result == "critical"

    def test_warning_1_failure(self):
        result = _job_health(
            consecutive_failures=1,
            success_rate=90.0,
            last_status="failed",
            last_success_at=_fresh_dt(2),
            schedule_cron="0 0 * * *",
            now=NOW,
        )
        assert result == "warning"

    def test_warning_last_status_failed(self):
        result = _job_health(
            consecutive_failures=0,
            success_rate=90.0,
            last_status="failed",
            last_success_at=_fresh_dt(2),
            schedule_cron="0 0 * * *",
            now=NOW,
        )
        assert result == "warning"

    def test_critical_missed_schedule(self):
        """Job hasn't succeeded in 3x expected gap."""
        result = _job_health(
            consecutive_failures=0,
            success_rate=100.0,
            last_status="success",
            # 3x 24h = 72h, so 80h ago is critical
            last_success_at=_fresh_dt(80),
            schedule_cron="0 0 * * *",
            now=NOW,
        )
        assert result == "critical"

    def test_warning_missed_schedule(self):
        """Job hasn't succeeded in 2x expected gap."""
        result = _job_health(
            consecutive_failures=0,
            success_rate=100.0,
            last_status="success",
            # 2x 24h = 48h, 50h ago is warning
            last_success_at=_fresh_dt(50),
            schedule_cron="0 0 * * *",
            now=NOW,
        )
        assert result == "warning"


# ==================================================================
# determine_severity
# ==================================================================


class TestDetermineSeverity:
    def test_healthy(self):
        jobs = [_make_job_row(health="healthy")]
        fresh = [_make_freshness_row()]
        code, label = determine_severity(jobs, fresh)
        assert code == 0
        assert label == "HEALTHY"

    def test_critical_from_stale_data(self):
        jobs = [_make_job_row(health="healthy")]
        fresh = [
            _make_freshness_row(
                table="stock_prices",
                is_stale=True,
                is_critical=True,
                status="CRITICAL",
            )
        ]
        code, label = determine_severity(jobs, fresh)
        assert code == 1
        assert label == "CRITICAL"

    def test_critical_from_job_health(self):
        jobs = [_make_job_row(health="critical", consecutive_failures=5)]
        fresh = [_make_freshness_row()]
        code, label = determine_severity(jobs, fresh)
        assert code == 1
        assert label == "CRITICAL"

    def test_warning_only(self):
        jobs = [_make_job_row(health="warning")]
        fresh = [_make_freshness_row()]
        code, label = determine_severity(jobs, fresh)
        assert code == 2
        assert label == "WARNING"

    def test_warning_from_non_critical_stale(self):
        jobs = [_make_job_row(health="healthy")]
        fresh = [
            _make_freshness_row(
                table="news_articles",
                is_stale=True,
                is_critical=False,
                status="WARNING",
            )
        ]
        code, label = determine_severity(jobs, fresh)
        assert code == 2
        assert label == "WARNING"

    def test_empty_inputs(self):
        code, label = determine_severity([], [])
        assert code == 0
        assert label == "HEALTHY"


# ==================================================================
# format_report
# ==================================================================


class TestFormatReport:
    def test_report_has_header(self):
        report = format_report([], [], (0, "HEALTHY"))
        assert "# Data Freshness & CronJob Health Report" in report

    def test_report_has_overall_status(self):
        report = format_report([], [], (1, "CRITICAL"))
        assert "CRITICAL" in report

    def test_report_has_cronjob_table(self):
        jobs = [_make_job_row()]
        report = format_report(jobs, [], (0, "HEALTHY"))
        assert "## CronJob Health" in report
        assert "ic-score-calculator" in report

    def test_report_has_freshness_table(self):
        fresh = [_make_freshness_row()]
        report = format_report([], fresh, (0, "HEALTHY"))
        assert "## Data Freshness" in report
        assert "stock_prices" in report

    def test_report_has_issues_section_when_problems(self):
        jobs = [_make_job_row(health="critical")]
        report = format_report(jobs, [], (1, "CRITICAL"))
        assert "## Issues" in report
        assert "CRITICAL" in report

    def test_report_no_issues_message(self):
        report = format_report([], [], (0, "HEALTHY"))
        assert "No issues detected." in report

    def test_report_shows_never_for_null_last_run(self):
        job = _make_job_row()
        job["last_run"] = None
        report = format_report([job], [], (0, "HEALTHY"))
        assert "never" in report

    def test_report_shows_no_data_for_null_latest(self):
        fresh = _make_freshness_row()
        fresh["latest_data"] = None
        fresh["age_hours"] = None
        report = format_report([], [fresh], (0, "HEALTHY"))
        assert "no data" in report


# ==================================================================
# check_cronjob_health (mocked DB)
# ==================================================================


class TestCheckCronjobHealth:
    def _mock_conn(self, schedules, latest_exec, stats):
        """Create a mock connection with cursor responses."""
        conn = MagicMock()
        cur = MagicMock()
        conn.cursor.return_value.__enter__ = MagicMock(return_value=cur)
        conn.cursor.return_value.__exit__ = MagicMock(return_value=False)

        # fetchall for schedules, then fetchone x2 per schedule
        call_results = [schedules]
        for i in range(len(schedules)):
            call_results.append(latest_exec[i])  # fetchone
            call_results.append(stats[i])  # fetchone

        side_effects = iter(call_results)

        def _fetch_dispatch(*args, **kwargs):
            return next(side_effects)

        cur.fetchall = MagicMock(side_effect=lambda: next(side_effects))
        cur.fetchone = MagicMock(side_effect=lambda: next(side_effects))

        # Override: fetchall called once, fetchone multiple times
        call_idx = {"i": 0}
        all_calls = call_results

        def _execute(*args, **kwargs):
            pass

        cur.execute = _execute

        # Simpler approach: chain fetchall + fetchone
        cur.fetchall.side_effect = None
        cur.fetchall.return_value = schedules

        fetchone_returns = []
        for i in range(len(schedules)):
            fetchone_returns.append(latest_exec[i])
            fetchone_returns.append(stats[i])
        cur.fetchone.side_effect = fetchone_returns

        return conn

    def test_healthy_job(self):
        schedules = [
            {
                "job_name": "test-job",
                "schedule_cron": "0 2 * * *",
                "schedule_description": "Daily",
                "last_success_at": _fresh_dt(2),
                "last_failure_at": None,
                "consecutive_failures": 0,
                "expected_duration_seconds": 300,
            }
        ]
        latest = {"status": "success", "last_run": _fresh_dt(2)}
        stats = {"total": 7, "ok": 7}

        conn = self._mock_conn(schedules, [latest], [stats])
        with patch("scripts.freshness_report.datetime") as mock_dt:
            mock_dt.now.return_value = NOW
            mock_dt.side_effect = lambda *a, **kw: datetime(*a, **kw)
            result = check_cronjob_health(conn)

        assert len(result) == 1
        assert result[0]["job_name"] == "test-job"
        assert result[0]["health"] == "healthy"

    def test_critical_job(self):
        schedules = [
            {
                "job_name": "failing-job",
                "schedule_cron": "0 2 * * *",
                "schedule_description": "Daily",
                "last_success_at": _fresh_dt(96),
                "last_failure_at": _fresh_dt(1),
                "consecutive_failures": 4,
                "expected_duration_seconds": 300,
            }
        ]
        latest = {"status": "failed", "last_run": _fresh_dt(1)}
        stats = {"total": 7, "ok": 2}

        conn = self._mock_conn(schedules, [latest], [stats])
        with patch("scripts.freshness_report.datetime") as mock_dt:
            mock_dt.now.return_value = NOW
            mock_dt.side_effect = lambda *a, **kw: datetime(*a, **kw)
            result = check_cronjob_health(conn)

        assert len(result) == 1
        assert result[0]["health"] == "critical"


# ==================================================================
# check_data_freshness (mocked DB)
# ==================================================================


class TestCheckDataFreshness:
    def _mock_conn(self, table_data):
        """table_data: dict of table -> latest datetime or None.

        The mock cursor's execute method extracts the table name
        from the SQL ``FROM <table>`` clause and stores it so
        fetchone returns the matching datetime.
        """
        conn = MagicMock()
        cur = MagicMock()
        conn.cursor.return_value.__enter__ = MagicMock(return_value=cur)
        conn.cursor.return_value.__exit__ = MagicMock(return_value=False)

        def _execute(query, *args):
            # Extract table name from "FROM <table>"
            cur._current_table = None
            for tbl in table_data:
                if f"FROM {tbl}" in query:
                    cur._current_table = tbl
                    return
            # Table not in our data — simulate missing table
            # by raising (which triggers rollback in the code)
            raise Exception(f"relation does not exist")

        cur.execute = _execute
        cur.connection = conn

        def _fetchone():
            tbl = getattr(cur, "_current_table", None)
            if tbl and tbl in table_data:
                return {"latest": table_data[tbl]}
            return {"latest": None}

        cur.fetchone = _fetchone

        return conn

    def test_all_fresh(self):
        """When all tables have recent data, none should be stale.

        Uses real datetimes relative to now (no datetime mock
        needed) to avoid breaking isinstance checks in the
        production code.
        """
        now = datetime.now(timezone.utc)
        table_data = {
            "stock_prices": now - timedelta(hours=10),
            "technical_indicators": now - timedelta(hours=10),
            "ic_scores": now - timedelta(hours=10),
            "valuation_ratios": now - timedelta(hours=10),
            "risk_metrics": now - timedelta(hours=10),
            "fundamental_metrics_extended": now - timedelta(hours=10),
            "treasury_rates": now - timedelta(hours=10),
            "benchmark_returns": now - timedelta(hours=10),
            "analyst_ratings": now - timedelta(hours=10),
            "news_articles": now - timedelta(hours=5),
            "insider_trades": now - timedelta(hours=10),
            "financials": now - timedelta(hours=10),
            "institutional_holdings": now - timedelta(hours=10),
            "ttm_financials": now - timedelta(hours=10),
            "reddit_posts": now - timedelta(hours=5),
        }

        conn = self._mock_conn(table_data)
        result = check_data_freshness(conn)

        stale = [r for r in result if r["is_stale"]]
        assert len(stale) == 0

    def test_stale_critical_table(self):
        """stock_prices older than 36h should be flagged CRITICAL."""
        now = datetime.now(timezone.utc)
        table_data = {
            "stock_prices": now - timedelta(hours=50),
        }

        conn = self._mock_conn(table_data)
        result = check_data_freshness(conn)

        # Find the stock_prices entry
        sp = [r for r in result if r["table"] == "stock_prices"]
        assert len(sp) == 1
        assert sp[0]["is_stale"] is True
        assert sp[0]["is_critical"] is True
        assert sp[0]["status"] == "CRITICAL"

    def test_missing_table_treated_as_stale(self):
        """Tables not in the DB should be flagged as stale."""
        conn = self._mock_conn({})  # no tables exist
        result = check_data_freshness(conn)

        # All should be stale (missing table exception -> None)
        for r in result:
            assert r["is_stale"] is True


# ==================================================================
# main (integration-level mock)
# ==================================================================


class TestMain:
    @patch("scripts.freshness_report.get_db_connection")
    @patch("scripts.freshness_report.check_data_freshness")
    @patch("scripts.freshness_report.check_cronjob_health")
    def test_healthy_exit_0(self, mock_jobs, mock_fresh, mock_conn):
        mock_conn.return_value = MagicMock()
        mock_jobs.return_value = [_make_job_row(health="healthy")]
        mock_fresh.return_value = [_make_freshness_row()]

        code = main()
        assert code == 0

    @patch("scripts.freshness_report.get_db_connection")
    @patch("scripts.freshness_report.check_data_freshness")
    @patch("scripts.freshness_report.check_cronjob_health")
    def test_critical_exit_1(self, mock_jobs, mock_fresh, mock_conn):
        mock_conn.return_value = MagicMock()
        mock_jobs.return_value = [_make_job_row(health="critical")]
        mock_fresh.return_value = [_make_freshness_row()]

        code = main()
        assert code == 1

    @patch("scripts.freshness_report.get_db_connection")
    @patch("scripts.freshness_report.check_data_freshness")
    @patch("scripts.freshness_report.check_cronjob_health")
    def test_warning_exit_2(self, mock_jobs, mock_fresh, mock_conn):
        mock_conn.return_value = MagicMock()
        mock_jobs.return_value = [_make_job_row(health="warning")]
        mock_fresh.return_value = [_make_freshness_row()]

        code = main()
        assert code == 2


# ==================================================================
# markdown_to_html
# ==================================================================


class TestMarkdownToHtml:
    def test_heading_conversion(self):
        html = markdown_to_html("# Title\n\n## Section")
        assert "<h1>Title</h1>" in html
        assert "<h2>Section</h2>" in html

    def test_bold_conversion(self):
        html = markdown_to_html("**bold text**")
        assert "<strong>bold text</strong>" in html

    def test_table_conversion(self):
        md = "| A | B |\n" "| --- | --- |\n" "| 1 | 2 |"
        html = markdown_to_html(md)
        assert "<table" in html
        assert "<th>A</th>" in html
        assert "<td>1</td>" in html

    def test_bullet_list(self):
        html = markdown_to_html("- item one\n- item two")
        assert "<li>item one</li>" in html
        assert "<li>item two</li>" in html


# ==================================================================
# send_email_report
# ==================================================================


class TestSendEmailReport:
    @patch("scripts.freshness_report.smtplib.SMTP")
    def test_email_sent_when_smtp_configured(self, mock_smtp):
        """Email is sent when SMTP env vars are set."""
        mock_server = MagicMock()
        mock_smtp.return_value.__enter__ = MagicMock(return_value=mock_server)
        mock_smtp.return_value.__exit__ = MagicMock(return_value=False)

        env = {
            "SMTP_HOST": "smtp.test.com",
            "SMTP_PORT": "587",
            "SMTP_USERNAME": "user",
            "SMTP_PASSWORD": "pass",
            "SMTP_FROM_EMAIL": "test@test.com",
        }
        with patch.dict(os.environ, env):
            result = send_email_report("# Report", "HEALTHY", "to@test.com")

        assert result is True
        mock_smtp.assert_called_once_with("smtp.test.com", 587)
        mock_server.sendmail.assert_called_once()

    def test_email_skipped_when_no_smtp(self):
        """Email is skipped when SMTP_HOST is not set."""
        env = {"SMTP_HOST": "", "SMTP_PASSWORD": ""}
        with patch.dict(os.environ, env, clear=False):
            result = send_email_report("# Report", "HEALTHY", "to@test.com")

        assert result is False

    @patch("scripts.freshness_report.smtplib.SMTP")
    def test_email_subject_contains_severity(self, mock_smtp):
        """Email subject includes the severity label."""
        mock_server = MagicMock()
        mock_smtp.return_value.__enter__ = MagicMock(return_value=mock_server)
        mock_smtp.return_value.__exit__ = MagicMock(return_value=False)

        env = {
            "SMTP_HOST": "smtp.test.com",
            "SMTP_PORT": "587",
            "SMTP_USERNAME": "user",
            "SMTP_PASSWORD": "pass",
        }
        with patch.dict(os.environ, env):
            send_email_report("# Report", "CRITICAL", "to@test.com")

        # Extract the sent message
        call_args = mock_server.sendmail.call_args
        msg_str = call_args[0][2]  # third arg is the message
        assert "CRITICAL" in msg_str

    @patch("scripts.freshness_report.smtplib.SMTP")
    def test_email_failure_returns_false(self, mock_smtp):
        """SMTP errors return False, don't raise."""
        mock_smtp.side_effect = Exception("connection refused")

        env = {
            "SMTP_HOST": "smtp.test.com",
            "SMTP_PORT": "587",
            "SMTP_USERNAME": "user",
            "SMTP_PASSWORD": "pass",
        }
        with patch.dict(os.environ, env):
            result = send_email_report("# Report", "HEALTHY", "to@test.com")

        assert result is False
