# Crypto Price Updater - AWS Deployment Guide

## Overview

This guide walks through deploying the crypto price updater to AWS EKS to ensure continuous 24/7 cryptocurrency price updates.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        AWS EKS Cluster                       │
│                                                               │
│  ┌────────────────────┐         ┌─────────────────────┐     │
│  │  Redis Deployment  │◄────────│  Crypto Updater     │     │
│  │  (cache service)   │         │  (Deployment)       │     │
│  │                    │         │                     │     │
│  │  - 512MB RAM       │         │  - Fetches prices   │     │
│  │  - Persistent Vol  │         │  - Updates Redis    │     │
│  │  - LRU eviction    │         │  - 3min intervals   │     │
│  └────────────────────┘         └─────────────────────┘     │
│           ▲                              │                   │
│           │                              │                   │
│           │                              ▼                   │
│           │                    ┌─────────────────────┐      │
│           └────────────────────│  Backend API        │      │
│                                │  (reads from Redis) │      │
│                                └─────────────────────┘      │
└─────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
                                   Frontend
```

## Prerequisites

1. **AWS CLI configured** with AdministratorAccess-360358043271 profile
2. **Docker** installed and running
3. **kubectl** configured for investorcenter-cluster
4. **Redis** must be deployed first (see step 2 below)

## Step-by-Step Deployment

### Step 1: Authenticate with AWS

```bash
# Login to AWS SSO
aws sso login --profile AdministratorAccess-360358043271

# Verify authentication
export AWS_PROFILE=AdministratorAccess-360358043271
aws sts get-caller-identity
```

### Step 2: Deploy Redis to EKS

Redis must be deployed BEFORE the crypto updater.

```bash
# Deploy Redis
kubectl apply -f k8s/redis-deployment.yaml

# Wait for Redis to be ready (this may take 1-2 minutes)
kubectl wait --for=condition=ready pod -l app=redis -n investorcenter --timeout=120s

# Verify Redis is running
kubectl get pods -n investorcenter | grep redis

# Test Redis connection
kubectl exec -n investorcenter deployment/redis -- redis-cli ping
# Expected output: PONG
```

### Step 3: Create ECR Repository (One-time setup)

```bash
# Create the ECR repository
export AWS_PROFILE=AdministratorAccess-360358043271
aws ecr create-repository \
  --repository-name investorcenter/crypto-price-updater \
  --region us-east-1

# Expected output: JSON with repository details
```

### Step 4: Build and Push Docker Image

```bash
# Build the Docker image
docker build -f Dockerfile.crypto-updater -t crypto-price-updater:latest .

# Tag for ECR
docker tag crypto-price-updater:latest \
  360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/crypto-price-updater:latest

# Login to ECR
export AWS_PROFILE=AdministratorAccess-360358043271
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin \
  360358043271.dkr.ecr.us-east-1.amazonaws.com

# Push to ECR
docker push 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/crypto-price-updater:latest
```

### Step 5: Deploy Crypto Price Updater

```bash
# Deploy the crypto updater
kubectl apply -f k8s/crypto-price-updater-deployment.yaml

# Wait for deployment to be ready
kubectl rollout status deployment/crypto-price-updater -n investorcenter --timeout=180s

# Check pod status
kubectl get pods -n investorcenter | grep crypto-price-updater
```

### Step 6: Verify Deployment

```bash
# Check logs (should see "🚀 Complete CoinGecko Service Starting")
kubectl logs -n investorcenter -l app=crypto-price-updater --tail=50

# Follow logs in real-time
kubectl logs -n investorcenter -l app=crypto-price-updater -f

# Test Redis has crypto data (wait 30 seconds after deployment)
kubectl exec -n investorcenter deployment/redis -- \
  redis-cli keys "crypto:quote:*" | head -10

# Get Bitcoin price from Redis
kubectl exec -n investorcenter deployment/redis -- \
  redis-cli get crypto:quote:BTC
```

### Step 7: Update Backend to Use Redis Service

Update backend deployment to point to Redis:

```bash
# Edit backend deployment
kubectl edit deployment backend-deployment -n investorcenter

# Add environment variable:
# - name: REDIS_HOST
#   value: "redis-service"
# - name: REDIS_PORT
#   value: "6379"

# Or apply updated manifest if you have it
kubectl apply -f k8s/backend-deployment.yaml

# Wait for backend to restart
kubectl rollout status deployment/backend-deployment -n investorcenter
```

### Step 8: Test End-to-End

```bash
# Port forward to backend
kubectl port-forward -n investorcenter svc/backend-service 8080:8080 &

# Test crypto price API
curl http://localhost:8080/api/v1/crypto/BTC/price | jq '{symbol, price, market_cap_rank}'

# Expected output:
# {
#   "symbol": "BTC",
#   "price": 122072,
#   "market_cap_rank": 1
# }
```

## Automated Deployment Script

For convenience, use the deployment script:

```bash
# Make it executable
chmod +x scripts/deploy-crypto-updater.sh

# Run it
./scripts/deploy-crypto-updater.sh

# Or with a custom tag
./scripts/deploy-crypto-updater.sh v1.0.0
```

## Monitoring

### Check Pod Health

```bash
# Get pod status
kubectl get pods -n investorcenter | grep crypto

# Describe pod for details
kubectl describe pod -n investorcenter -l app=crypto-price-updater

# Check resource usage
kubectl top pod -n investorcenter -l app=crypto-price-updater
```

### Check Logs

```bash
# Recent logs
kubectl logs -n investorcenter -l app=crypto-price-updater --tail=100

# Follow logs
kubectl logs -n investorcenter -l app=crypto-price-updater -f

# Get logs from previous pod (if it crashed)
kubectl logs -n investorcenter -l app=crypto-price-updater --previous
```

### Check Redis Data

```bash
# Count crypto keys in Redis
kubectl exec -n investorcenter deployment/redis -- \
  redis-cli keys "crypto:quote:*" | wc -l

# Get top 10 crypto symbols
kubectl exec -n investorcenter deployment/redis -- \
  redis-cli zrange crypto:symbols:ranked 0 9

# Check specific crypto
kubectl exec -n investorcenter deployment/redis -- \
  redis-cli get crypto:quote:ETH | jq '.current_price'
```

## Troubleshooting

### Pod CrashLoopBackOff

```bash
# Check logs
kubectl logs -n investorcenter -l app=crypto-price-updater --previous

# Common issues:
# 1. Redis not reachable → Check Redis is running
# 2. CoinGecko API rate limit → Wait 60 seconds and it will retry
# 3. Out of memory → Increase memory limit in deployment.yaml
```

### No Data in Redis

```bash
# Check if updater is running
kubectl get pods -n investorcenter | grep crypto-price-updater

# Check logs for API errors
kubectl logs -n investorcenter -l app=crypto-price-updater | grep ERROR

# Common issues:
# 1. DNS resolution failure → Check network policies
# 2. Rate limiting → Normal, it will retry
# 3. No internet access → Check EKS node group security groups
```

### Backend Can't Connect to Redis

```bash
# Test connectivity from backend pod
kubectl exec -n investorcenter deployment/backend-deployment -- \
  nc -zv redis-service 6379

# If it fails:
# 1. Check Redis service exists
kubectl get svc redis-service -n investorcenter

# 2. Check backend env vars
kubectl exec -n investorcenter deployment/backend-deployment -- env | grep REDIS
```

## Update/Redeploy

### Update the Image

```bash
# Build new image
docker build -f Dockerfile.crypto-updater -t crypto-price-updater:v2 .

# Tag and push
docker tag crypto-price-updater:v2 \
  360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/crypto-price-updater:v2
docker push 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/crypto-price-updater:v2

# Update deployment
kubectl set image deployment/crypto-price-updater \
  crypto-price-updater=360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/crypto-price-updater:v2 \
  -n investorcenter

# Watch rollout
kubectl rollout status deployment/crypto-price-updater -n investorcenter
```

### Restart the Pod

```bash
# Restart deployment (keeps same image)
kubectl rollout restart deployment/crypto-price-updater -n investorcenter

# Delete pod (will recreate automatically)
kubectl delete pod -n investorcenter -l app=crypto-price-updater
```

## Scaling Considerations

### DO NOT scale beyond 1 replica

The crypto updater should always have **exactly 1 replica** to avoid:
- Duplicate API calls to CoinGecko (wastes rate limit)
- Race conditions updating Redis
- Unnecessary resource usage

The HPA (Horizontal Pod Autoscaler) is commented out in the manifest for this reason.

### If you need higher availability

Use a **PodDisruptionBudget** instead:

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: crypto-price-updater-pdb
  namespace: investorcenter
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: crypto-price-updater
```

This ensures the pod is recreated quickly if it fails, but doesn't run multiple instances.

## Cost Optimization

### Current Resources

- **Redis**: ~$15/month (t3.small equivalent)
- **Crypto Updater**: ~$5/month (minimal CPU/memory)
- **Total**: ~$20/month

### To Reduce Costs

1. **Reduce update frequency**: Edit `coingecko_complete_service.py`
   - Change top 100 update from 3min to 5min
   - Change full sync from 30min to 60min

2. **Use Redis spot instances**: Not recommended (data loss risk)

3. **Reduce Redis memory**: Change from 512MB to 256MB if only caching top 100 cryptos

## Production Checklist

Before going live, ensure:

- [ ] Redis is deployed and has persistent volume
- [ ] Crypto updater is deployed and logs show successful updates
- [ ] Backend environment variables point to `redis-service`
- [ ] API endpoint `/api/v1/crypto/BTC/price` returns valid data
- [ ] Frontend displays crypto prices correctly
- [ ] Set up CloudWatch alarms for pod failures
- [ ] Configure log aggregation (CloudWatch Logs)
- [ ] Test failover (delete pod and verify it recreates)

## Next Steps

After deployment:

1. **Add monitoring**: Set up Prometheus metrics for API call success rate
2. **Add alerting**: Alert if CoinGecko API fails for >10 minutes
3. **Add backup**: Periodic Redis snapshots to S3
4. **Add more cryptos**: Edit script to fetch top 500 instead of top 100
