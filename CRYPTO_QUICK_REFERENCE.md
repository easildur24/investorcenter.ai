# Crypto Price Updater - Quick Reference

## 🚀 One-Command Deployment

```bash
./scripts/deploy-crypto-to-aws.sh
```

That's it! This deploys everything and ensures continuous 24/7 updates.

---

## ❓ How It Ensures Continuous Updates

### 1. **Runs as Kubernetes Deployment** (not CronJob)
   - Always running, never stops
   - Built-in `while True` loop in Python script
   - Kubernetes keeps it alive 24/7

### 2. **Auto-Restart on Failure**
   - Pod crashes → Kubernetes restarts within seconds
   - Network error → Script retries automatically
   - API rate limit → Waits 60s and retries

### 3. **Update Cadence (Built-in)**

| What | Frequency | Why |
|------|-----------|-----|
| **Top 100 cryptos** (BTC, ETH, etc.) | Every **3 minutes** | Most traded, users check often |
| **All 1,250+ cryptos** | Every **30 minutes** | Full market coverage |

**Timeline**:
```
00:00 → Full sync (all cryptos)
00:03 → Top 100 update
00:06 → Top 100 update
00:09 → Top 100 update
... (every 3 min)
00:30 → Full sync (all cryptos)
... repeats forever
```

---

## 📊 Data Flow

```
CoinGecko API
    ↓ (fetch every 3 min)
Python Script (runs 24/7 in Kubernetes)
    ↓ (store JSON)
Redis Cache (60s-600s TTL)
    ↓ (read on request)
Go Backend API
    ↓ (JSON response)
Frontend (displays prices)
```

---

## ✅ Verification Commands

### Check if it's running:
```bash
kubectl get pods -n investorcenter | grep crypto-price-updater
# Expected: crypto-price-updater-xxx   1/1   Running
```

### View update logs (real-time):
```bash
kubectl logs -n investorcenter -l app=crypto-price-updater -f
# Expected: Updates every 3 minutes
```

### Check Bitcoin price freshness:
```bash
kubectl exec -n investorcenter deployment/redis -- \
  redis-cli get crypto:quote:BTC | jq '.fetched_at'
# Expected: Timestamp within last 3 minutes
```

### Test API endpoint:
```bash
# Port forward first
kubectl port-forward -n investorcenter svc/backend-service 8080:8080 &

# Then test
curl http://localhost:8080/api/v1/crypto/BTC/price | jq '{symbol, price}'
# Expected: {"symbol": "BTC", "price": 122072}
```

---

## 🔧 Maintenance

### Restart the updater:
```bash
kubectl rollout restart deployment/crypto-price-updater -n investorcenter
```

### Update with new code:
```bash
# Make changes to scripts/coingecko_complete_service.py
# Then redeploy:
./scripts/deploy-crypto-to-aws.sh v2
```

### Check logs for errors:
```bash
kubectl logs -n investorcenter -l app=crypto-price-updater | grep ERROR
```

---

## 🚨 Troubleshooting

### No updates happening?
```bash
# 1. Check pod status
kubectl get pods -n investorcenter | grep crypto-price-updater

# 2. Check logs for errors
kubectl logs -n investorcenter -l app=crypto-price-updater --tail=50

# 3. Restart if needed
kubectl rollout restart deployment/crypto-price-updater -n investorcenter
```

### Prices showing $0?
```bash
# 1. Check if Redis has data
kubectl exec -n investorcenter deployment/redis -- \
  redis-cli keys "crypto:quote:*" | wc -l
# Should show > 0

# 2. Check backend is connected to Redis
kubectl logs -n investorcenter -l app=investorcenter-backend | grep -i redis
```

### CoinGecko rate limit errors?
```bash
# Check logs for "429" errors
kubectl logs -n investorcenter -l app=crypto-price-updater | grep 429

# Script automatically retries after 60 seconds
# If persistent, reduce update frequency
```

---

## 📈 Monitoring Checklist

Daily:
- [ ] Check pod is running: `kubectl get pods -n investorcenter`
- [ ] Verify recent updates in logs (within last 3 min)
- [ ] Test API returns valid prices

Weekly:
- [ ] Check for ERROR logs
- [ ] Verify all top 100 cryptos updating
- [ ] Check Redis memory usage

Monthly:
- [ ] Review CoinGecko API usage
- [ ] Update script dependencies if needed
- [ ] Test failover (delete pod, verify restart)

---

## 💰 Cost

- **Redis**: ~$15/month
- **Crypto Updater**: ~$5/month
- **Total**: ~$20/month

---

## 📚 Full Documentation

- **Deployment Guide**: [CRYPTO_DEPLOYMENT_GUIDE.md](CRYPTO_DEPLOYMENT_GUIDE.md)
- **Update Cadence**: [CRYPTO_UPDATE_CADENCE.md](CRYPTO_UPDATE_CADENCE.md)
- **Timeline Diagram**: [CRYPTO_UPDATE_TIMELINE.md](CRYPTO_UPDATE_TIMELINE.md)

---

## 🎯 Key Takeaways

1. ✅ **Runs continuously** (Deployment, not CronJob)
2. ✅ **Updates every 3 minutes** (top 100 cryptos)
3. ✅ **Auto-restarts** (Kubernetes ensures it's always running)
4. ✅ **No manual intervention needed** (set and forget)
5. ✅ **Built-in error handling** (retries automatically)

**Deploy once, runs forever!** 🚀
