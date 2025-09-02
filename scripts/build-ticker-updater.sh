#!/bin/bash
# Build and push ticker-updater Docker image

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

echo "Building ticker-updater Docker image..."
echo "======================================"

# Build context directory
BUILD_DIR="$ROOT_DIR/docker/ticker-updater"

# Copy required files to build context
echo "Preparing build context..."
cp "$ROOT_DIR/scripts/update_tickers_cron.py" "$BUILD_DIR/"
cp "$ROOT_DIR/scripts/ticker_import_to_db.py" "$BUILD_DIR/"
cp -r "$ROOT_DIR/scripts/us_tickers" "$BUILD_DIR/"

# Build Docker image
echo "Building Docker image..."
cd "$BUILD_DIR"
docker build -t investorcenter/ticker-updater:latest .

# Tag with timestamp for versioning
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
docker tag investorcenter/ticker-updater:latest investorcenter/ticker-updater:$TIMESTAMP

echo ""
echo "✅ Docker image built successfully!"
echo "Images created:"
echo "  - investorcenter/ticker-updater:latest"
echo "  - investorcenter/ticker-updater:$TIMESTAMP"
echo ""

# Production deployment requires pushing to registry
echo "PRODUCTION DEPLOYMENT STEPS:"
echo "1. Push to your container registry:"
echo "   docker tag investorcenter/ticker-updater:latest YOUR_REGISTRY/investorcenter/ticker-updater:latest"
echo "   docker push YOUR_REGISTRY/investorcenter/ticker-updater:latest"
echo ""
echo "2. Update k8s/ticker-update-cronjob.yaml image field to use your registry"
echo ""
echo "3. Deploy to production cluster:"
echo "   kubectl apply -f k8s/ticker-update-cronjob.yaml"
echo ""

# Clean up build context
echo "Cleaning up build context..."
rm -f "$BUILD_DIR/update_tickers_cron.py"
rm -f "$BUILD_DIR/ticker_import_to_db.py"
rm -rf "$BUILD_DIR/us_tickers"

echo "Build complete! Ready to deploy CronJob."
