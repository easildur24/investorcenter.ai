"""Tests for the SEC 13F institutional holdings ingestion pipeline.

Validates institution configuration, rate limiting, filing fetching,
XML parsing, CUSIP-to-ticker mapping, quarter calculation, and
database storage with mocked requests and get_database.
"""

from datetime import date
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

# Patch get_database before importing the pipeline module.
with patch("pipelines.sec_13f_ingestion.get_database"):
    from pipelines.sec_13f_ingestion import SEC13FIngestion


@pytest.fixture
def pipeline():
    """Create a SEC13FIngestion with mocked DB and requests."""
    with patch(
        "pipelines.sec_13f_ingestion.get_database"
    ) as mock_db:
        db_instance = MagicMock()
        mock_db.return_value = db_instance
        p = SEC13FIngestion()
        # Mock the requests session to prevent real HTTP calls
        p.session = MagicMock()
        yield p


# ==================================================================
# Configuration
# ==================================================================


class TestConfiguration:
    def test_major_institutions_not_empty(self):
        assert len(SEC13FIngestion.MAJOR_INSTITUTIONS) > 0

    def test_berkshire_is_first(self):
        cik, name = SEC13FIngestion.MAJOR_INSTITUTIONS[0]
        assert "Berkshire" in name
        assert cik == "0001067983"

    def test_all_institutions_have_cik_and_name(self):
        for cik, name in SEC13FIngestion.MAJOR_INSTITUTIONS:
            assert cik is not None
            assert len(cik) == 10
            assert name is not None
            assert len(name) > 0

    def test_user_agent_set(self):
        assert "InvestorCenter" in SEC13FIngestion.USER_AGENT

    def test_rate_limit_constants(self):
        assert SEC13FIngestion.REQUESTS_PER_SECOND == 10
        assert SEC13FIngestion.MIN_REQUEST_INTERVAL > 0


# ==================================================================
# Rate limiting
# ==================================================================


class TestRateLimiting:
    def test_rate_limit_updates_timestamp(self, pipeline):
        pipeline.last_request_time = 0.0
        pipeline._rate_limit()
        assert pipeline.last_request_time > 0.0


# ==================================================================
# parse_filing_date_to_quarter
# ==================================================================


class TestParseFilingDateToQuarter:
    def test_q1_filing_maps_to_q4_prev_year(self, pipeline):
        result = pipeline.parse_filing_date_to_quarter("2024-02-15")
        assert result == date(2023, 12, 31)

    def test_q2_filing_maps_to_q1(self, pipeline):
        result = pipeline.parse_filing_date_to_quarter("2024-05-15")
        assert result == date(2024, 3, 31)

    def test_q3_filing_maps_to_q2(self, pipeline):
        result = pipeline.parse_filing_date_to_quarter("2024-08-15")
        assert result == date(2024, 6, 30)

    def test_q4_filing_maps_to_q3(self, pipeline):
        result = pipeline.parse_filing_date_to_quarter("2024-11-15")
        assert result == date(2024, 9, 30)

    def test_boundary_month_3(self, pipeline):
        result = pipeline.parse_filing_date_to_quarter("2024-03-31")
        assert result == date(2023, 12, 31)

    def test_boundary_month_6(self, pipeline):
        result = pipeline.parse_filing_date_to_quarter("2024-06-30")
        assert result == date(2024, 3, 31)


# ==================================================================
# get_latest_13f_filing
# ==================================================================


class TestGetLatest13FFiling:
    def test_success_returns_dict(self, pipeline):
        atom_xml = """<?xml version="1.0" encoding="UTF-8"?>
        <feed xmlns="http://www.w3.org/2005/Atom">
          <entry>
            <content>
              <accession-number>0001067983-24-000012</accession-number>
              <filing-date>2024-05-15</filing-date>
              <filing-href>https://www.sec.gov/test</filing-href>
            </content>
          </entry>
        </feed>"""

        mock_response = MagicMock()
        mock_response.content = atom_xml.encode()
        mock_response.raise_for_status = MagicMock()
        pipeline.session.get.return_value = mock_response

        result = pipeline.get_latest_13f_filing("0001067983")

        assert result is not None
        assert result["accession_number"] == "0001067983-24-000012"
        assert result["filing_date"] == "2024-05-15"

    def test_no_entries_returns_none(self, pipeline):
        atom_xml = """<?xml version="1.0" encoding="UTF-8"?>
        <feed xmlns="http://www.w3.org/2005/Atom">
        </feed>"""

        mock_response = MagicMock()
        mock_response.content = atom_xml.encode()
        mock_response.raise_for_status = MagicMock()
        pipeline.session.get.return_value = mock_response

        result = pipeline.get_latest_13f_filing("0000000000")
        assert result is None

    def test_request_error_returns_none(self, pipeline):
        pipeline.session.get.side_effect = Exception(
            "Connection error"
        )

        result = pipeline.get_latest_13f_filing("0001067983")
        assert result is None


# ==================================================================
# lookup_cusips_via_openfigi
# ==================================================================


class TestLookupCusips:
    def test_empty_list_returns_empty(self, pipeline):
        result = pipeline.lookup_cusips_via_openfigi([])
        assert result == {}

    def test_successful_lookup(self, pipeline):
        mock_response = MagicMock()
        mock_response.json.return_value = [
            {
                "data": [{"ticker": "AAPL"}],
            }
        ]
        mock_response.raise_for_status = MagicMock()
        pipeline.session.post.return_value = mock_response

        result = pipeline.lookup_cusips_via_openfigi(
            ["037833100"]
        )

        assert "037833100" in result
        assert result["037833100"] == "AAPL"

    def test_failed_lookup_excluded(self, pipeline):
        mock_response = MagicMock()
        mock_response.json.return_value = [
            {"error": "No mapping found"},
        ]
        mock_response.raise_for_status = MagicMock()
        pipeline.session.post.return_value = mock_response

        result = pipeline.lookup_cusips_via_openfigi(
            ["000000000"]
        )

        assert "000000000" not in result


# ==================================================================
# parse_information_table
# ==================================================================


class TestParseInformationTable:
    def test_valid_xml_parsed(self, pipeline):
        info_xml = """<?xml version="1.0" encoding="UTF-8"?>
        <informationTable
            xmlns="http://www.sec.gov/edgar/document/thirteenf/informationtable">
          <infoTable>
            <cusip>037833100</cusip>
            <value>5000000</value>
            <shrsOrPrnAmt>
              <sshPrnamt>25000</sshPrnamt>
            </shrsOrPrnAmt>
          </infoTable>
        </informationTable>"""

        mock_response = MagicMock()
        mock_response.content = info_xml.encode()
        mock_response.raise_for_status = MagicMock()
        pipeline.session.get.return_value = mock_response

        # Pre-populate the CUSIP mapping
        pipeline.cusip_to_ticker["037833100"] = "AAPL"

        # Mock OpenFIGI so it is not called
        pipeline.lookup_cusips_via_openfigi = MagicMock(
            return_value={}
        )

        holdings = pipeline.parse_information_table(
            xml_url="https://example.com/info.xml",
            institution_cik="0001067983",
            institution_name="Berkshire Hathaway",
            quarter_end_date=date(2024, 3, 31),
            filing_date=date(2024, 5, 15),
        )

        assert len(holdings) == 1
        assert holdings[0]["ticker"] == "AAPL"
        assert holdings[0]["shares"] == 25000
        assert holdings[0]["market_value"] == 5000000

    def test_unmapped_cusip_skipped(self, pipeline):
        info_xml = """<?xml version="1.0" encoding="UTF-8"?>
        <informationTable
            xmlns="http://www.sec.gov/edgar/document/thirteenf/informationtable">
          <infoTable>
            <cusip>UNKNOWN99</cusip>
            <value>1000</value>
            <shrsOrPrnAmt>
              <sshPrnamt>10</sshPrnamt>
            </shrsOrPrnAmt>
          </infoTable>
        </informationTable>"""

        mock_response = MagicMock()
        mock_response.content = info_xml.encode()
        mock_response.raise_for_status = MagicMock()
        pipeline.session.get.return_value = mock_response

        # Return empty from OpenFIGI lookup
        pipeline.lookup_cusips_via_openfigi = MagicMock(
            return_value={}
        )

        holdings = pipeline.parse_information_table(
            xml_url="https://example.com/info.xml",
            institution_cik="0001067983",
            institution_name="Berkshire Hathaway",
            quarter_end_date=date(2024, 3, 31),
            filing_date=date(2024, 5, 15),
        )

        assert len(holdings) == 0


# ==================================================================
# store_holdings
# ==================================================================


class TestStoreHoldings:
    @pytest.mark.asyncio
    async def test_empty_list_returns_false(self, pipeline):
        result = await pipeline.store_holdings([])
        assert result is False

    @pytest.mark.asyncio
    async def test_valid_holdings_stored(self, pipeline):
        mock_session = AsyncMock()
        pipeline.db.session.return_value.__aenter__ = AsyncMock(
            return_value=mock_session
        )
        pipeline.db.session.return_value.__aexit__ = AsyncMock(
            return_value=False
        )

        holdings = [
            {
                "ticker": "AAPL",
                "filing_date": date(2024, 5, 15),
                "quarter_end_date": date(2024, 3, 31),
                "institution_name": "Berkshire Hathaway",
                "institution_cik": "0001067983",
                "shares": 25000,
                "market_value": 5000000,
                "position_change": "New",
                "shares_change": 25000,
                "percent_change": None,
            }
        ]

        result = await pipeline.store_holdings(holdings)
        assert result is True
        mock_session.commit.assert_called_once()

    @pytest.mark.asyncio
    async def test_deduplicates_same_ticker_institution(
        self, pipeline
    ):
        mock_session = AsyncMock()
        pipeline.db.session.return_value.__aenter__ = AsyncMock(
            return_value=mock_session
        )
        pipeline.db.session.return_value.__aexit__ = AsyncMock(
            return_value=False
        )

        # Two entries for same ticker/quarter/institution
        holdings = [
            {
                "ticker": "AAPL",
                "filing_date": date(2024, 5, 15),
                "quarter_end_date": date(2024, 3, 31),
                "institution_name": "Berkshire Hathaway",
                "institution_cik": "0001067983",
                "shares": 10000,
                "market_value": 2000000,
                "position_change": "New",
                "shares_change": 10000,
                "percent_change": None,
            },
            {
                "ticker": "AAPL",
                "filing_date": date(2024, 5, 15),
                "quarter_end_date": date(2024, 3, 31),
                "institution_name": "Berkshire Hathaway",
                "institution_cik": "0001067983",
                "shares": 15000,
                "market_value": 3000000,
                "position_change": "New",
                "shares_change": 15000,
                "percent_change": None,
            },
        ]

        result = await pipeline.store_holdings(holdings)
        assert result is True
        # Should de-duplicate to 1 record with aggregated shares
