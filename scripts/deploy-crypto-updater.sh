#!/bin/bash
set -e

echo "🚀 Deploying Crypto Price Updater to Kubernetes"
echo "================================================"

# Configuration
AWS_REGION="us-east-1"
AWS_ACCOUNT_ID="360358043271"
ECR_REPO="investorcenter/crypto-price-updater"
IMAGE_TAG="${1:-latest}"

echo "📦 Building Docker image for linux/amd64 (EKS)..."
docker build --platform linux/amd64 -f Dockerfile.crypto-updater -t crypto-price-updater:${IMAGE_TAG} .

echo "🏷️  Tagging image for ECR..."
docker tag crypto-price-updater:${IMAGE_TAG} \
    ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPO}:${IMAGE_TAG}

echo "🔑 Logging into ECR..."
AWS_PROFILE=AdministratorAccess-${AWS_ACCOUNT_ID} \
    aws ecr get-login-password --region ${AWS_REGION} | \
    docker login --username AWS --password-stdin \
    ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

echo "📤 Pushing image to ECR..."
docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPO}:${IMAGE_TAG}

echo "☸️  Deploying to Kubernetes..."

# Deploy Redis first (if not already deployed)
echo "  → Deploying Redis..."
kubectl apply -f k8s/redis-deployment.yaml

# Wait for Redis to be ready
echo "  → Waiting for Redis to be ready..."
kubectl wait --for=condition=ready pod -l app=redis -n investorcenter --timeout=120s

# Deploy crypto price updater
echo "  → Deploying Crypto Price Updater..."
kubectl apply -f k8s/crypto-price-updater-deployment.yaml

# Wait for deployment
echo "  → Waiting for deployment to be ready..."
kubectl rollout status deployment/crypto-price-updater -n investorcenter --timeout=180s

echo ""
echo "✅ Deployment complete!"
echo ""
echo "📊 Check logs with:"
echo "   kubectl logs -n investorcenter -l app=crypto-price-updater -f"
echo ""
echo "🔍 Check status with:"
echo "   kubectl get pods -n investorcenter | grep crypto"
echo ""
echo "🧪 Test Redis connection:"
echo "   kubectl exec -n investorcenter deployment/crypto-price-updater -- python3 -c 'import redis; r=redis.Redis(host=\"redis-service\"); print(r.ping())'"
