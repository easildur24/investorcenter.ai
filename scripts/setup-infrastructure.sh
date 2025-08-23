#!/bin/bash

# Setup AWS infrastructure using Terraform
set -e

# Configuration
AWS_REGION="us-east-1"

echo "Setting up AWS infrastructure..."

# Check if AWS CLI is configured
if ! aws sts get-caller-identity > /dev/null 2>&1; then
    echo "Error: AWS CLI is not configured. Please run 'aws configure' first."
    exit 1
fi

# Check if Terraform is installed
if ! command -v terraform &> /dev/null; then
    echo "Error: Terraform is not installed. Please install Terraform first."
    exit 1
fi

# Navigate to terraform directory
cd terraform

# Initialize Terraform
echo "Initializing Terraform..."
terraform init

# Plan the infrastructure
echo "Planning infrastructure changes..."
terraform plan -var="aws_region=${AWS_REGION}"

# Apply the infrastructure
echo "Applying infrastructure changes..."
read -p "Do you want to proceed with creating the infrastructure? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    terraform apply -var="aws_region=${AWS_REGION}" -auto-approve
    
    echo "Infrastructure created successfully!"
    
    # Output important information
    echo "Getting cluster information..."
    terraform output
    
    # Update kubeconfig
    echo "Updating kubeconfig..."
    aws eks update-kubeconfig --region ${AWS_REGION} --name investorcenter-eks
    
    echo "Setup completed! You can now deploy your application."
else
    echo "Infrastructure setup cancelled."
fi
