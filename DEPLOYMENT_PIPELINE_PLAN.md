# Deployment Pipeline Implementation Plan

## Overview
Implement automated CI/CD pipeline using GitHub Actions for both backend (Go/K8s) and frontend (Next.js/Vercel).

**Timeline**: 3-4 hours total
**Approach**: Hybrid - GitHub Actions + AWS EKS (backend) + Vercel (frontend)

---

## Prerequisites

### Required Access
- [ ] GitHub repository admin access
- [ ] AWS credentials with ECR and EKS permissions
- [ ] AWS CLI configured with profile: `AdministratorAccess-360358043271`
- [ ] kubectl configured for EKS cluster: `investorcenter-cluster`

### Required Secrets (GitHub Repository Settings)
Navigate to: `Settings → Secrets and variables → Actions → New repository secret`

**Backend Secrets:**
- `AWS_ACCESS_KEY_ID` - AWS access key
- `AWS_SECRET_ACCESS_KEY` - AWS secret key
- `AWS_REGION` - `us-east-1`
- `ECR_REPOSITORY` - `360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter-backend`
- `EKS_CLUSTER_NAME` - `investorcenter-cluster`
- `DB_PASSWORD_PROD` - Production database password
- `POLYGON_API_KEY` - Polygon.io API key

**Frontend Secrets:**
- `VERCEL_TOKEN` - Vercel deployment token (if using Vercel)
- `VERCEL_ORG_ID` - Vercel organization ID (if using Vercel)
- `VERCEL_PROJECT_ID` - Vercel project ID (if using Vercel)

---

## Phase 1: Backend CI/CD Pipeline (2-3 hours)

### Step 1.1: Add Go Tests to CI
**File**: `.github/workflows/ci.yml`

**Prompt:**
```
Add Go backend testing and linting to the existing CI workflow.

Requirements:
- Add a new job called "backend-test" that runs in parallel with existing jobs
- Use Go 1.21 or later
- Run tests with: cd backend && go test ./... -v
- Run linting with: cd backend && go vet ./...
- Run formatting check: cd backend && test -z $(gofmt -l .)
- Cache Go modules for faster builds
- Only run when backend/** files change or on pull requests
```

### Step 1.2: Create Backend Deployment Workflow
**File**: `.github/workflows/deploy-backend.yml`

**Prompt:**
```
Create a GitHub Actions workflow to build and deploy the Go backend to AWS EKS.

Requirements:
- Trigger on push to main branch when backend/** files change
- Trigger on manual workflow_dispatch
- Use ubuntu-latest runner
- Add jobs:
  1. build-and-push:
     - Checkout code
     - Set up Go 1.21
     - Run tests (cd backend && go test ./...)
     - Build Linux binary: GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o main .
     - Configure AWS credentials from secrets
     - Login to ECR using aws ecr get-login-password
     - Build Docker image using backend/Dockerfile.prebuilt
     - Tag with both 'latest' and git SHA
     - Push to ECR repository from secret ECR_REPOSITORY
  2. deploy-to-eks:
     - needs: build-and-push
     - Configure AWS credentials
     - Update kubeconfig for EKS cluster
     - Update deployment with new image: kubectl set image deployment/backend-deployment backend=$ECR_REPOSITORY:$GITHUB_SHA -n investorcenter
     - Wait for rollout: kubectl rollout status deployment/backend-deployment -n investorcenter
     - Show deployment status

Use GitHub secrets: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION, ECR_REPOSITORY, EKS_CLUSTER_NAME
```

### Step 1.3: Test Backend Pipeline
**Prompts:**

1. **Dry run test:**
```
Review the deploy-backend.yml workflow and check for any syntax errors or issues.
Use: gh workflow view or yamllint if available.
```

2. **Trigger test deployment:**
```
Create a minimal backend code change (add a comment to main.go) and push to a feature branch.
Create a PR to test that backend-test job runs correctly.
After merge to main, monitor the deployment workflow:
- Check GitHub Actions tab for deploy-backend workflow
- Verify ECR image was pushed
- Verify K8s deployment was updated
- Check pod logs: kubectl logs -n investorcenter -l app=backend --tail=50
```

---

## Phase 2: Frontend CI/CD Pipeline (1-2 hours)

### Step 2.1: Choose Frontend Deployment Target

**Option A: Vercel (Recommended - Easiest)**

**Setup Steps:**
1. Go to https://vercel.com and sign in with GitHub
2. Import the investorcenter.ai repository
3. Configure project:
   - Framework Preset: Next.js
   - Root Directory: ./
   - Build Command: npm run build
   - Output Directory: .next
4. Get deployment credentials:
   - Go to Account Settings → Tokens → Create Token
   - Copy token (this is VERCEL_TOKEN secret)
   - Copy org ID and project ID from project settings

**Prompt for Vercel workflow:**
```
Create a GitHub Actions workflow for Vercel deployment.

File: .github/workflows/deploy-frontend.yml

Requirements:
- Trigger on push to main when app/**, components/**, lib/**, or public/** files change
- Trigger on manual workflow_dispatch
- Jobs:
  1. deploy:
     - Checkout code
     - Set up Node.js 18
     - Install dependencies: npm ci
     - Run linting: npm run lint
     - Install Vercel CLI: npm i -g vercel
     - Pull Vercel environment: vercel pull --yes --environment=production --token=$VERCEL_TOKEN
     - Build project: vercel build --prod --token=$VERCEL_TOKEN
     - Deploy to Vercel: vercel deploy --prebuilt --prod --token=$VERCEL_TOKEN

Use secrets: VERCEL_TOKEN, VERCEL_ORG_ID, VERCEL_PROJECT_ID

Add environment variables to Vercel project settings:
- NEXT_PUBLIC_API_URL: https://api.investorcenter.ai (or your backend URL)
```

**Option B: AWS S3 + CloudFront (More Control)**

**Prompt for AWS workflow:**
```
Create a GitHub Actions workflow to deploy Next.js frontend to AWS S3 + CloudFront.

File: .github/workflows/deploy-frontend.yml

Requirements:
- Trigger on push to main when frontend files change
- Jobs:
  1. build-and-deploy:
     - Checkout code
     - Set up Node.js 18
     - Install dependencies: npm ci
     - Create .env.production with: NEXT_PUBLIC_API_URL=https://api.investorcenter.ai
     - Build Next.js: npm run build
     - Configure AWS credentials
     - Sync build to S3: aws s3 sync out/ s3://investorcenter-frontend --delete
     - Invalidate CloudFront: aws cloudfront create-invalidation --distribution-id $CLOUDFRONT_ID --paths "/*"

Additional setup required:
1. Create S3 bucket: investorcenter-frontend
2. Enable static website hosting
3. Create CloudFront distribution pointing to S3
4. Add GitHub secrets: S3_BUCKET, CLOUDFRONT_DISTRIBUTION_ID

Note: Requires Next.js static export config in next.config.js:
output: 'export'
images: { unoptimized: true }
```

### Step 2.2: Add Frontend Tests to CI
**File**: `.github/workflows/ci.yml`

**Prompt:**
```
Add Next.js frontend testing to the CI workflow.

Requirements:
- Add new job called "frontend-test"
- Use Node.js 18
- Install dependencies: npm ci
- Run linting: npm run lint
- Run build test: npm run build (to catch build errors)
- Cache node_modules for faster builds
- Only run when app/**, components/**, lib/** files change
```

### Step 2.3: Test Frontend Pipeline
**Prompts:**

1. **Test build locally:**
```
Run these commands to verify build works:
npm ci
npm run lint
npm run build

Fix any errors before pushing.
```

2. **Test deployment:**
```
Make a small frontend change (update homepage text in app/page.tsx).
Push to feature branch and create PR.
Verify frontend-test job passes.
Merge to main and monitor deploy-frontend workflow.
Verify deployment:
- For Vercel: Check Vercel dashboard for deployment
- For AWS: Check S3 bucket updated and CloudFront invalidation completed
Visit the live site to confirm changes.
```

---

## Phase 3: Environment Configuration (30 minutes)

### Step 3.1: Configure Environment Variables

**Backend (K8s ConfigMap/Secrets):**

**Prompt:**
```
Review and update Kubernetes ConfigMap for backend environment variables.

File: k8s/backend-deployment.yaml

Ensure these environment variables are set:
- DB_HOST (from postgres-service)
- DB_PORT (5432)
- DB_USER (from secret)
- DB_PASSWORD (from secret)
- DB_NAME (investorcenter_db)
- REDIS_HOST (redis-service)
- REDIS_PORT (6379)
- POLYGON_API_KEY (from secret)
- PORT (8080)

Create/update secret:
kubectl create secret generic backend-secrets \
  --from-literal=db-password=$DB_PASSWORD_PROD \
  --from-literal=polygon-api-key=$POLYGON_API_KEY \
  -n investorcenter \
  --dry-run=client -o yaml | kubectl apply -f -
```

**Frontend (Vercel or .env.production):**

**Prompt for Vercel:**
```
Add environment variables to Vercel project:
1. Go to Vercel project → Settings → Environment Variables
2. Add:
   - NEXT_PUBLIC_API_URL = https://api.investorcenter.ai (or your backend load balancer URL)
   - NODE_ENV = production

Redeploy after adding variables.
```

**Prompt for AWS CloudFront:**
```
Update next.config.js to use environment variables:

Add to next.config.js:
env: {
  NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
}

Create .env.production with:
NEXT_PUBLIC_API_URL=https://api.investorcenter.ai

Ensure this file is NOT in .gitignore so it gets built into the static export.
```

### Step 3.2: Setup Backend Load Balancer

**Prompt:**
```
Expose backend API via LoadBalancer service for frontend access.

Update k8s/backend-service.yaml:
- Change type from ClusterIP to LoadBalancer
- Add annotations for AWS:
  service.beta.kubernetes.io/aws-load-balancer-type: nlb
  service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing

Apply changes:
kubectl apply -f k8s/backend-service.yaml -n investorcenter

Get load balancer URL:
kubectl get svc backend-service -n investorcenter -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'

Update frontend NEXT_PUBLIC_API_URL with this URL.
```

---

## Phase 4: Advanced Features (Optional - 1-2 hours)

### Step 4.1: Add Staging Environment

**Prompt:**
```
Create staging deployment workflow that mirrors production.

Requirements:
- Trigger on push to 'develop' branch
- Deploy backend to 'investorcenter-staging' namespace
- Use separate database (or same DB with prefix)
- Deploy frontend to Vercel preview or separate S3 bucket
- Add GitHub environment protection rules:
  - Production: Require approval
  - Staging: Auto-deploy

Files to create:
- .github/workflows/deploy-backend-staging.yml
- .github/workflows/deploy-frontend-staging.yml
- k8s/staging/ (copy of k8s/ with namespace changes)
```

### Step 4.2: Add Database Migrations to Pipeline

**Prompt:**
```
Add database migration step to backend deployment workflow.

Update .github/workflows/deploy-backend.yml:

Add job before deploy-to-eks:
  migrate-database:
    runs-on: ubuntu-latest
    steps:
      - Checkout code
      - Configure AWS credentials
      - Update kubeconfig
      - Create migration job:
        kubectl create job migrate-$(date +%s) \
          --from=cronjob/database-migrations \
          -n investorcenter
      - Wait for completion:
        kubectl wait --for=condition=complete job/migrate-* -n investorcenter --timeout=300s

Create K8s CronJob for migrations:
File: k8s/database-migration-job.yaml
- Uses same backend image
- Runs: ./main --migrate
- Suspend: true (only run manually via job creation)

Requires updating main.go to support --migrate flag that runs SQL files.
```

### Step 4.3: Add Rollback Strategy

**Prompt:**
```
Add rollback capability to backend deployment.

Update .github/workflows/deploy-backend.yml:

Add manual workflow_dispatch inputs:
  rollback:
    description: 'Rollback to previous version'
    required: false
    default: 'false'

Add rollback job:
  rollback:
    if: github.event.inputs.rollback == 'true'
    runs-on: ubuntu-latest
    steps:
      - Configure AWS and kubectl
      - Rollback deployment:
        kubectl rollout undo deployment/backend-deployment -n investorcenter
      - Check status:
        kubectl rollout status deployment/backend-deployment -n investorcenter

To rollback: Go to Actions → deploy-backend → Run workflow → Check rollback box
```

### Step 4.4: Add Monitoring and Notifications

**Prompt:**
```
Add Slack notifications for deployment status.

Requirements:
1. Create Slack webhook:
   - Go to Slack workspace → Apps → Incoming Webhooks
   - Create webhook for #deployments channel
   - Copy webhook URL

2. Add GitHub secret: SLACK_WEBHOOK_URL

3. Update both deployment workflows to add notification steps:
   - On success: Send success message with deployment details
   - On failure: Send failure message with logs link

Use action: slackapi/slack-github-action@v1
```

---

## Verification Checklist

### Backend Pipeline
- [ ] Go tests run on every PR
- [ ] Docker image builds successfully
- [ ] Image pushed to ECR with correct tags
- [ ] K8s deployment updates automatically
- [ ] New pods start successfully
- [ ] Health check endpoint responds: `curl https://api.investorcenter.ai/health`
- [ ] Old pods terminated gracefully

### Frontend Pipeline
- [ ] Linting runs on every PR
- [ ] Build succeeds without errors
- [ ] Deployment completes successfully
- [ ] Frontend accessible at production URL
- [ ] API calls work (check Network tab)
- [ ] Environment variables loaded correctly

### Integration
- [ ] Frontend can communicate with backend
- [ ] CORS configured correctly
- [ ] Authentication works (if applicable)
- [ ] Real-time features work (WebSocket/SSE)
- [ ] Database connections stable

---

## Troubleshooting Guide

### Backend Deployment Fails

**Issue**: ECR login fails
```bash
# Fix: Refresh AWS credentials in GitHub secrets
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin $ECR_REPOSITORY
```

**Issue**: kubectl can't connect to cluster
```bash
# Fix: Update kubeconfig
aws eks update-kubeconfig --name investorcenter-cluster --region us-east-1
kubectl get nodes  # Verify connection
```

**Issue**: Pods crash after deployment
```bash
# Debug:
kubectl logs -n investorcenter -l app=backend --tail=100
kubectl describe pod -n investorcenter <pod-name>

# Common causes:
# - Missing environment variables (check ConfigMap/Secret)
# - Database connection issues (check DB_HOST, DB_PASSWORD)
# - Binary architecture mismatch (ensure GOOS=linux GOARCH=amd64)
```

### Frontend Deployment Fails

**Issue**: Vercel build fails
```bash
# Debug locally:
npm ci
npm run build

# Check for:
# - TypeScript errors
# - Missing environment variables
# - Import path issues
```

**Issue**: API calls fail (CORS errors)
```bash
# Fix backend CORS configuration:
# In backend/main.go, ensure frontend URL in allowed origins:
config := cors.DefaultConfig()
config.AllowOrigins = []string{
    "http://localhost:3000",
    "https://investorcenter.ai",
    "https://www.investorcenter.ai",
    "https://investorcenter.vercel.app"  // Add Vercel domain
}
```

**Issue**: Environment variables not loading
```bash
# For Vercel: Check project settings → Environment Variables
# For AWS: Ensure .env.production exists and correct

# Verify in browser console:
console.log(process.env.NEXT_PUBLIC_API_URL)
```

---

## Maintenance

### Regular Tasks
- **Weekly**: Review deployment logs for errors
- **Monthly**: Update dependencies (npm audit, go get -u)
- **Quarterly**: Review and rotate secrets

### Updating Dependencies
```bash
# Backend
cd backend
go get -u ./...
go mod tidy

# Frontend
npm update
npm audit fix

# Test locally before pushing
make test
```

### Monitoring Deployment Health
```bash
# Backend
kubectl get pods -n investorcenter -w
kubectl top pods -n investorcenter

# Frontend (Vercel)
# Check Vercel dashboard → Analytics

# Frontend (AWS)
aws cloudfront get-distribution --id $CLOUDFRONT_ID
aws s3 ls s3://investorcenter-frontend --summarize
```

---

## Cost Estimate

### GitHub Actions
- **Free tier**: 2,000 minutes/month for private repos
- **Estimated usage**: ~20 mins/day = 600 mins/month
- **Cost**: $0 (within free tier)

### AWS Resources
- **ECR**: $0.10/GB/month (~$1-5/month)
- **EKS**: $0.10/hour = ~$73/month (cluster cost)
- **Load Balancer**: ~$16/month (NLB)
- **CloudFront** (if used): ~$1-10/month depending on traffic
- **Total AWS**: ~$90-105/month

### Vercel (if used)
- **Hobby**: Free (1 concurrent build, 100 GB bandwidth)
- **Pro**: $20/month (unlimited builds, 1 TB bandwidth)
- Recommended: Start with Hobby tier

**Total Monthly Cost**: ~$90-125 (mostly AWS EKS)

---

## Next Steps After Implementation

1. **Set up monitoring**: Add Prometheus/Grafana for K8s monitoring
2. **Add logging**: Implement centralized logging (CloudWatch or ELK)
3. **Performance**: Add CDN caching rules
4. **Security**: Implement WAF rules, rate limiting
5. **Backup**: Automate database backups
6. **Documentation**: Create runbook for common operations

---

## Support

**Useful Commands:**
```bash
# Check all workflows
gh workflow list

# View workflow runs
gh run list --workflow=deploy-backend.yml

# Watch deployment live
gh run watch

# Manual trigger
gh workflow run deploy-backend.yml

# View logs
gh run view --log
```

**Key Documentation:**
- GitHub Actions: https://docs.github.com/en/actions
- AWS EKS: https://docs.aws.amazon.com/eks/
- Vercel: https://vercel.com/docs
- kubectl: https://kubernetes.io/docs/reference/kubectl/

---

## Implementation Order

**Recommended sequence:**
1. ✅ Phase 1.1: Add Go tests to CI (15 min)
2. ✅ Phase 1.2: Create backend deployment workflow (45 min)
3. ✅ Phase 1.3: Test backend pipeline (30 min)
4. ✅ Phase 2.1: Setup Vercel (15 min)
5. ✅ Phase 2.2: Create frontend deployment workflow (30 min)
6. ✅ Phase 2.3: Test frontend pipeline (20 min)
7. ✅ Phase 3: Configure environments (30 min)
8. ⏸️ Phase 4: Optional advanced features (as needed)

**Total time**: ~3 hours for core pipeline, +2 hours for advanced features

---

*Last Updated: 2025-09-30*
*Author: Claude Code Assistant*