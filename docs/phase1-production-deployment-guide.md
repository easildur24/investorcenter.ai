# Phase 1 Authentication - Production Deployment Guide

## Summary of Work Completed

✅ **Database Migration:** Successfully ran on production PostgreSQL
✅ **Kubernetes Secrets:** Updated app-secrets with JWT and SMTP configuration
✅ **Deployment Config:** Updated backend-deployment.yaml with auth environment variables
✅ **Code Merged:** All auth code merged to main branch
✅ **Configuration Committed:** Kubernetes manifests pushed to repository

---

## What's Been Deployed

### 1. Database Changes ✅ COMPLETE

The auth tables have been successfully created in production:
- `users` - User accounts with email/password
- `sessions` - Refresh token storage
- `oauth_providers` - OAuth accounts (Google, GitHub)
- All indexes and triggers created

**Verification:**
```bash
kubectl exec -n investorcenter postgres-simple-794f5cd8b7-qg96s -- \
  psql -U investorcenter -d investorcenter_db -c "\dt users;"
```

### 2. Kubernetes Secrets ✅ COMPLETE

Updated `app-secrets` secret with:
- **JWT Secret:** Strong 256-bit key generated
- **SMTP Host:** smtp.gmail.com (placeholder)
- **SMTP Port:** 587
- **SMTP Username:** noreply@investorcenter.ai (placeholder)
- **SMTP Password:** CONFIGURE_SMTP_PASSWORD (⚠️ **NEEDS CONFIGURATION**)

**⚠️ IMPORTANT:** You must update the SMTP password with real credentials:
```bash
kubectl patch secret app-secrets -n investorcenter --type=json -p='[
  {"op": "replace", "path": "/data/smtp-password", "value": "'$(echo -n 'YOUR_REAL_SMTP_PASSWORD' | base64)'"}
]'
```

### 3. Backend Deployment Configuration ✅ UPDATED

The `backend-deployment.yaml` now includes:
- JWT token expiry settings (1h access, 168h refresh)
- Bcrypt cost factor (12)
- SMTP configuration environment variables
- Frontend URL for email links

---

## Remaining Deployment Steps

### Step 1: Build and Push Docker Image

The backend code with authentication is now in the main branch. You need to build and push a new Docker image:

```bash
# Navigate to backend directory
cd backend

# Build Docker image
docker build -t 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/backend:phase1-auth .

# Login to ECR
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin \
  360358043271.dkr.ecr.us-east-1.amazonaws.com

# Push image
docker push 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/backend:phase1-auth

# Tag as latest
docker tag 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/backend:phase1-auth \
  360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/backend:latest

docker push 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/backend:latest
```

**Alternative: Use existing Dockerfile** (if you have one at project root):
```bash
# From project root
docker buildx build --platform linux/amd64 \
  -t 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/backend:latest \
  -f Dockerfile.backend .

docker push 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/backend:latest
```

### Step 2: Apply Updated Backend Deployment

Once the Docker image is built and pushed:

```bash
# Apply the updated deployment configuration
kubectl apply -f k8s/backend-deployment.yaml

# Watch the rollout
kubectl rollout status deployment/investorcenter-backend -n investorcenter

# Verify pods are running
kubectl get pods -n investorcenter | grep backend
```

### Step 3: Configure SMTP Credentials

⚠️ **CRITICAL:** Email verification and password reset won't work until SMTP is configured.

**Option A: Use SendGrid (Recommended)**
1. Create SendGrid account: https://sendgrid.com
2. Get API key from Settings → API Keys
3. Update secret:
```bash
kubectl patch secret app-secrets -n investorcenter --type=json -p='[
  {"op": "replace", "path": "/data/smtp-host", "value": "'$(echo -n 'smtp.sendgrid.net' | base64)'"},
  {"op": "replace", "path": "/data/smtp-username", "value": "'$(echo -n 'apikey' | base64)'"},
  {"op": "replace", "path": "/data/smtp-password", "value": "'$(echo -n 'YOUR_SENDGRID_API_KEY' | base64)'"}
]'
```

**Option B: Use Gmail**
1. Enable 2FA on Gmail account
2. Generate App Password: https://myaccount.google.com/apppasswords
3. Update secret:
```bash
kubectl patch secret app-secrets -n investorcenter --type=json -p='[
  {"op": "replace", "path": "/data/smtp-username", "value": "'$(echo -n 'your-gmail@gmail.com' | base64)'"},
  {"op": "replace", "path": "/data/smtp-password", "value": "'$(echo -n 'YOUR_APP_PASSWORD' | base64)'"}
]'
```

4. Restart backend pods to pick up new secret:
```bash
kubectl rollout restart deployment/investorcenter-backend -n investorcenter
```

### Step 4: Deploy Frontend Changes

The frontend has new auth pages (`/auth/login`, `/auth/signup`) that need to be deployed.

**If using Vercel:**
```bash
# Vercel will auto-deploy from main branch
# Or trigger manual deployment:
vercel --prod
```

**If using custom hosting:**
```bash
# Build frontend
npm run build

# Deploy built files to your hosting provider
# (specific commands depend on your setup)
```

**Update Frontend Environment Variables:**
Ensure the frontend has the correct API URL:
```bash
# In Vercel dashboard or your hosting provider:
NEXT_PUBLIC_API_URL=https://api.investorcenter.ai/api/v1
# OR if backend is on same domain:
NEXT_PUBLIC_API_URL=/api/v1
```

### Step 5: Update Ingress (if needed)

If you need to update CORS or routing, modify the Ingress configuration:

```bash
# Check current ingress
kubectl get ingress -n investorcenter

# If needed, update ingress to handle auth routes
kubectl edit ingress -n investorcenter
```

Add annotations for CORS if not already present:
```yaml
annotations:
  nginx.ingress.kubernetes.io/cors-allow-origin: "https://investorcenter.ai,https://www.investorcenter.ai"
  nginx.ingress.kubernetes.io/cors-allow-methods: "GET, POST, PUT, DELETE, OPTIONS"
  nginx.ingress.kubernetes.io/cors-allow-headers: "Origin, Content-Type, Accept, Authorization"
  nginx.ingress.kubernetes.io/cors-allow-credentials: "true"
```

---

## Verification Steps

### 1. Check Backend Health

```bash
# Check if backend pods are running
kubectl get pods -n investorcenter | grep backend

# Check pod logs for errors
kubectl logs -n investorcenter -l app=investorcenter-backend --tail=50

# Test health endpoint
curl https://api.investorcenter.ai/health
# Should show: {"database": "healthy", "status": "healthy"}
```

### 2. Test Authentication Endpoints

**Test Signup:**
```bash
curl -X POST https://api.investorcenter.ai/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "testpassword123",
    "full_name": "Test User"
  }'
```

Expected response:
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "expires_in": 3600,
  "user": {
    "id": "...",
    "email": "test@example.com",
    "full_name": "Test User",
    "email_verified": false
  }
}
```

**Test Login:**
```bash
curl -X POST https://api.investorcenter.ai/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "testpassword123"
  }'
```

**Test Protected Route:**
```bash
# Use access_token from above
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  https://api.investorcenter.ai/api/v1/user/me
```

### 3. Test Frontend

1. Visit https://investorcenter.ai/auth/login
2. Try signing up with a new account
3. Check email for verification link (if SMTP configured)
4. Try logging in
5. Verify user menu appears in header when logged in
6. Test logout

### 4. Check Database

```bash
# Connect to production database
kubectl exec -it -n investorcenter postgres-simple-794f5cd8b7-qg96s -- \
  psql -U investorcenter -d investorcenter_db

# Check if users were created
SELECT id, email, full_name, email_verified, created_at FROM users;

# Check sessions
SELECT id, user_id, expires_at, created_at FROM sessions LIMIT 5;

# Exit
\q
```

---

## Rollback Plan

If something goes wrong, you can rollback:

### Rollback Backend Deployment

```bash
# View deployment history
kubectl rollout history deployment/investorcenter-backend -n investorcenter

# Rollback to previous version
kubectl rollout undo deployment/investorcenter-backend -n investorcenter

# Or rollback to specific revision
kubectl rollout undo deployment/investorcenter-backend -n investorcenter --to-revision=X
```

### Rollback Database Migration

⚠️ Only if absolutely necessary (will lose user data):

```bash
kubectl exec -n investorcenter postgres-simple-794f5cd8b7-qg96s -- \
  psql -U investorcenter -d investorcenter_db <<'EOF'
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS oauth_providers CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP FUNCTION IF EXISTS cleanup_expired_sessions CASCADE;
DROP FUNCTION IF EXISTS update_updated_at_column CASCADE;
EOF
```

---

## Monitoring & Logs

### View Backend Logs

```bash
# Tail logs from all backend pods
kubectl logs -f -n investorcenter -l app=investorcenter-backend

# View logs from specific pod
kubectl logs -n investorcenter investorcenter-backend-XXXXX-XXXXX

# View logs with timestamps
kubectl logs -n investorcenter -l app=investorcenter-backend --timestamps=true
```

### Monitor Authentication Events

Look for these log entries:
- `"User {email} signed up"` - New user registration
- `"Failed login attempt for {email}"` - Failed login
- `"User {email} logged in"` - Successful login
- `"Too many requests"` - Rate limiting triggered

### Set Up Alerts (Recommended)

Monitor these metrics:
- Failed login attempts > 10/minute (possible brute force)
- Signup rate spike (possible spam/bots)
- High error rate on auth endpoints
- Database connection failures

---

## Security Checklist

Before going live:

- [ ] JWT_SECRET is strong (32+ bytes, randomly generated) ✅
- [ ] SMTP credentials are configured (not placeholder)
- [ ] HTTPS is enforced (no HTTP traffic)
- [ ] CORS is properly configured (whitelist only)
- [ ] Rate limiting is working (test with 6 failed logins)
- [ ] Email verification is sent (check spam folder)
- [ ] Password reset flow works
- [ ] Sessions expire correctly
- [ ] Tokens cannot be reused after logout
- [ ] Database backups are enabled
- [ ] Logs are being collected (CloudWatch, Datadog, etc.)

---

## Common Issues & Solutions

### Issue: Pods in CrashLoopBackOff

**Cause:** Missing environment variables or database connection failure

**Solution:**
```bash
# Check pod events
kubectl describe pod -n investorcenter investorcenter-backend-XXXXX

# Check if secret exists
kubectl get secret app-secrets -n investorcenter

# Verify database connectivity
kubectl exec -n investorcenter postgres-simple-794f5cd8b7-qg96s -- psql -U investorcenter -d investorcenter_db -c "SELECT 1;"
```

### Issue: "Authorization header required"

**Cause:** Frontend not sending Authorization header

**Solution:**
- Check that `lib/api.ts` is including the token in headers
- Verify token is stored in localStorage
- Check browser console for errors

### Issue: "Invalid or expired token"

**Cause:** JWT_SECRET mismatch or token expired

**Solution:**
- Verify JWT_SECRET in backend matches what was used to sign tokens
- Check token expiry time
- Try logging out and back in to get fresh tokens

### Issue: Email not sending

**Cause:** SMTP credentials not configured or incorrect

**Solution:**
- Check SMTP logs in backend pods
- Verify SMTP credentials in secret
- Test SMTP connection manually
- Check if email is in spam folder

### Issue: CORS errors

**Cause:** Frontend domain not whitelisted

**Solution:**
- Update CORS config in `backend/main.go`:
```go
config.AllowOrigins = []string{
    "https://investorcenter.ai",
    "https://www.investorcenter.ai",
}
```
- Rebuild and redeploy backend

---

## Next Steps After Deployment

1. **Monitor for 24 hours** - Watch logs and metrics
2. **Test all auth flows** - Signup, login, password reset, email verification
3. **Configure SMTP properly** - Replace placeholder with real credentials
4. **Set up monitoring alerts** - For failed logins, errors, etc.
5. **Document for team** - Share this guide with your team
6. **Plan Phase 2** - Watch List Management (next milestone)

---

## Support & Resources

- **Backend Logs:** `kubectl logs -f -n investorcenter -l app=investorcenter-backend`
- **Database Access:** `kubectl exec -it -n investorcenter postgres-simple-794f5cd8b7-qg96s -- psql`
- **Code Review:** See [docs/code-review-phase1-auth.md](./code-review-phase1-auth.md)
- **Tech Spec:** See [docs/phase1-auth-tech-spec.md](./phase1-auth-tech-spec.md)

---

**Deployment Status:** 🟡 Partially Complete

✅ Database migrated
✅ Secrets configured
✅ Deployment updated
⚠️ Docker image needs to be built and pushed
⚠️ SMTP credentials need to be configured
⚠️ Frontend needs to be deployed

**Ready to complete deployment? Follow Steps 1-5 above!**
