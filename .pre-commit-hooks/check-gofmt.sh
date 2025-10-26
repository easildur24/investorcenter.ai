#!/bin/bash
# Check if Go files are formatted with gofmt
# Uses Docker if gofmt is not installed locally

set -e

if command -v gofmt &> /dev/null; then
    # Use local gofmt
    cd backend
    unformatted=$(gofmt -l .)
    if [ -n "$unformatted" ]; then
        echo "❌ The following Go files are not formatted:"
        echo "$unformatted"
        echo ""
        echo "Run: cd backend && gofmt -w ."
        exit 1
    fi
    echo "✅ All Go files are properly formatted"
else
    # Use Docker
    echo "Using Docker to check Go formatting..."
    docker run --rm -v "$(pwd)":/app -w /app/backend golang:1.21 sh -c '
        unformatted=$(gofmt -l .)
        if [ -n "$unformatted" ]; then
            echo "❌ The following Go files are not formatted:"
            echo "$unformatted"
            echo ""
            echo "Run: docker run --rm -v $(pwd):/app -w /app/backend golang:1.21 gofmt -w ."
            exit 1
        fi
        echo "✅ All Go files are properly formatted"
    '
fi
