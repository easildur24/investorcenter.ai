# Polygon API Migration - Testing Guide 🧪

## Overview
This guide covers all tests created to ensure the Polygon API migration doesn't cause any regressions and maintains backward compatibility.

## Test Files Created

### 1. **Unit Tests** (`backend/services/polygon_test.go`)
Tests core Polygon service functionality:
- Client creation with/without API key
- Exchange code mapping (XNAS → NASDAQ)
- Asset type mapping (CS → stock, ETF → etf)
- Ticker serialization/deserialization
- Mock server integration
- Real API integration (optional)

### 2. **Integration Tests** (`backend/cmd/import-tickers/main_test.go`)
Tests database operations:
- `tickerExists()` - Check if ticker exists in DB
- `insertTicker()` - Insert new tickers
- `updateTicker()` - Update existing tickers
- `shouldUpdate()` - Logic for update decisions
- `mapSICToSector()` - SIC code to sector mapping

### 3. **Regression Tests** (`backend/tests/regression_test.go`)
Ensures no breaking changes:
- Existing functionality still works (GetQuote, GetHistoricalData, etc.)
- Backward compatibility maintained
- New functionality works correctly
- Data integrity for popular tickers (AAPL, SPY, etc.)
- Performance hasn't degraded

### 4. **Test Runner** (`scripts/run_polygon_tests.sh`)
Automated test execution script that runs all tests and provides a summary.

## Running Tests

### Quick Test (No API Calls)
```bash
# Run unit tests only
cd backend
go test ./services -v

# Run with coverage
go test ./services -cover
```

### Full Test Suite
```bash
# Run all tests (includes API calls)
./scripts/run_polygon_tests.sh

# With database tests
RUN_DB_TESTS=true ./scripts/run_polygon_tests.sh

# With regression tests (makes many API calls)
RUN_REGRESSION_TESTS=true ./scripts/run_polygon_tests.sh
```

### Individual Test Categories

#### Unit Tests
```bash
cd backend

# Test Polygon client creation
go test -v ./services -run TestNewPolygonClient

# Test mapping functions
go test -v ./services -run TestMapExchangeCode
go test -v ./services -run TestMapAssetType

# Test with mock server
go test -v ./services -run TestGetAllTickers_MockServer
```

#### Integration Tests (Database)
```bash
# Requires database connection
export RUN_INTEGRATION_TESTS=true
export TEST_DB_PASSWORD=your_password

cd backend
go test -v ./cmd/import-tickers/
```

#### Performance Benchmarks
```bash
cd backend

# Run all benchmarks
go test -bench=. ./services

# Run specific benchmark
go test -bench=BenchmarkMapExchangeCode ./services

# With memory allocation stats
go test -bench=. -benchmem ./services
```

#### Regression Tests
```bash
# These make real API calls - use sparingly
export RUN_REGRESSION_TESTS=true
export POLYGON_API_KEY=zapuIgaTVLJoanfEuimZYQ2xRlZmoU1m

cd backend
go test -v ./tests -timeout 10m
```

## Test Coverage Areas

### ✅ What's Tested

1. **Core Functionality**
   - Polygon client initialization
   - API key management
   - HTTP client configuration

2. **New Features**
   - `GetAllTickers()` with pagination
   - `GetTickersByType()` filtering
   - Exchange code mapping
   - Asset type classification

3. **Backward Compatibility**
   - Existing methods still work
   - Struct definitions unchanged
   - API responses parse correctly

4. **Data Integrity**
   - Stocks identified as type "CS"
   - ETFs identified as type "ETF"
   - Popular tickers (AAPL, SPY) work correctly

5. **Performance**
   - Mapping functions are fast (<1ms for 10k calls)
   - No memory leaks
   - Efficient pagination handling

6. **Error Handling**
   - Rate limit detection
   - Invalid ticker handling
   - Network error recovery

### 🔄 Test Data

The tests use a mix of:
- **Mock data** for unit tests
- **Real API calls** with test API key (zapuIgaTVLJoanfEuimZYQ2xRlZmoU1m)
- **Test database** for integration tests

## CI/CD Integration

### GitHub Actions
Add to `.github/workflows/test.yml`:
```yaml
- name: Run Polygon Tests
  env:
    POLYGON_API_KEY: ${{ secrets.POLYGON_API_KEY }}
    SKIP_INTEGRATION_TESTS: true
  run: |
    cd backend
    go test ./services -v
    go test -bench=. ./services
```

### Pre-commit Hook
Add to `.pre-commit-config.yaml`:
```yaml
- repo: local
  hooks:
    - id: polygon-tests
      name: Polygon API Tests
      entry: cd backend && go test ./services
      language: system
      pass_filenames: false
      files: ^backend/services/polygon
```

## Test Results Interpretation

### Success Indicators
- ✅ All unit tests pass
- ✅ Mapping functions work correctly
- ✅ ETFs properly identified
- ✅ No performance regression
- ✅ API responds with valid data

### Warning Signs
- ⚠️ Rate limit errors (normal with free tier)
- ⚠️ Some tickers not found (data availability)
- ⚠️ Slow API responses (network dependent)

### Failure Indicators
- ❌ Mapping functions return wrong values
- ❌ ETFs classified as stocks
- ❌ Database operations fail
- ❌ Existing methods break
- ❌ Performance degradation >10x

## Troubleshooting

### Common Issues

1. **Rate Limit Errors**
   ```
   Error: You've exceeded the maximum requests per minute
   ```
   Solution: Wait 60 seconds between test runs or upgrade API plan

2. **Database Connection Failed**
   ```
   Failed to connect to test database
   ```
   Solution: Check database credentials and ensure PostgreSQL is running

3. **API Key Invalid**
   ```
   Error: Unknown API Key
   ```
   Solution: Verify POLYGON_API_KEY environment variable is set correctly

4. **Test Timeout**
   ```
   panic: test timed out after 10m0s
   ```
   Solution: Increase timeout or skip regression tests

## Performance Benchmarks

Expected performance metrics:

| Function | Operations/sec | ns/op | Memory/op |
|----------|---------------|-------|-----------|
| MapExchangeCode | >10,000,000 | <100 | 0 B |
| MapAssetType | >10,000,000 | <100 | 0 B |
| GetAllTickers | 5-10 | ~200ms | <10 KB |

## Monitoring Production

After deployment, monitor:

1. **API Usage**
   - Request count vs rate limits
   - Error rates
   - Response times

2. **Data Quality**
   - ETF classification accuracy
   - Missing tickers
   - Stale data

3. **Performance**
   - Import duration
   - Query response times
   - Database size growth

## Test Maintenance

### Regular Updates Needed
- Update test API key if it expires
- Add tests for new asset types
- Update mock data periodically
- Review performance benchmarks

### When to Run Full Tests
- Before major deployments
- After Polygon API updates
- When adding new features
- During debugging sessions

## Summary

The test suite ensures:
1. ✅ **No regressions** - Existing functionality preserved
2. ✅ **ETF support works** - Proper type identification
3. ✅ **Performance maintained** - No slowdowns
4. ✅ **Data integrity** - Correct asset classification
5. ✅ **Error handling** - Graceful failure modes

Run `./scripts/run_polygon_tests.sh` before any deployment to verify everything works correctly!