#!/bin/bash

# Script to check for exposed secrets and API keys
# Run this before committing code

echo "🔍 Checking for exposed secrets..."

# Patterns to search for
PATTERNS=(
    "AKIA[0-9A-Z]{16}"  # AWS Access Key IDs
    "(?i)api[_-]?key.*=.*['\"][0-9a-zA-Z]{16,}"  # Generic API keys
    "(?i)secret.*=.*['\"][0-9a-zA-Z]{16,}"  # Generic secrets
    "[0-9a-fA-F]{40}"  # GitHub tokens
    "sk_live_[0-9a-zA-Z]{24}"  # Stripe keys
    "6MVGMJ4FCAGF2ONU"  # Known Alpha Vantage key
)

FOUND_ISSUES=0

for pattern in "${PATTERNS[@]}"; do
    echo "Checking for pattern: $pattern"
    if grep -r -E "$pattern" . --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=venv --exclude="*.example" --exclude="check_secrets.sh" 2>/dev/null; then
        echo "❌ Found potential secret matching pattern: $pattern"
        FOUND_ISSUES=1
    fi
done

if [ $FOUND_ISSUES -eq 0 ]; then
    echo "✅ No exposed secrets found"
else
    echo "⚠️  Potential secrets found! Please review and remove them before committing."
    exit 1
fi