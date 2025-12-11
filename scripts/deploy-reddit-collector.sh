#!/bin/bash
# Deploy Reddit Collector to AWS EKS
set -e

echo "🚀 Deploying Reddit Collector to AWS EKS"

# Configuration
AWS_PROFILE="AdministratorAccess-360358043271"
AWS_REGION="us-east-1"
ECR_REGISTRY="360358043271.dkr.ecr.us-east-1.amazonaws.com"
IMAGE_NAME="investorcenter/reddit-collector"
TAG="latest"

# Step 1: Build Docker image for linux/amd64 platform (EKS runs on AMD64)
echo ""
echo "📦 Building Docker image..."
docker build \
  --platform linux/amd64 \
  -f scripts/Dockerfile.reddit-collector \
  -t ${IMAGE_NAME}:${TAG} \
  .

# Step 2: Tag for ECR
echo ""
echo "🏷️  Tagging image for ECR..."
docker tag ${IMAGE_NAME}:${TAG} ${ECR_REGISTRY}/${IMAGE_NAME}:${TAG}

# Step 3: Login to ECR
echo ""
echo "🔐 Logging in to ECR..."
AWS_PROFILE=${AWS_PROFILE} aws ecr get-login-password --region ${AWS_REGION} | \
  docker login --username AWS --password-stdin ${ECR_REGISTRY}

# Step 4: Create ECR repository if it doesn't exist
echo ""
echo "📂 Ensuring ECR repository exists..."
AWS_PROFILE=${AWS_PROFILE} aws ecr describe-repositories \
  --repository-names ${IMAGE_NAME} \
  --region ${AWS_REGION} 2>/dev/null || \
AWS_PROFILE=${AWS_PROFILE} aws ecr create-repository \
  --repository-name ${IMAGE_NAME} \
  --region ${AWS_REGION}

# Step 5: Push to ECR
echo ""
echo "⬆️  Pushing image to ECR..."
docker push ${ECR_REGISTRY}/${IMAGE_NAME}:${TAG}

# Step 6: Deploy to Kubernetes
echo ""
echo "☸️  Deploying to Kubernetes..."
kubectl apply -f k8s/reddit-collector-cronjob.yaml

# Step 7: Verify deployment
echo ""
echo "✅ Verifying CronJob..."
kubectl get cronjob reddit-collector -n investorcenter

echo ""
echo "✓ Deployment complete!"
echo ""
echo "📊 Useful commands:"
echo "  # View CronJob schedule:"
echo "  kubectl get cronjob reddit-collector -n investorcenter"
echo ""
echo "  # Manually trigger a job:"
echo "  kubectl create job --from=cronjob/reddit-collector reddit-collector-manual -n investorcenter"
echo ""
echo "  # View job logs:"
echo "  kubectl logs -n investorcenter -l app=reddit-collector --tail=100"
echo ""
echo "  # Check job history:"
echo "  kubectl get jobs -n investorcenter -l app=reddit-collector"
