"""Tests for SIC-to-sector mapping and industry name formatting."""

import pytest

from scripts.backfill_sectors_from_sec import (
    format_industry,
    get_sector_from_sic,
)


# ---------------------------------------------------------------------------
# get_sector_from_sic
# ---------------------------------------------------------------------------


class TestGetSectorFromSIC:
    """Test SIC code to GICS sector mapping."""

    def test_4digit_match_takes_priority(self):
        # SIC 3674 -> Technology (4-digit map)
        assert get_sector_from_sic("3674") == "Technology"

    def test_2digit_fallback(self):
        # SIC 2011 not in 4-digit map -> "20" -> Consumer Defensive
        assert get_sector_from_sic("2011") == "Consumer Defensive"

    def test_known_tech_codes(self):
        assert get_sector_from_sic("7372") == "Technology"
        assert get_sector_from_sic("3571") == "Technology"

    def test_known_healthcare_codes(self):
        assert get_sector_from_sic("2834") == "Healthcare"
        assert get_sector_from_sic("8062") == "Healthcare"

    def test_known_financial_codes(self):
        assert get_sector_from_sic("6020") == "Financial Services"
        assert get_sector_from_sic("6199") == "Financial Services"

    def test_known_energy_codes(self):
        assert get_sector_from_sic("1311") == "Energy"
        assert get_sector_from_sic("2911") == "Energy"

    def test_real_estate(self):
        assert get_sector_from_sic("6512") == "Real Estate"

    def test_communication_services(self):
        assert get_sector_from_sic("4813") == "Communication Services"
        assert get_sector_from_sic("4841") == "Communication Services"

    def test_utilities(self):
        assert get_sector_from_sic("4911") == "Utilities"

    def test_empty_returns_empty(self):
        assert get_sector_from_sic("") == ""
        assert get_sector_from_sic(None) == ""

    def test_unmapped_code_returns_empty(self):
        assert get_sector_from_sic("0000") == ""

    def test_whitespace_stripped(self):
        assert get_sector_from_sic(" 7372 ") == "Technology"


# ---------------------------------------------------------------------------
# format_industry
# ---------------------------------------------------------------------------


class TestFormatIndustry:
    """Test SEC sicDescription -> industry name formatting."""

    def test_basic_title_case(self):
        assert format_industry("ELECTRONIC COMPUTERS") == (
            "Electronic Computers"
        )

    def test_pharmaceutical(self):
        assert format_industry("PHARMACEUTICAL PREPARATIONS") == (
            "Pharmaceutical Preparations"
        )

    def test_prefix_stripping_services(self):
        result = format_industry(
            "SERVICES-COMPUTER PROGRAMMING, DATA PROCESSING"
        )
        assert result == "Computer Programming, Data Processing"

    def test_prefix_stripping_retail(self):
        assert format_industry("RETAIL-EATING PLACES") == "Eating Places"

    def test_small_words_lowercased(self):
        result = format_industry("SURGICAL & MEDICAL INSTRUMENTS & APPARATUS")
        assert "& Medical" in result
        assert "& Apparatus" in result

    def test_small_word_of(self):
        result = format_industry("OFFICES OF DOCTORS OF MEDICINE")
        assert result == "Offices of Doctors of Medicine"

    def test_first_word_stays_capitalized(self):
        result = format_industry("IN VITRO & IN VIVO DIAGNOSTICS")
        # "In" at start stays capitalized
        assert result.startswith("In ")

    def test_empty_returns_empty(self):
        assert format_industry("") == ""
        assert format_industry(None) == ""

    def test_whitespace_only(self):
        assert format_industry("   ") == ""

    def test_ampersand_preserved(self):
        result = format_industry("SEMICONDUCTORS & RELATED DEVICES")
        assert "& Related" in result

    def test_no_prefix_no_stripping(self):
        assert format_industry("SEMICONDUCTORS") == "Semiconductors"

    def test_possessive_apostrophe(self):
        result = format_industry("WOMEN'S CLOTHING STORES")
        assert result == "Women's Clothing Stores"
