#!/usr/bin/env python3
"""
Helper functions for parsing YCharts data formats.
"""

import re
from datetime import datetime
from typing import Optional, Union


def parse_dollar_amount(value: str) -> Optional[int]:
    """
    Parse YCharts dollar amounts to integer (in cents).
    
    Examples:
        "187.14B" -> 187140000000
        "99.20B" -> 99200000000
        "3.979M" -> 3979000
        "-28.56B" -> -28560000000
        "11.33B" -> 11330000000
    
    Returns None if unable to parse.
    """
    if not value or value == "--":
        return None
    
    # Remove commas and whitespace
    value = value.replace(",", "").strip()
    
    # Extract number and suffix
    match = re.match(r'^(-?[\d.]+)([BKMGT])?$', value, re.IGNORECASE)
    if not match:
        return None
    
    number_str, suffix = match.groups()
    
    try:
        number = float(number_str)
    except ValueError:
        return None
    
    # Apply multiplier based on suffix
    multipliers = {
        'T': 1_000_000_000_000,
        'B': 1_000_000_000,
        'M': 1_000_000,
        'K': 1_000,
        None: 1
    }
    
    multiplier = multipliers.get(suffix.upper() if suffix else None, 1)
    return int(number * multiplier)


def parse_percentage(value: str) -> Optional[float]:
    """
    Parse YCharts percentages to decimal.
    
    Examples:
        "62.49%" -> 0.6249
        "0.02%" -> 0.0002
        "-1.60%" -> -0.0160
    
    Returns None if unable to parse.
    """
    if not value or value == "--":
        return None
    
    # Remove % sign and whitespace
    value = value.replace("%", "").strip()
    
    try:
        return float(value) / 100
    except ValueError:
        return None


def parse_float(value: str) -> Optional[float]:
    """
    Parse plain float values.
    
    Examples:
        "4.038" -> 4.038
        "186.96" -> 186.96
        "2.313" -> 2.313
    
    Returns None if unable to parse.
    """
    if not value or value == "--":
        return None
    
    # Remove commas
    value = value.replace(",", "").strip()
    
    try:
        return float(value)
    except ValueError:
        return None


def parse_integer(value: str) -> Optional[int]:
    """
    Parse plain integer values.
    
    Examples:
        "36000" -> 36000
        "1,234,567" -> 1234567
    
    Returns None if unable to parse.
    """
    if not value or value == "--":
        return None
    
    # Remove commas
    value = value.replace(",", "").strip()
    
    try:
        return int(float(value))  # Use float() to handle "36000.0"
    except ValueError:
        return None


def parse_ycharts_date(value: str) -> Optional[str]:
    """
    Parse YCharts date formats to ISO 8601 (YYYY-MM-DD).
    
    Examples:
        "Oct. 29, 2025" -> "2025-10-29"
        "Dec. 04, 2025" -> "2025-12-04"
        "Apr. 07, 2025" -> "2025-04-07"
    
    Returns None if unable to parse.
    """
    if not value or value == "--":
        return None
    
    # Replace abbreviated month names with full names
    value = value.replace(".", "").strip()
    
    try:
        # Try parsing "Oct 29, 2025" format
        dt = datetime.strptime(value, "%b %d, %Y")
        return dt.strftime("%Y-%m-%d")
    except ValueError:
        pass
    
    try:
        # Try parsing "October 29, 2025" format
        dt = datetime.strptime(value, "%B %d, %Y")
        return dt.strftime("%Y-%m-%d")
    except ValueError:
        return None


def parse_ycharts_timestamp(value: str) -> Optional[str]:
    """
    Parse YCharts timestamp formats to ISO 8601.
    
    Examples:
        "USD | NASDAQ | Feb 12, 16:00" -> extract timestamp
    
    Returns None if unable to parse.
    """
    if not value:
        return None
    
    # Extract date and time parts
    parts = [p.strip() for p in value.split("|")]
    
    if len(parts) >= 3:
        date_time_str = parts[2]  # "Feb 12, 16:00"
        
        try:
            # Try parsing
            dt = datetime.strptime(date_time_str, "%b %d, %H:%M")
            # Assume current year if not specified
            dt = dt.replace(year=datetime.now().year)
            return dt.isoformat() + "Z"
        except ValueError:
            pass
    
    return None


# Example usage and tests
if __name__ == "__main__":
    # Test dollar amounts
    assert parse_dollar_amount("187.14B") == 187140000000
    assert parse_dollar_amount("99.20B") == 99200000000
    assert parse_dollar_amount("3.979M") == 3979000
    assert parse_dollar_amount("-28.56B") == -28560000000
    assert parse_dollar_amount("11.33B") == 11330000000
    assert parse_dollar_amount("--") is None
    
    # Test percentages
    assert parse_percentage("62.49%") == 0.6249
    assert parse_percentage("0.02%") == 0.0002
    assert parse_percentage("-1.60%") == -0.0160
    assert parse_percentage("--") is None
    
    # Test floats
    assert parse_float("4.038") == 4.038
    assert parse_float("186.96") == 186.96
    assert parse_float("2.313") == 2.313
    assert parse_float("--") is None
    
    # Test integers
    assert parse_integer("36000") == 36000
    assert parse_integer("1,234,567") == 1234567
    assert parse_integer("--") is None
    
    # Test dates
    assert parse_ycharts_date("Oct. 29, 2025") == "2025-10-29"
    assert parse_ycharts_date("Dec. 04, 2025") == "2025-12-04"
    assert parse_ycharts_date("Apr. 07, 2025") == "2025-04-07"
    assert parse_ycharts_date("--") is None
    
    print("✅ All tests passed!")
