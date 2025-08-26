#!/usr/bin/env python3
"""Simple test script to verify package installation and functionality."""

import sys
import os

# Add the current directory to Python path for testing
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

def test_import():
    """Test that the package can be imported."""
    try:
        import us_tickers
        print("✓ Package imported successfully")
        print(f"  Version: {us_tickers.__version__}")
        return True
    except ImportError as e:
        print(f"✗ Failed to import package: {e}")
        return False

def test_function_import():
    """Test that the main function can be imported."""
    try:
        from us_tickers import get_exchange_listed_tickers
        print("✓ Main function imported successfully")
        return True
    except ImportError as e:
        print(f"✗ Failed to import main function: {e}")
        return False

def test_cli_import():
    """Test that the CLI module can be imported."""
    try:
        import us_tickers.cli
        print("✓ CLI module imported successfully")
        return True
    except ImportError as e:
        print(f"✗ Failed to import CLI module: {e}")
        return False

def test_cache_import():
    """Test that the cache module can be imported."""
    try:
        import us_tickers.cache
        print("✓ Cache module imported successfully")
        return True
    except ImportError as e:
        print(f"✗ Failed to import cache module: {e}")
        return False

def test_constants():
    """Test that package constants are accessible."""
    try:
        from us_tickers.fetch import EXCHANGE_CODES
        print("✓ Exchange codes accessible")
        print(f"  Available exchanges: {list(EXCHANGE_CODES.keys())}")
        return True
    except Exception as e:
        print(f"✗ Failed to access constants: {e}")
        return False

def test_cli_help():
    """Test that CLI help works."""
    try:
        import subprocess
        result = subprocess.run(
            [sys.executable, "-m", "us_tickers.cli", "--help"],
            capture_output=True,
            text=True,
            timeout=10
        )
        if result.returncode == 0:
            print("✓ CLI help works")
            return True
        else:
            print(f"✗ CLI help failed: {result.stderr}")
            return False
    except Exception as e:
        print(f"✗ CLI help test failed: {e}")
        return False

def main():
    """Run all tests."""
    print("US Tickers Package Test")
    print("=" * 30)
    print()
    
    tests = [
        test_import,
        test_function_import,
        test_cli_import,
        test_cache_import,
        test_constants,
        test_cli_help,
    ]
    
    passed = 0
    total = len(tests)
    
    for test in tests:
        try:
            if test():
                passed += 1
        except Exception as e:
            print(f"✗ Test {test.__name__} failed with exception: {e}")
        print()
    
    print(f"Results: {passed}/{total} tests passed")
    
    if passed == total:
        print("🎉 All tests passed! Package is working correctly.")
        return 0
    else:
        print("❌ Some tests failed. Please check the errors above.")
        return 1

if __name__ == "__main__":
    sys.exit(main())
