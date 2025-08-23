#!/bin/bash

# Deploy application to EKS
set -e

# Configuration
AWS_REGION="us-east-1"
CLUSTER_NAME="investorcenter-eks"
NAMESPACE="investorcenter"

echo "Deploying application to EKS cluster: ${CLUSTER_NAME}"

# Update kubeconfig
echo "Updating kubeconfig..."
aws eks update-kubeconfig --region ${AWS_REGION} --name ${CLUSTER_NAME}

# Create namespace if it doesn't exist
echo "Creating namespace..."
kubectl apply -f k8s/namespace.yaml

# Apply secrets first
echo "Applying secrets..."
kubectl apply -f k8s/secrets.yaml

# Apply database manifests
echo "Applying database manifests..."
kubectl apply -f k8s/postgres-deployment.yaml
kubectl apply -f k8s/redis-deployment.yaml

# Wait for databases to be ready
echo "Waiting for databases to be ready..."
kubectl rollout status deployment/postgres -n ${NAMESPACE}
kubectl rollout status deployment/redis -n ${NAMESPACE}

# Apply backend manifests
echo "Applying backend manifests..."
kubectl apply -f k8s/backend-deployment.yaml
kubectl apply -f k8s/backend-service.yaml

# Wait for backend to be ready
echo "Waiting for backend to be ready..."
kubectl rollout status deployment/investorcenter-backend -n ${NAMESPACE}

# Apply frontend manifests
echo "Applying frontend manifests..."
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml

# Wait for frontend deployment to be ready
echo "Waiting for frontend deployment to be ready..."
kubectl rollout status deployment/investorcenter-app -n ${NAMESPACE}

# Apply ingress (after ALB controller is ready)
echo "Applying ingress..."
kubectl apply -f k8s/ingress.yaml

echo "Deployment completed successfully!"

# Show deployment status
echo "Deployment status:"
kubectl get pods -n ${NAMESPACE}
kubectl get services -n ${NAMESPACE}
kubectl get ingress -n ${NAMESPACE}
