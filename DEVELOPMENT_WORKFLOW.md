# Development Workflow & Code Quality

This document explains how to maintain code quality and catch CI issues locally before pushing.

## Quick Commands

### Before Every Push
```bash
# Format, lint, test everything - catch CI issues early
make check-all
```

### Individual Operations
```bash
# Format all code automatically
make format

# Run linting checks
make lint

# Run tests only
make test

# Run security checks (temporarily disabled)
make safety-check
```

## Automatic Code Quality

### Pre-commit Hooks (Recommended)
Pre-commit hooks are automatically installed with `make setup-local` and run on every commit:

```bash
# Install hooks manually if needed
make pre-commit-install

# Run hooks manually
make pre-commit-run
```

### What Gets Checked
- **Python formatting**: Black code formatter
- **Import sorting**: isort for clean imports
- **Linting**: flake8 for style issues (configured to work with black)
- **Type checking**: mypy for type safety
- **Security**: bandit for security vulnerabilities
- **Testing**: 60 comprehensive tests across all modules

## CI Compatibility

The local validation matches CI exactly:
- Same flake8 configuration (ignores E203,W503 for black compatibility)
- Same dependencies and versions
- Same test suite
- MyPy configured for practical usage (--ignore-missing-imports, --no-strict-optional)

## Workflow

### Standard Development Flow
```bash
# 1. Make your changes
vim some_file.py

# 2. Format and validate
make check-all

# 3. Commit (pre-commit hooks run automatically)
git commit -m "your changes"

# 4. Push (CI will pass)
git push
```

### Quick Fix Flow
```bash
# Just format if you're in a hurry
make format

# Quick lint check
make lint-no-format
```

## Troubleshooting

### Black/Flake8 Conflicts
The Makefile is configured to handle the known E203 conflict between black and flake8. If you see formatting issues:

```bash
make format  # This fixes most issues
```

### MyPy Issues
MyPy is configured to be practical rather than strict. For stubborn type issues:
- Types are checked but missing imports are ignored
- Optional types are handled gracefully

### Test Failures
All tests should pass locally before pushing:

```bash
make test-python  # Run just Python tests
make test-go      # Run just Go tests
```

## Benefits

- **Zero CI Surprises**: Catch all issues locally
- **Automatic Formatting**: Never worry about code style
- **Comprehensive Testing**: 60+ tests ensure reliability
- **Security Scanning**: Automatic vulnerability detection
- **Professional Quality**: Code follows industry standards

## Dependencies

All required dependencies are in `requirements.txt`:
- Code quality tools (black, flake8, mypy, bandit)
- Database dependencies (psycopg2-binary, python-dotenv)
- Type stubs (types-psycopg2)
- Testing tools (pytest, pytest-cov)
