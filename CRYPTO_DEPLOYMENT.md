# Cryptocurrency Implementation & Deployment Guide

## Overview
This guide documents the cryptocurrency real-time pricing implementation for InvestorCenter.

## Architecture

### Components
1. **Python Service** (`scripts/crypto_complete_service.py`)
   - Downloads real-time prices for 14,000+ cryptocurrencies from CoinLore API
   - Stores data in Redis with tiered update intervals
   - Currently tracks 3,783 active cryptocurrencies

2. **Redis Cache**
   - Stores real-time crypto prices
   - Keys: `crypto:quote:{SYMBOL}`
   - TTL managed by service based on tier

3. **PostgreSQL Sync** (`scripts/crypto_postgres_sync.py`)
   - Syncs top 1,000 cryptos to PostgreSQL
   - Updates ticker metadata
   - Stores historical snapshots

4. **Backend API** (`backend/handlers/crypto_realtime_handlers.go`)
   - `/api/v1/crypto/{symbol}/price` - Get single crypto price
   - `/api/v1/crypto/prices` - Get all crypto prices
   - `/api/v1/crypto/stream` - SSE endpoint for real-time updates

5. **Frontend**
   - Crypto-specific UI components
   - Real-time price updates via hooks
   - 24/7 market status (never shows "Market Closed")

## Deployment Steps

### 1. Prerequisites
- Redis server running
- PostgreSQL database
- Python 3.8+
- Go 1.19+
- Node.js 18+

### 2. Environment Variables

#### Backend (Go)
```bash
export REDIS_ADDR=localhost:6379        # Redis address
export REDIS_PASSWORD=                  # Redis password (optional)
export BACKEND_URL=http://localhost:8080  # For frontend SSR
```

#### Python Services
```bash
export DB_HOST=localhost
export DB_PORT=5433
export DB_USER=investorcenter
export DB_PASSWORD=password123
export DB_NAME=investorcenter_db
```

### 3. Start Services

#### A. Start Redis
```bash
redis-server
```

#### B. Start Crypto Price Service
```bash
cd scripts
python3 crypto_complete_service.py
```
This service should run continuously to maintain real-time prices.

#### C. Start PostgreSQL Sync (via cron)
```bash
# Add to crontab for hourly sync
0 * * * * /path/to/scripts/crypto_sync_cron.sh
```

#### D. Start Backend API
```bash
cd backend
go run main.go
```

#### E. Start Frontend
```bash
npm run build
npm start
```

## Kubernetes Deployment

### 1. Create ConfigMap for Environment
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: crypto-config
  namespace: investorcenter
data:
  REDIS_ADDR: "redis-service:6379"
  DB_HOST: "postgres-simple-service"
  DB_PORT: "5432"
```

### 2. Deploy Crypto Service
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: crypto-service
  namespace: investorcenter
spec:
  replicas: 1
  selector:
    matchLabels:
      app: crypto-service
  template:
    metadata:
      labels:
        app: crypto-service
    spec:
      containers:
      - name: crypto-service
        image: investorcenter/crypto-service:latest
        envFrom:
        - configMapRef:
            name: crypto-config
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "500m"
```

### 3. Build Docker Images
```dockerfile
# Dockerfile.crypto-service
FROM python:3.9-slim
WORKDIR /app
COPY scripts/requirements.txt .
RUN pip install -r requirements.txt
COPY scripts/crypto_complete_service.py .
CMD ["python", "crypto_complete_service.py"]
```

Build and push:
```bash
docker build -f Dockerfile.crypto-service -t investorcenter/crypto-service:latest .
docker push investorcenter/crypto-service:latest
```

## Monitoring

### Health Checks
1. **Redis Connection**
   ```bash
   redis-cli ping
   redis-cli get crypto:quote:BTC
   ```

2. **Service Status**
   ```bash
   curl http://localhost:8080/api/v1/crypto/BTC/price
   ```

3. **Database Sync**
   ```sql
   SELECT COUNT(*) FROM tickers WHERE asset_type = 'crypto';
   SELECT * FROM tickers WHERE symbol = 'BTC';
   ```

### Metrics to Monitor
- Active cryptocurrency count (~3,783)
- Redis memory usage
- API response times
- Update intervals per tier

## Troubleshooting

### Issue: No crypto prices
1. Check Redis connection
2. Verify crypto_complete_service.py is running
3. Check logs: `tail -f scripts/logs/crypto_service.log`

### Issue: Wrong prices in frontend
1. Clear browser cache
2. Check Redis data: `redis-cli get crypto:quote:BTC`
3. Verify API endpoint: `curl http://localhost:8080/api/v1/crypto/BTC/price`

### Issue: High memory usage
1. Monitor Redis: `redis-cli info memory`
2. Reduce tracked cryptos in service
3. Adjust TTL values

## Update Tiers
- **Tier 1** (Top 100): Updates every 5 seconds
- **Tier 2** (101-500): Updates every 10 seconds
- **Tier 3** (501-1000): Updates every 20 seconds
- **Tier 4** (1001-5000): Updates every 30 seconds
- **Tier 5** (5001+): Updates every 60 seconds

## Security Considerations
1. Use Redis password in production
2. Restrict Redis network access
3. Use HTTPS for all API endpoints
4. Implement rate limiting on API
5. Monitor for unusual request patterns

## Performance Optimization
1. Redis persistence: Use AOF for durability
2. Connection pooling in Go handlers
3. Batch updates in PostgreSQL sync
4. CDN for frontend assets
5. Consider Redis Cluster for scaling

## Rollback Plan
If issues occur:
1. Stop crypto_complete_service.py
2. Revert to previous backend deployment
3. Clear Redis crypto keys: `redis-cli --scan --pattern crypto:* | xargs redis-cli del`
4. Restore from database backup if needed

## Contact
For issues or questions about the crypto implementation, contact the development team.