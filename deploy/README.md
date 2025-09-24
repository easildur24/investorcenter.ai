# InvestorCenter Deployment Tools

## Overview

This directory contains deployment tools and automation scripts to simplify the deployment process for InvestorCenter services.

## Directory Structure

```
deploy/
├── deploy-crypto.sh      # Deploy crypto service to Kubernetes
├── deploy-backend.sh     # Deploy backend API to Kubernetes  
├── deploy-frontend.sh    # Deploy frontend to S3/CloudFront
└── README.md            # This file

helm/investorcenter/
├── Chart.yaml           # Helm chart metadata
├── values.yaml          # Default configuration values
└── templates/           # Kubernetes resource templates
    └── crypto-service.yaml

.github/workflows/
├── deploy-crypto.yml    # GitHub Actions for crypto service CI/CD
└── deploy-backend.yml   # GitHub Actions for backend CI/CD

backend/k8s/
├── redis-deployment.yaml              # Redis cache deployment
├── crypto-service-deployment.yaml     # Crypto service deployment
└── crypto-postgres-sync-cronjob.yaml  # PostgreSQL sync CronJob

scripts/
├── Dockerfile.crypto-service          # Docker image for crypto service
├── Dockerfile.crypto-postgres-sync    # Docker image for sync job
└── requirements-crypto.txt            # Python dependencies
```

## Quick Start

### 1. Deploy Crypto Service

```bash
# Deploy crypto service with latest image
./deploy/deploy-crypto.sh

# Deploy with specific version
./deploy/deploy-crypto.sh v1.2.3
```

### 2. Deploy Backend API

```bash
# Deploy backend service
./deploy/deploy-backend.sh
```

### 3. Deploy Frontend

```bash
# Deploy frontend to S3/CloudFront
./deploy/deploy-frontend.sh
```

## Using Helm Charts

The Helm chart provides a more advanced deployment option with better configuration management.

### Install/Upgrade with Helm

```bash
# Add Bitnami repo for dependencies
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Install the chart
helm install investorcenter ./helm/investorcenter \
  --namespace investorcenter \
  --create-namespace

# Upgrade with custom values
helm upgrade investorcenter ./helm/investorcenter \
  --namespace investorcenter \
  --values custom-values.yaml
```

### Custom Values

Create a `custom-values.yaml` file to override defaults:

```yaml
crypto:
  replicaCount: 2
  resources:
    limits:
      cpu: 500m
      memory: 1Gi

backend:
  ingress:
    host: api.yourdomain.com
```

## GitHub Actions CI/CD

Automated deployments are triggered on push to main branch:

- **Crypto Service**: Deploys when `scripts/crypto_*.py` files change
- **Backend API**: Deploys when `backend/` files change
- **Frontend**: Manual deployment via workflow dispatch

### Required GitHub Secrets

```
AWS_ACCESS_KEY_ID       # AWS access key
AWS_SECRET_ACCESS_KEY   # AWS secret key
```

## Manual Kubernetes Operations

### Deploy Individual Components

```bash
# Deploy Redis
kubectl apply -f backend/k8s/redis-deployment.yaml

# Deploy Crypto Service
kubectl apply -f backend/k8s/crypto-service-deployment.yaml

# Deploy PostgreSQL Sync CronJob
kubectl apply -f backend/k8s/crypto-postgres-sync-cronjob.yaml
```

### Check Status

```bash
# View all pods
kubectl get pods -n investorcenter

# View crypto service logs
kubectl logs -n investorcenter -l app=crypto-service -f

# Check CronJob status
kubectl get cronjobs -n investorcenter
```

### Scale Services

```bash
# Scale crypto service
kubectl scale deployment crypto-service -n investorcenter --replicas=3

# Scale backend
kubectl scale deployment backend -n investorcenter --replicas=5
```

## Docker Images

### Build Images Locally

```bash
# Build crypto service image
cd scripts
docker build -f Dockerfile.crypto-service -t crypto-service:local .

# Build sync job image
docker build -f Dockerfile.crypto-postgres-sync -t crypto-sync:local .
```

### Push to ECR

```bash
# Login to ECR
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin 360358043271.dkr.ecr.us-east-1.amazonaws.com

# Tag and push
docker tag crypto-service:local 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/crypto-service:latest
docker push 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/crypto-service:latest
```

## Monitoring

### View Metrics

```bash
# CPU and Memory usage
kubectl top pods -n investorcenter

# Node resource usage
kubectl top nodes
```

### Health Checks

```bash
# Check Redis connection
kubectl exec -it -n investorcenter deploy/redis -- redis-cli ping

# Check crypto service health
kubectl exec -it -n investorcenter deploy/crypto-service -- curl localhost:8080/health
```

## Troubleshooting

### Common Issues

1. **Pods not starting**
   ```bash
   kubectl describe pod <pod-name> -n investorcenter
   kubectl logs <pod-name> -n investorcenter --previous
   ```

2. **Image pull errors**
   ```bash
   # Verify ECR login
   aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 360358043271.dkr.ecr.us-east-1.amazonaws.com
   ```

3. **Database connection issues**
   ```bash
   # Check secrets
   kubectl get secret postgres-credentials -n investorcenter -o yaml
   ```

## Rollback

### Rollback Deployment

```bash
# View rollout history
kubectl rollout history deployment/crypto-service -n investorcenter

# Rollback to previous version
kubectl rollout undo deployment/crypto-service -n investorcenter

# Rollback to specific revision
kubectl rollout undo deployment/crypto-service -n investorcenter --to-revision=2
```

### Rollback with Helm

```bash
# List releases
helm list -n investorcenter

# View history
helm history investorcenter -n investorcenter

# Rollback to previous release
helm rollback investorcenter -n investorcenter
```

## Best Practices

1. **Always test in staging first**
2. **Use version tags instead of `latest`**
3. **Monitor logs during deployment**
4. **Keep secrets in AWS Secrets Manager**
5. **Use Helm for production deployments**
6. **Enable autoscaling for production workloads**
7. **Set up alerts for critical services**

## Support

For issues or questions:
1. Check the logs: `kubectl logs -n investorcenter <pod-name>`
2. Review events: `kubectl get events -n investorcenter`
3. Contact the DevOps team