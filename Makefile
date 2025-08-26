.PHONY: help install install-dev test test-cov lint format clean build check

help:  ## Show this help message
	@echo "US Tickers - Development Commands"
	@echo "=================================="
	@echo ""
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

install:  ## Install package in development mode
	pip install -e .

install-dev:  ## Install package with development dependencies
	pip install -e ".[dev]"

test:  ## Run tests
	pytest scripts/us_tickers/tests/ -v

test-cov:  ## Run tests with coverage
	pytest scripts/us_tickers/tests/ -v --cov=us_tickers --cov-report=term-missing --cov-report=html

lint:  ## Run linting checks
	flake8 scripts/us_tickers/ scripts/us_tickers/tests/
	mypy scripts/us_tickers/

format:  ## Format code with black
	black scripts/us_tickers/ scripts/us_tickers/tests/

check: format lint test  ## Run all checks (format, lint, test)

clean:  ## Clean up generated files
	rm -rf build/
	rm -rf dist/
	rm -rf *.egg-info/
	rm -rf .pytest_cache/
	rm -rf htmlcov/
	rm -rf .coverage
	find . -type f -name "*.pyc" -delete
	find . -type d -name "__pycache__" -delete

build:  ## Build package distribution
	python -m build

demo:  ## Run a quick demo
	@echo "Running US Tickers demo..."
	@python -c "from us_tickers import get_exchange_listed_tickers; tickers, df = get_exchange_listed_tickers(exchanges=('Q', 'N'), include_etfs=False, include_test_issues=False); print(f'Found {len(tickers)} tickers'); print(f'Sample: {tickers[:10]}'); print(f'DataFrame shape: {df.shape}')"

demo-cli:  ## Run CLI demo
	@echo "Running CLI demo..."
	@cd scripts/us_tickers && python -m us_tickers.cli fetch --exchanges Q,N --out demo_tickers.csv --format csv
	@echo "Demo complete! Check demo/us_tickers/demo_tickers.csv"

install-hooks:  ## Install pre-commit hooks
	pre-commit install

run-hooks:  ## Run pre-commit hooks on all files
	pre-commit run --all-files

docs:  ## Generate documentation
	@echo "Documentation is in README.md"
	@echo "For API docs, run: python -c 'import us_tickers; help(us_tickers)'"

release: check build  ## Prepare release (run checks and build)
	@echo "Release preparation complete!"
	@echo "Next steps:"
	@echo "1. Update version in pyproject.toml"
	@echo "2. Tag the release: git tag v0.1.0"
	@echo "3. Push tags: git push --tags"
	@echo "4. Upload to PyPI: python -m twine upload dist/*"

.PHONY: help install install-dev test test-cov lint format clean build check demo demo-cli install-hooks run-hooks docs release
