"""Unit tests for SEC EDGAR API client.

Tests cover all public and private methods of SECClient and AsyncSECClient:
- __init__: user agent configuration, session creation
- _create_session: retry strategy, headers
- _rate_limit: sleep enforcement when requests are too fast
- fetch_company_facts: success, HTTP 404, HTTP 500, timeout, bad JSON, invalid CIK
- parse_financial_data: empty/missing facts, period grouping
- _parse_fiscal_period: fiscal period string parsing
- _correct_fiscal_years: fiscal year correction for non-calendar FYE
- _correct_shares_outstanding: heuristic correction for shares in millions
- _calculate_metrics: derived financial metric calculations, revenue synthesis
- get_financials_for_ticker: end-to-end orchestration
- close: session cleanup
- AsyncSECClient: async fetch, rate limiting, error handling
"""

import asyncio
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
import requests

from pipelines.utils.sec_client import AsyncSECClient, SECClient


# =====================================================================
# Fixtures
# =====================================================================


@pytest.fixture
def client():
    """Create a SECClient instance with mocked session."""
    with patch.object(SECClient, "_create_session") as mock_create:
        mock_session = MagicMock(spec=requests.Session)
        mock_create.return_value = mock_session
        c = SECClient()
    return c


def _mock_response(status_code=200, json_data=None, raise_for_status=None):
    """Helper to create a mock response object."""
    mock_resp = MagicMock()
    mock_resp.status_code = status_code
    mock_resp.json.return_value = json_data or {}

    if raise_for_status:
        mock_resp.raise_for_status.side_effect = raise_for_status
    else:
        mock_resp.raise_for_status.return_value = None

    return mock_resp


def _make_company_facts(revenue_val=100_000_000_000, net_income_val=20_000_000_000,
                        form="10-K", period_end="2024-12-31", filed="2025-02-15",
                        fy=2024, fp="FY"):
    """Helper to create a minimal company_facts structure for testing."""
    facts = {
        "facts": {
            "us-gaap": {}
        }
    }
    us_gaap = facts["facts"]["us-gaap"]

    if revenue_val is not None:
        us_gaap["Revenues"] = {
            "units": {
                "USD": [
                    {
                        "form": form,
                        "end": period_end,
                        "filed": filed,
                        "fy": fy,
                        "fp": fp,
                        "val": revenue_val,
                    }
                ]
            }
        }

    if net_income_val is not None:
        us_gaap["NetIncomeLoss"] = {
            "units": {
                "USD": [
                    {
                        "form": form,
                        "end": period_end,
                        "filed": filed,
                        "fy": fy,
                        "fp": fp,
                        "val": net_income_val,
                    }
                ]
            }
        }

    return facts


# =====================================================================
# TestInit
# =====================================================================


class TestInit:
    """Tests for SECClient initialization."""

    def test_init_default_user_agent(self):
        """Client uses default user agent when none specified."""
        with patch.object(SECClient, "_create_session"):
            client = SECClient()
        assert client.user_agent == "InvestorCenter.ai admin@investorcenter.ai"

    def test_init_custom_user_agent(self):
        """Client accepts a custom user agent string."""
        with patch.object(SECClient, "_create_session"):
            client = SECClient(user_agent="MyApp dev@example.com")
        assert client.user_agent == "MyApp dev@example.com"

    def test_init_sets_last_request_time_to_zero(self, client):
        """Initial last_request_time is 0.0."""
        assert client.last_request_time == 0.0

    def test_init_creates_session(self, client):
        """Session is created during initialization."""
        assert client.session is not None


# =====================================================================
# TestCreateSession
# =====================================================================


class TestCreateSession:
    """Tests for _create_session (real session creation)."""

    def test_creates_real_session(self):
        """_create_session returns a requests.Session with proper headers."""
        client = SECClient()
        assert isinstance(client.session, requests.Session)
        assert client.session.headers["User-Agent"] == "InvestorCenter.ai admin@investorcenter.ai"
        assert client.session.headers["Accept"] == "application/json"
        client.close()

    def test_creates_session_with_custom_user_agent(self):
        """Session headers use custom user agent when specified."""
        client = SECClient(user_agent="TestAgent test@test.com")
        assert client.session.headers["User-Agent"] == "TestAgent test@test.com"
        client.close()

    def test_session_has_retry_adapter(self):
        """Session mounts retry adapters for http and https."""
        client = SECClient()
        # Check that adapters are mounted
        assert "https://" in client.session.adapters
        assert "http://" in client.session.adapters
        client.close()


# =====================================================================
# TestRateLimit
# =====================================================================


class TestRateLimit:
    """Tests for _rate_limit enforcement."""

    @patch("pipelines.utils.sec_client.time")
    def test_rate_limit_sleeps_when_too_fast(self, mock_time, client):
        """Rate limiter sleeps when requests are faster than MIN_REQUEST_INTERVAL."""
        # First call at time 1.0, second call at 1.02 (only 0.02s later)
        mock_time.time.side_effect = [1.02, 1.12]
        client.last_request_time = 1.0

        client._rate_limit()

        # Should have slept: MIN_REQUEST_INTERVAL - 0.02 = 0.1 - 0.02 = 0.08
        mock_time.sleep.assert_called_once()
        sleep_arg = mock_time.sleep.call_args[0][0]
        assert abs(sleep_arg - 0.08) < 0.01

    @patch("pipelines.utils.sec_client.time")
    def test_rate_limit_no_sleep_when_enough_time_elapsed(self, mock_time, client):
        """Rate limiter does not sleep when enough time has passed."""
        mock_time.time.side_effect = [2.0, 2.0]
        client.last_request_time = 1.0  # 1.0 second ago, well beyond MIN_REQUEST_INTERVAL

        client._rate_limit()

        mock_time.sleep.assert_not_called()

    @patch("pipelines.utils.sec_client.time")
    def test_rate_limit_updates_last_request_time(self, mock_time, client):
        """Rate limiter updates last_request_time after each call."""
        mock_time.time.side_effect = [5.0, 5.0]
        client.last_request_time = 0.0

        client._rate_limit()

        assert client.last_request_time == 5.0


# =====================================================================
# TestFetchCompanyFacts
# =====================================================================


class TestFetchCompanyFacts:
    """Tests for fetch_company_facts."""

    @patch("pipelines.utils.sec_client.time")
    def test_success(self, mock_time, client):
        """Successful request returns parsed JSON."""
        mock_time.time.return_value = 100.0
        expected_data = {"facts": {"us-gaap": {"Revenues": {}}}}
        mock_resp = _mock_response(json_data=expected_data)
        client.session.get.return_value = mock_resp

        result = client.fetch_company_facts("320193")

        assert result == expected_data
        client.session.get.assert_called_once()

    @patch("pipelines.utils.sec_client.time")
    def test_cik_zero_padding(self, mock_time, client):
        """CIK is zero-padded to 10 digits in URL."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"facts": {}})
        client.session.get.return_value = mock_resp

        client.fetch_company_facts("320193")

        call_args = client.session.get.call_args
        url = call_args[0][0]
        assert "CIK0000320193.json" in url

    @patch("pipelines.utils.sec_client.time")
    def test_cik_with_leading_zeros(self, mock_time, client):
        """CIK with leading zeros is handled correctly."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"facts": {}})
        client.session.get.return_value = mock_resp

        client.fetch_company_facts("0000320193")

        call_args = client.session.get.call_args
        url = call_args[0][0]
        assert "CIK0000320193.json" in url

    @patch("pipelines.utils.sec_client.time")
    def test_cik_as_integer_string(self, mock_time, client):
        """CIK as a simple integer string works correctly."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"facts": {}})
        client.session.get.return_value = mock_resp

        client.fetch_company_facts("789019")

        call_args = client.session.get.call_args
        url = call_args[0][0]
        assert "CIK0000789019.json" in url

    @patch("pipelines.utils.sec_client.time")
    def test_url_construction(self, mock_time, client):
        """Full URL is correctly formed."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"facts": {}})
        client.session.get.return_value = mock_resp

        client.fetch_company_facts("320193")

        call_args = client.session.get.call_args
        url = call_args[0][0]
        assert url == "https://data.sec.gov/api/xbrl/companyfacts/CIK0000320193.json"

    @patch("pipelines.utils.sec_client.time")
    def test_timeout_parameter(self, mock_time, client):
        """Request uses 30-second timeout."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"facts": {}})
        client.session.get.return_value = mock_resp

        client.fetch_company_facts("320193")

        call_kwargs = client.session.get.call_args
        assert call_kwargs[1]["timeout"] == 30

    @patch("pipelines.utils.sec_client.time")
    def test_http_404_returns_none(self, mock_time, client):
        """HTTP 404 returns None (company not found in EDGAR)."""
        mock_time.time.return_value = 100.0
        http_error = requests.exceptions.HTTPError(response=MagicMock(status_code=404))
        mock_resp = _mock_response(status_code=404, raise_for_status=http_error)
        client.session.get.return_value = mock_resp

        result = client.fetch_company_facts("9999999")

        assert result is None

    @patch("pipelines.utils.sec_client.time")
    def test_http_500_returns_none(self, mock_time, client):
        """HTTP 500 server error returns None."""
        mock_time.time.return_value = 100.0
        http_error = requests.exceptions.HTTPError(response=MagicMock(status_code=500))
        mock_resp = _mock_response(status_code=500, raise_for_status=http_error)
        client.session.get.return_value = mock_resp

        result = client.fetch_company_facts("320193")

        assert result is None

    @patch("pipelines.utils.sec_client.time")
    def test_http_429_rate_limit_returns_none(self, mock_time, client):
        """HTTP 429 rate limit error returns None."""
        mock_time.time.return_value = 100.0
        http_error = requests.exceptions.HTTPError(response=MagicMock(status_code=429))
        mock_resp = _mock_response(status_code=429, raise_for_status=http_error)
        client.session.get.return_value = mock_resp

        result = client.fetch_company_facts("320193")

        assert result is None

    @patch("pipelines.utils.sec_client.time")
    def test_timeout_returns_none(self, mock_time, client):
        """Request timeout returns None."""
        mock_time.time.return_value = 100.0
        client.session.get.side_effect = requests.exceptions.Timeout("Connection timed out")

        result = client.fetch_company_facts("320193")

        assert result is None

    @patch("pipelines.utils.sec_client.time")
    def test_connection_error_returns_none(self, mock_time, client):
        """Connection error returns None."""
        mock_time.time.return_value = 100.0
        client.session.get.side_effect = requests.exceptions.ConnectionError("No route")

        result = client.fetch_company_facts("320193")

        assert result is None

    @patch("pipelines.utils.sec_client.time")
    def test_unexpected_exception_returns_none(self, mock_time, client):
        """Unexpected exception returns None."""
        mock_time.time.return_value = 100.0
        client.session.get.side_effect = RuntimeError("Something unexpected")

        result = client.fetch_company_facts("320193")

        assert result is None

    def test_invalid_cik_format_returns_none(self, client):
        """Non-numeric CIK returns None."""
        result = client.fetch_company_facts("not_a_number")

        assert result is None

    @patch("pipelines.utils.sec_client.time")
    def test_bad_json_returns_none(self, mock_time, client):
        """Malformed JSON response returns None."""
        mock_time.time.return_value = 100.0
        mock_resp = MagicMock()
        mock_resp.raise_for_status.return_value = None
        mock_resp.json.side_effect = ValueError("No JSON object could be decoded")
        client.session.get.return_value = mock_resp

        result = client.fetch_company_facts("320193")

        assert result is None

    @patch("pipelines.utils.sec_client.time")
    def test_cik_zero(self, mock_time, client):
        """CIK of '0' should be handled gracefully."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"facts": {}})
        client.session.get.return_value = mock_resp

        result = client.fetch_company_facts("0")

        # Should not crash; CIK 0 would be formatted as 0000000000
        call_args = client.session.get.call_args
        url = call_args[0][0]
        assert "CIK0000000000.json" in url

    @patch("pipelines.utils.sec_client.time")
    def test_all_zeros_cik(self, mock_time, client):
        """CIK of all zeros ('0000000000') is handled correctly."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"facts": {}})
        client.session.get.return_value = mock_resp

        result = client.fetch_company_facts("0000000000")

        call_args = client.session.get.call_args
        url = call_args[0][0]
        assert "CIK0000000000.json" in url


# =====================================================================
# TestParseFinancialData
# =====================================================================


class TestParseFinancialData:
    """Tests for parse_financial_data."""

    def setup_method(self):
        self.client = SECClient()

    def test_empty_company_facts_returns_empty(self):
        """Empty company facts returns empty list."""
        result = self.client.parse_financial_data({})
        assert result == []

    def test_none_company_facts_returns_empty(self):
        """None company facts returns empty list."""
        result = self.client.parse_financial_data(None)
        assert result == []

    def test_no_facts_key_returns_empty(self):
        """Company facts without 'facts' key returns empty list."""
        result = self.client.parse_financial_data({"cik": "320193"})
        assert result == []

    def test_no_us_gaap_returns_empty(self):
        """Company facts without 'us-gaap' returns empty list."""
        result = self.client.parse_financial_data({"facts": {"dei": {}}})
        assert result == []

    def test_empty_us_gaap_returns_empty(self):
        """Company facts with empty 'us-gaap' returns empty list."""
        result = self.client.parse_financial_data({"facts": {"us-gaap": {}}})
        assert result == []

    def test_filters_non_10q_10k_forms(self):
        """Only 10-Q and 10-K forms are included."""
        company_facts = {
            "facts": {
                "us-gaap": {
                    "Revenues": {
                        "units": {
                            "USD": [
                                # 10-K should be included
                                {
                                    "form": "10-K",
                                    "end": "2024-12-31",
                                    "filed": "2025-02-15",
                                    "fy": 2024,
                                    "fp": "FY",
                                    "val": 100_000_000_000,
                                },
                                # 8-K should be filtered out
                                {
                                    "form": "8-K",
                                    "end": "2024-12-31",
                                    "filed": "2025-01-05",
                                    "fy": 2024,
                                    "fp": "FY",
                                    "val": 999999,
                                },
                                # 20-F should be filtered out
                                {
                                    "form": "20-F",
                                    "end": "2024-12-31",
                                    "filed": "2025-02-15",
                                    "fy": 2024,
                                    "fp": "FY",
                                    "val": 888888,
                                },
                            ]
                        }
                    },
                }
            }
        }

        result = self.client.parse_financial_data(company_facts)

        assert len(result) == 1
        assert result[0]["revenue"] == 100_000_000_000

    def test_parses_basic_10k_filing(self):
        """Parses a basic 10-K filing with revenue and net income."""
        company_facts = _make_company_facts(
            revenue_val=100_000_000_000,
            net_income_val=20_000_000_000,
        )

        result = self.client.parse_financial_data(company_facts)

        assert len(result) == 1
        assert result[0]["revenue"] == 100_000_000_000
        assert result[0]["net_income"] == 20_000_000_000
        assert result[0]["period_end_date"] == "2024-12-31"
        assert result[0]["fiscal_year"] == 2024
        assert result[0]["fiscal_quarter"] is None
        assert result[0]["statement_type"] == "10-K"

    def test_num_periods_limits_results(self):
        """num_periods parameter limits the number of results returned."""
        company_facts = {
            "facts": {
                "us-gaap": {
                    "Revenues": {
                        "units": {
                            "USD": [
                                {
                                    "form": "10-Q",
                                    "end": f"2024-{month:02d}-30",
                                    "filed": f"2024-{month + 1:02d}-15",
                                    "fy": 2024,
                                    "fp": f"Q{i + 1}",
                                    "val": (i + 1) * 10_000_000_000,
                                }
                                for i, month in enumerate([3, 6, 9])
                            ]
                        }
                    },
                }
            }
        }

        result = self.client.parse_financial_data(company_facts, num_periods=2)

        assert len(result) == 2

    def test_sorted_by_date_descending(self):
        """Results are sorted by period_end_date descending (newest first)."""
        company_facts = {
            "facts": {
                "us-gaap": {
                    "Revenues": {
                        "units": {
                            "USD": [
                                {
                                    "form": "10-Q",
                                    "end": "2024-03-31",
                                    "filed": "2024-05-01",
                                    "fy": 2024,
                                    "fp": "Q1",
                                    "val": 25_000_000_000,
                                },
                                {
                                    "form": "10-Q",
                                    "end": "2024-09-30",
                                    "filed": "2024-11-01",
                                    "fy": 2024,
                                    "fp": "Q3",
                                    "val": 75_000_000_000,
                                },
                                {
                                    "form": "10-Q",
                                    "end": "2024-06-30",
                                    "filed": "2024-08-01",
                                    "fy": 2024,
                                    "fp": "Q2",
                                    "val": 50_000_000_000,
                                },
                            ]
                        }
                    },
                }
            }
        }

        result = self.client.parse_financial_data(company_facts)

        assert result[0]["period_end_date"] == "2024-09-30"
        assert result[1]["period_end_date"] == "2024-06-30"
        assert result[2]["period_end_date"] == "2024-03-31"

    def test_skips_datapoints_missing_required_fields(self):
        """Datapoints missing required fields (end, filed, fy, val) are skipped."""
        company_facts = {
            "facts": {
                "us-gaap": {
                    "Revenues": {
                        "units": {
                            "USD": [
                                # Missing 'end'
                                {
                                    "form": "10-K",
                                    "filed": "2025-02-15",
                                    "fy": 2024,
                                    "fp": "FY",
                                    "val": 100_000_000,
                                },
                                # Missing 'filed'
                                {
                                    "form": "10-K",
                                    "end": "2024-12-31",
                                    "fy": 2024,
                                    "fp": "FY",
                                    "val": 100_000_000,
                                },
                                # Missing 'fy'
                                {
                                    "form": "10-K",
                                    "end": "2024-12-31",
                                    "filed": "2025-02-15",
                                    "fp": "FY",
                                    "val": 100_000_000,
                                },
                                # val is None
                                {
                                    "form": "10-K",
                                    "end": "2024-12-31",
                                    "filed": "2025-02-15",
                                    "fy": 2024,
                                    "fp": "FY",
                                    "val": None,
                                },
                                # Valid record
                                {
                                    "form": "10-K",
                                    "end": "2024-12-31",
                                    "filed": "2025-02-15",
                                    "fy": 2024,
                                    "fp": "FY",
                                    "val": 50_000_000_000,
                                },
                            ]
                        }
                    },
                }
            }
        }

        result = self.client.parse_financial_data(company_facts)

        assert len(result) == 1
        assert result[0]["revenue"] == 50_000_000_000

    def test_parses_shares_from_shares_unit(self):
        """Shares outstanding are parsed from 'shares' unit type."""
        company_facts = {
            "facts": {
                "us-gaap": {
                    "Revenues": {
                        "units": {
                            "USD": [
                                {
                                    "form": "10-K",
                                    "end": "2024-12-31",
                                    "filed": "2025-02-15",
                                    "fy": 2024,
                                    "fp": "FY",
                                    "val": 100_000_000_000,
                                },
                            ]
                        }
                    },
                    "CommonStockSharesOutstanding": {
                        "units": {
                            "shares": [
                                {
                                    "form": "10-K",
                                    "end": "2024-12-31",
                                    "filed": "2025-02-15",
                                    "fy": 2024,
                                    "fp": "FY",
                                    "val": 15_000_000_000,
                                },
                            ]
                        }
                    },
                }
            }
        }

        result = self.client.parse_financial_data(company_facts)

        assert len(result) == 1
        assert result[0]["shares_outstanding"] == 15_000_000_000

    def test_metrics_calculated_for_returned_periods(self):
        """Derived metrics are calculated for returned periods."""
        company_facts = _make_company_facts(
            revenue_val=100_000_000_000,
            net_income_val=20_000_000_000,
        )

        result = self.client.parse_financial_data(company_facts, num_periods=5)

        assert len(result) == 1
        # net_margin should be calculated: 20B / 100B * 100 = 20%
        assert abs(result[0]["net_margin"] - 20.0) < 0.01


# =====================================================================
# TestParseFiscalPeriod
# =====================================================================


class TestParseFiscalPeriod:
    """Tests for SECClient._parse_fiscal_period."""

    def setup_method(self):
        self.client = SECClient()

    def test_q1(self):
        """Q1 parses to quarter 1."""
        assert self.client._parse_fiscal_period("Q1") == 1

    def test_q2(self):
        """Q2 parses to quarter 2."""
        assert self.client._parse_fiscal_period("Q2") == 2

    def test_q3(self):
        """Q3 parses to quarter 3."""
        assert self.client._parse_fiscal_period("Q3") == 3

    def test_q4(self):
        """Q4 parses to quarter 4."""
        assert self.client._parse_fiscal_period("Q4") == 4

    def test_fy(self):
        """FY (annual) parses to None."""
        assert self.client._parse_fiscal_period("FY") is None

    def test_q1i_variant(self):
        """Q1I interim variant parses to quarter 1."""
        assert self.client._parse_fiscal_period("Q1I") == 1

    def test_q2i_variant(self):
        """Q2I interim variant parses to quarter 2."""
        assert self.client._parse_fiscal_period("Q2I") == 2

    def test_q3i_variant(self):
        """Q3I interim variant parses to quarter 3."""
        assert self.client._parse_fiscal_period("Q3I") == 3

    def test_q4i_variant(self):
        """Q4I interim variant parses to quarter 4."""
        assert self.client._parse_fiscal_period("Q4I") == 4

    def test_none_input(self):
        """None input returns None."""
        assert self.client._parse_fiscal_period(None) is None

    def test_lowercase_input(self):
        """Lowercase inputs are handled by uppercasing."""
        assert self.client._parse_fiscal_period("q1") == 1
        assert self.client._parse_fiscal_period("fy") is None

    def test_unknown_period_returns_none(self):
        """Unknown fiscal period strings return None."""
        assert self.client._parse_fiscal_period("H1") is None
        assert self.client._parse_fiscal_period("YTD") is None
        assert self.client._parse_fiscal_period("") is None


# =====================================================================
# TestCorrectFiscalYears
# =====================================================================


class TestCorrectFiscalYears:
    """Tests for SECClient._correct_fiscal_years.

    This method detects the fiscal year-end (FYE) month from annual filings
    and corrects quarterly fiscal_year values when the company has a
    non-December fiscal year end (e.g., Apple's September FYE).
    """

    def setup_method(self):
        self.client = SECClient()

    def test_calendar_year_no_correction(self):
        """Companies with December FYE should have no fiscal year corrections."""
        financials = [
            # Annual filing ending December (FYE = December)
            {
                "period_end_date": "2024-12-31",
                "fiscal_year": 2024,
                "fiscal_quarter": None,
            },
            # Q1 ending March
            {
                "period_end_date": "2024-03-31",
                "fiscal_year": 2024,
                "fiscal_quarter": 1,
            },
            # Q2 ending June
            {
                "period_end_date": "2024-06-30",
                "fiscal_year": 2024,
                "fiscal_quarter": 2,
            },
            # Q3 ending September
            {
                "period_end_date": "2024-09-30",
                "fiscal_year": 2024,
                "fiscal_quarter": 3,
            },
        ]

        self.client._correct_fiscal_years(financials)

        # All fiscal years should remain unchanged
        assert financials[0]["fiscal_year"] == 2024
        assert financials[1]["fiscal_year"] == 2024
        assert financials[2]["fiscal_year"] == 2024
        assert financials[3]["fiscal_year"] == 2024

    def test_non_calendar_fy_correction(self):
        """AAPL-like company with September FYE: October period belongs to next FY."""
        financials = [
            # Annual filing ending September (FYE = September)
            {
                "period_end_date": "2024-09-28",
                "fiscal_year": 2024,
                "fiscal_quarter": None,
            },
            # Quarterly filing ending in October -- period_end_month (10) > fye_month (9)
            # So correct fiscal year = 2024 + 1 = 2025
            {
                "period_end_date": "2024-10-31",
                "fiscal_year": 2024,
                "fiscal_quarter": 1,
            },
        ]

        self.client._correct_fiscal_years(financials)

        # Annual record should stay the same
        assert financials[0]["fiscal_year"] == 2024
        # October quarter should be corrected to FY 2025
        assert financials[1]["fiscal_year"] == 2025

    def test_no_annual_filings_skips(self):
        """When there are no annual filings, no corrections should be applied."""
        financials = [
            {
                "period_end_date": "2024-10-31",
                "fiscal_year": 2024,
                "fiscal_quarter": 1,
            },
            {
                "period_end_date": "2025-01-31",
                "fiscal_year": 2025,
                "fiscal_quarter": 2,
            },
        ]

        original_fy_values = [f["fiscal_year"] for f in financials]
        self.client._correct_fiscal_years(financials)
        new_fy_values = [f["fiscal_year"] for f in financials]

        assert original_fy_values == new_fy_values

    def test_multiple_quarters_corrected(self):
        """All quarterly records for a non-calendar FYE are corrected consistently."""
        financials = [
            # Annual filing ending June (FYE = June)
            {
                "period_end_date": "2024-06-30",
                "fiscal_year": 2024,
                "fiscal_quarter": None,
            },
            # Q1: period ends September -- month 9 > fye_month 6, so FY = 2024+1 = 2025
            {
                "period_end_date": "2024-09-30",
                "fiscal_year": 2024,
                "fiscal_quarter": 1,
            },
            # Q2: period ends December -- month 12 > fye_month 6, so FY = 2024+1 = 2025
            {
                "period_end_date": "2024-12-31",
                "fiscal_year": 2024,
                "fiscal_quarter": 2,
            },
            # Q3: period ends March -- month 3 <= fye_month 6, so FY = 2025
            {
                "period_end_date": "2025-03-31",
                "fiscal_year": 2025,
                "fiscal_quarter": 3,
            },
        ]

        self.client._correct_fiscal_years(financials)

        # Q1 and Q2 with period_end_month > 6 should become FY 2025
        assert financials[1]["fiscal_year"] == 2025
        assert financials[2]["fiscal_year"] == 2025
        # Q3 with period_end_month (3) <= 6 stays FY 2025
        assert financials[3]["fiscal_year"] == 2025

    def test_annual_records_not_modified(self):
        """Annual records (fiscal_quarter=None) should not have fiscal_year changed."""
        financials = [
            # Annual filing with September FYE
            {
                "period_end_date": "2024-09-28",
                "fiscal_year": 2024,
                "fiscal_quarter": None,
            },
            # Another annual filing
            {
                "period_end_date": "2023-09-30",
                "fiscal_year": 2023,
                "fiscal_quarter": None,
            },
        ]

        self.client._correct_fiscal_years(financials)

        assert financials[0]["fiscal_year"] == 2024
        assert financials[1]["fiscal_year"] == 2023

    def test_empty_list(self):
        """Empty financials list is handled gracefully."""
        self.client._correct_fiscal_years([])
        # Should not raise


# =====================================================================
# TestCorrectSharesOutstanding
# =====================================================================


class TestCorrectSharesOutstanding:
    """Tests for SECClient._correct_shares_outstanding.

    Heuristic: if shares_outstanding < 10,000 but revenue > 1B,
    multiply shares by 1,000,000 (company reported in millions).
    """

    def setup_method(self):
        self.client = SECClient()

    def test_shares_in_millions_corrected(self):
        """Shares < 10,000 with revenue > 1B are multiplied by 1,000,000."""
        financials = [
            {
                "shares_outstanding": 750,
                "revenue": 50_000_000_000,  # 50B
            }
        ]

        self.client._correct_shares_outstanding(financials)

        assert financials[0]["shares_outstanding"] == 750_000_000

    def test_normal_shares_not_corrected(self):
        """Shares >= 10,000 are left unchanged even with high revenue."""
        financials = [
            {
                "shares_outstanding": 15_000_000_000,
                "revenue": 50_000_000_000,
            }
        ]

        self.client._correct_shares_outstanding(financials)

        assert financials[0]["shares_outstanding"] == 15_000_000_000

    def test_no_shares_skipped(self):
        """Records with shares_outstanding=None are skipped without error."""
        financials = [
            {
                "shares_outstanding": None,
                "revenue": 50_000_000_000,
            }
        ]

        self.client._correct_shares_outstanding(financials)

        assert financials[0]["shares_outstanding"] is None

    def test_low_revenue_not_corrected(self):
        """Shares < 10,000 with revenue <= 1B are not corrected (small company)."""
        financials = [
            {
                "shares_outstanding": 500,
                "revenue": 500_000_000,  # 500M, below 1B threshold
            }
        ]

        self.client._correct_shares_outstanding(financials)

        assert financials[0]["shares_outstanding"] == 500

    def test_no_revenue_not_corrected(self):
        """Shares with no revenue field are not corrected."""
        financials = [
            {
                "shares_outstanding": 500,
            }
        ]

        self.client._correct_shares_outstanding(financials)

        assert financials[0]["shares_outstanding"] == 500

    def test_boundary_shares_9999(self):
        """Shares at 9999 (just under 10000) with revenue > 1B are corrected."""
        financials = [
            {
                "shares_outstanding": 9999,
                "revenue": 2_000_000_000,
            }
        ]

        self.client._correct_shares_outstanding(financials)

        assert financials[0]["shares_outstanding"] == 9_999_000_000

    def test_boundary_shares_10000(self):
        """Shares at 10000 (threshold) with revenue > 1B are NOT corrected."""
        financials = [
            {
                "shares_outstanding": 10000,
                "revenue": 2_000_000_000,
            }
        ]

        self.client._correct_shares_outstanding(financials)

        assert financials[0]["shares_outstanding"] == 10000


# =====================================================================
# TestCalculateMetrics
# =====================================================================


class TestCalculateMetrics:
    """Tests for SECClient._calculate_metrics.

    This method calculates derived financial ratios/metrics in-place.
    """

    def setup_method(self):
        self.client = SECClient()

    def _make_financial(self, **kwargs):
        """Helper to create a financial dict with sensible defaults."""
        base = {
            "revenue": 100_000_000_000,       # 100B
            "cost_of_revenue": 60_000_000_000,  # 60B
            "operating_income": 25_000_000_000,  # 25B
            "net_income": 20_000_000_000,        # 20B
            "total_assets": 400_000_000_000,     # 400B
            "total_liabilities": 250_000_000_000,
            "shareholders_equity": 150_000_000_000,
            "short_term_debt": 10_000_000_000,
            "long_term_debt": 90_000_000_000,
            "operating_cash_flow": 30_000_000_000,
            "capex": -8_000_000_000,  # capex is typically negative
        }
        base.update(kwargs)
        return base

    def test_gross_profit(self):
        """Gross profit = revenue - cost_of_revenue when not already present."""
        financial = self._make_financial()
        # Remove pre-existing gross_profit to force calculation
        financial.pop("gross_profit", None)

        self.client._calculate_metrics(financial)

        # 100B - 60B = 40B
        assert financial["gross_profit"] == 40_000_000_000.0

    def test_free_cash_flow(self):
        """Free cash flow = operating_cash_flow - abs(capex)."""
        financial = self._make_financial()

        self.client._calculate_metrics(financial)

        # 30B - abs(-8B) = 22B
        assert financial["free_cash_flow"] == 22_000_000_000.0

    def test_gross_margin(self):
        """Gross margin = gross_profit / revenue * 100."""
        financial = self._make_financial()
        financial.pop("gross_profit", None)

        self.client._calculate_metrics(financial)

        # gross_profit = 40B, revenue = 100B -> 40%
        assert abs(financial["gross_margin"] - 40.0) < 0.01

    def test_operating_margin(self):
        """Operating margin = operating_income / revenue * 100."""
        financial = self._make_financial()

        self.client._calculate_metrics(financial)

        # 25B / 100B * 100 = 25%
        assert abs(financial["operating_margin"] - 25.0) < 0.01

    def test_net_margin(self):
        """Net margin = net_income / revenue * 100."""
        financial = self._make_financial()

        self.client._calculate_metrics(financial)

        # 20B / 100B * 100 = 20%
        assert abs(financial["net_margin"] - 20.0) < 0.01

    def test_debt_to_equity(self):
        """Debt to equity = (short_term_debt + long_term_debt) / shareholders_equity."""
        financial = self._make_financial()

        self.client._calculate_metrics(financial)

        # (10B + 90B) / 150B = 0.6667
        expected = (10_000_000_000 + 90_000_000_000) / 150_000_000_000
        assert abs(financial["debt_to_equity"] - expected) < 0.0001

    def test_roa(self):
        """ROA = net_income / total_assets * 100."""
        financial = self._make_financial()

        self.client._calculate_metrics(financial)

        # 20B / 400B * 100 = 5%
        assert abs(financial["roa"] - 5.0) < 0.01

    def test_roe(self):
        """ROE = net_income / shareholders_equity * 100."""
        financial = self._make_financial()

        self.client._calculate_metrics(financial)

        # 20B / 150B * 100 = 13.333...%
        expected = (20_000_000_000 / 150_000_000_000) * 100
        assert abs(financial["roe"] - expected) < 0.01

    def test_zero_revenue_safe(self):
        """Zero revenue should not cause division errors; margins should not be set."""
        financial = self._make_financial(revenue=0, cost_of_revenue=0)

        self.client._calculate_metrics(financial)

        # No division by zero error should occur
        assert "gross_margin" not in financial
        assert "operating_margin" not in financial
        assert "net_margin" not in financial

    def test_none_revenue_safe(self):
        """None revenue should not cause errors."""
        financial = self._make_financial(revenue=None, cost_of_revenue=None)

        self.client._calculate_metrics(financial)

        assert "gross_margin" not in financial
        assert "operating_margin" not in financial

    def test_missing_capex_no_fcf(self):
        """Free cash flow is not calculated when capex is missing."""
        financial = self._make_financial(capex=None)

        self.client._calculate_metrics(financial)

        assert "free_cash_flow" not in financial

    def test_missing_operating_cash_flow_no_fcf(self):
        """Free cash flow is not calculated when operating_cash_flow is missing."""
        financial = self._make_financial(operating_cash_flow=None)

        self.client._calculate_metrics(financial)

        assert "free_cash_flow" not in financial

    def test_zero_shareholders_equity_no_roe(self):
        """ROE is not calculated when shareholders_equity is zero."""
        financial = self._make_financial(shareholders_equity=0)

        self.client._calculate_metrics(financial)

        assert "roe" not in financial

    def test_none_shareholders_equity_no_roe(self):
        """ROE is not calculated when shareholders_equity is None."""
        financial = self._make_financial(shareholders_equity=None)

        self.client._calculate_metrics(financial)

        assert "roe" not in financial
        assert "debt_to_equity" not in financial

    def test_zero_total_assets_no_roa(self):
        """ROA is not calculated when total_assets is zero."""
        financial = self._make_financial(total_assets=0)

        self.client._calculate_metrics(financial)

        assert "roa" not in financial

    def test_missing_cost_of_revenue_no_gross_profit_calc(self):
        """Gross profit is not calculated when cost_of_revenue is missing."""
        financial = self._make_financial(cost_of_revenue=None)
        financial.pop("gross_profit", None)

        self.client._calculate_metrics(financial)

        assert "gross_profit" not in financial

    def test_debt_defaults_to_zero(self):
        """Missing short_term_debt and long_term_debt default to 0 for D/E calc."""
        financial = self._make_financial(short_term_debt=None, long_term_debt=None)

        self.client._calculate_metrics(financial)

        # 0 / 150B = 0
        assert financial["debt_to_equity"] == 0.0

    def test_revenue_synthesis_from_banking_fields(self):
        """Revenue is synthesized from net_interest_income + noninterest_income."""
        financial = {
            "net_interest_income": 23_500_000_000,
            "noninterest_income": 23_600_000_000,
            "net_income": 14_000_000_000,
        }

        self.client._calculate_metrics(financial)

        expected_revenue = 23_500_000_000 + 23_600_000_000
        assert financial["revenue"] == float(expected_revenue)
        # Net margin should be calculated from synthesized revenue
        expected_margin = (14_000_000_000 / expected_revenue) * 100
        assert abs(financial["net_margin"] - expected_margin) < 0.01

    def test_revenue_synthesis_from_insurance_premiums(self):
        """Revenue is synthesized from premiums_earned when standard revenue is missing."""
        financial = {
            "premiums_earned": 42_000_000_000,
            "net_income": 6_300_000_000,
        }

        self.client._calculate_metrics(financial)

        assert financial["revenue"] == float(42_000_000_000)

    def test_revenue_synthesis_from_real_estate(self):
        """Revenue is synthesized from real_estate_revenue when standard revenue is missing."""
        financial = {
            "real_estate_revenue": 8_000_000_000,
            "net_income": 2_000_000_000,
        }

        self.client._calculate_metrics(financial)

        assert financial["revenue"] == float(8_000_000_000)

    def test_standard_revenue_not_overridden(self):
        """When standard revenue exists, industry-specific fields don't override it."""
        financial = {
            "revenue": 50_000_000_000,
            "net_interest_income": 25_000_000_000,
            "noninterest_income": 20_000_000_000,
            "net_income": 10_000_000_000,
        }

        self.client._calculate_metrics(financial)

        # Revenue should remain as the standard value, not be overridden
        # (synthesis only happens when revenue is None)
        assert "net_margin" in financial
        expected_margin = (10_000_000_000 / 50_000_000_000) * 100
        assert abs(financial["net_margin"] - expected_margin) < 0.01

    def test_banking_revenue_with_only_nii(self):
        """Revenue can be synthesized with only net_interest_income (no noninterest)."""
        financial = {
            "net_interest_income": 23_500_000_000,
            "net_income": 10_000_000_000,
        }

        self.client._calculate_metrics(financial)

        assert financial["revenue"] == float(23_500_000_000)

    def test_negative_net_income(self):
        """Negative net income produces negative margins."""
        financial = self._make_financial(net_income=-5_000_000_000)

        self.client._calculate_metrics(financial)

        assert financial["net_margin"] < 0
        assert abs(financial["net_margin"] - (-5.0)) < 0.01


# =====================================================================
# TestGetFinancialsForTicker
# =====================================================================


class TestGetFinancialsForTicker:
    """Tests for get_financials_for_ticker (end-to-end orchestration)."""

    @patch("pipelines.utils.sec_client.time")
    def test_success(self, mock_time, client):
        """Successfully fetches and parses financial data."""
        mock_time.time.return_value = 100.0
        company_facts = _make_company_facts()
        mock_resp = _mock_response(json_data=company_facts)
        client.session.get.return_value = mock_resp

        result = client.get_financials_for_ticker("320193", num_periods=5)

        assert isinstance(result, list)
        assert len(result) >= 1
        assert result[0]["revenue"] == 100_000_000_000

    @patch("pipelines.utils.sec_client.time")
    def test_api_failure_returns_empty(self, mock_time, client):
        """Returns empty list when API request fails."""
        mock_time.time.return_value = 100.0
        client.session.get.side_effect = requests.exceptions.ConnectionError()

        result = client.get_financials_for_ticker("320193")

        assert result == []

    @patch("pipelines.utils.sec_client.time")
    def test_404_returns_empty(self, mock_time, client):
        """Returns empty list when company not found."""
        mock_time.time.return_value = 100.0
        http_error = requests.exceptions.HTTPError(response=MagicMock(status_code=404))
        mock_resp = _mock_response(status_code=404, raise_for_status=http_error)
        client.session.get.return_value = mock_resp

        result = client.get_financials_for_ticker("9999999")

        assert result == []

    def test_invalid_cik_returns_empty(self, client):
        """Returns empty list when CIK is invalid."""
        result = client.get_financials_for_ticker("not_a_number")

        assert result == []

    @patch("pipelines.utils.sec_client.time")
    def test_empty_facts_returns_empty(self, mock_time, client):
        """Returns empty list when company facts has no data."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"facts": {"us-gaap": {}}})
        client.session.get.return_value = mock_resp

        result = client.get_financials_for_ticker("320193")

        assert result == []

    @patch("pipelines.utils.sec_client.time")
    def test_respects_num_periods(self, mock_time, client):
        """Respects the num_periods parameter."""
        mock_time.time.return_value = 100.0
        company_facts = {
            "facts": {
                "us-gaap": {
                    "Revenues": {
                        "units": {
                            "USD": [
                                {
                                    "form": "10-Q",
                                    "end": f"2024-{month:02d}-30",
                                    "filed": f"2024-{month + 1:02d}-15",
                                    "fy": 2024,
                                    "fp": f"Q{i + 1}",
                                    "val": (i + 1) * 25_000_000_000,
                                }
                                for i, month in enumerate([3, 6, 9])
                            ]
                        }
                    },
                }
            }
        }
        mock_resp = _mock_response(json_data=company_facts)
        client.session.get.return_value = mock_resp

        result = client.get_financials_for_ticker("320193", num_periods=2)

        assert len(result) == 2


# =====================================================================
# TestClose
# =====================================================================


class TestClose:
    """Tests for close method."""

    def test_close_calls_session_close(self, client):
        """close() calls session.close()."""
        client.close()

        client.session.close.assert_called_once()

    def test_close_with_none_session(self):
        """close() handles None session gracefully."""
        with patch.object(SECClient, "_create_session"):
            c = SECClient()
        c.session = None

        # Should not raise
        c.close()


# =====================================================================
# TestAsyncSECClientInit
# =====================================================================


class TestAsyncSECClientInit:
    """Tests for AsyncSECClient initialization."""

    def test_init_default_user_agent(self):
        """Async client uses default user agent."""
        client = AsyncSECClient()
        assert client.user_agent == "InvestorCenter.ai admin@investorcenter.ai"

    def test_init_custom_user_agent(self):
        """Async client accepts custom user agent."""
        client = AsyncSECClient(user_agent="MyApp dev@example.com")
        assert client.user_agent == "MyApp dev@example.com"

    def test_init_creates_sync_client(self):
        """Async client creates an internal sync client for parsing."""
        client = AsyncSECClient()
        assert isinstance(client.sync_client, SECClient)

    def test_init_sets_last_request_time_to_zero(self):
        """Initial last_request_time is 0.0."""
        client = AsyncSECClient()
        assert client.last_request_time == 0.0

    def test_init_creates_semaphore(self):
        """Semaphore is created with REQUESTS_PER_SECOND capacity."""
        client = AsyncSECClient()
        assert isinstance(client.semaphore, asyncio.Semaphore)


# =====================================================================
# TestAsyncSECClientFetchCompanyFacts
# =====================================================================


def _make_async_aiohttp_mocks(status=200, json_data=None):
    """Helper to create properly nested async context manager mocks for aiohttp.

    aiohttp uses two nested async context managers:
        async with aiohttp.ClientSession() as session:
            async with session.get(url, ...) as response:
                ...

    Returns:
        (mock_client_session_class, mock_session, mock_response)
    """
    # Mock the response object
    mock_response = MagicMock()
    mock_response.status = status
    mock_response.json = AsyncMock(return_value=json_data or {})
    mock_response.raise_for_status = MagicMock()

    # session.get(...) returns an async context manager yielding mock_response
    mock_get_ctx = MagicMock()
    mock_get_ctx.__aenter__ = AsyncMock(return_value=mock_response)
    mock_get_ctx.__aexit__ = AsyncMock(return_value=False)

    # The session itself (has .get method)
    mock_session = MagicMock()
    mock_session.get.return_value = mock_get_ctx

    # aiohttp.ClientSession() returns an async context manager yielding mock_session
    mock_client_session_instance = MagicMock()
    mock_client_session_instance.__aenter__ = AsyncMock(return_value=mock_session)
    mock_client_session_instance.__aexit__ = AsyncMock(return_value=False)

    return mock_client_session_instance, mock_session, mock_response


class TestAsyncSECClientFetchCompanyFacts:
    """Tests for AsyncSECClient.fetch_company_facts."""

    @pytest.fixture
    def async_client(self):
        """Create an AsyncSECClient instance."""
        return AsyncSECClient()

    @pytest.mark.asyncio
    async def test_invalid_cik_returns_none(self, async_client):
        """Invalid CIK returns None without making any HTTP request."""
        result = await async_client.fetch_company_facts("not_a_number")
        assert result is None

    @pytest.mark.asyncio
    async def test_success(self, async_client):
        """Successful async fetch returns parsed JSON."""
        expected_data = {"facts": {"us-gaap": {"Revenues": {}}}}
        mock_cs, _, _ = _make_async_aiohttp_mocks(status=200, json_data=expected_data)

        with patch("pipelines.utils.sec_client.aiohttp.ClientSession",
                    return_value=mock_cs):
            with patch("pipelines.utils.sec_client.time") as mock_time:
                mock_time.time.return_value = 100.0
                result = await async_client.fetch_company_facts("320193")

        assert result == expected_data

    @pytest.mark.asyncio
    async def test_404_returns_none(self, async_client):
        """HTTP 404 returns None for async client."""
        mock_cs, _, _ = _make_async_aiohttp_mocks(status=404)

        with patch("pipelines.utils.sec_client.aiohttp.ClientSession",
                    return_value=mock_cs):
            with patch("pipelines.utils.sec_client.time") as mock_time:
                mock_time.time.return_value = 100.0
                result = await async_client.fetch_company_facts("9999999")

        assert result is None

    @pytest.mark.asyncio
    async def test_exception_returns_none(self, async_client):
        """General exceptions during async fetch return None."""
        with patch("pipelines.utils.sec_client.aiohttp.ClientSession",
                    side_effect=Exception("Connection failed")):
            with patch("pipelines.utils.sec_client.time") as mock_time:
                mock_time.time.return_value = 100.0
                result = await async_client.fetch_company_facts("320193")

        assert result is None

    @pytest.mark.asyncio
    async def test_cik_zero_padding(self, async_client):
        """CIK is zero-padded in URL for async client."""
        mock_cs, mock_session, _ = _make_async_aiohttp_mocks(
            status=200, json_data={"facts": {}}
        )

        with patch("pipelines.utils.sec_client.aiohttp.ClientSession",
                    return_value=mock_cs):
            with patch("pipelines.utils.sec_client.time") as mock_time:
                mock_time.time.return_value = 100.0
                await async_client.fetch_company_facts("320193")

        # Verify URL was correctly constructed
        call_args = mock_session.get.call_args
        url = call_args[0][0]
        assert "CIK0000320193.json" in url

    @pytest.mark.asyncio
    async def test_headers_include_user_agent(self, async_client):
        """Async client sends correct User-Agent and Accept headers."""
        mock_cs, mock_session, _ = _make_async_aiohttp_mocks(
            status=200, json_data={"facts": {}}
        )

        with patch("pipelines.utils.sec_client.aiohttp.ClientSession",
                    return_value=mock_cs):
            with patch("pipelines.utils.sec_client.time") as mock_time:
                mock_time.time.return_value = 100.0
                await async_client.fetch_company_facts("320193")

        call_kwargs = mock_session.get.call_args
        headers = call_kwargs[1]["headers"]
        assert headers["User-Agent"] == "InvestorCenter.ai admin@investorcenter.ai"
        assert headers["Accept"] == "application/json"


# =====================================================================
# TestAsyncSECClientGetFinancialsForTicker
# =====================================================================


class TestAsyncSECClientGetFinancialsForTicker:
    """Tests for AsyncSECClient.get_financials_for_ticker."""

    @pytest.fixture
    def async_client(self):
        """Create an AsyncSECClient instance."""
        return AsyncSECClient()

    @pytest.mark.asyncio
    async def test_returns_empty_when_fetch_fails(self, async_client):
        """Returns empty list when async fetch fails."""
        with patch.object(async_client, "fetch_company_facts",
                          new_callable=AsyncMock, return_value=None):
            result = await async_client.get_financials_for_ticker("320193")

        assert result == []

    @pytest.mark.asyncio
    async def test_returns_parsed_financials(self, async_client):
        """Returns parsed financial data from sync parser."""
        company_facts = _make_company_facts()

        with patch.object(async_client, "fetch_company_facts",
                          new_callable=AsyncMock, return_value=company_facts):
            result = await async_client.get_financials_for_ticker("320193")

        assert isinstance(result, list)
        assert len(result) >= 1
        assert result[0]["revenue"] == 100_000_000_000

    @pytest.mark.asyncio
    async def test_respects_num_periods(self, async_client):
        """Passes num_periods to sync parser."""
        company_facts = {
            "facts": {
                "us-gaap": {
                    "Revenues": {
                        "units": {
                            "USD": [
                                {
                                    "form": "10-Q",
                                    "end": f"2024-{month:02d}-30",
                                    "filed": f"2024-{month + 1:02d}-15",
                                    "fy": 2024,
                                    "fp": f"Q{i + 1}",
                                    "val": (i + 1) * 25_000_000_000,
                                }
                                for i, month in enumerate([3, 6, 9])
                            ]
                        }
                    },
                }
            }
        }

        with patch.object(async_client, "fetch_company_facts",
                          new_callable=AsyncMock, return_value=company_facts):
            result = await async_client.get_financials_for_ticker("320193", num_periods=2)

        assert len(result) == 2


# =====================================================================
# TestAsyncSECClientRateLimit
# =====================================================================


class TestAsyncSECClientRateLimit:
    """Tests for AsyncSECClient._rate_limit."""

    @pytest.mark.asyncio
    async def test_rate_limit_updates_last_request_time(self):
        """Async rate limiter updates last_request_time."""
        client = AsyncSECClient()
        client.last_request_time = 0.0

        with patch("pipelines.utils.sec_client.time") as mock_time:
            mock_time.time.side_effect = [5.0, 5.0]
            await client._rate_limit()

        assert client.last_request_time == 5.0

    @pytest.mark.asyncio
    async def test_rate_limit_sleeps_when_too_fast(self):
        """Async rate limiter sleeps when requests are too fast."""
        client = AsyncSECClient()
        client.last_request_time = 1.0

        with patch("pipelines.utils.sec_client.time") as mock_time:
            mock_time.time.side_effect = [1.02, 1.12]
            with patch("pipelines.utils.sec_client.asyncio.sleep",
                        new_callable=AsyncMock) as mock_sleep:
                await client._rate_limit()

                mock_sleep.assert_called_once()
                sleep_arg = mock_sleep.call_args[0][0]
                assert abs(sleep_arg - 0.08) < 0.01

    @pytest.mark.asyncio
    async def test_rate_limit_no_sleep_when_enough_time(self):
        """Async rate limiter does not sleep when enough time has passed."""
        client = AsyncSECClient()
        client.last_request_time = 1.0

        with patch("pipelines.utils.sec_client.time") as mock_time:
            mock_time.time.side_effect = [2.0, 2.0]
            with patch("pipelines.utils.sec_client.asyncio.sleep",
                        new_callable=AsyncMock) as mock_sleep:
                await client._rate_limit()

                mock_sleep.assert_not_called()


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
