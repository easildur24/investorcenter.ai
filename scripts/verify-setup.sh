#!/bin/bash
# Verify Complete InvestorCenter Database Setup
# Checks both local and production databases

set -e

echo "InvestorCenter Database Setup Verification"
echo "==========================================="

# Add PostgreSQL to PATH
export PATH="/opt/homebrew/opt/postgresql@15/bin:$PATH"

echo ""
echo "LOCAL DATABASE (Development)"
echo "-----------------------------"

# Test local database
source scripts/switch-env.sh local >/dev/null 2>&1

if psql investorcenter_db -c "SELECT 1;" >/dev/null 2>&1; then
    LOCAL_COUNT=$(psql investorcenter_db -t -c "SELECT COUNT(*) FROM stocks;" | xargs)
    LOCAL_NASDAQ=$(psql investorcenter_db -t -c "SELECT COUNT(*) FROM stocks WHERE exchange = 'Nasdaq';" | xargs)
    LOCAL_NYSE=$(psql investorcenter_db -t -c "SELECT COUNT(*) FROM stocks WHERE exchange = 'NYSE';" | xargs)

        echo "Connection: SUCCESS"
    echo "Total stocks: $LOCAL_COUNT"
    echo "Nasdaq: $LOCAL_NASDAQ"
    echo "NYSE: $LOCAL_NYSE"

    # Test some famous stocks
    echo "Sample companies:"
    psql investorcenter_db -t -c "SELECT '  ' || symbol || ' - ' || name FROM stocks WHERE symbol IN ('AAPL', 'MSFT', 'GOOGL', 'TSLA') ORDER BY symbol;" | head -4
else
    echo "❌ Connection: FAILED"
    echo "   Run: brew services start postgresql@15"
fi

echo ""
echo "🚀 PRODUCTION DATABASE (Kubernetes)"
echo "------------------------------------"

# Check if Kubernetes is available
if kubectl get pods -n investorcenter >/dev/null 2>&1; then
    # Check if PostgreSQL pod is running
    if kubectl get pod -n investorcenter -l app=postgres | grep Running >/dev/null 2>&1; then
        # Test production database
        PROD_PASSWORD=$(kubectl get secret postgres-secret -n investorcenter -o jsonpath='{.data.password}' | base64 -d)

        if PGPASSWORD="$PROD_PASSWORD" psql -h localhost -p 5433 -U investorcenter -d investorcenter_db -c "SELECT 1;" >/dev/null 2>&1; then
            PROD_COUNT=$(PGPASSWORD="$PROD_PASSWORD" psql -h localhost -p 5433 -U investorcenter -d investorcenter_db -t -c "SELECT COUNT(*) FROM stocks;" | xargs)
            PROD_NASDAQ=$(PGPASSWORD="$PROD_PASSWORD" psql -h localhost -p 5433 -U investorcenter -d investorcenter_db -t -c "SELECT COUNT(*) FROM stocks WHERE exchange = 'Nasdaq';" | xargs)
            PROD_NYSE=$(PGPASSWORD="$PROD_PASSWORD" psql -h localhost -p 5433 -U investorcenter -d investorcenter_db -t -c "SELECT COUNT(*) FROM stocks WHERE exchange = 'NYSE';" | xargs)

            echo "✅ Connection: SUCCESS (via port-forward)"
            echo "📈 Total stocks: $PROD_COUNT"
            echo "🔵 Nasdaq: $PROD_NASDAQ"
            echo "🔴 NYSE: $PROD_NYSE"

            # Test some famous stocks
            echo "🏢 Sample companies:"
            PGPASSWORD="$PROD_PASSWORD" psql -h localhost -p 5433 -U investorcenter -d investorcenter_db -t -c "SELECT '  ' || symbol || ' - ' || name FROM stocks WHERE symbol IN ('AAPL', 'MSFT', 'GOOGL', 'TSLA') ORDER BY symbol;" | head -4
        else
            echo "❌ Connection: FAILED"
            echo "   Check port-forward: kubectl port-forward -n investorcenter svc/postgres-service 5433:5432"
        fi
    else
        echo "❌ PostgreSQL pod not running"
        echo "   Deploy: kubectl apply -f k8s/postgres-deployment.yaml"
    fi
else
    echo "❌ Kubernetes cluster not available"
    echo "   Start cluster or check kubectl configuration"
fi

echo ""
echo "🔧 ENVIRONMENT USAGE"
echo "--------------------"
echo "Switch to local development:"
echo "  source scripts/switch-env.sh local"
echo "  python scripts/ticker_import_to_db.py"
echo ""
echo "Switch to production:"
echo "  source scripts/switch-env.sh prod"
echo "  python scripts/ticker_import_to_db.py"
echo ""
echo "Periodic updates:"
echo "  python scripts/update_tickers_cron.py"

echo ""
echo "🚀 SETUP STATUS"
echo "---------------"

if [[ "$LOCAL_COUNT" -gt 0 ]] && [[ "$PROD_COUNT" -gt 0 ]]; then
    echo "🎉 COMPLETE: Both local and production databases ready!"
    echo "   Local: $LOCAL_COUNT stocks"
    echo "   Production: $PROD_COUNT stocks"
    echo ""
    echo "🎯 Next steps:"
    echo "   - Start developing with local database"
    echo "   - Deploy to production with K8s database"
    echo "   - Set up periodic ticker updates"
elif [[ "$LOCAL_COUNT" -gt 0 ]]; then
    echo "⚠️  LOCAL ONLY: Local database ready, production needs setup"
elif [[ "$PROD_COUNT" -gt 0 ]]; then
    echo "⚠️  PRODUCTION ONLY: Production ready, local needs setup"
else
    echo "❌ SETUP INCOMPLETE: Both databases need ticker import"
fi
