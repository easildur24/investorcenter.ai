#!/bin/bash
# Environment Switcher for InvestorCenter
# Switches between local development and production database environments

set -e

ENV=${1:-local}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

case $ENV in
  local)
    echo "🏠 Switching to local development environment"

    # Export local database environment
    export DB_HOST=localhost
    export DB_PORT=5432
    export DB_USER=investorcenter
    export DB_PASSWORD=investorcenter123
    export DB_NAME=investorcenter_db
    export DB_SSLMODE=disable

    # Add PostgreSQL to PATH
    export PATH="/opt/homebrew/opt/postgresql@15/bin:$PATH"

    # Test local database
    if psql $DB_NAME -c "SELECT 1;" >/dev/null 2>&1; then
        STOCK_COUNT=$(psql $DB_NAME -t -c "SELECT COUNT(*) FROM stocks;" 2>/dev/null | xargs || echo "0")
        echo "✅ Local PostgreSQL: $STOCK_COUNT stocks"
    else
        echo "⚠️  Local PostgreSQL not available"
        echo "   Run: brew services start postgresql@15"
    fi

    # Environment variables ready
    echo "🔧 Environment variables set:"
    echo "   DB_HOST=$DB_HOST"
    echo "   DB_PORT=$DB_PORT"
    echo "   DB_USER=$DB_USER"
    echo "   DB_NAME=$DB_NAME"
    ;;

  prod)
    echo "🚀 Switching to production environment (via port-forward)"

    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        echo "❌ kubectl not found - install Kubernetes CLI tools"
        exit 1
    fi

    # Check if investorcenter namespace exists
    if ! kubectl get namespace investorcenter >/dev/null 2>&1; then
        echo "❌ investorcenter namespace not found"
        echo "   Create with: kubectl apply -f k8s/namespace.yaml"
        exit 1
    fi

    # Check if PostgreSQL pod is running
    if ! kubectl get pod -n investorcenter -l app=postgres | grep Running >/dev/null 2>&1; then
        echo "❌ PostgreSQL pod not running in production"
        echo "   Deploy with: kubectl apply -f k8s/postgres-deployment.yaml"
        exit 1
    fi

    # Kill any existing port-forwards
    pkill -f "kubectl port-forward.*postgres-service" 2>/dev/null || true

    # Start port-forward to production database
    echo "🔌 Starting port-forward to production PostgreSQL..."
    kubectl port-forward -n investorcenter svc/postgres-service 5433:5432 &
    sleep 3

    # Get database password from Kubernetes secret
    DB_PASSWORD=$(kubectl get secret postgres-secret -n investorcenter -o jsonpath='{.data.password}' | base64 -d)

    # Export production database environment
    export DB_HOST=localhost
    export DB_PORT=5433
    export DB_USER=investorcenter
    export DB_PASSWORD="$DB_PASSWORD"
    export DB_NAME=investorcenter_db
    export DB_SSLMODE=disable

    # Test production database
    echo "🧪 Testing production database connection..."
    if PGPASSWORD="$DB_PASSWORD" psql -h localhost -p 5433 -U investorcenter -d investorcenter_db -c "SELECT 1;" >/dev/null 2>&1; then
        STOCK_COUNT=$(PGPASSWORD="$DB_PASSWORD" psql -h localhost -p 5433 -U investorcenter -d investorcenter_db -t -c "SELECT COUNT(*) FROM stocks;" 2>/dev/null | xargs || echo "0")
        echo "✅ Production PostgreSQL: $STOCK_COUNT stocks"
    else
        echo "❌ Production database connection failed"
        exit 1
    fi

    echo "🔧 Environment variables set:"
    echo "   DB_HOST=$DB_HOST"
    echo "   DB_PORT=$DB_PORT (port-forwarded)"
    echo "   DB_USER=$DB_USER"
    echo "   DB_NAME=$DB_NAME"
    echo ""
    echo "⚠️  Note: Port-forward active on port 5433"
    echo "   To stop: pkill -f 'kubectl port-forward.*postgres-service'"
    ;;

  *)
    echo "❌ Invalid environment: $ENV"
    echo ""
    echo "Usage: $0 [local|prod]"
    echo ""
    echo "Environments:"
    echo "  local - Local PostgreSQL (localhost:5432)"
    echo "  prod  - Production PostgreSQL via port-forward (localhost:5433)"
    echo ""
    echo "Examples:"
    echo "  source scripts/switch-env.sh local"
    echo "  source scripts/switch-env.sh prod"
    echo "  python scripts/ticker_import_to_db.py"
    exit 1
    ;;
esac

echo ""
echo "🎯 Environment ready! You can now:"
echo "   python scripts/ticker_import_to_db.py --dry-run"
echo "   python scripts/ticker_import_to_db.py"
echo "   python scripts/update_tickers_cron.py"
