#!/bin/bash

# Crypto PostgreSQL Sync Cron Script
# Run this hourly to keep crypto metadata in sync

# Configuration
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
LOG_DIR="$SCRIPT_DIR/logs"
LOG_FILE="$LOG_DIR/crypto_sync_$(date +%Y%m%d).log"

# Database configuration
export DB_HOST=localhost
export DB_PORT=5433
export DB_USER=investorcenter
export DB_PASSWORD=password123
export DB_NAME=investorcenter_db

# Create log directory if it doesn't exist
mkdir -p "$LOG_DIR"

echo "================================================" >> "$LOG_FILE"
echo "Starting crypto sync at $(date)" >> "$LOG_FILE"
echo "================================================" >> "$LOG_FILE"

# Run the sync (top 1000 cryptos, no history)
cd "$SCRIPT_DIR"
./venv/bin/python crypto_postgres_sync.py --limit 1000 --no-history >> "$LOG_FILE" 2>&1

if [ $? -eq 0 ]; then
    echo "✅ Sync completed successfully at $(date)" >> "$LOG_FILE"
else
    echo "❌ Sync failed at $(date)" >> "$LOG_FILE"
fi

# Rotate logs (keep last 7 days)
find "$LOG_DIR" -name "crypto_sync_*.log" -type f -mtime +7 -delete

echo "" >> "$LOG_FILE"