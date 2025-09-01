# Production Setup (Kubernetes)

This guide covers deploying InvestorCenter to production using Kubernetes with PostgreSQL and Redis.

## Infrastructure Overview

### Architecture
- Kubernetes Cluster (AWS EKS recommended)
- PostgreSQL 15 with persistent storage (10Gi)
- Redis 7 for caching (5Gi)
- Go API Backend with auto-scaling
- Next.js Frontend
- Load Balancer with SSL/TLS

### Key Components
- k8s/postgres-deployment.yaml - PostgreSQL database
- k8s/redis-deployment.yaml - Redis cache
- k8s/backend-deployment.yaml - Go API
- k8s/secrets.yaml - Database credentials
- k8s/ingress.yaml - Load balancer & SSL

## Quick Start

### Complete Production Setup
```bash
# Deploy full production infrastructure
make setup-prod

# Verify deployment
make status
```

## Manual Deployment Steps

### 1. Prerequisites
```bash
# Install required tools
brew install kubectl terraform eksctl

# Configure AWS CLI
aws configure
```

### 2. Setup Infrastructure
```bash
# Create AWS infrastructure (EKS cluster)
cd terraform
terraform init
terraform plan
terraform apply

# Or use automated script
./scripts/setup-infrastructure.sh
```

### 3. Deploy Database
```bash
# Setup production PostgreSQL
make db-setup-prod

# Run migrations
make prod db-migrate

# Import ticker data
make prod db-import
```

### 4. Deploy Application
```bash
# Build and push Docker images
make docker-build
make docker-push

# Deploy to Kubernetes
make k8s-deploy
```

## Database Management

### Import Data
```bash
# Import to production database
make prod db-import

# Update with latest data
make prod db-update
```

### Access Production Database
```bash
# Via port-forward (automatic)
python scripts/env-manager.py set prod

# Manual port-forward
kubectl port-forward -n investorcenter svc/postgres-service 5433:5432

# Direct database access
export PGPASSWORD=$(kubectl get secret postgres-secret -n investorcenter -o jsonpath='{.data.password}' | base64 -d)
psql -h localhost -p 5433 -U investorcenter -d investorcenter_db
```

### Data Migration Between Environments
```bash
# Export from local
pg_dump investorcenter_db --data-only --table=stocks > stocks_export.sql

# Import to production
make prod db-migrate
kubectl port-forward -n investorcenter svc/postgres-service 5433:5432 &
export PGPASSWORD=$(kubectl get secret postgres-secret -n investorcenter -o jsonpath='{.data.password}' | base64 -d)
psql -h localhost -p 5433 -U investorcenter -d investorcenter_db -f stocks_export.sql
```

## Monitoring and Maintenance

### Health Checks
```bash
# Check all components
make status

# Check specific pods
kubectl get pods -n investorcenter

# Check database directly
kubectl exec -it deployment/postgres -n investorcenter -- psql -U investorcenter -d investorcenter_db -c "SELECT COUNT(*) FROM stocks;"
```

### Log Monitoring
```bash
# Backend application logs
kubectl logs -f deployment/backend -n investorcenter

# Database logs
kubectl logs -f deployment/postgres -n investorcenter

# All pod logs
kubectl logs -f -l app=investorcenter -n investorcenter
```

### Scaling
```bash
# Scale backend
kubectl scale deployment/backend --replicas=3 -n investorcenter

# Scale frontend
kubectl scale deployment/frontend --replicas=3 -n investorcenter
```

## Updates and Maintenance

### Application Updates
```bash
# Update backend
make docker-build-backend
make docker-push
kubectl rollout restart deployment/backend -n investorcenter

# Update frontend
make docker-build-frontend
make docker-push
kubectl rollout restart deployment/frontend -n investorcenter
```

### Data Updates
```bash
# Manual ticker data update
make prod db-update

# Automated updates (CronJob)
kubectl apply -f k8s/ticker-update-cronjob.yaml
```

### Backup and Restore
```bash
# Backup production database
make prod db-backup

# Restore from backup
kubectl port-forward -n investorcenter svc/postgres-service 5433:5432 &
export PGPASSWORD=$(kubectl get secret postgres-secret -n investorcenter -o jsonpath='{.data.password}' | base64 -d)
psql -h localhost -p 5433 -U investorcenter -d investorcenter_db -f backup_prod_20250831.sql
```

## Environment Configuration

### Production Secrets
```bash
# Create database credentials
kubectl create secret generic postgres-secret \
  --from-literal=username=investorcenter \
  --from-literal=password=YOUR_SECURE_PASSWORD \
  -n investorcenter

# Create application secrets
kubectl create secret generic backend-secrets \
  --from-literal=jwt-secret=YOUR_JWT_SECRET \
  --from-literal=market-api-key=YOUR_API_KEY \
  -n investorcenter
```

### Configuration Files
- k8s/backend-config.yaml - Application configuration
- k8s/secrets.yaml - Sensitive credentials
- terraform/variables.tf - Infrastructure settings

## Security Considerations

- SSL/TLS encryption with AWS Certificate Manager
- Private subnets for worker nodes
- Security groups with minimal access
- ECR image scanning enabled
- Kubernetes RBAC configured
- Database password stored in Kubernetes secrets

## Cost Optimization

Current setup costs approximately:
- EKS cluster: ~$73/month
- EC2 instances (2x t3.medium): ~$60/month
- ALB: ~$20/month
- NAT Gateway: ~$45/month
- Total: ~$200/month

Cost reduction strategies:
- Use t3.small instances for development
- Implement cluster autoscaling
- Use Spot instances for non-production workloads

## Troubleshooting

### Common Issues

**Pod not starting:**
```bash
kubectl describe pod <pod-name> -n investorcenter
kubectl logs <pod-name> -n investorcenter
```

**Database connection issues:**
```bash
# Test connection
make prod verify

# Check database pod
kubectl get pod -n investorcenter -l app=postgres
kubectl logs -f deployment/postgres -n investorcenter
```

**Image pull issues:**
```bash
# Check ECR authentication
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin <account-id>.dkr.ecr.us-east-1.amazonaws.com

# Rebuild and push
make docker-build
make docker-push
```

### Cleanup
```bash
# Remove all Kubernetes resources
make k8s-cleanup

# Clean local artifacts
make clean
```

This setup provides enterprise-grade infrastructure with automated deployment and management capabilities.
