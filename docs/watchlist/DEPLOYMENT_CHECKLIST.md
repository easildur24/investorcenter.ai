# Watchlist Phase 2 - Production Deployment Checklist

## Pre-Deployment Checks

### Code Quality
- [x] Migration syntax error fixed
- [x] Backend builds successfully
- [x] Frontend builds successfully
- [x] Go code formatted (`go fmt`)
- [x] Integration tests written
- [x] Local migration tested successfully

### Feature Completeness
- [x] Watch list CRUD operations
- [x] Watch list item management (add/remove/update)
- [x] Real-time price integration
- [x] Free tier limit enforcement (10 tickers)
- [x] Default watch list auto-creation
- [x] Bulk import functionality
- [x] Display order/reordering
- [x] Toast notifications (UX enhancement)
- [x] Target price alerts (visual indicators)

## Database Migration

### Step 1: Backup Production Database
```bash
# SSH into database pod or use pg_dump remotely
kubectl exec -it postgres-0 -n investorcenter -- pg_dump -U postgres investorcenter_db > backup_before_watchlist_$(date +%Y%m%d_%H%M%S).sql
```

### Step 2: Apply Migration
```bash
# Copy migration file to pod
kubectl cp backend/migrations/010_watchlist_tables.sql investorcenter/postgres-0:/tmp/

# Execute migration
kubectl exec -it postgres-0 -n investorcenter -- psql -U postgres investorcenter_db -f /tmp/010_watchlist_tables.sql
```

### Step 3: Verify Migration
```bash
# Check tables created
kubectl exec -it postgres-0 -n investorcenter -- psql -U postgres investorcenter_db -c "\d watch_lists"
kubectl exec -it postgres-0 -n investorcenter -- psql -U postgres investorcenter_db -c "\d watch_list_items"

# Check triggers
kubectl exec -it postgres-0 -n investorcenter -- psql -U postgres investorcenter_db -c "\d watch_lists" | grep -i trigger
kubectl exec -it postgres-0 -n investorcenter -- psql -U postgres investorcenter_db -c "\d watch_list_items" | grep -i trigger

# Test auto-create trigger (create test user, verify watch list created)
kubectl exec -it postgres-0 -n investorcenter -- psql -U postgres investorcenter_db << 'EOF'
INSERT INTO users (email, password_hash, full_name) VALUES ('test_migration@example.com', 'hash', 'Test');
SELECT name, is_default FROM watch_lists WHERE user_id = (SELECT id FROM users WHERE email = 'test_migration@example.com');
DELETE FROM users WHERE email = 'test_migration@example.com';
EOF
```

## Backend Deployment

### Step 1: Build Docker Image
```bash
# From project root
docker buildx build --platform linux/amd64 -t investorcenter-api:watchlist-v2 -f backend/Dockerfile .

# Tag for ECR
aws ecr get-login-password --region us-east-1 --profile investorcenter | docker login --username AWS --password-stdin 360358043270.dkr.ecr.us-east-1.amazonaws.com
docker tag investorcenter-api:watchlist-v2 360358043270.dkr.ecr.us-east-1.amazonaws.com/investorcenter-api:watchlist-v2
docker push 360358043270.dkr.ecr.us-east-1.amazonaws.com/investorcenter-api:watchlist-v2
```

### Step 2: Update Kubernetes Deployment
```bash
# Update backend deployment to use new image
kubectl set image deployment/investorcenter-api investorcenter-api=360358043270.dkr.ecr.us-east-1.amazonaws.com/investorcenter-api:watchlist-v2 -n investorcenter

# Watch rollout
kubectl rollout status deployment/investorcenter-api -n investorcenter
```

### Step 3: Verify Backend
```bash
# Check logs
kubectl logs -f deployment/investorcenter-api -n investorcenter --tail=100

# Test health endpoint
kubectl port-forward service/investorcenter-api 8080:8080 -n investorcenter
curl http://localhost:8080/health

# Test watch list endpoints (requires auth token)
# Get auth token from frontend login
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/watchlists
```

## Frontend Deployment

### Step 1: Build Frontend Image
```bash
# From project root
docker buildx build --platform linux/amd64 -t investorcenter-frontend:watchlist-v2 -f Dockerfile .

# Tag for ECR
docker tag investorcenter-frontend:watchlist-v2 360358043270.dkr.ecr.us-east-1.amazonaws.com/investorcenter-frontend:watchlist-v2
docker push 360358043270.dkr.ecr.us-east-1.amazonaws.com/investorcenter-frontend:watchlist-v2
```

### Step 2: Update Kubernetes Deployment
```bash
# Update frontend deployment
kubectl set image deployment/investorcenter-frontend investorcenter-frontend=360358043270.dkr.ecr.us-east-1.amazonaws.com/investorcenter-frontend:watchlist-v2 -n investorcenter

# Watch rollout
kubectl rollout status deployment/investorcenter-frontend -n investorcenter
```

### Step 3: Verify Frontend
```bash
# Check logs
kubectl logs -f deployment/investorcenter-frontend -n investorcenter --tail=100

# Test frontend access
curl https://investorcenter.ai/watchlist
```

## Post-Deployment Testing

### Manual Testing Checklist

#### Authentication
- [ ] Log in with existing user
- [ ] Verify auth middleware protects watch list routes
- [ ] Test unauthorized access (401 response)

#### Watch List Management
- [ ] List watch lists (should show default watch list)
- [ ] Create new watch list
- [ ] Update watch list name/description
- [ ] Delete watch list (confirm cannot delete if has items)
- [ ] Verify default watch list created for new users

#### Watch List Items
- [ ] Add ticker to watch list (valid symbol)
- [ ] Add ticker with notes, tags, target prices
- [ ] Try to add invalid ticker (should fail)
- [ ] Try to add duplicate ticker (should fail with 409)
- [ ] Remove ticker from watch list
- [ ] Update ticker metadata (notes, tags, targets)
- [ ] Add 10 tickers (free tier limit)
- [ ] Try to add 11th ticker (should fail with limit error)

#### Real-Time Prices
- [ ] Verify prices displayed in watch list table
- [ ] Verify price changes color-coded (green/red)
- [ ] Wait 30 seconds, verify auto-refresh works
- [ ] Test with crypto symbols (should use Redis cache)
- [ ] Test with stock symbols (should use Polygon API)

#### Target Price Alerts
- [ ] Set target buy price below current price
- [ ] Verify row highlighted when price at/below target
- [ ] Set target sell price above current price
- [ ] Verify row highlighted when price at/above target
- [ ] Verify alert badge shows correct message

#### Toast Notifications
- [ ] Add ticker → success toast appears
- [ ] Remove ticker → success toast appears
- [ ] Update ticker → success toast appears
- [ ] Try invalid operation → error toast appears
- [ ] Verify toasts auto-dismiss after 5 seconds

#### Bulk Operations
- [ ] Bulk add multiple valid tickers
- [ ] Bulk add mix of valid/invalid tickers
- [ ] Verify response shows added vs failed counts

#### UI/UX
- [ ] Empty state displays when no tickers
- [ ] Loading states show during API calls
- [ ] Error messages display clearly
- [ ] Modal forms validate input
- [ ] Responsive design works on mobile
- [ ] Links to ticker detail pages work

### Database Verification
```bash
# Check data integrity
kubectl exec -it postgres-0 -n investorcenter -- psql -U postgres investorcenter_db << 'EOF'
-- Count watch lists
SELECT COUNT(*) as total_watch_lists FROM watch_lists;

-- Count items
SELECT COUNT(*) as total_items FROM watch_list_items;

-- Check foreign key constraints
SELECT conname, contype FROM pg_constraint WHERE conrelid = 'watch_lists'::regclass;
SELECT conname, contype FROM pg_constraint WHERE conrelid = 'watch_list_items'::regclass;

-- Verify triggers
SELECT tgname, tgtype FROM pg_trigger WHERE tgrelid = 'watch_lists'::regclass;
SELECT tgname, tgtype FROM pg_trigger WHERE tgrelid = 'watch_list_items'::regclass;

-- Check indexes
SELECT indexname, indexdef FROM pg_indexes WHERE tablename = 'watch_lists';
SELECT indexname, indexdef FROM pg_indexes WHERE tablename = 'watch_list_items';
EOF
```

## Monitoring

### Metrics to Monitor
- [ ] API response times for watch list endpoints
- [ ] Watch list creation rate
- [ ] Free tier limit hit rate
- [ ] Error rates (400, 500 responses)
- [ ] Database query performance
- [ ] Frontend page load times

### Logs to Check
```bash
# Backend logs
kubectl logs -f deployment/investorcenter-api -n investorcenter | grep -i watchlist

# Frontend logs
kubectl logs -f deployment/investorcenter-frontend -n investorcenter | grep -i watchlist

# Database logs (if issues)
kubectl logs -f statefulset/postgres -n investorcenter
```

## Rollback Plan

### If Issues Detected

#### Rollback Database Migration
```bash
# Restore from backup
kubectl exec -i postgres-0 -n investorcenter -- psql -U postgres investorcenter_db < backup_before_watchlist_<timestamp>.sql
```

#### Rollback Backend
```bash
# Revert to previous image
kubectl set image deployment/investorcenter-api investorcenter-api=360358043270.dkr.ecr.us-east-1.amazonaws.com/investorcenter-api:<previous-tag> -n investorcenter
kubectl rollout status deployment/investorcenter-api -n investorcenter
```

#### Rollback Frontend
```bash
# Revert to previous image
kubectl set image deployment/investorcenter-frontend investorcenter-frontend=360358043270.dkr.ecr.us-east-1.amazonaws.com/investorcenter-frontend:<previous-tag> -n investorcenter
kubectl rollout status deployment/investorcenter-frontend -n investorcenter
```

## Post-Deployment Tasks

### Documentation
- [ ] Update API documentation with new endpoints
- [ ] Update user guide with watch list feature
- [ ] Document any known issues/limitations
- [ ] Update changelog/release notes

### Communication
- [ ] Notify team of successful deployment
- [ ] Announce new feature to users (if applicable)
- [ ] Update status page (if applicable)

### Next Phase Prep
- [ ] Review Phase 2 completion
- [ ] Plan Phase 3 (Heatmap visualization)
- [ ] Gather user feedback on watch lists
- [ ] Identify any improvements needed

## Troubleshooting

### Common Issues

**Issue: Migration fails**
- Check database connection
- Verify PostgreSQL version compatibility
- Review error logs
- Ensure no conflicting table names

**Issue: Backend won't start**
- Check environment variables
- Verify database connectivity
- Review Go binary logs
- Check image architecture (amd64 vs arm64)

**Issue: Real-time prices not updating**
- Verify Polygon API key set
- Check Redis connectivity for crypto
- Review cache service logs
- Test API endpoints directly

**Issue: Free tier limit not enforcing**
- Verify trigger created successfully
- Check user `is_premium` field
- Review trigger function logic
- Test with non-premium user

**Issue: Frontend errors**
- Check CORS configuration
- Verify API_URL environment variable
- Review browser console errors
- Test API endpoints with curl

## Success Criteria

Deployment is successful when:
- [x] Migration applied without errors
- [ ] All database constraints and triggers working
- [ ] Backend deployed and healthy
- [ ] Frontend deployed and accessible
- [ ] All manual tests passing
- [ ] No critical errors in logs
- [ ] Real-time prices updating
- [ ] Free tier limits enforced
- [ ] Target price alerts working
- [ ] Toast notifications working
- [ ] No performance degradation

## Deployment Log

| Date | Time | Action | Status | Notes |
|------|------|--------|--------|-------|
| | | Database backup | | |
| | | Migration applied | | |
| | | Backend deployed | | |
| | | Frontend deployed | | |
| | | Manual testing | | |
| | | Monitoring enabled | | |

## Sign-Off

- [ ] Tech Lead Review
- [ ] QA Testing Complete
- [ ] Production Deployment Complete
- [ ] Monitoring Confirmed

Deployed by: _______________
Date: _______________
Time: _______________
