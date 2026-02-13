#!/bin/bash
# Deploy Ticker Update CronJob to AWS EKS Production

set -e

echo "🚀 InvestorCenter.ai Production CronJob Deployment"
echo "=================================================="
echo ""

# Check prerequisites
echo "🔍 Checking prerequisites..."

# Check if we're on production context
CURRENT_CONTEXT=$(kubectl config current-context)
echo "Current kubectl context: $CURRENT_CONTEXT"

if [[ "$CURRENT_CONTEXT" == *"rancher"* ]] || [[ "$CURRENT_CONTEXT" == *"docker"* ]] || [[ "$CURRENT_CONTEXT" == *"minikube"* ]]; then
    echo "❌ ERROR: You're connected to a LOCAL Kubernetes cluster!"
    echo "   Current context: $CURRENT_CONTEXT"
    echo ""
    echo "   Switch to production cluster first:"
    echo "   aws eks update-kubeconfig --region us-east-1 --name investorcenter-cluster"
    echo "   kubectl config use-context arn:aws:eks:us-east-1:ACCOUNT:cluster/investorcenter-cluster"
    exit 1
fi

# Check AWS CLI
if ! command -v aws &> /dev/null; then
    echo "❌ AWS CLI not found. Install with: brew install awscli"
    exit 1
fi

# Check Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker not found. Install Docker Desktop"
    exit 1
fi

echo "✅ Prerequisites check passed"
echo ""

# Get AWS account and region info
AWS_ACCOUNT=$(aws sts get-caller-identity --query Account --output text)
AWS_REGION=$(aws configure get region || echo "us-east-1")
ECR_REGISTRY="$AWS_ACCOUNT.dkr.ecr.$AWS_REGION.amazonaws.com"

echo "📋 Deployment Configuration:"
echo "   AWS Account: $AWS_ACCOUNT"
echo "   AWS Region: $AWS_REGION"
echo "   ECR Registry: $ECR_REGISTRY"
echo "   K8s Context: $CURRENT_CONTEXT"
echo ""

# Confirm deployment
read -p "🤔 Continue with production deployment? (y/N): " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Deployment cancelled"
    exit 1
fi

echo ""
echo "🏗️  Step 1: Building Polygon Ticker Updater image..."
echo "===================================================="

# Build the Polygon ticker updater image for linux/amd64 platform (EKS runs on AMD64)
docker build --platform linux/amd64 -f docker/polygon-ticker-updater/Dockerfile -t polygon-ticker-updater:latest .

echo ""
echo "🔐 Step 2: Authenticating with ECR..."
echo "====================================="

# Login to ECR
aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $ECR_REGISTRY

echo ""
echo "🏷️  Step 3: Tagging and pushing image..."
echo "======================================="

# Tag for ECR
ECR_IMAGE="$ECR_REGISTRY/investorcenter/polygon-ticker-updater:latest"
docker tag polygon-ticker-updater:latest $ECR_IMAGE

# Push to ECR
echo "Pushing to: $ECR_IMAGE"
docker push $ECR_IMAGE

echo ""
echo "⚙️  Step 4: Deploying to production cluster..."
echo "============================================"

# Deploy namespace if it doesn't exist
kubectl apply -f k8s/namespace.yaml

# Check for Polygon API secret
if ! kubectl get secret polygon-api-secret -n investorcenter &> /dev/null; then
    echo "❌ ERROR: Polygon API secret not found!"
    echo "   Create it with: kubectl create secret generic polygon-api-secret --from-literal=api-key=YOUR_API_KEY -n investorcenter"
    exit 1
fi

# Check for PostgreSQL secret
if [ -z "$PROD_DB_PASSWORD" ]; then
    echo "❌ ERROR: PROD_DB_PASSWORD environment variable is not set!"
    echo "   Export it before running: export PROD_DB_PASSWORD='your-secure-password'"
    exit 1
fi
kubectl create secret generic postgres-secret \
    --from-literal=username=investorcenter \
    --from-literal=password="$PROD_DB_PASSWORD" \
    -n investorcenter || echo "PostgreSQL secret already exists"

# Deploy the Polygon CronJob
kubectl apply -f k8s/polygon-ticker-update-cronjob.yaml

echo ""
echo "✅ Production CronJob Deployment Complete!"
echo "========================================"
echo ""
echo "📊 Status Check:"
kubectl get cronjobs -n investorcenter
echo ""
echo "🔍 Next Steps:"
echo "1. Monitor CronJob: kubectl get cronjobs -n investorcenter"
echo "2. Test manual run: kubectl create job --from=cronjob/polygon-ticker-update test-prod-$(date +%s) -n investorcenter"
echo "3. View logs: kubectl logs -n investorcenter -l app=polygon-ticker-update --tail=50"
echo ""
echo "📅 Scheduled: Daily at 6:30 AM UTC"
echo "🔄 Next run: $(kubectl get cronjob polygon-ticker-update -n investorcenter -o jsonpath='{.status.lastScheduleTime}' 2>/dev/null || echo 'Not scheduled yet')"
