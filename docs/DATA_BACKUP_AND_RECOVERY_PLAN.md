# InvestorCenter.ai Data Backup and Recovery Plan

**Last Updated:** November 29, 2025
**Document Owner:** Platform Engineering Team

---

## Table of Contents
1. [Executive Summary](#executive-summary)
2. [Data Classification and Recovery Priority](#data-classification-and-recovery-priority)
3. [Recovery Procedures](#recovery-procedures)
4. [Prevention Strategies](#prevention-strategies)
5. [Monitoring and Alerting](#monitoring-and-alerting)
6. [Recovery Runbook](#recovery-runbook)

---

## Executive Summary

### Incident Context
On November 29, 2025, we experienced complete data loss due to:
- **Root Cause:** PostgreSQL deployment (`postgres-simple`) using EmptyDir storage (ephemeral)
- **Trigger:** Pod restart caused by OOMKilled or node issues
- **Impact:** All database tables emptied; screener returned "Failed to Load Data"
- **Contributing Factor:** PVC (`postgres-pvc`) was Pending for 76 days due to missing IAM permissions

### Lessons Learned
1. EmptyDir volumes are ephemeral and unsuitable for production databases
2. PVC provisioning failures must be monitored and alerted on
3. Data can be reconstructed from external sources (Polygon, SEC EDGAR)
4. Recovery procedures need to be documented and tested

---

## Data Classification and Recovery Priority

### Priority 1: Critical (Required for Basic Functionality)
| Table | Source | Recovery Time | Recovery Method |
|-------|--------|---------------|-----------------|
| `tickers` | Polygon.io | 30 min | polygon_incremental_update.py |
| `stock_prices` | Polygon.io | 4-6 hours | historical_price_backfill.py |
| `financials` | SEC EDGAR | 4-8 hours | sec_financials_ingestion.py |

### Priority 2: High (Required for Full Features)
| Table | Source | Recovery Time | Recovery Method |
|-------|--------|---------------|-----------------|
| `ttm_financials` | Calculated | 30 min | ttm_financials_calculator.py |
| `valuation_ratios` | Calculated | 30 min | valuation_ratios_calculator.py |
| `fundamental_metrics_extended` | Calculated | 1 hour | fundamental_metrics_pipeline.py |
| `technical_indicators` | Calculated | 2 hours | technical_indicators_calculator.py |

### Priority 3: Medium (Enhanced Features)
| Table | Source | Recovery Time | Recovery Method |
|-------|--------|---------------|-----------------|
| `risk_metrics` | Calculated | 2 hours | risk_metrics_calculator.py |
| `analyst_ratings` | Polygon.io | 1 hour | analyst_ratings_ingestion.py |
| `insider_trades` | SEC EDGAR | 2 hours | sec_insider_trades_ingestion.py |
| `benchmark_returns` | Polygon.io | 30 min | benchmark_data_pipeline.py |
| `treasury_rates` | FRED | 10 min | treasury_rates_pipeline.py |

### Priority 4: Low (IC Score Features)
| Table | Source | Recovery Time | Recovery Method |
|-------|--------|---------------|-----------------|
| `institutional_holdings` | SEC EDGAR | 4 hours | sec_13f_ingestion.py |
| `news_articles` | Polygon.io | 2 hours | news_sentiment_ingestion.py |
| `ic_scores` | Calculated | 1 hour | ic_score_calculator.py |

### User Data (Cannot be Recovered)
| Table | Notes |
|-------|-------|
| `users` | Requires user re-registration |
| `watchlists` | Lost permanently |
| `portfolios` | Lost permanently |
| `alerts` | Lost permanently |

---

## Recovery Procedures

### Phase 1: Infrastructure Setup (15 minutes)

```bash
# 1. Verify database is running
kubectl get pods -n investorcenter | grep postgres

# 2. Verify PVC is bound (not Pending)
kubectl get pvc -n investorcenter
# Expected: STATUS = Bound

# 3. If PVC is Pending, fix IAM permissions first
# See: terraform/eks.tf - ensure AmazonEBSCSIDriverPolicy is attached

# 4. Port forward for local access
kubectl port-forward svc/postgres-simple-service 15432:5432 -n investorcenter &
```

### Phase 2: Schema Creation (5 minutes)

```bash
# Apply all migrations
kubectl exec -n investorcenter deployment/postgres-simple -- psql -U investorcenter -d investorcenter_db -f /migrations/001_initial.sql
# ... repeat for all migration files

# Or create tables via script
kubectl exec -n investorcenter deployment/postgres-simple -- psql -U investorcenter -d investorcenter_db <<'EOF'
-- Core tables (tickers, stock_prices, financials, etc.)
-- See: backend/migrations/ and ic-score-service/migrations/
EOF
```

### Phase 3: Priority 1 Recovery (4-6 hours)

#### Step 3.1: Tickers (30 minutes)
```bash
# Option A: Import from CSV
cd /path/to/investorcenter.ai
python scripts/ticker_import_to_db.py

# Option B: Fetch from Polygon API
kubectl create job --from=cronjob/polygon-ticker-update manual-ticker-update -n investorcenter

# Verify
kubectl exec -n investorcenter deployment/postgres-simple -- \
  psql -U investorcenter -d investorcenter_db -c "SELECT COUNT(*) FROM tickers;"
# Expected: ~25,000 rows
```

#### Step 3.2: Stock Prices - 10 Year Backfill (4-6 hours)
```bash
# Set up port forward
kubectl port-forward svc/postgres-simple-service 15432:5432 -n investorcenter &

# Run backfill (in background with nohup)
cd ic-score-service
nohup env \
  DB_HOST=localhost DB_PORT=15432 \
  DB_USER=investorcenter DB_PASSWORD=password123 \
  DB_NAME=investorcenter_db \
  POLYGON_API_KEY=$POLYGON_API_KEY \
  python -m pipelines.historical_price_backfill --all --years 10 --resume \
  > /tmp/historical_price_backfill.log 2>&1 &

# Monitor progress
tail -f /tmp/historical_price_backfill.log
grep "Progress:" /tmp/historical_price_backfill.log | tail -1
```

#### Step 3.3: SEC Financials (4-8 hours)
```bash
# Option A: Run locally (if cluster resources insufficient)
cd ic-score-service
python -m pipelines.sec_financials_ingestion --all --resume

# Option B: Run as Kubernetes job
kubectl create job --from=cronjob/ic-score-sec-financials manual-sec-financials -n investorcenter
```

### Phase 4: Priority 2 Recovery (2-4 hours)

Run after Phase 3 completes (dependencies: stock_prices, financials):

```bash
# TTM Financials (requires financials)
kubectl create job --from=cronjob/ic-score-ttm-financials manual-ttm -n investorcenter

# Valuation Ratios (requires stock_prices + ttm_financials)
kubectl create job --from=cronjob/ic-score-valuation-ratios manual-valuation -n investorcenter

# Fundamental Metrics (requires ttm_financials)
kubectl create job --from=cronjob/ic-score-fundamental-metrics manual-fundamentals -n investorcenter

# Technical Indicators (requires stock_prices)
kubectl create job --from=cronjob/ic-score-technical-indicators manual-technical -n investorcenter
```

### Phase 5: Priority 3 & 4 Recovery (4-8 hours)

Run in parallel after Phase 4:

```bash
# Benchmark data
kubectl create job --from=cronjob/ic-score-benchmark-data manual-benchmark -n investorcenter

# Treasury rates
kubectl create job --from=cronjob/ic-score-treasury-rates manual-treasury -n investorcenter

# Risk metrics (requires stock_prices + benchmark + treasury)
kubectl create job --from=cronjob/ic-score-risk-metrics manual-risk -n investorcenter

# Analyst ratings
kubectl create job --from=cronjob/ic-score-analyst-ratings manual-analyst -n investorcenter

# Fair value calculations
kubectl create job --from=cronjob/ic-score-fair-value manual-fair-value -n investorcenter

# IC Scores (final step, requires all above)
kubectl create job --from=cronjob/ic-score-calculator manual-ic-score -n investorcenter
```

---

## Prevention Strategies

### 1. Use Persistent Storage (Critical)

**Current Problem:**
```yaml
# postgres-simple uses EmptyDir (EPHEMERAL - DO NOT USE)
volumes:
- name: data
  emptyDir: {}  # ❌ Data lost on pod restart
```

**Solution:**
```yaml
# Use PersistentVolumeClaim with EBS storage
volumes:
- name: data
  persistentVolumeClaim:
    claimName: postgres-pvc  # ✅ Data persists across restarts
```

**Required IAM Policy:**
```hcl
# terraform/eks.tf
resource "aws_iam_role_policy_attachment" "eks_ebs_csi_policy" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"
  role       = aws_iam_role.eks_node_group.name
}
```

### 2. Automated Backups

#### Option A: PostgreSQL pg_dump (Daily)
```yaml
# k8s/postgres-backup-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-backup
  namespace: investorcenter
spec:
  schedule: "0 4 * * *"  # Daily at 4 AM UTC
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:15
            command:
            - /bin/sh
            - -c
            - |
              TIMESTAMP=$(date +%Y%m%d_%H%M%S)
              pg_dump -h postgres-simple-service -U investorcenter investorcenter_db | \
                gzip > /backup/investorcenter_${TIMESTAMP}.sql.gz
              # Upload to S3
              aws s3 cp /backup/investorcenter_${TIMESTAMP}.sql.gz \
                s3://investorcenter-backups/postgres/
              # Keep only last 7 days locally
              find /backup -mtime +7 -delete
            env:
            - name: PGPASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: password
            volumeMounts:
            - name: backup
              mountPath: /backup
          volumes:
          - name: backup
            persistentVolumeClaim:
              claimName: postgres-backup-pvc
```

#### Option B: AWS RDS with Automated Backups
Consider migrating to RDS for production:
- Automated daily backups with 35-day retention
- Point-in-time recovery
- Multi-AZ for high availability
- No PVC/EBS management required

### 3. EBS Snapshots (Weekly)
```bash
# Create snapshot of postgres-pvc volume
aws ec2 create-snapshot \
  --volume-id vol-xxxxxxxxxx \
  --description "InvestorCenter Postgres Weekly Backup" \
  --tag-specifications 'ResourceType=snapshot,Tags=[{Key=Name,Value=postgres-weekly}]'
```

### 4. Monitoring and Alerting

#### PVC Status Monitoring
```yaml
# Add to Prometheus/Grafana
- alert: PVCPending
  expr: kube_persistentvolumeclaim_status_phase{phase="Pending"} == 1
  for: 10m
  labels:
    severity: critical
  annotations:
    summary: "PVC {{ $labels.persistentvolumeclaim }} is Pending"
    description: "PVC has been Pending for more than 10 minutes. Check EBS CSI driver and IAM permissions."
```

#### Data Freshness Monitoring
```sql
-- Add to monitoring dashboard
SELECT
  'stock_prices' as table_name,
  MAX(time)::date as latest_data,
  CASE WHEN MAX(time) < NOW() - INTERVAL '2 days' THEN 'STALE' ELSE 'OK' END as status
FROM stock_prices
UNION ALL
SELECT 'ic_scores', MAX(date),
  CASE WHEN MAX(date) < NOW() - INTERVAL '2 days' THEN 'STALE' ELSE 'OK' END
FROM ic_scores;
```

### 5. Infrastructure as Code Validation

```bash
# Pre-commit hook for terraform
terraform plan -detailed-exitcode

# Validate EBS CSI policy is included
grep -q "AmazonEBSCSIDriverPolicy" terraform/eks.tf || echo "WARNING: Missing EBS CSI policy"
```

### 6. Disaster Recovery Testing (Quarterly)

```bash
# Test recovery procedure quarterly
1. Create test namespace
2. Deploy postgres with PVC
3. Run recovery scripts
4. Validate data integrity
5. Document any issues
```

---

## Monitoring and Alerting

### Key Metrics to Monitor

| Metric | Alert Threshold | Action |
|--------|-----------------|--------|
| PVC Status | Pending > 10 min | Check IAM, EBS CSI driver |
| Database Connection | Failed > 1 min | Restart pod, check logs |
| Stock Prices Age | > 2 days stale | Trigger manual backfill |
| CronJob Failures | > 2 consecutive | Investigate and re-run |
| Disk Usage | > 80% | Expand PVC or clean old data |

### Slack/PagerDuty Alerts

```yaml
# alertmanager.yaml
receivers:
- name: 'platform-team'
  slack_configs:
  - channel: '#platform-alerts'
    send_resolved: true
    title: 'InvestorCenter Alert'
    text: '{{ .CommonAnnotations.description }}'
```

---

## Recovery Runbook

### Quick Recovery Checklist

```
□ 1. Verify PVC is Bound (not Pending)
□ 2. Verify postgres pod is Running
□ 3. Check if tables exist (run schema migrations if not)
□ 4. Count rows in tickers table (should be ~25,000)
□ 5. If tickers empty, run polygon_incremental_update
□ 6. Start historical_price_backfill (background, 4-6 hours)
□ 7. After prices complete, run TTM and valuation pipelines
□ 8. Run remaining pipelines in dependency order
□ 9. Verify screener returns data
□ 10. Document any issues encountered
```

### Recovery Commands Cheat Sheet

```bash
# Check database status
kubectl exec -n investorcenter deployment/postgres-simple -- \
  psql -U investorcenter -d investorcenter_db -c "
    SELECT 'tickers' as tbl, COUNT(*) FROM tickers
    UNION ALL SELECT 'stock_prices', COUNT(*) FROM stock_prices
    UNION ALL SELECT 'financials', COUNT(*) FROM financials;"

# Check PVC status
kubectl get pvc -n investorcenter

# Check job status
kubectl get jobs -n investorcenter | head -20

# View job logs
kubectl logs -n investorcenter job/manual-price-backfill

# Monitor backfill progress
grep "Progress:" /tmp/historical_price_backfill.log | tail -1
```

---

## Estimated Full Recovery Time

| Phase | Duration | Notes |
|-------|----------|-------|
| Infrastructure Setup | 15 min | PVC, port forward |
| Schema Creation | 5 min | Run migrations |
| Tickers Import | 30 min | ~25,000 tickers |
| Stock Prices (10yr) | 4-6 hours | ~23,000 tickers × 2,500 days |
| SEC Financials | 4-8 hours | Can run in parallel |
| Calculated Metrics | 2-4 hours | After source data complete |
| **Total** | **8-16 hours** | Depends on API rate limits |

---

## Document History

| Date | Author | Changes |
|------|--------|---------|
| 2025-11-29 | Platform Team | Initial version after data loss incident |

