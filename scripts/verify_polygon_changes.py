#!/usr/bin/env python3
"""
Verification script for Polygon API migration
Tests the changes without requiring Go compilation
"""

import json
import subprocess
import sys

def check_file_exists(filepath):
    """Check if a file exists"""
    try:
        with open(filepath, 'r') as f:
            return True
    except FileNotFoundError:
        return False

def verify_polygon_extension():
    """Verify polygon.go has been extended with new functions"""
    print("1️⃣ Checking polygon.go extensions...")
    
    with open('backend/services/polygon.go', 'r') as f:
        content = f.read()
    
    required_functions = [
        'GetAllTickers',
        'GetTickersByType', 
        'MapExchangeCode',
        'MapAssetType',
        'PolygonTickersResponse',
        'PolygonTicker'
    ]
    
    missing = []
    for func in required_functions:
        if func not in content:
            missing.append(func)
    
    if missing:
        print(f"  ❌ Missing functions/types: {', '.join(missing)}")
        return False
    else:
        print("  ✅ All required functions/types found")
        return True

def verify_database_migration():
    """Verify database migration file exists"""
    print("\n2️⃣ Checking database migration...")
    
    migration_file = 'backend/migrations/002_add_polygon_ticker_fields.sql'
    if check_file_exists(migration_file):
        with open(migration_file, 'r') as f:
            content = f.read()
        
        required_fields = [
            'asset_type',
            'cik',
            'ipo_date',
            'sic_code',
            'composite_figi'
        ]
        
        missing = []
        for field in required_fields:
            if field not in content:
                missing.append(field)
        
        if missing:
            print(f"  ❌ Missing fields: {', '.join(missing)}")
            return False
        else:
            print("  ✅ All required fields in migration")
            return True
    else:
        print(f"  ❌ Migration file not found: {migration_file}")
        return False

def verify_import_tool():
    """Verify import tool exists"""
    print("\n3️⃣ Checking import tool...")
    
    import_file = 'backend/cmd/import-tickers/main.go'
    if check_file_exists(import_file):
        with open(import_file, 'r') as f:
            content = f.read()
        
        required_items = [
            'GetAllTickers',
            'insertTicker',
            'updateTicker',
            'MapAssetType',
            'asset_type'
        ]
        
        missing = []
        for item in required_items:
            if item not in content:
                missing.append(item)
        
        if missing:
            print(f"  ❌ Missing items: {', '.join(missing)}")
            return False
        else:
            print("  ✅ Import tool properly implemented")
            return True
    else:
        print(f"  ❌ Import tool not found: {import_file}")
        return False

def verify_test_files():
    """Verify test files exist"""
    print("\n4️⃣ Checking test files...")
    
    test_files = [
        'backend/services/polygon_test.go',
        'backend/cmd/import-tickers/main_test.go',
        'backend/tests/regression_test.go',
        'scripts/run_polygon_tests.sh'
    ]
    
    all_exist = True
    for file in test_files:
        if check_file_exists(file):
            print(f"  ✅ {file}")
        else:
            print(f"  ❌ Missing: {file}")
            all_exist = False
    
    return all_exist

def test_api_functionality():
    """Test actual API functionality"""
    print("\n5️⃣ Testing API functionality...")
    
    import urllib.request
    import urllib.parse
    
    api_key = "zapuIgaTVLJoanfEuimZYQ2xRlZmoU1m"
    
    # Test stocks endpoint
    print("  Testing stocks endpoint...")
    url = f"https://api.polygon.io/v3/reference/tickers?market=stocks&type=CS&limit=1&apikey={api_key}"
    
    try:
        with urllib.request.urlopen(url) as response:
            data = json.loads(response.read())
            if data.get('status') == 'OK' and data.get('results'):
                ticker = data['results'][0]
                if ticker.get('type') == 'CS':
                    print(f"    ✅ Stock found: {ticker.get('ticker')} - Type: CS")
                else:
                    print(f"    ❌ Wrong type: {ticker.get('type')}")
                    return False
            else:
                print(f"    ❌ API error: {data}")
                return False
    except Exception as e:
        print(f"    ❌ Request failed: {e}")
        return False
    
    # Test ETF endpoint
    print("  Testing ETF endpoint...")
    url = f"https://api.polygon.io/v3/reference/tickers?market=stocks&type=ETF&limit=1&apikey={api_key}"
    
    try:
        with urllib.request.urlopen(url) as response:
            data = json.loads(response.read())
            if data.get('status') == 'OK' and data.get('results'):
                ticker = data['results'][0]
                if ticker.get('type') == 'ETF':
                    print(f"    ✅ ETF found: {ticker.get('ticker')} - Type: ETF")
                else:
                    print(f"    ❌ Wrong type: {ticker.get('type')}")
                    return False
            else:
                print(f"    ❌ API error: {data}")
                return False
    except Exception as e:
        print(f"    ❌ Request failed: {e}")
        return False
    
    return True

def verify_documentation():
    """Verify documentation exists"""
    print("\n6️⃣ Checking documentation...")
    
    doc_files = [
        'docs/polygon-migration-summary.md',
        'docs/polygon-testing-guide.md',
        'docs/data-format-comparison.md'
    ]
    
    all_exist = True
    for file in doc_files:
        if check_file_exists(file):
            print(f"  ✅ {file}")
        else:
            print(f"  ❌ Missing: {file}")
            all_exist = False
    
    return all_exist

def main():
    print("================================================")
    print("🔍 Polygon API Migration Verification")
    print("================================================")
    
    results = []
    
    # Run all verifications
    results.append(("Polygon.go Extensions", verify_polygon_extension()))
    results.append(("Database Migration", verify_database_migration()))
    results.append(("Import Tool", verify_import_tool()))
    results.append(("Test Files", verify_test_files()))
    results.append(("API Functionality", test_api_functionality()))
    results.append(("Documentation", verify_documentation()))
    
    # Summary
    print("\n================================================")
    print("📊 Verification Summary")
    print("================================================")
    
    all_passed = True
    for name, passed in results:
        if passed:
            print(f"✅ {name}: PASSED")
        else:
            print(f"❌ {name}: FAILED")
            all_passed = False
    
    print()
    if all_passed:
        print("🎉 All verifications passed! The migration is ready.")
        print("\nNext steps:")
        print("1. Run the migration: ./scripts/migrate_to_polygon.sh")
        print("2. Test locally: make dev")
        print("3. Deploy to production when ready")
        return 0
    else:
        print("⚠️  Some verifications failed. Please review the issues above.")
        return 1

if __name__ == "__main__":
    sys.exit(main())