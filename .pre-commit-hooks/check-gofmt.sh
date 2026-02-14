#!/bin/bash
# Check if Go files are formatted with gofmt
# Checks all Go service directories: backend, data-ingestion-service

set -e

GO_DIRS="backend data-ingestion-service"
failed=0

if command -v gofmt &> /dev/null; then
    for dir in $GO_DIRS; do
        if [ -d "$dir" ]; then
            unformatted=$(gofmt -l "$dir")
            if [ -n "$unformatted" ]; then
                echo "❌ The following Go files in $dir/ are not formatted:"
                echo "$unformatted"
                echo ""
                echo "Run: gofmt -w $dir/"
                failed=1
            fi
        fi
    done

    if [ "$failed" -eq 1 ]; then
        exit 1
    fi
    echo "✅ All Go files are properly formatted"
else
    # Use Docker
    echo "Using Docker to check Go formatting..."
    for dir in $GO_DIRS; do
        if [ -d "$dir" ]; then
            docker run --rm -v "$(pwd)":/app -w /app golang:1.21 sh -c "
                unformatted=\$(gofmt -l $dir)
                if [ -n \"\$unformatted\" ]; then
                    echo \"❌ The following Go files in $dir/ are not formatted:\"
                    echo \"\$unformatted\"
                    exit 1
                fi
            " || failed=1
        fi
    done

    if [ "$failed" -eq 1 ]; then
        echo ""
        echo "Run: gofmt -w backend/ data-ingestion-service/"
        exit 1
    fi
    echo "✅ All Go files are properly formatted"
fi
