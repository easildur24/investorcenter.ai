# 🗄️ InvestorCenter Database Setup

Complete setup guide for both local development and production Kubernetes databases.

## 🎯 **Current Status**

✅ **Local Development Database**: PostgreSQL 15 via Homebrew
✅ **Production Database**: PostgreSQL 15 in Kubernetes
✅ **Ticker Data**: 4,643 US stocks imported to both
✅ **Environment Switcher**: Seamless local ↔ production switching

## 🚀 **Quick Start**

### **Verify Your Setup**
```bash
./scripts/verify-setup.sh
```

### **Switch Between Environments**
```bash
# Local development
source scripts/switch-env.sh local
python scripts/ticker_import_to_db.py --dry-run

# Production (via port-forward)
source scripts/switch-env.sh prod
python scripts/ticker_import_to_db.py --dry-run
```

## 📊 **Database Details**

### **Local Development**
- **Host**: localhost:5432
- **Database**: investorcenter_db
- **User**: investorcenter / investorcenter123
- **SSL**: disabled
- **Storage**: Local filesystem
- **Use for**: Development, testing, ticker imports

### **Production (Kubernetes)**
- **Host**: localhost:5433 (via port-forward)
- **Database**: investorcenter_db
- **User**: investorcenter / prod_investorcenter_456
- **SSL**: disabled (local port-forward)
- **Storage**: Kubernetes PVC (10Gi)
- **Use for**: Production deployment, staging tests

## 🔄 **Ticker Management**

### **Import Fresh Data**
```bash
# Local
source scripts/switch-env.sh local
python scripts/ticker_import_to_db.py

# Production
source scripts/switch-env.sh prod
python scripts/ticker_import_to_db.py
```

### **Periodic Updates (Recommended)**
```bash
# Weekly updates - only adds new listings
python scripts/update_tickers_cron.py

# Add to crontab:
0 2 * * 0 cd /path/to/investorcenter.ai && source scripts/switch-env.sh local && python scripts/update_tickers_cron.py >> logs/ticker_updates.log 2>&1
```

## 🏗️ **Schema Overview**

Both databases contain identical schemas:

### **Core Tables**
- `stocks` - **4,643 US companies** (symbol, name, exchange, country, currency)
- `stock_prices` - Historical price data (OHLCV)
- `fundamentals` - Financial metrics (PE, ROE, revenue, etc.)
- `earnings` - Quarterly earnings with estimates
- `analyst_ratings` - Analyst recommendations
- `news_articles` - News with sentiment analysis
- `dividends` - Dividend payment history
- `insider_trading` - Insider buy/sell activity
- `technical_indicators` - RSI, MACD, moving averages

### **Data Quality**
✅ **Only common stocks** - No warrants, preferred, notes
✅ **Clean names** - Standardized company names
✅ **Exchange mapping** - Full names (Nasdaq, NYSE, etc.)
✅ **US focus** - Country=US, Currency=USD
✅ **Incremental safe** - ON CONFLICT DO NOTHING

## 🔧 **Environment Variables**

### **Local Development**
```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=investorcenter
DB_PASSWORD=investorcenter123
DB_NAME=investorcenter_db
DB_SSLMODE=disable
```

### **Production (K8s)**
```bash
DB_HOST=localhost
DB_PORT=5433  # Port-forwarded
DB_USER=investorcenter
DB_PASSWORD=prod_investorcenter_456
DB_NAME=investorcenter_db
DB_SSLMODE=disable  # For port-forward access
```

## 🚀 **Deployment Guide**

### **Local Development Workflow**
1. **Start local PostgreSQL**: `brew services start postgresql@15`
2. **Switch environment**: `source scripts/switch-env.sh local`
3. **Start backend**: `cd backend && ./investorcenter-api`
4. **Develop**: API available at `http://localhost:8080`

### **Production Deployment**
1. **Deploy to K8s**: `kubectl apply -f k8s/`
2. **Access via port-forward**: `source scripts/switch-env.sh prod`
3. **Import data**: `python scripts/ticker_import_to_db.py`
4. **Monitor**: `kubectl get pods -n investorcenter`

## 📈 **API Endpoints**

With database connected, these endpoints now serve real data:

```bash
# Health check (shows database status)
GET /health

# Stock management
GET /api/v1/tickers/                    # List all stocks (paginated)
GET /api/v1/tickers/?search=APPLE       # Search stocks
GET /api/v1/tickers/?limit=100&offset=0 # Pagination
POST /api/v1/tickers/import             # Upload CSV import
GET /api/v1/tickers/AAPL                # Ticker details (still mock data)
```

## 🔄 **Data Sync**

### **Export/Import Between Environments**
```bash
# Export from local
export PATH="/opt/homebrew/opt/postgresql@15/bin:$PATH"
pg_dump investorcenter_db --data-only --table=stocks > stocks_backup.sql

# Import to production
source scripts/switch-env.sh prod
export PGPASSWORD="prod_investorcenter_456"
psql -h localhost -p 5433 -U investorcenter -d investorcenter_db -f stocks_backup.sql
```

### **Keep Environments in Sync**
```bash
# Update both environments weekly
source scripts/switch-env.sh local && python scripts/update_tickers_cron.py
source scripts/switch-env.sh prod && python scripts/update_tickers_cron.py
```

## 🎯 **Next Development Steps**

1. **Populate additional data**: Sector, industry, market cap
2. **Import price data**: Historical stock prices
3. **Add fundamentals**: Financial metrics and ratios
4. **News integration**: Real-time news feeds
5. **Real-time prices**: Live market data APIs

Your InvestorCenter platform now has a **solid foundation** with complete US stock coverage! 🚀
