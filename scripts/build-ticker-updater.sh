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

# Optional: Push to registry (uncomment when ready)
echo "To push to registry:"
echo "  docker push investorcenter/ticker-updater:latest"
echo "  docker push investorcenter/ticker-updater:$TIMESTAMP"
echo ""

# Clean up build context
echo "Cleaning up build context..."
rm -f "$BUILD_DIR/update_tickers_cron.py"
rm -f "$BUILD_DIR/ticker_import_to_db.py"
rm -rf "$BUILD_DIR/us_tickers"

echo "Build complete! Ready to deploy CronJob."
