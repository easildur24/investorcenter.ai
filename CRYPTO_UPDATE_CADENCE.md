# Crypto Price Update Cadence & Reliability

## 🔄 How Continuous Updates Work

### Architecture: Deployment (Not CronJob)

The crypto updater runs as a **Kubernetes Deployment**, which means:

✅ **Runs continuously 24/7** (not on a schedule)
✅ **Automatically restarts** if it crashes
✅ **Kubernetes ensures it's always running** (replicas: 1)
✅ **Built-in infinite loop** that never exits

### Update Schedule (Built into the Script)

The `coingecko_complete_service.py` script has a **built-in update loop**:

```python
# Main service loop (lines 387-411)
while True:  # ← Runs forever!
    try:
        # 1. Full market data update
        await self.fetch_all_market_data(session)

        # 2. Then do targeted updates for 30 minutes
        targeted_update_end = time.time() + 1800  # 30 min

        while time.time() < targeted_update_end:
            # Update top 100 coins
            top_coins = await self.fetch_market_data(session, page=1, per_page=100)
            self.store_coin_data(top_coins)

            # Wait 3 minutes before next update
            await asyncio.sleep(180)  # 3 min = 180 seconds

    except Exception as e:
        logger.error(f"Error: {e}")
        await asyncio.sleep(60)  # Wait 1 min before retry
```

## 📊 Update Frequency

### Top 100 Cryptocurrencies
- **Frequency**: Every **3 minutes** (180 seconds)
- **Example**: BTC, ETH, BNB, XRP, SOL, ADA, DOGE, etc.
- **Why**: Most actively traded, users check frequently

### All Cryptocurrencies (1,250+)
- **Frequency**: Every **30 minutes** (full sync)
- **Coverage**: All coins from CoinGecko
- **Why**: Less frequently traded, saves API rate limits

### Update Pattern (Repeating Cycle)

```
Time 00:00 → Full sync of ALL cryptos (1,250+)
Time 00:03 → Update top 100
Time 00:06 → Update top 100
Time 00:09 → Update top 100
Time 00:12 → Update top 100
Time 00:15 → Update top 100
Time 00:18 → Update top 100
Time 00:21 → Update top 100
Time 00:24 → Update top 100
Time 00:27 → Update top 100
Time 00:30 → Full sync of ALL cryptos (1,250+) again
Time 00:33 → Update top 100
... (repeats forever)
```

## ⏱️ Redis Cache TTL (Time-To-Live)

Prices are cached in Redis with different TTLs based on market cap rank:

| Rank | TTL | Reason |
|------|-----|--------|
| Top 10 | 60 seconds | Most volatile, high volume |
| Top 50 | 120 seconds | Still very active |
| Top 100 | 180 seconds | Matches update frequency |
| Top 500 | 300 seconds | Less active |
| Others | 600 seconds | Low trading volume |

**Example**: Bitcoin (rank #1) stays in cache for 60 seconds, then Redis auto-deletes it. But the script updates it every 3 minutes, so there's always a fresh copy.

## 🔒 Reliability & Failover

### 1. Kubernetes Auto-Restart

If the pod crashes or stops:

```yaml
# From k8s/crypto-price-updater-deployment.yaml
spec:
  replicas: 1  # Always keep 1 pod running
  template:
    spec:
      restartPolicy: Always  # Auto-restart on failure
```

**What happens**:
1. Pod crashes → Kubernetes detects it within seconds
2. Kubernetes starts a new pod → Takes ~30 seconds
3. New pod starts fetching prices → Resumes within 1 minute

**Downtime**: Maximum ~1 minute

### 2. Script-Level Error Handling

The script handles errors gracefully:

```python
try:
    # Fetch and update prices
except Exception as e:
    logger.error(f"Error: {e}")
    await asyncio.sleep(60)  # Wait 1 minute
    # Then retry automatically (while True loop)
```

**Common errors handled**:
- CoinGecko API rate limit → Waits 60 seconds, then retries
- Network timeout → Logs error, retries in 1 minute
- Redis connection lost → Retries automatically

### 3. Liveness Probe

Kubernetes checks if the pod is healthy:

```yaml
livenessProbe:
  exec:
    command:
    - python3
    - -c
    - "import redis; r = redis.Redis(host='redis-service'); r.ping()"
  initialDelaySeconds: 60
  periodSeconds: 300  # Check every 5 minutes
```

**What happens**:
- Every 5 minutes, Kubernetes runs: `redis.ping()`
- If it fails 3 times → Pod is killed and restarted
- Ensures the pod isn't "stuck" or frozen

### 4. Readiness Probe

Kubernetes checks if data is being populated:

```yaml
readinessProbe:
  exec:
    command:
    - python3
    - -c
    - "import redis; r = redis.Redis(host='redis-service'); r.exists('crypto:symbols:ranked')"
  initialDelaySeconds: 30
  periodSeconds: 60
```

**What happens**:
- Checks if Redis has crypto data
- If not → Pod marked "not ready"
- Prevents backend from using empty Redis

## 🚨 What Could Go Wrong?

### Scenario 1: CoinGecko API Down

**Problem**: CoinGecko API returns errors
**Impact**: No new price updates
**Mitigation**:
- Old prices stay in Redis until TTL expires
- Script retries every 60 seconds automatically
- Logs show errors for monitoring

**Recovery**: Automatic when CoinGecko comes back online

### Scenario 2: Redis Pod Crashes

**Problem**: Redis pod dies
**Impact**: All cached prices lost
**Mitigation**:
- Redis has persistent volume (data survives pod restart)
- Redis restarts automatically (Kubernetes)
- Crypto updater refills Redis within 3 minutes

**Recovery**: ~3 minutes (time to restart Redis + fetch prices)

### Scenario 3: Network Issues

**Problem**: Pod can't reach CoinGecko or Redis
**Impact**: Updates stop
**Mitigation**:
- Script logs errors
- Kubernetes restarts pod if livenessProbe fails
- Retries automatically every 60 seconds

**Recovery**: Automatic when network recovers

### Scenario 4: Out of Memory

**Problem**: Pod exceeds memory limit (512MB)
**Impact**: Kubernetes kills the pod
**Mitigation**:
- Kubernetes restarts pod immediately
- Memory limit can be increased if needed

**Recovery**: ~30 seconds (pod restart time)

## 📈 Monitoring & Alerts

### Check if Updates Are Running

```bash
# 1. Check pod status
kubectl get pods -n investorcenter | grep crypto-price-updater

# Expected output:
# crypto-price-updater-xxx   1/1   Running   0   2d

# 2. Check recent logs (should see updates every 3 min)
kubectl logs -n investorcenter -l app=crypto-price-updater --tail=20

# Expected output:
# 2025-10-03 12:00:00 - INFO - 📈 Updating top 100 coins...
# 2025-10-03 12:00:01 - INFO - 📊 Fetched page 1 with 100 coins
# 2025-10-03 12:00:01 - INFO - 💾 Stored 100 coins in Redis

# 3. Check Redis data freshness
kubectl exec -n investorcenter deployment/redis -- \
  redis-cli get crypto:stats:last_update

# Expected output: timestamp within last 30 minutes

# 4. Check specific crypto price
kubectl exec -n investorcenter deployment/redis -- \
  redis-cli get crypto:quote:BTC | jq '.fetched_at'

# Expected output: timestamp within last 3 minutes
```

### Set Up Alerts (Recommended)

**CloudWatch Alarm 1**: Pod Not Running
```bash
# Alert if crypto-price-updater pod count < 1
Metric: kube_deployment_status_replicas_available
Threshold: < 1
Action: SNS notification
```

**CloudWatch Alarm 2**: No Updates in 10 Minutes
```bash
# Check Redis last update timestamp
# Alert if no updates in 10 minutes
Custom metric from logs
Action: SNS notification
```

**CloudWatch Alarm 3**: High Error Rate
```bash
# Alert if ERROR log count > 10 in 5 minutes
Metric: CloudWatch Logs filter pattern "ERROR"
Threshold: > 10
Action: SNS notification
```

## 🔧 Tuning Update Frequency

### To Update MORE Frequently (Higher API Usage)

Edit `scripts/coingecko_complete_service.py`:

```python
# Current: Top 100 every 3 minutes
await asyncio.sleep(180)  # Change to 120 for 2 minutes

# Current: Full sync every 30 minutes
targeted_update_end = time.time() + 1800  # Change to 900 for 15 min
```

**⚠️ Warning**: CoinGecko free tier = 10 calls/minute
- Current setup uses ~2 calls/3min = safe
- 2-minute updates = ~3 calls/2min = still safe
- 1-minute updates = rate limit errors!

### To Update LESS Frequently (Save API Calls)

```python
# Every 5 minutes for top 100
await asyncio.sleep(300)

# Full sync every 60 minutes
targeted_update_end = time.time() + 3600
```

**Trade-off**: Less API usage, but stale prices for users

## 📋 Deployment Checklist

Before deploying to production:

- [ ] Run `./scripts/deploy-crypto-to-aws.sh`
- [ ] Verify pod is running: `kubectl get pods -n investorcenter`
- [ ] Check logs show updates: `kubectl logs -n investorcenter -l app=crypto-price-updater -f`
- [ ] Verify Redis has data: `kubectl exec -n investorcenter deployment/redis -- redis-cli keys "crypto:*"`
- [ ] Test API endpoint: `curl http://<backend-ip>/api/v1/crypto/BTC/price`
- [ ] Set up CloudWatch alarms (see above)
- [ ] Document on-call procedures for failures

## 🎯 Recommended Cadence

Based on typical cryptocurrency trading patterns:

| Use Case | Top 100 Frequency | Full Sync Frequency |
|----------|------------------|---------------------|
| **High-frequency trading** | 1-2 minutes | 15 minutes |
| **Active trading (current)** | 3 minutes | 30 minutes |
| **Casual investors** | 5 minutes | 60 minutes |
| **Portfolio tracking only** | 15 minutes | 120 minutes |

**Current setting**: ✅ **Active trading** (good balance)

## 💡 Pro Tips

1. **Monitor API rate limits**:
   ```bash
   # Check CoinGecko rate limit headers in logs
   kubectl logs -n investorcenter -l app=crypto-price-updater | grep "429"
   ```

2. **Scale Redis if needed**:
   ```bash
   # If storing more cryptos, increase Redis memory
   # Edit k8s/redis-deployment.yaml:
   # --maxmemory 1gb  (from 512mb)
   ```

3. **Add backup data source**:
   - Consider adding Binance or Coinbase APIs as fallback
   - Update script to try multiple sources if CoinGecko fails

4. **Log aggregation**:
   - Ship logs to CloudWatch Logs or DataDog
   - Makes debugging failures easier

## Summary

✅ **How it stays running**: Kubernetes Deployment with auto-restart
✅ **Update frequency**: Top 100 every 3 min, all cryptos every 30 min
✅ **Reliability**: Auto-restart, error handling, health checks
✅ **Monitoring**: Check logs, Redis data, set up CloudWatch alarms
✅ **Recommended cadence**: Current (3 min / 30 min) is optimal

**No CronJob needed** - the script runs continuously with a `while True` loop! 🎉
