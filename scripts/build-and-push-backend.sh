#!/bin/bash

# Build and push Backend Docker image to ECR
set -e

# Configuration
AWS_REGION="us-east-1"
ECR_REPOSITORY="investorcenter/backend"
IMAGE_TAG="latest"

# Get AWS account ID
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

# ECR repository URI
ECR_URI="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY}"

echo "Building and pushing Backend Docker image to ECR..."
echo "Repository: ${ECR_URI}"

# Login to ECR
echo "Logging in to ECR..."
aws ecr get-login-password --region ${AWS_REGION} | docker login --username AWS --password-stdin ${ECR_URI}

# Build the Docker image
echo "Building Docker image..."
docker build -t investorcenter/backend:${IMAGE_TAG} -f backend/Dockerfile backend/

# Tag the image for ECR
echo "Tagging image for ECR..."
docker tag investorcenter/backend:${IMAGE_TAG} ${ECR_URI}:${IMAGE_TAG}

# Push the image to ECR
echo "Pushing image to ECR..."
docker push ${ECR_URI}:${IMAGE_TAG}

echo "Image pushed successfully!"
echo "Image URI: ${ECR_URI}:${IMAGE_TAG}"
