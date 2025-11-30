#!/bin/bash
# Deploy Backend to AWS EKS Production
# This script builds and deploys the Go backend with the market overview fix

set -e

echo "🚀 InvestorCenter Backend Deployment"
echo "====================================="
echo ""

# Configuration
AWS_REGION="us-east-1"
ECR_REGISTRY="360358043271.dkr.ecr.us-east-1.amazonaws.com"
IMAGE_NAME="investorcenter/backend"
IMAGE_TAG="latest"
FULL_IMAGE="$ECR_REGISTRY/$IMAGE_NAME:$IMAGE_TAG"
NAMESPACE="investorcenter"

# Check prerequisites
echo "🔍 Checking prerequisites..."

# Check kubectl context
CURRENT_CONTEXT=$(kubectl config current-context)
echo "Current kubectl context: $CURRENT_CONTEXT"

if [[ "$CURRENT_CONTEXT" != *"investorcenter-eks"* ]]; then
    echo "⚠️  WARNING: Not connected to investorcenter-eks cluster"
    echo "   Current context: $CURRENT_CONTEXT"
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "❌ Deployment cancelled"
        exit 1
    fi
fi

# Check AWS credentials
if ! aws sts get-caller-identity &> /dev/null; then
    echo "❌ AWS authentication failed"
    echo "   Please run: aws sso login --profile investorcenter"
    exit 1
fi

echo "✅ Prerequisites check passed"
echo ""

# Confirm deployment
echo "📋 Deployment Configuration:"
echo "   Image: $FULL_IMAGE"
echo "   Cluster: $CURRENT_CONTEXT"
echo "   Namespace: $NAMESPACE"
echo ""
read -p "🤔 Continue with deployment? (y/N): " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Deployment cancelled"
    exit 1
fi

echo ""
echo "🏗️  Step 1: Building backend Docker image..."
echo "============================================"

# Build from backend directory for linux/amd64 platform (EKS runs on AMD64)
cd backend
docker build --platform linux/amd64 -t $IMAGE_NAME:$IMAGE_TAG .

echo ""
echo "🔐 Step 2: Authenticating with ECR..."
echo "====================================="

# Login to ECR
aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $ECR_REGISTRY

echo ""
echo "🏷️  Step 3: Tagging and pushing image..."
echo "======================================="

# Tag and push
docker tag $IMAGE_NAME:$IMAGE_TAG $FULL_IMAGE
echo "Pushing to: $FULL_IMAGE"
docker push $FULL_IMAGE

echo ""
echo "⚙️  Step 4: Updating Kubernetes deployment..."
echo "============================================"

# Go back to root
cd ..

# Apply the deployment (this will trigger a rolling update)
kubectl apply -f k8s/backend-deployment.yaml

# Wait for rollout to complete
echo ""
echo "⏳ Waiting for deployment to complete..."
kubectl rollout status deployment/investorcenter-backend -n $NAMESPACE --timeout=5m

echo ""
echo "✅ Backend Deployment Complete!"
echo "==============================="
echo ""

# Show status
echo "📊 Deployment Status:"
kubectl get pods -n $NAMESPACE -l app=investorcenter-backend
echo ""

echo "🔍 Recent logs:"
kubectl logs -n $NAMESPACE -l app=investorcenter-backend --tail=20
echo ""

echo "✅ Done! The market overview fix is now live."
echo ""
echo "🔍 Next Steps:"
echo "1. Test the API: curl https://investorcenter.ai/api/v1/markets/indices"
echo "2. Monitor logs: kubectl logs -n $NAMESPACE -l app=investorcenter-backend -f"
echo "3. Check status: kubectl get pods -n $NAMESPACE"
echo ""
