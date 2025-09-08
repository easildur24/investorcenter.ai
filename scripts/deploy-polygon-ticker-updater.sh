#!/bin/bash

# Deploy Polygon Ticker Updater CronJob
# This script builds and deploys the Polygon ticker updater to EKS

set -e

echo "🚀 Deploying Polygon Ticker Updater CronJob"

# Configuration
AWS_ACCOUNT_ID="360358043271"
AWS_REGION="us-east-1"
ECR_REPOSITORY="investorcenter/polygon-ticker-updater"
IMAGE_TAG="latest"
NAMESPACE="investorcenter"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Check prerequisites
echo "Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed"
    exit 1
fi

if ! command -v kubectl &> /dev/null; then
    print_error "kubectl is not installed"
    exit 1
fi

if ! command -v aws &> /dev/null; then
    print_error "AWS CLI is not installed"
    exit 1
fi

print_status "Prerequisites check passed"

# Step 1: Build Docker image
echo -e "\n📦 Building Docker image..."
docker build -f docker/polygon-ticker-updater/Dockerfile -t ${ECR_REPOSITORY}:${IMAGE_TAG} .
print_status "Docker image built successfully"

# Step 2: Login to ECR
echo -e "\n🔐 Logging in to Amazon ECR..."
aws ecr get-login-password --region ${AWS_REGION} | docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com
print_status "Logged in to ECR"

# Step 3: Tag and push image
echo -e "\n📤 Pushing image to ECR..."
docker tag ${ECR_REPOSITORY}:${IMAGE_TAG} ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY}:${IMAGE_TAG}
docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${ECR_REPOSITORY}:${IMAGE_TAG}
print_status "Image pushed to ECR"

# Step 4: Check if Polygon API secret exists
echo -e "\n🔑 Checking Polygon API secret..."
if kubectl get secret polygon-api-secret -n ${NAMESPACE} &> /dev/null; then
    print_status "Polygon API secret exists"
else
    print_warning "Polygon API secret not found!"
    echo "Please create the secret with your Polygon API key:"
    echo ""
    echo "kubectl create secret generic polygon-api-secret \\"
    echo "  --from-literal=api-key=YOUR_POLYGON_API_KEY \\"
    echo "  -n ${NAMESPACE}"
    echo ""
    read -p "Do you want to continue without the secret? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Step 5: Delete old cronjob if exists
echo -e "\n🗑️  Checking for existing cronjob..."
if kubectl get cronjob ticker-update -n ${NAMESPACE} &> /dev/null; then
    print_warning "Found existing ticker-update cronjob, deleting..."
    kubectl delete cronjob ticker-update -n ${NAMESPACE}
    print_status "Old cronjob deleted"
fi

if kubectl get cronjob polygon-ticker-update -n ${NAMESPACE} &> /dev/null; then
    print_warning "Found existing polygon-ticker-update cronjob, deleting..."
    kubectl delete cronjob polygon-ticker-update -n ${NAMESPACE}
    print_status "Old cronjob deleted"
fi

# Step 6: Apply new cronjob
echo -e "\n🚀 Deploying new Polygon ticker update cronjob..."
kubectl apply -f k8s/polygon-ticker-update-cronjob.yaml
print_status "Cronjob deployed successfully"

# Step 7: Verify deployment
echo -e "\n✅ Verifying deployment..."
kubectl get cronjob polygon-ticker-update -n ${NAMESPACE}

# Step 8: Create a test job to run immediately (optional)
echo -e "\n🧪 Do you want to create a test job to run immediately? (y/n)"
read -p "" -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Creating test job..."
    kubectl create job --from=cronjob/polygon-ticker-update polygon-ticker-test -n ${NAMESPACE}
    print_status "Test job created"
    
    echo "Waiting for job to start..."
    sleep 5
    
    echo "Job status:"
    kubectl get jobs -n ${NAMESPACE} | grep polygon-ticker-test
    
    echo -e "\nTo view logs, run:"
    echo "kubectl logs -f job/polygon-ticker-test -n ${NAMESPACE}"
fi

echo -e "\n${GREEN}✨ Deployment complete!${NC}"
echo ""
echo "The Polygon ticker updater will run daily at 6:30 AM UTC"
echo "It will fetch only new and updated tickers since the last sync"
echo ""
echo "Useful commands:"
echo "  View cronjob: kubectl get cronjob polygon-ticker-update -n ${NAMESPACE}"
echo "  View jobs:    kubectl get jobs -n ${NAMESPACE}"
echo "  View logs:    kubectl logs -f job/<job-name> -n ${NAMESPACE}"
echo "  Manual run:   kubectl create job --from=cronjob/polygon-ticker-update manual-run -n ${NAMESPACE}"