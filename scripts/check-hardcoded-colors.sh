#!/bin/bash
# Check for hardcoded Tailwind color classes that should use theme variables instead.
# These hardcoded colors break dark mode. Use ic-* theme classes or alpha-blended
# variants (e.g., bg-green-500/20 text-ic-positive) instead.
#
# Usage: scripts/check-hardcoded-colors.sh [--warn-only]
#
# Exit codes:
#   0 - No violations found (or --warn-only mode)
#   1 - Violations found

set -euo pipefail

WARN_ONLY=false
if [[ "${1:-}" == "--warn-only" ]]; then
  WARN_ONLY=true
fi

# Pattern matches hardcoded Tailwind color classes with specific shade numbers
# Uses ERE (grep -E) for cross-platform compatibility (macOS + Linux)
PATTERN='(bg|text|border|ring|from|to|via)-(red|green|blue|yellow|gray|slate|zinc|neutral|stone|orange|amber|lime|emerald|teal|cyan|sky|indigo|violet|purple|fuchsia|pink|rose)-(50|100|200|300|400|500|600|700|800|900)[^/]'

# Directories to check
DIRS="app/ components/ lib/"

# Files to exclude (tests, config files)
EXCLUDE_PATTERN='__tests__|\.test\.|\.spec\.|node_modules|\.next'

count=0
violations=""

for dir in $DIRS; do
  if [ -d "$dir" ]; then
    while IFS= read -r line; do
      [ -z "$line" ] && continue
      violations+="$line"$'\n'
      count=$((count + 1))
    done < <(grep -rnE "$PATTERN" "$dir" --include="*.tsx" --include="*.ts" 2>/dev/null | grep -vE "$EXCLUDE_PATTERN" || true)
  fi
done

if [ "$count" -gt 0 ]; then
  echo "============================================="
  echo "  Hardcoded Tailwind Colors Found: $count"
  echo "============================================="
  echo ""
  echo "These hardcoded color classes break dark mode."
  echo "Use theme-aware alternatives instead:"
  echo "  bg-red-100     -> bg-red-500/10 or bg-ic-negative/10"
  echo "  text-green-800 -> text-ic-positive"
  echo "  bg-blue-50     -> bg-blue-500/10 or bg-ic-blue/10"
  echo "  text-gray-600  -> text-ic-text-muted"
  echo ""
  echo "$violations"

  if [ "$WARN_ONLY" = true ]; then
    echo "::warning::Found $count hardcoded Tailwind color classes that may break dark mode"
    exit 0
  else
    exit 1
  fi
else
  echo "No hardcoded Tailwind color violations found."
fi
