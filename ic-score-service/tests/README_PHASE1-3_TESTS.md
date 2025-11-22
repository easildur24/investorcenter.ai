# Financial Metrics Test Suite

Comprehensive test suite for Phases 1-3 of the financial metrics implementation.

## Overview

This test suite validates the correctness of:
- **Phase 1**: TTM Performance Ratios (ROA, ROE, ROIC, Margins)
- **Phase 2**: EBITDA and EV/EBITDA calculations
- **Phase 3**: Quarter-over-Quarter Growth Metrics

## Test Files

### 1. `test_ttm_performance_ratios.py`
Tests for TTM performance ratio calculations.

**Coverage**:
- ROA (Return on Assets) calculation
- ROE (Return on Equity) calculation
- ROIC (Return on Invested Capital) calculation
- Gross, Operating, and Net Margin calculations
- Edge cases: division by zero, negative values, missing data
- Real-world scenarios: Apple-like metrics, negative equity companies

**Key Test Cases**:
```python
# ROA Test
assert (1M / avg(10M, 8M)) * 100 ≈ 11.11%

# ROE Test
assert (1M / avg(5M, 4.5M)) * 100 ≈ 21.05%

# ROIC Test
assert (NOPAT / avg_invested_capital) * 100 ≈ 16.36%
```

### 2. `test_ebitda_calculations.py`
Tests for EBITDA and Enterprise Value calculations.

**Coverage**:
- EBITDA = Operating Income + D&A
- TTM EBITDA = Sum of 4 quarters
- Enterprise Value = Market Cap + Debt - Cash
- EV/EBITDA ratio calculations
- Negative EBITDA handling
- Comparative valuation scenarios

**Key Test Cases**:
```python
# EBITDA Test
assert (5M operating_income + 1M D&A) == 6M

# EV Test
assert (100M market_cap + 20M debt - 10M cash) == 110M

# EV/EBITDA Test
assert (110M EV / 11M EBITDA) ≈ 10.0
```

### 3. `test_quarterly_growth.py`
Tests for quarter-over-quarter growth metrics.

**Coverage**:
- QoQ growth formula: ((Current - Previous) / |Previous|) × 100
- Positive and negative growth
- Division by zero handling
- Negative-to-positive transitions (returns NULL)
- Both quarters negative (loss reduction)
- Seasonal business patterns

**Key Test Cases**:
```python
# Basic Growth
assert ((110M - 100M) / 100M) * 100 == 10.0%

# Decline
assert ((90M - 100M) / 100M) * 100 == -10.0%

# Sign Change
assert growth(10M, -5M) is None  # Returns NULL
```

## Running Tests

### Run All Tests
```bash
cd ic-score-service
pytest tests/test_ttm_performance_ratios.py -v
pytest tests/test_ebitda_calculations.py -v
pytest tests/test_quarterly_growth.py -v
```

### Run Specific Test Class
```bash
pytest tests/test_ttm_performance_ratios.py::TestTTMPerformanceRatios -v
pytest tests/test_ebitda_calculations.py::TestEnterpriseValue -v
pytest tests/test_quarterly_growth.py::TestQoQGrowthCalculations -v
```

### Run Single Test
```bash
pytest tests/test_ttm_performance_ratios.py::TestTTMPerformanceRatios::test_roa_calculation -v
```

### Run with Coverage
```bash
pytest tests/ --cov=pipelines --cov-report=html
```

## Test Coverage Goals

- **Unit Tests**: 90%+ coverage of calculation logic
- **Edge Cases**: All NULL, zero, and sign-change scenarios
- **Integration Tests**: End-to-end pipeline testing (separate file)
- **Real-World**: Tests with actual AAPL, MSFT-like data

## Expected Test Results

All tests should pass with the following characteristics:

### TTM Performance Ratios
- ✅ Calculations match manual verification
- ✅ Uses averaged balance sheet items (4 quarters apart)
- ✅ Handles missing data gracefully
- ✅ Returns NULL for invalid scenarios
- ✅ Precision: 4 decimal places

### EBITDA Calculations
- ✅ EBITDA always >= Operating Income
- ✅ TTM EBITDA = sum of 4 quarters
- ✅ EV accounts for debt and cash
- ✅ EV/EBITDA returns NULL for negative EBITDA
- ✅ Precision: 2 decimal places for ratios

### QoQ Growth
- ✅ Growth formula matches specification
- ✅ Sign changes return NULL
- ✅ Division by zero returns NULL
- ✅ Both negative values handled correctly
- ✅ Precision: 2 decimal places

## Common Test Patterns

### 1. Edge Case Testing
```python
def test_division_by_zero():
    # Should handle gracefully, not crash
    result = calculate_ratio(numerator=100, denominator=0)
    assert result is None
```

### 2. Precision Testing
```python
def test_precision_rounding():
    value = 12.3456789
    rounded = round(value, 4)
    assert rounded == 12.3457
```

### 3. Real-World Validation
```python
def test_apple_like_metrics():
    # Compare against known good values
    assert calculated_roe == pytest.approx(expected_roe, rel=0.01)
```

## Troubleshooting

### Tests Fail on Division by Zero
- Check that code returns `None` instead of raising exception
- Ensure validation before calculation

### Floating Point Precision Issues
- Use `pytest.approx()` for decimal comparisons
- Round values consistently (4 decimals for ratios, 2 for growth)

### Real-World Test Failures
- Verify test data matches actual financial statements
- Account for rounding differences in source data
- Consider using relative tolerances (`rel=0.01` for 1%)

## Integration Testing

For end-to-end testing of the full pipeline:

```bash
# Test TTM calculator with single ticker
python pipelines/ttm_financials_calculator.py --ticker AAPL

# Test quarterly growth calculator
python pipelines/quarterly_growth_calculator.py --ticker MSFT

# Test valuation ratios calculator
python pipelines/valuation_ratios_calculator.py --ticker GOOGL
```

## Continuous Integration

Add to CI/CD pipeline:

```yaml
# .github/workflows/test.yml
- name: Run Financial Metrics Tests
  run: |
    cd ic-score-service
    pytest tests/test_ttm_performance_ratios.py -v
    pytest tests/test_ebitda_calculations.py -v
    pytest tests/test_quarterly_growth.py -v
```

## Test Data

Test data should cover:
- ✅ Normal cases (profitable, growing companies)
- ✅ Edge cases (zero, negative, NULL values)
- ✅ Sign transitions (loss to profit, profit to loss)
- ✅ Extreme values (micro-cap to mega-cap)
- ✅ Real companies (AAPL, MSFT, GOOGL metrics)

## Success Criteria

Tests are considered passing when:
1. All assertions pass
2. No unexpected exceptions
3. Edge cases return NULL (not crash)
4. Real-world examples match expected values within tolerance
5. Performance is acceptable (<100ms per test)

## Next Steps

1. **Run tests locally**: Verify all tests pass
2. **Add integration tests**: Test full pipeline end-to-end
3. **Performance tests**: Ensure calculations are fast
4. **Database tests**: Test actual database operations
5. **Load tests**: Test with 4,600+ stocks

## Contact

For questions about tests or failures:
- Check test file comments for expected behavior
- Review calculation formulas in specification
- Verify database schema matches expected structure
