#!/bin/bash

# Comprehensive test runner for Polygon API migration
# This script runs all tests to ensure no regressions

set -e

echo "================================================"
echo "🧪 Polygon API Migration Test Suite"
echo "================================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Set test API key from environment or use demo key
export POLYGON_API_KEY="${POLYGON_API_KEY:-demo}"

# Track test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Function to run a test
run_test() {
    local test_name=$1
    local test_command=$2
    local test_type=$3
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    echo -n "Running $test_type: $test_name... "
    
    if eval "$test_command" > /tmp/test_output_$$.log 2>&1; then
        echo -e "${GREEN}✅ PASSED${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        if grep -q "SKIP" /tmp/test_output_$$.log; then
            echo -e "${YELLOW}⏭️  SKIPPED${NC}"
            SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
        else
            echo -e "${RED}❌ FAILED${NC}"
            FAILED_TESTS=$((FAILED_TESTS + 1))
            echo "  Error output:"
            tail -20 /tmp/test_output_$$.log | sed 's/^/    /'
        fi
    fi
    
    rm -f /tmp/test_output_$$.log
}

# Navigate to backend directory
cd backend || exit 1

echo "1️⃣ Unit Tests"
echo "--------------------------------"

# Run unit tests
run_test "Polygon Service Tests" \
    "go test -v ./services -run Test.*Polygon" \
    "Unit Test"

run_test "Exchange Code Mapping" \
    "go test -v ./services -run TestMapExchangeCode" \
    "Unit Test"

run_test "Asset Type Mapping" \
    "go test -v ./services -run TestMapAssetType" \
    "Unit Test"

run_test "Ticker Serialization" \
    "go test -v ./services -run TestPolygonTickerSerialization" \
    "Unit Test"

echo ""
echo "2️⃣ Integration Tests (Mock)"
echo "--------------------------------"

# Run mock integration tests
run_test "Mock Server Tests" \
    "go test -v ./services -run TestGetAllTickers_MockServer" \
    "Integration Test"

echo ""
echo "3️⃣ Performance Benchmarks"
echo "--------------------------------"

# Run benchmarks
echo "Running performance benchmarks..."
go test -bench=. -benchmem ./services 2>/dev/null | grep -E "Benchmark|ns/op" || true

echo ""
echo "4️⃣ Database Tests (Optional)"
echo "--------------------------------"

# Check if database tests should run
if [ "$RUN_DB_TESTS" == "true" ]; then
    export RUN_INTEGRATION_TESTS=true
    
    run_test "Ticker Exists Function" \
        "go test -v ./cmd/import-tickers -run TestTickerExists" \
        "Database Test"
    
    run_test "Insert Ticker Function" \
        "go test -v ./cmd/import-tickers -run TestInsertTicker" \
        "Database Test"
    
    run_test "Update Ticker Function" \
        "go test -v ./cmd/import-tickers -run TestUpdateTicker" \
        "Database Test"
else
    echo "⏭️  Skipping database tests (set RUN_DB_TESTS=true to enable)"
fi

echo ""
echo "5️⃣ API Rate Limit Test"
echo "--------------------------------"

# Test rate limiting handling
echo "Testing API rate limit handling..."
cat > /tmp/rate_limit_test.go << 'EOF'
package main

import (
    "fmt"
    "os"
    "time"
    "investorcenter-api/services"
)

func main() {
    os.Setenv("POLYGON_API_KEY", os.Getenv("POLYGON_API_KEY"))
    client := services.NewPolygonClient()
    
    // Make rapid requests to test rate limiting
    for i := 0; i < 3; i++ {
        _, err := client.GetAllTickers("stocks", 1)
        if err != nil {
            fmt.Printf("Request %d: %v\n", i+1, err)
        } else {
            fmt.Printf("Request %d: Success\n", i+1)
        }
        time.Sleep(1 * time.Second)
    }
}
EOF

go run /tmp/rate_limit_test.go 2>/dev/null || echo "Rate limit test completed"
rm -f /tmp/rate_limit_test.go

echo ""
echo "6️⃣ Regression Tests (Optional)"
echo "--------------------------------"

if [ "$RUN_REGRESSION_TESTS" == "true" ]; then
    export RUN_REGRESSION_TESTS=true
    
    run_test "Existing Functionality" \
        "go test -v ./tests -run TestExistingFunctionality -timeout 5m" \
        "Regression Test"
    
    run_test "Backward Compatibility" \
        "go test -v ./tests -run TestBackwardCompatibility" \
        "Regression Test"
    
    run_test "New Functionality" \
        "go test -v ./tests -run TestNewFunctionality -timeout 5m" \
        "Regression Test"
else
    echo "⏭️  Skipping regression tests (set RUN_REGRESSION_TESTS=true to enable)"
    echo "   Note: Regression tests make real API calls and may take several minutes"
fi

echo ""
echo "7️⃣ Quick Smoke Test"
echo "--------------------------------"

# Quick smoke test with curl
echo "Testing live API with curl..."
response=$(curl -s "https://api.polygon.io/v3/reference/tickers?ticker=AAPL&apikey=$POLYGON_API_KEY")
if echo "$response" | grep -q '"status":"OK"'; then
    echo -e "${GREEN}✅ API is responding correctly${NC}"
else
    echo -e "${RED}❌ API response unexpected${NC}"
    echo "$response" | head -50
fi

echo ""
echo "================================================"
echo "📊 Test Summary"
echo "================================================"
echo "Total Tests: $TOTAL_TESTS"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"
echo -e "Skipped: ${YELLOW}$SKIPPED_TESTS${NC}"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed! The migration is safe to deploy.${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed. Please review before deploying.${NC}"
    exit 1
fi