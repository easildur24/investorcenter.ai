#!/bin/bash
set -e

echo "🚀 Deploying Crypto Price Infrastructure to AWS EKS"
echo "===================================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configuration
AWS_REGION="us-east-1"
AWS_ACCOUNT_ID="360358043271"
ECR_REPO="investorcenter/crypto-price-updater"
IMAGE_TAG="${1:-latest}"

# Check prerequisites
echo "🔍 Checking prerequisites..."

if ! command -v aws &> /dev/null; then
    echo -e "${RED}❌ AWS CLI not found${NC}"
    exit 1
fi

if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}❌ kubectl not found${NC}"
    exit 1
fi

if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker not found${NC}"
    exit 1
fi

# Check AWS authentication
echo "🔑 Checking AWS authentication..."
export AWS_PROFILE=AdministratorAccess-${AWS_ACCOUNT_ID}

if ! aws sts get-caller-identity &> /dev/null; then
    echo -e "${YELLOW}⚠️  AWS SSO token expired. Please login:${NC}"
    echo "   aws sso login --profile AdministratorAccess-${AWS_ACCOUNT_ID}"
    exit 1
fi

echo -e "${GREEN}✅ Prerequisites OK${NC}"
echo ""

# Step 1: Create ECR repository if it doesn't exist
echo "📦 Step 1: Creating ECR repository..."
if aws ecr describe-repositories --region ${AWS_REGION} --repository-names ${ECR_REPO} &> /dev/null; then
    echo -e "${YELLOW}   Repository already exists${NC}"
else
    aws ecr create-repository \
        --repository-name ${ECR_REPO} \
        --region ${AWS_REGION} > /dev/null
    echo -e "${GREEN}   ✅ Repository created${NC}"
fi
echo ""

# Step 2: Build Docker image
echo "🐳 Step 2: Building Docker image..."
docker build -f Dockerfile.crypto-updater -t crypto-price-updater:${IMAGE_TAG} .
echo -e "${GREEN}   ✅ Image built${NC}"
echo ""

# Step 3: Tag and push to ECR
echo "📤 Step 3: Pushing to ECR..."
docker tag crypto-price-updater:${IMAGE_TAG} \
    ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPO}:${IMAGE_TAG}

echo "   Logging into ECR..."
aws ecr get-login-password --region ${AWS_REGION} | \
    docker login --username AWS --password-stdin \
    ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

echo "   Pushing image..."
docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPO}:${IMAGE_TAG}
echo -e "${GREEN}   ✅ Image pushed${NC}"
echo ""

# Step 4: Deploy Redis
echo "💾 Step 4: Deploying Redis..."
kubectl apply -f k8s/redis-deployment.yaml

echo "   Waiting for Redis to be ready..."
if kubectl wait --for=condition=ready pod -l app=redis -n investorcenter --timeout=120s &> /dev/null; then
    echo -e "${GREEN}   ✅ Redis is ready${NC}"
else
    echo -e "${YELLOW}   ⚠️  Redis not ready yet (may still be starting)${NC}"
fi
echo ""

# Step 5: Deploy crypto price updater
echo "🔄 Step 5: Deploying Crypto Price Updater..."
kubectl apply -f k8s/crypto-price-updater-deployment.yaml

echo "   Waiting for deployment..."
if kubectl rollout status deployment/crypto-price-updater -n investorcenter --timeout=180s &> /dev/null; then
    echo -e "${GREEN}   ✅ Deployment ready${NC}"
else
    echo -e "${YELLOW}   ⚠️  Deployment not ready yet (check logs)${NC}"
fi
echo ""

# Step 6: Update backend deployment
echo "🔧 Step 6: Updating backend to use Redis..."
kubectl apply -f k8s/backend-deployment.yaml
echo "   Waiting for backend rollout..."
kubectl rollout status deployment/investorcenter-backend -n investorcenter --timeout=120s &> /dev/null || true
echo -e "${GREEN}   ✅ Backend updated${NC}"
echo ""

# Step 7: Verify deployment
echo "✅ Step 7: Verifying deployment..."
echo ""

# Check pods
echo "   Pod Status:"
kubectl get pods -n investorcenter | grep -E "redis|crypto-price-updater" || echo "   No pods found"
echo ""

# Check logs (last 10 lines)
echo "   Recent Crypto Updater Logs:"
kubectl logs -n investorcenter -l app=crypto-price-updater --tail=10 2>/dev/null || echo "   No logs yet"
echo ""

# Check Redis data
echo "   Checking Redis data (waiting 30 seconds for first update)..."
sleep 30
CRYPTO_COUNT=$(kubectl exec -n investorcenter deployment/redis -- redis-cli keys "crypto:quote:*" 2>/dev/null | wc -l | tr -d ' ')
if [ "$CRYPTO_COUNT" -gt 0 ]; then
    echo -e "${GREEN}   ✅ Found ${CRYPTO_COUNT} cryptocurrencies in Redis${NC}"

    # Get Bitcoin price
    BTC_PRICE=$(kubectl exec -n investorcenter deployment/redis -- redis-cli get crypto:quote:BTC 2>/dev/null | jq -r '.current_price' 2>/dev/null || echo "")
    if [ -n "$BTC_PRICE" ]; then
        echo -e "${GREEN}   ✅ Bitcoin price: \$${BTC_PRICE}${NC}"
    fi
else
    echo -e "${YELLOW}   ⚠️  No crypto data yet (may still be fetching)${NC}"
fi
echo ""

echo "=================================================="
echo -e "${GREEN}🎉 Deployment Complete!${NC}"
echo "=================================================="
echo ""
echo "Next steps:"
echo ""
echo "1. Monitor logs:"
echo "   kubectl logs -n investorcenter -l app=crypto-price-updater -f"
echo ""
echo "2. Check Redis data:"
echo "   kubectl exec -n investorcenter deployment/redis -- redis-cli keys 'crypto:quote:*'"
echo ""
echo "3. Test API (port-forward first):"
echo "   kubectl port-forward -n investorcenter svc/backend-service 8080:8080 &"
echo "   curl http://localhost:8080/api/v1/crypto/BTC/price | jq"
echo ""
echo "4. View full deployment guide:"
echo "   cat CRYPTO_DEPLOYMENT_GUIDE.md"
echo ""
