#!/bin/bash

# Script to migrate ticker data from exchange files to Polygon API
set -e

echo "================================================"
echo "🚀 Polygon Ticker Migration Script"
echo "================================================"

# Check if API key is set
if [ -z "$POLYGON_API_KEY" ]; then
    echo "❌ Error: POLYGON_API_KEY environment variable is not set"
    echo ""
    echo "Please set your API key:"
    echo "  export POLYGON_API_KEY=your_api_key_here"
    echo ""
    echo "Or update the Kubernetes secret:"
    echo "  ./scripts/update-api-key.sh your_api_key_here"
    exit 1
fi

# Check database credentials
if [ -z "$DB_PASSWORD" ]; then
    echo "⚠️  Warning: DB_PASSWORD not set. Using default from .env file"
    export DB_HOST=${DB_HOST:-localhost}
    export DB_PORT=${DB_PORT:-5432}
    export DB_USER=${DB_USER:-investorcenter}
    export DB_NAME=${DB_NAME:-investorcenter_db}
    export DB_PASSWORD=${DB_PASSWORD:-your_password_here}
fi

echo "📋 Configuration:"
echo "  API Key: ${POLYGON_API_KEY:0:10}..."
echo "  Database: $DB_USER@$DB_HOST:$DB_PORT/$DB_NAME"
echo ""

# Step 1: Run database migration
echo "1️⃣ Running database migration..."
echo "--------------------------------"
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f backend/migrations/002_add_polygon_ticker_fields.sql 2>/dev/null || {
    echo "⚠️  Migration may have already been applied or failed. Continuing..."
}

# Step 2: Build the import tool
echo ""
echo "2️⃣ Building ticker import tool..."
echo "--------------------------------"
cd backend
go build -o ../bin/import-tickers ./cmd/import-tickers/main.go
cd ..

# Step 3: Test with dry run
echo ""
echo "3️⃣ Testing import (dry run)..."
echo "--------------------------------"
./bin/import-tickers -type=stocks -limit=10 -dry-run -verbose

echo ""
read -p "📝 Dry run complete. Proceed with actual import? (y/N) " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Import cancelled"
    exit 1
fi

# Step 4: Import each asset type
echo ""
echo "4️⃣ Starting ticker import..."
echo "--------------------------------"

# Import stocks (most important)
echo ""
echo "📈 Importing US stocks..."
./bin/import-tickers -type=stocks -limit=0 || {
    echo "⚠️  Stock import failed. Check API key and rate limits."
}

# Wait to avoid rate limiting
echo "⏳ Waiting 10 seconds to avoid rate limits..."
sleep 10

# Import ETFs
echo ""
echo "💼 Importing ETFs..."
./bin/import-tickers -type=etf -limit=0 || {
    echo "⚠️  ETF import failed. Check API key and rate limits."
}

# Wait to avoid rate limiting
echo "⏳ Waiting 10 seconds to avoid rate limits..."
sleep 10

# Import crypto (optional - lots of pairs)
echo ""
read -p "🪙 Import crypto pairs? This may take a while (y/N) " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    ./bin/import-tickers -type=crypto -limit=1000 || {
        echo "⚠️  Crypto import failed. Check API key and rate limits."
    }
    sleep 10
fi

# Import indices
echo ""
echo "📊 Importing indices..."
./bin/import-tickers -type=indices -limit=0 || {
    echo "⚠️  Index import failed. Check API key and rate limits."
}

# Step 5: Show summary
echo ""
echo "5️⃣ Import Summary"
echo "--------------------------------"
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "
    SELECT 
        asset_type,
        COUNT(*) as count,
        COUNT(CASE WHEN market_cap IS NOT NULL THEN 1 END) as with_market_cap,
        COUNT(CASE WHEN cik IS NOT NULL THEN 1 END) as with_cik
    FROM stocks
    WHERE asset_type IS NOT NULL
    GROUP BY asset_type
    ORDER BY count DESC;
" 2>/dev/null || echo "Could not fetch summary"

echo ""
echo "✅ Migration complete!"
echo ""
echo "Next steps:"
echo "1. Restart the backend to use new ticker data:"
echo "   make dev"
echo ""
echo "2. Test the API endpoints:"
echo "   curl http://localhost:8080/api/v1/tickers?type=etf"
echo "   curl http://localhost:8080/api/v1/tickers/SPY"
echo ""
echo "3. For production, update the Kubernetes deployment:"
echo "   kubectl rollout restart deployment/investorcenter-backend -n investorcenter"