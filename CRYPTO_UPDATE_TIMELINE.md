# Crypto Price Update Timeline

## 📊 Visual Timeline (30-Minute Cycle)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     CONTINUOUS UPDATE CYCLE (Repeats Forever)               │
└─────────────────────────────────────────────────────────────────────────────┘

00:00  ████████████  FULL SYNC (All 1,250+ cryptos)
       │
       │  [Fetches pages 1-5 from CoinGecko]
       │  [Stores: BTC, ETH, BNB, XRP, SOL, ADA... + 1,244 more]
       │
00:03  ▓▓▓▓  Top 100 Update (BTC, ETH, BNB, XRP, SOL, ADA, DOGE...)
       │
       │  [3 minute wait]
       │
00:06  ▓▓▓▓  Top 100 Update
       │
       │  [3 minute wait]
       │
00:09  ▓▓▓▓  Top 100 Update
       │
       │  [3 minute wait]
       │
00:12  ▓▓▓▓  Top 100 Update
       │
       │  [3 minute wait]
       │
00:15  ▓▓▓▓  Top 100 Update
       │
       │  [3 minute wait]
       │
00:18  ▓▓▓▓  Top 100 Update
       │
       │  [3 minute wait]
       │
00:21  ▓▓▓▓  Top 100 Update
       │
       │  [3 minute wait]
       │
00:24  ▓▓▓▓  Top 100 Update
       │
       │  [3 minute wait]
       │
00:27  ▓▓▓▓  Top 100 Update
       │
       │  [3 minute wait]
       │
00:30  ████████████  FULL SYNC (All 1,250+ cryptos) ← Cycle repeats
       │
       │  ... continues forever ...
       │
```

## 🔄 What Happens Each Update

### Full Sync (00:00, 00:30, 01:00...)
```
1. Fetch page 1 (250 cryptos) → Store in Redis
2. Fetch page 2 (250 cryptos) → Store in Redis
3. Fetch page 3 (250 cryptos) → Store in Redis
4. Fetch page 4 (250 cryptos) → Store in Redis
5. Fetch page 5 (250 cryptos) → Store in Redis
6. [Hit rate limit, wait 60s]

Total time: ~90 seconds
Total cryptos: 1,250
```

### Top 100 Update (00:03, 00:06, 00:09...)
```
1. Fetch page 1 (100 cryptos) → Store in Redis

Total time: ~5 seconds
Total cryptos: 100
```

## 📈 Price Freshness Guarantee

| Crypto Rank | Update Frequency | Max Staleness | Redis TTL |
|-------------|-----------------|---------------|-----------|
| **BTC (#1)** | Every 3 min | 3 min old | 60s |
| **ETH (#2)** | Every 3 min | 3 min old | 60s |
| **Top 10** | Every 3 min | 3 min old | 60s |
| **Top 50** | Every 3 min | 3 min old | 120s |
| **Top 100** | Every 3 min | 3 min old | 180s |
| **Rank 101-500** | Every 30 min | 30 min old | 300s |
| **Rank 500+** | Every 30 min | 30 min old | 600s |

## 🕐 Hour-by-Hour Example (Bitcoin)

```
00:00:00  BTC price fetched: $122,072  (FULL SYNC)
          ↓ Stored in Redis with 60s TTL

00:01:00  BTC expires from Redis (60s TTL)
          ↓ But API can still serve from backend cache

00:03:00  BTC price fetched: $122,185  (TOP 100 UPDATE)
          ↓ New price stored in Redis

00:04:00  BTC expires from Redis

00:06:00  BTC price fetched: $122,290  (TOP 100 UPDATE)
          ↓ New price stored

00:09:00  BTC price fetched: $122,150  (TOP 100 UPDATE)
          ↓ New price stored

... repeats every 3 minutes, 24/7 ...
```

## 🌍 Global Trading Hours Coverage

Crypto markets are **24/7/365**, so the updater runs continuously:

```
Monday    00:00 ████████████████████████ 24:00
Tuesday   00:00 ████████████████████████ 24:00
Wednesday 00:00 ████████████████████████ 24:00
Thursday  00:00 ████████████████████████ 24:00
Friday    00:00 ████████████████████████ 24:00
Saturday  00:00 ████████████████████████ 24:00
Sunday    00:00 ████████████████████████ 24:00

Legend: ████ = Continuous updates (no downtime)
```

**Stock market** updaters would only run during market hours:
```
Monday    00:00 ░░░░░░░████████░░░░░░░░░ 24:00
                      ↑        ↑
                    9:30am   4:00pm EST
```

## 🔄 Deployment vs CronJob Comparison

### ❌ CronJob Approach (NOT used)
```
00:00  Run script → Update prices → Exit
00:03  [nothing]
00:06  [nothing]
00:09  [nothing]
00:12  [nothing]
00:15  Run script → Update prices → Exit
00:18  [nothing]
...

Problems:
- Pod starts/stops repeatedly (wasteful)
- Cold start delay every run (~30s)
- No continuous monitoring
- Complex error handling
```

### ✅ Deployment Approach (USED)
```
00:00  ████ Running ████ Running ████ Running ████
       │                                          │
       └─── Single pod, never stops, while True ──┘

Benefits:
- Always running (no cold starts)
- Immediate updates (no wait for next cron)
- Built-in error handling (try/except + retry)
- Easy monitoring (check pod status)
- Auto-restart on failure (Kubernetes)
```

## 🚀 Quick Verification Commands

```bash
# 1. Check if updater is running continuously
kubectl get pods -n investorcenter | grep crypto-price-updater
# Should show: Running for X hours/days

# 2. Verify updates are happening (check logs for timestamps)
kubectl logs -n investorcenter -l app=crypto-price-updater --tail=10
# Should show updates every 3 minutes

# 3. Check how fresh Bitcoin price is
kubectl exec -n investorcenter deployment/redis -- \
  redis-cli get crypto:quote:BTC | jq '.fetched_at'
# Should be within last 3 minutes

# 4. See full update cycle in real-time
kubectl logs -n investorcenter -l app=crypto-price-updater -f
# Watch as it updates every 3 minutes
```

## 📊 API Rate Limit Budget (CoinGecko Free Tier)

**Limit**: 10 API calls per minute

### Current Usage:
```
Full Sync:
- 5 pages × 1 call = 5 calls
- Spread over 90 seconds
- Rate: ~3.3 calls/min (within limit ✅)

Top 100 Update:
- 1 page × 1 call = 1 call
- Every 3 minutes
- Rate: 0.33 calls/min (well within limit ✅)

Total Average: ~4 calls/min (40% of limit)
```

**Headroom**: 6 calls/min unused → Can add more features!

## 🎯 Summary: How to Ensure Continuous Updates

### ✅ It's Already Configured!

1. **Deployment (not CronJob)** → Runs 24/7 with `while True` loop
2. **Auto-restart** → Kubernetes restarts if it crashes
3. **Error handling** → Script retries on failure
4. **Health checks** → Liveness/readiness probes
5. **Persistent Redis** → Data survives pod restarts

### 🚀 To Deploy:

```bash
# Just run this once:
./scripts/deploy-crypto-to-aws.sh

# Then forget about it - it runs forever! ✨
```

### 📈 To Monitor:

```bash
# Check it's running:
kubectl get pods -n investorcenter | grep crypto

# View logs:
kubectl logs -n investorcenter -l app=crypto-price-updater -f

# Check data freshness:
kubectl exec -n investorcenter deployment/redis -- \
  redis-cli get crypto:quote:BTC | jq '.fetched_at'
```

## 🔧 Tuning Options

### Want More Frequent Updates?

Edit `scripts/coingecko_complete_service.py`:
```python
# Line 407: Change from 180 to 120
await asyncio.sleep(120)  # 2 minutes instead of 3
```

Then rebuild and redeploy:
```bash
./scripts/deploy-crypto-to-aws.sh v2
```

### Want Less Frequent Updates?

```python
# Line 407: Change from 180 to 300
await asyncio.sleep(300)  # 5 minutes instead of 3
```

**Recommendation**: Keep current (3 min) - it's optimal for active trading! ✅
