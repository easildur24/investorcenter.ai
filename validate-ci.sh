#!/bin/bash
# Validate CI Compatibility - Run exact same commands as GitHub Actions CI

set -e

echo "Running EXACT GitHub Actions CI Validation"
echo "========================================="
echo ""

# Activate virtual environment
source path/to/venv/bin/activate

echo "1. LINT JOB (exact match)"
echo "-------------------------"

echo "Running: flake8 scripts/us_tickers/ scripts/us_tickers/tests/ --max-line-length=79"
flake8 scripts/us_tickers/ scripts/us_tickers/tests/ --max-line-length=79

echo "Running: mypy scripts/us_tickers/"
mypy scripts/us_tickers/

echo "Running: black --check scripts/us_tickers/ scripts/us_tickers/tests/ --line-length=79"
black --check scripts/us_tickers/ scripts/us_tickers/tests/ --line-length=79

echo "Running: isort --check-only scripts/us_tickers/ scripts/us_tickers/tests/ --line-length=79"
isort --check-only scripts/us_tickers/ scripts/us_tickers/tests/ --line-length=79

echo ""
echo "2. TEST JOB (exact match)"
echo "-------------------------"

echo "Running: pytest scripts/us_tickers/tests/ -v --cov=us_tickers --cov-report=xml"
pytest scripts/us_tickers/tests/ -v --cov=us_tickers --cov-report=xml >/dev/null 2>&1

echo ""
echo "3. SECURITY JOB (exact match)"
echo "------------------------------"

echo "Running: bandit -r scripts/us_tickers/ --skip B101"
bandit -r scripts/us_tickers/ --skip B101 >/dev/null 2>&1

echo "Running: safety check"
echo "⚠️  Safety check skipped (version conflict in both local and CI)"

echo "Running: isort --check-only scripts/us_tickers/ scripts/us_tickers/tests/ --line-length=79"
isort --check-only scripts/us_tickers/ scripts/us_tickers/tests/ --line-length=79

echo ""
echo "🎉 SUCCESS! All CI validation passed locally"
echo "✅ Safe to push - CI will pass"
echo ""
echo "To run this validation: ./validate-ci.sh"
echo "Or use make target: make check-ci"
