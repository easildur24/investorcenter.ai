# Production Deployment Guide

Complete guide for deploying InvestorCenter.ai to production with automated ticker updates.

## Production Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   AWS EKS       │    │   PostgreSQL     │    │   Ticker        │
│   Cluster       │    │   Database       │    │   CronJob       │
│                 │    │   (RDS/K8s)      │    │   (Weekly)      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Prerequisites

### 1. Production Kubernetes Cluster
- **AWS EKS** (recommended) or other cloud Kubernetes
- **NOT local Kubernetes** (Docker Desktop, Rancher Desktop, etc.)
- Cluster should have persistent storage and networking configured

### 2. Container Registry
- **AWS ECR**, Docker Hub, or other container registry
- Ability to push/pull Docker images from production cluster

### 3. Database Setup
Choose one:
- **Option A**: PostgreSQL in Kubernetes (current k8s/postgres-deployment.yaml)
- **Option B**: AWS RDS PostgreSQL (recommended for production)

## Deployment Steps

### Step 1: Verify Production Context
```bash
# Check current kubectl context
kubectl config current-context

# Should show production cluster (e.g., aws-eks-cluster), NOT:
# - rancher-desktop
# - docker-desktop  
# - minikube

# Switch to production if needed
kubectl config use-context your-production-cluster
```

### Step 2: Deploy Database Infrastructure
```bash
# Deploy to production cluster
make prod-k8s-setup

# Verify PostgreSQL is running
kubectl get pods -n investorcenter
```

### Step 3: Import Initial Data
```bash
# Import ticker data to production database
make db-import-prod

# Verify data was imported
kubectl port-forward -n investorcenter svc/postgres-service 5433:5432 &
export PGPASSWORD="prod_investorcenter_456"
psql -h localhost -p 5433 -U investorcenter -d investorcenter_db -c "SELECT COUNT(*) FROM stocks;"
```

### Step 4: Build and Push Docker Image
```bash
# Build ticker updater image
./scripts/build-ticker-updater.sh

# Tag for your registry (replace with your registry URL)
docker tag investorcenter/ticker-updater:latest your-registry.com/investorcenter/ticker-updater:latest

# Push to registry
docker push your-registry.com/investorcenter/ticker-updater:latest
```

### Step 5: Update CronJob Configuration
Edit `k8s/ticker-update-cronjob.yaml`:
```yaml
# Update image to use your registry
image: your-registry.com/investorcenter/ticker-updater:latest

# For AWS EKS with RDS, update database connection:
env:
- name: DB_HOST
  value: "your-rds-endpoint.amazonaws.com"  # Instead of postgres-service
- name: DB_SSLMODE  
  value: "require"  # Enable SSL for RDS
```

### Step 6: Deploy CronJob
```bash
# Deploy ticker update CronJob to production
kubectl apply -f k8s/ticker-update-cronjob.yaml

# Verify deployment
make prod-cron-status
```

### Step 7: Test Production CronJob
```bash
# Trigger manual test run
kubectl create job --from=cronjob/ticker-update test-prod-update -n investorcenter

# Monitor test run
kubectl logs job/test-prod-update -n investorcenter -f

# Check results
make prod-cron-logs
```

## Configuration for Different Environments

### AWS EKS + RDS PostgreSQL (Recommended)
```yaml
# k8s/ticker-update-cronjob.yaml
env:
- name: DB_HOST
  value: "investorcenter-db.cluster-xyz.us-east-1.rds.amazonaws.com"
- name: DB_PORT
  value: "5432"
- name: DB_SSLMODE
  value: "require"
- name: DB_USER
  valueFrom:
    secretKeyRef:
      name: rds-secret
      key: username
- name: DB_PASSWORD
  valueFrom:
    secretKeyRef:
      name: rds-secret
      key: password
```

### AWS EKS + PostgreSQL in K8s
```yaml
# k8s/ticker-update-cronjob.yaml (current configuration)
env:
- name: DB_HOST
  value: "postgres-service"
- name: DB_SSLMODE
  value: "disable"  # Internal cluster communication
```

## Monitoring

### CronJob Status
```bash
# Check CronJob schedule and status
make prod-cron-status

# View recent logs
make prod-cron-logs

# Check next scheduled run
kubectl describe cronjob ticker-update -n investorcenter
```

### Expected Behavior

#### Normal Weekly Run (No New IPOs)
```
INFO - Downloaded 6916 raw ticker records
INFO - Filtered to 4643 valid tickers  
INFO - Import completed: 0 inserted, 4643 skipped
INFO - Update completed in 1.5 seconds
```

#### When New IPOs Available
```
INFO - Downloaded 6920 raw ticker records  
INFO - Filtered to 4647 valid tickers
INFO - Import completed: 4 inserted, 4643 skipped
INFO - New companies added: NEWCO, STARTUP, GROWTH, TECH
```

## Security Considerations

- **Non-root containers**: CronJob runs as user 1000
- **Secret management**: Database credentials stored in K8s secrets
- **Network policies**: Consider restricting CronJob network access
- **Image scanning**: Scan Docker images for vulnerabilities before deployment

## Troubleshooting

### CronJob Not Running
1. Check cluster has sufficient resources
2. Verify image can be pulled from registry
3. Check database connectivity from cluster
4. Review K8s events: `kubectl get events -n investorcenter`

### Database Connection Issues  
1. Verify database is accessible from cluster
2. Check security groups/firewall rules
3. Validate database credentials in secrets
4. Test connection manually from pod

## Important Notes

- **Local Development**: CronJob should NOT be deployed to local clusters
- **Production Only**: Automated updates only run in production environment
- **Manual Updates**: Use `make db-import` for local development data updates
- **Registry Required**: Production clusters need image registry access
