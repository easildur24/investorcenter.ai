#!/bin/bash
# Run SEC filing fetcher with proper virtual environment

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# Create venv if it doesn't exist
if [ ! -d "venv" ]; then
    echo "Creating virtual environment..."
    python3 -m venv venv
fi

# Install dependencies if needed
if ! venv/bin/python -c "import psycopg2" 2>/dev/null; then
    echo "Installing dependencies..."
    venv/bin/pip install psycopg2-binary requests
fi

# Run the fetcher with passed arguments
echo "Running SEC filing fetcher..."
venv/bin/python sec_filing_fetcher.py "$@"