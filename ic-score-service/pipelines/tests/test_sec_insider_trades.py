"""Tests for the SEC insider trades ingestion pipeline.

Validates Form 4 RSS fetching, XML parsing, transaction type
determination, and database storage with mocked requests and
get_database.
"""

from datetime import date
from decimal import Decimal
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

# Patch get_database before importing the pipeline module.
with patch("pipelines.sec_insider_trades_ingestion.get_database"):
    from pipelines.sec_insider_trades_ingestion import (
        InsiderTradesIngestion,
    )


@pytest.fixture
def pipeline():
    """Create an InsiderTradesIngestion with mocked DB and requests."""
    with patch(
        "pipelines.sec_insider_trades_ingestion.get_database"
    ) as mock_db:
        db_instance = MagicMock()
        mock_db.return_value = db_instance
        p = InsiderTradesIngestion()
        # Mock the requests session to prevent real HTTP calls
        p.session = MagicMock()
        yield p


def _make_form4_xml(
    ticker="AAPL",
    owner_name="Tim Cook",
    is_officer=True,
    officer_title="CEO",
    trans_code="S",
    shares="50000",
    price="185.00",
    acq_disp="D",
    shares_after="100000",
    trans_date="2024-06-15",
):
    """Build a Form 4 XML string."""
    is_officer_val = "1" if is_officer else "0"
    return f"""<?xml version="1.0" encoding="UTF-8"?>
    <ownershipDocument>
      <issuer>
        <issuerTradingSymbol>{ticker}</issuerTradingSymbol>
      </issuer>
      <reportingOwner>
        <reportingOwnerId>
          <rptOwnerName>{owner_name}</rptOwnerName>
        </reportingOwnerId>
        <reportingOwnerRelationship>
          <isOfficer>{is_officer_val}</isOfficer>
          <officerTitle>{officer_title}</officerTitle>
        </reportingOwnerRelationship>
      </reportingOwner>
      <nonDerivativeTable>
        <nonDerivativeTransaction>
          <transactionDate>
            <value>{trans_date}</value>
          </transactionDate>
          <transactionCoding>
            <transactionCode>{trans_code}</transactionCode>
          </transactionCoding>
          <transactionAmounts>
            <transactionShares>
              <value>{shares}</value>
            </transactionShares>
            <transactionPricePerShare>
              <value>{price}</value>
            </transactionPricePerShare>
            <transactionAcquiredDisposedCode>
              <value>{acq_disp}</value>
            </transactionAcquiredDisposedCode>
          </transactionAmounts>
          <postTransactionAmounts>
            <sharesOwnedFollowingTransaction>
              <value>{shares_after}</value>
            </sharesOwnedFollowingTransaction>
          </postTransactionAmounts>
        </nonDerivativeTransaction>
      </nonDerivativeTable>
    </ownershipDocument>"""


# ==================================================================
# Configuration
# ==================================================================


class TestConfiguration:
    def test_rss_url_set(self):
        assert "sec.gov" in InsiderTradesIngestion.RSS_URL

    def test_user_agent_set(self):
        assert "InvestorCenter" in InsiderTradesIngestion.USER_AGENT

    def test_rate_limit_interval_positive(self):
        assert InsiderTradesIngestion.MIN_REQUEST_INTERVAL > 0


# ==================================================================
# Rate limiting
# ==================================================================


class TestRateLimiting:
    def test_rate_limit_updates_timestamp(self, pipeline):
        pipeline.last_request_time = 0.0
        pipeline._rate_limit()
        assert pipeline.last_request_time > 0.0


# ==================================================================
# parse_form4_xml
# ==================================================================


class TestParseForm4Xml:
    def test_sale_transaction(self, pipeline):
        xml = _make_form4_xml(
            trans_code="S", acq_disp="D", shares="50000"
        )
        result = pipeline.parse_form4_xml(
            xml, "https://sec.gov/test"
        )

        assert len(result) == 1
        tx = result[0]
        assert tx["ticker"] == "AAPL"
        assert tx["insider_name"] == "Tim Cook"
        assert tx["transaction_type"] == "Sale"
        assert tx["shares"] < 0  # Disposed shares are negative
        assert abs(tx["shares"]) == 50000

    def test_purchase_transaction(self, pipeline):
        xml = _make_form4_xml(
            trans_code="P", acq_disp="A", shares="10000"
        )
        result = pipeline.parse_form4_xml(
            xml, "https://sec.gov/test"
        )

        assert len(result) == 1
        tx = result[0]
        assert tx["transaction_type"] == "Purchase"
        assert tx["shares"] > 0
        assert tx["shares"] == 10000

    def test_price_per_share_parsed(self, pipeline):
        xml = _make_form4_xml(price="185.50")
        result = pipeline.parse_form4_xml(
            xml, "https://sec.gov/test"
        )

        assert len(result) == 1
        assert result[0]["price_per_share"] == Decimal("185.50")

    def test_total_value_calculated(self, pipeline):
        xml = _make_form4_xml(
            shares="100", price="200.00", acq_disp="D"
        )
        result = pipeline.parse_form4_xml(
            xml, "https://sec.gov/test"
        )

        assert len(result) == 1
        # total_value = abs(price * shares) = 200 * 100 = 20000
        assert result[0]["total_value"] == 20000

    def test_shares_owned_after(self, pipeline):
        xml = _make_form4_xml(shares_after="250000")
        result = pipeline.parse_form4_xml(
            xml, "https://sec.gov/test"
        )

        assert result[0]["shares_owned_after"] == 250000

    def test_missing_ticker_returns_empty(self, pipeline):
        xml = """<?xml version="1.0" encoding="UTF-8"?>
        <ownershipDocument>
          <issuer>
            <issuerTradingSymbol></issuerTradingSymbol>
          </issuer>
          <reportingOwner>
            <reportingOwnerId>
              <rptOwnerName>Test</rptOwnerName>
            </reportingOwnerId>
          </reportingOwner>
        </ownershipDocument>"""

        result = pipeline.parse_form4_xml(
            xml, "https://sec.gov/test"
        )
        assert result == []

    def test_missing_owner_returns_empty(self, pipeline):
        xml = """<?xml version="1.0" encoding="UTF-8"?>
        <ownershipDocument>
          <issuer>
            <issuerTradingSymbol>AAPL</issuerTradingSymbol>
          </issuer>
          <reportingOwner>
            <reportingOwnerId>
              <rptOwnerName></rptOwnerName>
            </reportingOwnerId>
          </reportingOwner>
        </ownershipDocument>"""

        result = pipeline.parse_form4_xml(
            xml, "https://sec.gov/test"
        )
        assert result == []

    def test_zero_shares_transaction_skipped(self, pipeline):
        xml = _make_form4_xml(shares="0")
        result = pipeline.parse_form4_xml(
            xml, "https://sec.gov/test"
        )
        assert len(result) == 0


# ==================================================================
# fetch_form4_filings (RSS)
# ==================================================================


class TestFetchForm4Filings:
    def test_empty_feed(self, pipeline):
        atom_xml = """<?xml version="1.0" encoding="UTF-8"?>
        <feed xmlns="http://www.w3.org/2005/Atom">
        </feed>"""

        mock_response = MagicMock()
        mock_response.content = atom_xml.encode()
        mock_response.raise_for_status = MagicMock()
        pipeline.session.get.return_value = mock_response

        result = pipeline.fetch_form4_filings(hours_back=24)
        assert result == []

    def test_request_error_returns_empty(self, pipeline):
        pipeline.session.get.side_effect = Exception(
            "Connection error"
        )
        result = pipeline.fetch_form4_filings(hours_back=24)
        assert result == []


# ==================================================================
# store_insider_trades
# ==================================================================


class TestStoreInsiderTrades:
    @pytest.mark.asyncio
    async def test_empty_list_returns_zero(self, pipeline):
        result = await pipeline.store_insider_trades([])
        assert result == 0

    @pytest.mark.asyncio
    async def test_valid_trades_stored(self, pipeline):
        mock_session = AsyncMock()
        mock_result = MagicMock()
        mock_result.rowcount = 1
        mock_session.execute.return_value = mock_result
        pipeline.db.session.return_value.__aenter__ = AsyncMock(
            return_value=mock_session
        )
        pipeline.db.session.return_value.__aexit__ = AsyncMock(
            return_value=False
        )

        trades = [
            {
                "ticker": "AAPL",
                "filing_date": date.today(),
                "transaction_date": date(2024, 6, 15),
                "insider_name": "Tim Cook",
                "insider_title": "CEO",
                "transaction_type": "Sale",
                "shares": -50000,
                "price_per_share": Decimal("185.00"),
                "total_value": 9250000,
                "shares_owned_after": 100000,
                "is_derivative": False,
                "form_type": "4",
                "sec_filing_url": "https://sec.gov/test",
            }
        ]

        result = await pipeline.store_insider_trades(trades)
        assert result == 1
        mock_session.commit.assert_called_once()
