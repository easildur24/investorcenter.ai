#!/bin/bash

# Deploy Crypto Service to Kubernetes
# This script handles the complete deployment of the crypto service

set -e

# Configuration
AWS_ACCOUNT_ID="360358043271"
AWS_REGION="us-east-1"
ECR_REPO="investorcenter/crypto-service"
K8S_NAMESPACE="investorcenter"
IMAGE_TAG="${1:-latest}"

echo "🚀 Deploying Crypto Service v${IMAGE_TAG}"
echo "============================================"

# Step 1: Build Docker Image
echo "📦 Building Docker image..."
cd scripts
docker build -f Dockerfile.crypto-service -t crypto-service:${IMAGE_TAG} .

# Step 2: Tag for ECR
echo "🏷️  Tagging image for ECR..."
docker tag crypto-service:${IMAGE_TAG} ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPO}:${IMAGE_TAG}

# Step 3: Login to ECR
echo "🔐 Logging into AWS ECR..."
aws ecr get-login-password --region ${AWS_REGION} | docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

# Step 4: Push to ECR
echo "⬆️  Pushing image to ECR..."
docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPO}:${IMAGE_TAG}

# Step 5: Deploy Redis if not exists
echo "🗄️  Checking Redis deployment..."
if ! kubectl get deployment redis -n ${K8S_NAMESPACE} &>/dev/null; then
    echo "   Installing Redis..."
    kubectl apply -f ../backend/k8s/redis-deployment.yaml
    kubectl wait --for=condition=available --timeout=300s deployment/redis -n ${K8S_NAMESPACE}
else
    echo "   Redis already deployed"
fi

# Step 6: Deploy Crypto Service
echo "🚀 Deploying Crypto Service to Kubernetes..."
kubectl apply -f ../backend/k8s/crypto-service-deployment.yaml

# Step 7: Update image if using latest tag
if [ "${IMAGE_TAG}" == "latest" ]; then
    echo "🔄 Forcing pod restart for latest tag..."
    kubectl rollout restart deployment/crypto-service -n ${K8S_NAMESPACE}
fi

# Step 8: Wait for deployment
echo "⏳ Waiting for deployment to be ready..."
kubectl rollout status deployment/crypto-service -n ${K8S_NAMESPACE} --timeout=300s

# Step 9: Check pod status
echo "✅ Deployment complete! Checking status..."
kubectl get pods -n ${K8S_NAMESPACE} -l app=crypto-service

# Step 10: Show logs
echo ""
echo "📋 Recent logs from crypto service:"
kubectl logs -n ${K8S_NAMESPACE} -l app=crypto-service --tail=20

echo ""
echo "✨ Crypto Service deployed successfully!"
echo "   Use 'kubectl logs -f -n ${K8S_NAMESPACE} -l app=crypto-service' to follow logs"