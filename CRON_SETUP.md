# Ticker Update CronJob Setup

Automated ticker data updates using Kubernetes CronJob.

## Overview

The ticker update system runs weekly to:
- Fetch latest ticker data from Nasdaq and NYSE exchanges
- Add new IPOs and listings to the database
- Skip existing tickers (incremental updates only)
- Log all operations for monitoring

## Schedule

- **Frequency**: Weekly on Sunday at 2 AM UTC
- **Retry Policy**: 3 attempts on failure
- **Concurrency**: Only one job runs at a time
- **History**: Keep last 3 successful + 1 failed job for debugging

## Deployment

### 1. Build Docker Image
```bash
# Build ticker-updater container
./scripts/build-ticker-updater.sh
```

### 2. Deploy CronJob
```bash
# Deploy to Kubernetes
make k8s-deploy-cron
```

### 3. Verify Deployment
```bash
# Check CronJob status
make k8s-cron-status

# View recent logs
make k8s-cron-logs
```

## Manual Operations

### Trigger Manual Update
```bash
# Create one-time job from CronJob template
kubectl create job --from=cronjob/ticker-update manual-ticker-update -n investorcenter

# Check job status
kubectl get jobs -n investorcenter

# View logs
kubectl logs job/manual-ticker-update -n investorcenter
```

### Monitor CronJob
```bash
# List all CronJobs
kubectl get cronjobs -n investorcenter

# Describe CronJob details
kubectl describe cronjob ticker-update -n investorcenter

# View recent job history
kubectl get jobs -n investorcenter -l app=ticker-update
```

## Configuration

### Environment Variables
- `DB_HOST`: postgres-service (internal K8s service)
- `DB_PORT`: 5432
- `DB_NAME`: investorcenter_db
- `DB_USER`: From postgres-secret
- `DB_PASSWORD`: From postgres-secret
- `DB_SSLMODE`: require

### Resource Limits
- **Memory**: 256Mi request, 512Mi limit
- **CPU**: 100m request, 500m limit
- **Runtime**: Typically completes in 2-5 minutes

## Troubleshooting

### CronJob Not Running
```bash
# Check CronJob exists
kubectl get cronjobs -n investorcenter

# Check events
kubectl get events -n investorcenter --sort-by=.metadata.creationTimestamp

# Check if next run is scheduled
kubectl describe cronjob ticker-update -n investorcenter
```

### Job Failures
```bash
# View failed job logs
kubectl logs -n investorcenter -l app=ticker-update --previous

# Check job details
kubectl describe job <job-name> -n investorcenter

# Common issues:
# - Database connection timeout
# - Exchange API rate limiting
# - Network connectivity issues
```

### Update Schedule
```bash
# Edit CronJob schedule
kubectl patch cronjob ticker-update -n investorcenter -p '{"spec":{"schedule":"0 6 * * 1"}}'

# Suspend CronJob temporarily
kubectl patch cronjob ticker-update -n investorcenter -p '{"spec":{"suspend":true}}'

# Resume CronJob
kubectl patch cronjob ticker-update -n investorcenter -p '{"spec":{"suspend":false}}'
```

## Expected Behavior

### Normal Operation
```
2025-09-01 02:00:00 - Starting ticker update job
2025-09-01 02:00:05 - Connected to database: 4,643 existing stocks
2025-09-01 02:00:10 - Fetching from Nasdaq exchange...
2025-09-01 02:00:45 - Fetching from NYSE exchange...
2025-09-01 02:01:20 - Downloaded 6,916 raw tickers
2025-09-01 02:01:25 - Filtered to 4,645 valid tickers  
2025-09-01 02:01:30 - Import completed: 2 inserted, 4,643 skipped
2025-09-01 02:01:35 - Job completed successfully
```

### When New IPOs Available
```
2025-09-01 02:01:30 - Import completed: 15 inserted, 4,628 skipped
2025-09-01 02:01:35 - New companies added:
  - NEWCO (NEW COMPANY INC.)
  - STARTUP (STARTUP CORP)
  - ...
```

## Future Enhancements (TODOs)

- **Alerting**: Slack/email notifications on consecutive failures
- **Graceful degradation**: Handle exchange API downtime gracefully  
- **Delisted companies**: Archive/soft-delete companies no longer listed
- **Metrics**: Export job metrics to monitoring system
- **Scheduling**: Dynamic scheduling based on market calendar
