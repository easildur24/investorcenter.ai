# Deploy Crypto Price Updater to Production

## 🚨 Docker Issue on Local Machine

Docker daemon is not running on your local machine. Here are 2 ways to deploy:

---

## ✅ Option 1: Deploy via GitHub Actions (Recommended)

### Step 1: Commit and Push Files

```bash
# Add all crypto updater files
git add scripts/coingecko_complete_service.py
git add Dockerfile.crypto-updater
git add k8s/crypto-price-updater-deployment.yaml
git add k8s/redis-deployment.yaml
git add k8s/backend-deployment.yaml
git add .github/workflows/deploy-crypto-updater.yml

# Commit
git commit -m "feat: add crypto price updater deployment"

# Push to trigger deployment
git push origin main
```

### Step 2: Monitor Deployment

1. Go to GitHub Actions: https://github.com/easildur24/investorcenter.ai/actions
2. Watch "Deploy Crypto Price Updater" workflow
3. It will automatically:
   - ✅ Build Docker image
   - ✅ Push to ECR
   - ✅ Deploy Redis
   - ✅ Deploy Crypto Updater
   - ✅ Update Backend
   - ✅ Verify deployment

### Step 3: Verify It's Working

After GitHub Actions completes (~5 minutes):

```bash
# Check if pod is running
kubectl get pods -n investorcenter | grep crypto-price-updater

# View logs
kubectl logs -n investorcenter -l app=crypto-price-updater -f

# Check Redis has data
kubectl exec -n investorcenter deployment/redis -- redis-cli keys "crypto:quote:*" | wc -l

# Test API
kubectl port-forward -n investorcenter svc/backend-service 8080:8080 &
curl http://localhost:8080/api/v1/crypto/BTC/price | jq
```

---

## Option 2: Manual Deploy (Requires Docker)

If you want to deploy manually, you need to start Docker first:

### Step 1: Start Docker/Rancher Desktop

1. **Open Rancher Desktop manually** from Applications
2. **Wait 2-3 minutes** for Docker daemon to fully start
3. **Verify**:
   ```bash
   docker ps
   ```

### Step 2: Run Deployment Script

Once Docker is running:

```bash
./scripts/deploy-crypto-to-aws.sh latest
```

This will:
1. Create ECR repository
2. Build Docker image
3. Push to ECR
4. Deploy Redis
5. Deploy Crypto Updater
6. Update Backend
7. Verify everything works

---

## 🎯 Quick Deploy Commands (Recommended: GitHub Actions)

```bash
# 1. Commit crypto updater files
git add -A
git commit -m "feat: deploy crypto price updater to production"

# 2. Push to main (triggers automatic deployment)
git push origin main

# 3. Wait 5 minutes, then verify
kubectl get pods -n investorcenter | grep crypto-price-updater
kubectl logs -n investorcenter -l app=crypto-price-updater --tail=20

# 4. Test API
kubectl port-forward -n investorcenter svc/backend-service 8080:8080 &
curl http://localhost:8080/api/v1/crypto/BTC/price | jq '.price'
```

---

## 📊 What Gets Deployed

1. **Redis** (if not already deployed)
   - 512MB memory
   - Persistent volume
   - LRU eviction policy

2. **Crypto Price Updater**
   - Continuous pod (runs 24/7)
   - Updates top 100 cryptos every 3 minutes
   - Full sync all cryptos every 30 minutes

3. **Backend Update**
   - Adds `REDIS_HOST` and `REDIS_PORT` env vars
   - Restarts to connect to Redis

---

## ✅ Expected Timeline

- **00:00** - Push to GitHub
- **00:01** - GitHub Actions starts
- **00:03** - Docker image built and pushed to ECR
- **00:04** - Redis deployed (or skipped if exists)
- **00:05** - Crypto updater deployed
- **00:06** - Backend updated
- **00:07** - First crypto prices fetched and cached
- **00:08** - ✅ **Production ready!**

---

## 🔍 Troubleshooting

### GitHub Actions fails?

Check:
1. AWS credentials are configured in GitHub Secrets
2. `AWS_ROLE_ARN` secret exists
3. Role has ECR and EKS permissions

### Pod not starting?

```bash
# Check pod status
kubectl describe pod -n investorcenter -l app=crypto-price-updater

# Check logs
kubectl logs -n investorcenter -l app=crypto-price-updater
```

### No prices in Redis?

```bash
# Check if updater is running
kubectl logs -n investorcenter -l app=crypto-price-updater --tail=50

# Check for CoinGecko API errors
kubectl logs -n investorcenter -l app=crypto-price-updater | grep ERROR
```

---

## 📚 Documentation

- [Deployment Guide](CRYPTO_DEPLOYMENT_GUIDE.md)
- [Update Cadence](CRYPTO_UPDATE_CADENCE.md)
- [Timeline](CRYPTO_UPDATE_TIMELINE.md)
- [Quick Reference](CRYPTO_QUICK_REFERENCE.md)
