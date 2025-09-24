#!/bin/bash

# Deploy Frontend to S3/CloudFront
# This script handles the complete deployment of the Next.js frontend

set -e

# Configuration
S3_BUCKET="investorcenter.ai"
CLOUDFRONT_DISTRIBUTION_ID="E1R2S3T4U5V6W7"  # Replace with actual ID
AWS_REGION="us-east-1"

echo "🚀 Deploying Frontend to S3/CloudFront"
echo "============================================"

# Step 1: Install dependencies
echo "📦 Installing dependencies..."
npm ci

# Step 2: Build Next.js app
echo "🔨 Building Next.js application..."
NEXT_PUBLIC_BACKEND_URL=https://api.investorcenter.ai npm run build

# Step 3: Export static files
echo "📁 Exporting static files..."
npx next export

# Step 4: Sync to S3
echo "⬆️  Uploading to S3..."
aws s3 sync out/ s3://${S3_BUCKET}/ \
    --delete \
    --cache-control "public, max-age=31536000, immutable" \
    --exclude "*.html" \
    --exclude "_next/data/*" \
    --exclude "_next/static/chunks/pages/*"

# Upload HTML files with shorter cache
aws s3 sync out/ s3://${S3_BUCKET}/ \
    --exclude "*" \
    --include "*.html" \
    --include "_next/data/*" \
    --include "_next/static/chunks/pages/*" \
    --cache-control "public, max-age=0, must-revalidate"

# Step 5: Invalidate CloudFront cache
echo "🔄 Invalidating CloudFront cache..."
aws cloudfront create-invalidation \
    --distribution-id ${CLOUDFRONT_DISTRIBUTION_ID} \
    --paths "/*" \
    --query 'Invalidation.Id' \
    --output text

echo ""
echo "✨ Frontend deployed successfully!"
echo "   URL: https://investorcenter.ai"
echo "   CloudFront distribution will take a few minutes to propagate"