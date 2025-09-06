#!/bin/bash

# Script to update Polygon.io API key
set -e

if [ $# -eq 0 ]; then
    echo "Usage: $0 <your-polygon-api-key>"
    echo "Example: $0 ABCD1234EFGH5678"
    exit 1
fi

API_KEY="$1"
NAMESPACE="investorcenter"

echo "🔑 Updating Polygon.io API key..."

# Encode the API key in base64
ENCODED_KEY=$(echo -n "$API_KEY" | base64)

echo "📝 Base64 encoded key: $ENCODED_KEY"

# Update the secret
kubectl patch secret app-secrets -n $NAMESPACE --type='merge' -p="{\"data\":{\"polygon-api-key\":\"$ENCODED_KEY\"}}"

echo "✅ API key updated successfully!"
echo "🔄 Restarting backend to pick up new API key..."

# Restart backend deployment to pick up new secret
kubectl rollout restart deployment/investorcenter-backend -n $NAMESPACE

echo "⏳ Waiting for backend to restart..."
kubectl rollout status deployment/investorcenter-backend -n $NAMESPACE

echo "🎉 Backend updated with new Polygon.io API key!"
echo "🧪 Testing API connection..."

# Wait a moment for the service to be ready
sleep 10

# Test the API
echo "Testing real-time price data for AAPL..."
kubectl exec -n $NAMESPACE deployment/investorcenter-backend -- wget -qO- "http://localhost:8080/api/v1/tickers/AAPL" | head -5

echo "✅ Setup complete! Your ticker pages now have unlimited real-time price data via Polygon.io!"
