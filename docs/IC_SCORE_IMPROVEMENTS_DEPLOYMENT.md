# IC Score Improvements - Test & Deployment Guide

## Overview

This document covers testing and deployment of the IC Score improvements:
- **B**: Aligned grading system (A=Strong Buy, C=Hold, etc.)
- **C**: Factor score transparency with calculation breakdowns
- **D**: Score composition display
- **A**: AI-powered contextual analysis (Gemini)

---

## 1. Local Testing

### Prerequisites
```bash
# Frontend (Next.js)
cd /home/user/investorcenter.ai
npm install

# IC Score Service (Python)
cd ic-score-service
pip install -r requirements.txt
pip install google-generativeai  # For AI analysis
```

### Environment Setup

**Frontend** (`.env.local`):
```bash
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_IC_SCORE_API_URL=http://localhost:8001
```

**IC Score Service** (`ic-score-service/.env`):
```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=investorcenter_db

# AI Analysis (required for AI feature)
GEMINI_API_KEY=AIzaSyAF9KZfX4UFx1snjM1rndJ8ulgMA0upb4A
```

### Start Services

```bash
# Terminal 1: Go Backend
cd backend && go run main.go

# Terminal 2: IC Score API
cd ic-score-service
uvicorn api.main:app --reload --port 8001

# Terminal 3: Frontend
npm run dev
```

### Test Checklist

#### B. Aligned Grading System
- [ ] Visit `/ticker/AAPL` and check IC Score card
- [ ] Score of 50 should show **Grade: C-** (not F)
- [ ] Score of 65 should show **Grade: B-** (not D)
- [ ] Score of 80 should show **Grade: A-** (not B-)

#### C. Factor Transparency
- [ ] Click on a factor card (e.g., "Value")
- [ ] Should expand to show detailed metrics (P/E, P/B, P/S)
- [ ] Each metric shows value and individual score contribution

#### D. Score Composition
- [ ] Click "Score Composition" in the IC Score widget
- [ ] Should show each factor with: `Score × Weight = Contribution`
- [ ] Missing factors should be listed at bottom

#### A. AI Analysis
```bash
# Test API endpoint directly
curl -X POST "http://localhost:8001/api/scores/AAPL/ai-analysis" \
  -H "Content-Type: application/json" \
  -d '{"ticker": "AAPL"}'
```
- [ ] Response includes: analysis, key_strengths, key_concerns, investment_thesis, risk_factors
- [ ] Frontend: Click "Get AI Analysis" button on IC Score card
- [ ] Analysis displays with colored sections (green strengths, amber concerns, etc.)

---

## 2. Staging Deployment

### Build Docker Image

```bash
cd ic-score-service

# Build with new dependencies
docker build -t ic-score-service:staging .

# Tag for ECR
docker tag ic-score-service:staging \
  360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/ic-score-service:staging
```

### Push to ECR

```bash
# Login to ECR
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin 360358043271.dkr.ecr.us-east-1.amazonaws.com

# Push
docker push 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/ic-score-service:staging
```

### Deploy to Staging Namespace

```bash
# Create staging secret with Gemini API key
kubectl create secret generic ic-score-ai-secrets \
  --from-literal=GEMINI_API_KEY=AIzaSyAF9KZfX4UFx1snjM1rndJ8ulgMA0upb4A \
  -n staging --dry-run=client -o yaml | kubectl apply -f -

# Deploy with staging image tag
kubectl set image deployment/ic-score-api \
  ic-score-api=360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/ic-score-service:staging \
  -n staging

# Verify rollout
kubectl rollout status deployment/ic-score-api -n staging
```

---

## 3. Production Deployment

### Step 1: Create Gemini API Secret

```bash
# Create secret in production namespace
kubectl create secret generic ic-score-ai-secrets \
  --from-literal=GEMINI_API_KEY=<YOUR_PRODUCTION_API_KEY> \
  -n investorcenter
```

### Step 2: Update Deployment to Use Secret

Create/update `ic-score-service/k8s/ic-score-api-deployment.yaml`:

Add this env var section after the existing DB env vars:

```yaml
        # AI Analysis (Gemini)
        - name: GEMINI_API_KEY
          valueFrom:
            secretKeyRef:
              name: ic-score-ai-secrets
              key: GEMINI_API_KEY
```

### Step 3: Build & Push Production Image

```bash
cd ic-score-service

# Build production image
docker build -t ic-score-service:latest .

# Tag for ECR
docker tag ic-score-service:latest \
  360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/ic-score-service:latest

# Push
docker push 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/ic-score-service:latest
```

### Step 4: Apply Deployment

```bash
# Apply updated deployment with Gemini secret
kubectl apply -f ic-score-service/k8s/ic-score-api-deployment.yaml

# Watch rollout
kubectl rollout status deployment/ic-score-api -n investorcenter

# Verify pods are running
kubectl get pods -n investorcenter -l app=ic-score-api
```

### Step 5: Deploy Frontend

```bash
# Build frontend with production env
npm run build

# Build Docker image
docker build -t frontend:latest .

# Tag and push
docker tag frontend:latest \
  360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/frontend:latest
docker push 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/frontend:latest

# Deploy
kubectl rollout restart deployment/frontend -n investorcenter
```

### Step 6: Deploy Go Backend

```bash
cd backend

# Build
docker build -t backend:latest .

# Tag and push
docker tag backend:latest \
  360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/backend:latest
docker push 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/backend:latest

# Deploy
kubectl rollout restart deployment/backend -n investorcenter
```

---

## 4. Verification

### API Health Checks

```bash
# IC Score API health
curl https://api.investorcenter.ai/ic-score/health

# Test AI analysis endpoint
curl -X POST "https://api.investorcenter.ai/ic-score/api/scores/AAPL/ai-analysis" \
  -H "Content-Type: application/json" \
  -d '{"ticker": "AAPL"}'
```

### Frontend Verification

1. Visit `https://investorcenter.ai/ticker/AAPL`
2. Scroll to IC Score section
3. Verify:
   - [ ] Grades align with ratings (C = Hold)
   - [ ] Factor cards expand on click
   - [ ] Score composition shows breakdown
   - [ ] "Get AI Analysis" button works

### Logs

```bash
# IC Score API logs
kubectl logs -f deployment/ic-score-api -n investorcenter

# Check for AI analysis errors
kubectl logs deployment/ic-score-api -n investorcenter | grep -i "ai\|gemini"
```

---

## 5. Rollback (if needed)

```bash
# Rollback IC Score API
kubectl rollout undo deployment/ic-score-api -n investorcenter

# Rollback Frontend
kubectl rollout undo deployment/frontend -n investorcenter

# Rollback Backend
kubectl rollout undo deployment/backend -n investorcenter
```

---

## 6. Cost Considerations

### Gemini API Pricing (2.0 Flash-Lite)
- Input: $0.075 per 1M tokens
- Output: $0.30 per 1M tokens

### Estimated Monthly Cost
- ~500 tokens per analysis request
- If 1,000 analyses/day = 30,000/month
- Cost: ~$2-5/month

### Rate Limiting Recommendation
Consider adding rate limiting to the AI analysis endpoint to prevent abuse:
```python
# In main.py, add rate limiting middleware or per-user limits
```

---

## Files Changed

| File | Change |
|------|--------|
| `lib/api/ic-score.ts` | New types, aligned grading functions |
| `backend/models/ic_score.go` | Expose calculation_metadata |
| `components/ic-score/FactorBreakdown.tsx` | Expandable cards with metrics |
| `components/ic-score/ICScoreWidget.tsx` | Score composition display |
| `components/ic-score/ICScoreExplainer.tsx` | Updated grading table |
| `components/ic-score/ICScoreAIAnalysis.tsx` | NEW - AI analysis component |
| `ic-score-service/api/main.py` | AI analysis endpoint |
| `ic-score-service/requirements.txt` | Added google-generativeai |
