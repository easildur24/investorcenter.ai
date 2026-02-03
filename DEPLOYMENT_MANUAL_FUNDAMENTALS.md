# Manual Fundamentals Deployment Guide

## 📦 Ready for Production!

All frontend and backend changes are complete and ready to deploy.

---

## 📋 Pre-Deployment Checklist

### 1. Test Locally First
```bash
# Terminal 1: Start backend
cd backend
go run main.go

# Terminal 2: Start frontend (if separate)
npm run dev

# Test:
# 1. Go to http://localhost:3000/ticker/AAPL
# 2. Click "Input" tab
# 3. Load sample template
# 4. Save data
# 5. Verify data appears in "Key Metrics" sidebar
```

### 2. Files to Commit
```bash
# Backend
backend/migrations/022_create_manual_fundamentals.sql
backend/handlers/manual_fundamentals.go
backend/main.go (modified)

# Frontend
components/ticker/tabs/InputTab.tsx
app/ticker/[symbol]/page.tsx (modified)
components/ticker/TickerFundamentals.tsx (modified)

# Documentation (optional)
MANUAL_FUNDAMENTALS_IMPLEMENTATION.md
TICKER_PAGE_DATA_FLOW.md
DEPLOYMENT_MANUAL_FUNDAMENTALS.md
```

---

## 🚀 Deployment Steps

### Step 1: Commit Changes
```bash
# Add all manual fundamentals files
git add backend/migrations/022_create_manual_fundamentals.sql
git add backend/handlers/manual_fundamentals.go
git add backend/main.go
git add components/ticker/tabs/InputTab.tsx
git add app/ticker/[symbol]/page.tsx
git add components/ticker/TickerFundamentals.tsx

# Optional: Add documentation
git add MANUAL_FUNDAMENTALS_IMPLEMENTATION.md
git add TICKER_PAGE_DATA_FLOW.md
git add DEPLOYMENT_MANUAL_FUNDAMENTALS.md

# Commit
git commit -m "feat: Add manual fundamentals data ingestion system

- Add manual_fundamentals table for flexible JSON storage
- Create POST/GET/DELETE API endpoints for manual data
- Add Input tab to ticker pages for data entry
- Integrate manual data as highest priority in TickerFundamentals
- Add comprehensive documentation

This allows manually calculated fundamental metrics to be stored
and displayed with highest priority, giving full control over
data accuracy and definitions."

# Push to main (or your deployment branch)
git push origin main
```

### Step 2: Database Migration (Production)

**⚠️ CRITICAL: Run migration on production database BEFORE deploying code**

```bash
# SSH into production database server or use your DB admin tool
# Option A: Direct psql
psql -h your-production-db-host \
     -U your-db-user \
     -d investorcenter \
     -f backend/migrations/022_create_manual_fundamentals.sql

# Option B: If you use a migration tool
# (adjust based on your tool: golang-migrate, goose, etc.)
migrate -path backend/migrations -database "postgres://..." up

# Verify migration
psql -h your-production-db-host -U your-db-user -d investorcenter
\d manual_fundamentals
# Should show the table with ticker, data, created_at, updated_at columns
```

### Step 3: Deploy Backend

**Your backend deployment depends on your infrastructure:**

#### Option A: Kubernetes (from your k8s/ folder)
```bash
# Build new backend image
cd backend
docker build -t investorcenter-backend:manual-fundamentals .

# Tag and push to registry
docker tag investorcenter-backend:manual-fundamentals your-registry/investorcenter-backend:latest
docker push your-registry/investorcenter-backend:latest

# Update Kubernetes deployment
kubectl rollout restart deployment investorcenter-backend-deployment -n investorcenter

# Monitor rollout
kubectl rollout status deployment/investorcenter-backend-deployment -n investorcenter

# Check logs
kubectl logs -f deployment/investorcenter-backend-deployment -n investorcenter
```

#### Option B: Direct Server Deployment
```bash
# Pull latest code on server
ssh your-server
cd /path/to/investorcenter.ai
git pull origin main

# Rebuild backend
cd backend
go build -o investorcenter-api

# Restart service
sudo systemctl restart investorcenter-backend
# OR
pm2 restart investorcenter-backend

# Check logs
sudo journalctl -u investorcenter-backend -f
# OR
pm2 logs investorcenter-backend
```

### Step 4: Deploy Frontend

**Your frontend deployment:**

#### Option A: Next.js on Vercel/Netlify
```bash
# If auto-deploy is enabled:
# Push to main triggers automatic deployment

# Manual deploy:
vercel --prod
# OR
netlify deploy --prod
```

#### Option B: Kubernetes/Docker
```bash
# Build frontend
npm run build

# Build Docker image
docker build -t investorcenter-frontend:manual-fundamentals -f Dockerfile .

# Push and deploy
docker tag investorcenter-frontend:manual-fundamentals your-registry/investorcenter-frontend:latest
docker push your-registry/investorcenter-frontend:latest

kubectl rollout restart deployment investorcenter-frontend-deployment -n investorcenter
```

#### Option C: Direct Server
```bash
ssh your-server
cd /path/to/investorcenter.ai
git pull origin main

npm install
npm run build

# Restart Next.js (if using PM2)
pm2 restart investorcenter-frontend
```

### Step 5: Verify Deployment

```bash
# Check backend API
curl https://investorcenter.ai/api/v1/tickers/AAPL/manual-fundamentals
# Should return 404 or empty data (expected before data is added)

# Check frontend
# 1. Open https://investorcenter.ai/ticker/AAPL
# 2. Look for "Input" tab (should be after "Ownership")
# 3. Click Input tab
# 4. Should see JSON editor with buttons

# Test full flow
# 1. Click "Load Sample Template"
# 2. Click "Save Fundamental Data"
# 3. Check "Key Metrics" sidebar - should show your data
# 4. Refresh page - data should persist
```

---

## 🔧 Rollback Plan (If Needed)

### If something goes wrong:

```bash
# 1. Revert code
git revert HEAD
git push origin main

# 2. Redeploy previous version
kubectl rollout undo deployment/investorcenter-backend-deployment -n investorcenter
kubectl rollout undo deployment/investorcenter-frontend-deployment -n investorcenter

# 3. (Optional) Drop table if needed
psql -h prod-db -U user -d investorcenter
DROP TABLE IF EXISTS manual_fundamentals;
```

---

## 🔒 Security Considerations

### Production Recommendations:

1. **Add Authentication** (Future)
   ```go
   // Protect write endpoints
   tickers.POST("/:symbol/manual-fundamentals", 
       auth.RequireAdmin(), 
       handlers.PostManualFundamentals)
   ```

2. **Rate Limiting**
   - Add rate limiting to POST endpoint
   - Prevent abuse of data uploads

3. **Input Validation**
   - Consider adding JSON schema validation
   - Validate data types and ranges

4. **Audit Logging**
   - Log who uploaded what data when
   - Track data changes for compliance

---

## 📊 Post-Deployment Monitoring

```bash
# Check backend logs for errors
kubectl logs -f deployment/investorcenter-backend-deployment -n investorcenter | grep "manual-fundamentals"

# Monitor database
psql -h prod-db -U user -d investorcenter
SELECT ticker, created_at, updated_at FROM manual_fundamentals ORDER BY updated_at DESC LIMIT 10;

# Check frontend errors (browser console)
# Look for failed API calls to /manual-fundamentals endpoints
```

---

## 🎯 Post-Deployment Tasks

1. **Test with Real Data**
   - Upload fundamental data for 5-10 tickers
   - Verify display on their ticker pages
   - Check that manual data overrides other sources

2. **User Communication** (if applicable)
   - Notify team about new Input tab
   - Provide training on how to use it
   - Share JSON format documentation

3. **Monitoring Setup**
   - Set up alerts for API errors
   - Monitor database growth
   - Track usage of new endpoints

---

## 📈 Success Metrics

After deployment, verify:
- ✅ Input tab visible on all ticker pages
- ✅ Can save data via UI
- ✅ Data persists after page refresh
- ✅ Manual data displays in Key Metrics sidebar
- ✅ API endpoints respond correctly
- ✅ No errors in backend logs
- ✅ No errors in browser console

---

## 🆘 Troubleshooting

### Issue: "Input tab not showing"
- **Cause**: Frontend not rebuilt/deployed
- **Fix**: Rebuild frontend and clear browser cache

### Issue: "Failed to save data"
- **Cause**: Database migration not run
- **Fix**: Run migration on production database

### Issue: "Data not displaying in sidebar"
- **Cause**: API not returning data or frontend not fetching
- **Fix**: Check browser network tab, verify API response

### Issue: "500 error on POST"
- **Cause**: Database connection issue or table doesn't exist
- **Fix**: Check backend logs, verify migration

---

## ✅ Final Checklist Before Deploying

- [ ] Tested locally and everything works
- [ ] All files committed to git
- [ ] Migration SQL reviewed and tested
- [ ] Database backup taken (recommended)
- [ ] Migration run on production database
- [ ] Backend deployed and healthy
- [ ] Frontend deployed and accessible
- [ ] "Input" tab visible on ticker pages
- [ ] Can save and retrieve data
- [ ] Manual data displays in sidebar
- [ ] No errors in logs
- [ ] Team notified (if applicable)

---

## 🎉 You're Ready!

This is a **safe, non-breaking deployment**:
- ✅ New table (doesn't affect existing data)
- ✅ New API endpoints (backward compatible)
- ✅ New UI tab (doesn't break existing tabs)
- ✅ Graceful fallback (if no manual data, uses existing sources)

**Ready when you are!** 🚀
