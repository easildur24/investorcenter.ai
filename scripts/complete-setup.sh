#!/bin/bash

# Complete setup script for InvestorCenter.ai
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
AWS_REGION="us-east-1"
CLUSTER_NAME="investorcenter-eks"
NAMESPACE="investorcenter"
DOMAIN_NAME="investorcenter.com"

echo -e "${BLUE}🚀 InvestorCenter.ai Complete Setup${NC}"
echo "=================================="

# Function to print status
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check prerequisites
echo -e "${BLUE}📋 Checking prerequisites...${NC}"

# Check AWS CLI
if ! command -v aws &> /dev/null; then
    print_error "AWS CLI is not installed"
    exit 1
fi

# Check if AWS is configured
if ! aws sts get-caller-identity > /dev/null 2>&1; then
    print_error "AWS CLI is not configured. Please run 'aws configure' first."
    exit 1
fi

# Check Docker
if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed"
    exit 1
fi

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker."
    exit 1
fi

# Check kubectl
if ! command -v kubectl &> /dev/null; then
    print_error "kubectl is not installed"
    exit 1
fi

# Check Terraform
if ! command -v terraform &> /dev/null; then
    print_error "Terraform is not installed"
    exit 1
fi

# Check Node.js
if ! command -v node &> /dev/null; then
    print_error "Node.js is not installed"
    exit 1
fi

print_status "All prerequisites are met"

# Get AWS account info
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
print_status "AWS Account ID: $AWS_ACCOUNT_ID"

# Install Node.js dependencies
echo -e "${BLUE}📦 Installing Node.js dependencies...${NC}"
npm install
print_status "Dependencies installed"

# Setup infrastructure
echo -e "${BLUE}🏗️  Setting up AWS infrastructure...${NC}"
cd terraform

# Initialize Terraform
terraform init

# Plan infrastructure
echo -e "${YELLOW}Planning infrastructure changes...${NC}"
terraform plan -var="aws_region=${AWS_REGION}" -var="domain_name=${DOMAIN_NAME}"

# Ask for confirmation
echo -e "${YELLOW}Do you want to proceed with creating the infrastructure? This will create AWS resources that incur costs.${NC}"
read -p "Continue? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_error "Infrastructure setup cancelled"
    exit 1
fi

# Apply infrastructure
echo -e "${BLUE}Creating infrastructure...${NC}"
terraform apply -var="aws_region=${AWS_REGION}" -var="domain_name=${DOMAIN_NAME}" -auto-approve

print_status "Infrastructure created successfully"

# Get certificate ARN
CERT_ARN=$(terraform output -raw certificate_arn 2>/dev/null || echo "")

# Go back to root directory
cd ..

# Update kubeconfig
echo -e "${BLUE}🔧 Updating kubeconfig...${NC}"
aws eks update-kubeconfig --region ${AWS_REGION} --name ${CLUSTER_NAME}
print_status "Kubeconfig updated"

# Wait for cluster to be ready
echo -e "${BLUE}⏳ Waiting for EKS cluster to be ready...${NC}"
while true; do
    CLUSTER_STATUS=$(aws eks describe-cluster --name ${CLUSTER_NAME} --query cluster.status --output text)
    if [ "$CLUSTER_STATUS" = "ACTIVE" ]; then
        break
    fi
    echo "Cluster status: $CLUSTER_STATUS. Waiting..."
    sleep 30
done
print_status "EKS cluster is ready"

# Wait for node group to be ready
echo -e "${BLUE}⏳ Waiting for node group to be ready...${NC}"
while true; do
    NODE_STATUS=$(aws eks describe-nodegroup --cluster-name ${CLUSTER_NAME} --nodegroup-name ${CLUSTER_NAME}-node-group --query nodegroup.status --output text)
    if [ "$NODE_STATUS" = "ACTIVE" ]; then
        break
    fi
    echo "Node group status: $NODE_STATUS. Waiting..."
    sleep 30
done
print_status "Node group is ready"

# Build and push Docker images
echo -e "${BLUE}🐳 Building and pushing Docker images...${NC}"
./scripts/build-and-push.sh
./scripts/build-and-push-backend.sh
print_status "Docker images built and pushed"

# Update deployments with correct image URIs
echo -e "${BLUE}📝 Updating Kubernetes deployments...${NC}"
FRONTEND_ECR_URI="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/investorcenter/app:latest"
BACKEND_ECR_URI="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/investorcenter/backend:latest"

sed -i.bak "s|investorcenter/app:latest|${FRONTEND_ECR_URI}|g" k8s/deployment.yaml
sed -i.bak "s|investorcenter/backend:latest|${BACKEND_ECR_URI}|g" k8s/backend-deployment.yaml
print_status "Deployments updated with ECR image URIs"

# Update ingress with certificate ARN if available
if [ ! -z "$CERT_ARN" ]; then
    echo -e "${BLUE}🔒 Updating ingress with SSL certificate...${NC}"
    sed -i.bak "s|arn:aws:acm:us-east-1:ACCOUNT_ID:certificate/CERTIFICATE_ID|${CERT_ARN}|g" k8s/ingress.yaml
    print_status "Ingress updated with certificate ARN"
else
    print_warning "Certificate ARN not found. You may need to update the ingress manually."
fi

# Deploy application
echo -e "${BLUE}🚀 Deploying application to Kubernetes...${NC}"
./scripts/deploy.sh
print_status "Application deployed"

# Wait for pods to be ready
echo -e "${BLUE}⏳ Waiting for pods to be ready...${NC}"
kubectl wait --for=condition=ready pod -l app=investorcenter-app -n ${NAMESPACE} --timeout=300s
print_status "Pods are ready"

# Get ingress information
echo -e "${BLUE}🌐 Getting ingress information...${NC}"
sleep 30  # Wait a bit for ingress to be processed

INGRESS_HOST=$(kubectl get ingress investorcenter-ingress -n ${NAMESPACE} -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>/dev/null || echo "")

if [ ! -z "$INGRESS_HOST" ]; then
    print_status "Load balancer hostname: $INGRESS_HOST"
    
    # Create Route53 records
    echo -e "${BLUE}📍 Creating Route53 records...${NC}"
    
    # Get hosted zone ID
    ZONE_ID=$(aws route53 list-hosted-zones-by-name --dns-name ${DOMAIN_NAME} --query "HostedZones[0].Id" --output text | sed 's|/hostedzone/||')
    
    if [ "$ZONE_ID" != "None" ] && [ ! -z "$ZONE_ID" ]; then
        # Get ALB zone ID
        ALB_ZONE_ID=$(aws elbv2 describe-load-balancers --query "LoadBalancers[?DNSName=='${INGRESS_HOST}'].CanonicalHostedZoneId" --output text)
        
        if [ ! -z "$ALB_ZONE_ID" ]; then
            # Create Route53 record for root domain
            cat > /tmp/route53-record.json << EOF
{
    "Changes": [{
        "Action": "UPSERT",
        "ResourceRecordSet": {
            "Name": "${DOMAIN_NAME}",
            "Type": "A",
            "AliasTarget": {
                "DNSName": "${INGRESS_HOST}",
                "EvaluateTargetHealth": true,
                "HostedZoneId": "${ALB_ZONE_ID}"
            }
        }
    }]
}
EOF
            
            aws route53 change-resource-record-sets --hosted-zone-id ${ZONE_ID} --change-batch file:///tmp/route53-record.json
            print_status "Route53 record created for ${DOMAIN_NAME}"
            
            # Create Route53 record for www subdomain
            cat > /tmp/route53-record-www.json << EOF
{
    "Changes": [{
        "Action": "UPSERT",
        "ResourceRecordSet": {
            "Name": "www.${DOMAIN_NAME}",
            "Type": "A",
            "AliasTarget": {
                "DNSName": "${INGRESS_HOST}",
                "EvaluateTargetHealth": true,
                "HostedZoneId": "${ALB_ZONE_ID}"
            }
        }
    }]
}
EOF
            
            aws route53 change-resource-record-sets --hosted-zone-id ${ZONE_ID} --change-batch file:///tmp/route53-record-www.json
            print_status "Route53 record created for www.${DOMAIN_NAME}"
            
            # Clean up temp files
            rm -f /tmp/route53-record.json /tmp/route53-record-www.json
        else
            print_warning "Could not get ALB zone ID. You may need to create Route53 records manually."
        fi
    else
        print_warning "Could not find hosted zone for ${DOMAIN_NAME}. You may need to create Route53 records manually."
    fi
else
    print_warning "Ingress hostname not available yet. It may take a few minutes to provision."
fi

# Final status
echo -e "${GREEN}🎉 Setup completed successfully!${NC}"
echo "=================================="
echo -e "${BLUE}📊 Deployment Summary:${NC}"
echo "• EKS Cluster: ${CLUSTER_NAME}"
echo "• Namespace: ${NAMESPACE}"
echo "• Domain: ${DOMAIN_NAME}"
echo "• Region: ${AWS_REGION}"

if [ ! -z "$INGRESS_HOST" ]; then
    echo "• Load Balancer: ${INGRESS_HOST}"
fi

echo ""
echo -e "${BLUE}🔍 Useful Commands:${NC}"
echo "• Check pods: kubectl get pods -n ${NAMESPACE}"
echo "• Check services: kubectl get services -n ${NAMESPACE}"
echo "• Check ingress: kubectl get ingress -n ${NAMESPACE}"
echo "• View logs: kubectl logs -f deployment/investorcenter-app -n ${NAMESPACE}"

echo ""
echo -e "${YELLOW}⏰ Note: It may take 5-10 minutes for DNS to propagate and SSL certificate to be ready.${NC}"
echo -e "${YELLOW}Visit https://${DOMAIN_NAME} once DNS propagation is complete.${NC}"

echo ""
echo -e "${GREEN}✨ Your InvestorCenter.ai application is now live!${NC}"
