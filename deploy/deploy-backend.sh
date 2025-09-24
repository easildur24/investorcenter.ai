#!/bin/bash

# Deploy Backend Service to Kubernetes
# This script handles the complete deployment of the backend API service

set -e

# Configuration
AWS_ACCOUNT_ID="360358043271"
AWS_REGION="us-east-1"
ECR_REPO="investorcenter/backend"
K8S_NAMESPACE="investorcenter"
IMAGE_TAG="${1:-latest}"

echo "🚀 Deploying Backend Service v${IMAGE_TAG}"
echo "============================================"

# Step 1: Build Go binary
echo "🔨 Building Go binary..."
cd backend
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o main .

# Step 2: Build Docker Image
echo "📦 Building Docker image..."
docker build -t backend:${IMAGE_TAG} .

# Step 3: Tag for ECR
echo "🏷️  Tagging image for ECR..."
docker tag backend:${IMAGE_TAG} ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPO}:${IMAGE_TAG}

# Step 4: Login to ECR
echo "🔐 Logging into AWS ECR..."
aws ecr get-login-password --region ${AWS_REGION} | docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

# Step 5: Push to ECR
echo "⬆️  Pushing image to ECR..."
docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPO}:${IMAGE_TAG}

# Step 6: Update Kubernetes deployment
echo "🚀 Updating Kubernetes deployment..."
kubectl set image deployment/backend backend=${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPO}:${IMAGE_TAG} -n ${K8S_NAMESPACE}

# Step 7: Wait for rollout
echo "⏳ Waiting for rollout to complete..."
kubectl rollout status deployment/backend -n ${K8S_NAMESPACE} --timeout=300s

# Step 8: Check pod status
echo "✅ Deployment complete! Checking status..."
kubectl get pods -n ${K8S_NAMESPACE} -l app=backend

echo ""
echo "✨ Backend Service deployed successfully!"
echo "   API endpoint: https://api.investorcenter.ai"
echo "   Use 'kubectl logs -f -n ${K8S_NAMESPACE} -l app=backend' to follow logs"