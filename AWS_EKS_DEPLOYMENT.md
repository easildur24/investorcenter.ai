# AWS EKS Production Deployment

Step-by-step guide to deploy InvestorCenter.ai with automated ticker updates to AWS EKS.

## Prerequisites

1. **AWS Account** with appropriate permissions
2. **AWS CLI** configured with credentials
3. **Terraform** installed
4. **kubectl** installed
5. **Docker** installed

## Step-by-Step Deployment

### 1. Deploy AWS Infrastructure
```bash
# Navigate to terraform directory
cd terraform

# Initialize Terraform
terraform init

# Review planned changes
terraform plan

# Deploy infrastructure (EKS cluster, VPC, ECR repositories)
terraform apply
```

This creates:
- **EKS Cluster** with worker nodes
- **VPC** with public/private subnets
- **ECR Repositories** for container images
- **Security Groups** and IAM roles
- **Route53** and SSL certificates

### 2. Configure kubectl for EKS
```bash
# Get cluster name from terraform output
CLUSTER_NAME=$(terraform output -raw cluster_id)
AWS_REGION=$(terraform output -raw aws_region || echo "us-east-1")

# Configure kubectl to use EKS cluster
aws eks update-kubeconfig --region $AWS_REGION --name $CLUSTER_NAME

# Verify connection to production cluster
kubectl get nodes
```

### 3. Deploy Database to EKS
```bash
# Return to project root
cd ..

# Deploy PostgreSQL to production cluster
make prod-k8s-setup

# Verify PostgreSQL is running
kubectl get pods -n investorcenter
```

### 4. Import Initial Ticker Data
```bash
# Import 4,600+ stock tickers to production database
make db-import-prod

# Verify data import
kubectl port-forward -n investorcenter svc/postgres-service 5433:5432 &
export PGPASSWORD="prod_investorcenter_456"
psql -h localhost -p 5433 -U investorcenter -d investorcenter_db -c "SELECT COUNT(*) FROM stocks;"
```

### 5. Deploy Automated Ticker Updates
```bash
# Deploy ticker update CronJob to production
make prod-deploy-cron
```

This will:
- Build Docker image with ticker update script
- Push image to AWS ECR repository
- Deploy Kubernetes CronJob for weekly updates
- Configure database connection and secrets

### 6. Verify CronJob Deployment
```bash
# Check CronJob status
make prod-cron-status

# Trigger manual test run
kubectl create job --from=cronjob/ticker-update test-prod-update -n investorcenter

# Monitor test execution
kubectl logs job/test-prod-update -n investorcenter -f

# View recent logs
make prod-cron-logs
```

## Production Configuration

### ECR Repositories Created
- `investorcenter/app` - Frontend application
- `investorcenter/backend` - Go API backend
- `investorcenter/ticker-updater` - Ticker update CronJob

### Kubernetes Resources
- **Namespace**: `investorcenter`
- **PostgreSQL**: Deployment with persistent storage
- **CronJob**: Weekly ticker updates (Sunday 2 AM UTC)
- **Secrets**: Database credentials
- **Services**: Database service for internal communication

### Database Configuration
- **Host**: `postgres-service` (internal K8s service)
- **Port**: `5432`
- **Database**: `investorcenter_db`
- **User**: `investorcenter`
- **SSL**: Disabled for internal cluster communication

## Monitoring Production CronJob

### Check Status
```bash
# CronJob schedule and recent runs
make prod-cron-status

# Recent execution logs
make prod-cron-logs

# Detailed CronJob information
kubectl describe cronjob ticker-update -n investorcenter
```

### Manual Operations
```bash
# Trigger immediate update
kubectl create job --from=cronjob/ticker-update manual-update-$(date +%s) -n investorcenter

# Monitor specific job
kubectl logs job/manual-update-XXXXX -n investorcenter

# Delete old test jobs
kubectl delete jobs -n investorcenter -l app=ticker-update --field-selector status.phase=Succeeded
```

## Expected Behavior

### Weekly Automatic Updates
```
Schedule: Every Sunday at 2 AM UTC
Duration: 1-5 seconds (normal), 30-60 seconds (with new IPOs)
Result: New tickers added to database, existing tickers skipped
```

### Normal Run (No New IPOs)
```
INFO - Downloaded 6916 raw ticker records
INFO - Filtered to 4643 valid tickers
INFO - Import completed: 0 inserted, 4643 skipped
INFO - Update completed in 1.5 seconds
```

### Run with New IPOs
```
INFO - Downloaded 6920 raw ticker records
INFO - Filtered to 4647 valid tickers
INFO - Import completed: 4 inserted, 4643 skipped
INFO - New companies: NEWCO, STARTUP, GROWTH, TECH
```

## Troubleshooting

### CronJob Not Running
1. **Check cluster resources**: `kubectl top nodes`
2. **Verify image pull**: `kubectl describe job <job-name> -n investorcenter`
3. **Check events**: `kubectl get events -n investorcenter --sort-by=.metadata.creationTimestamp`

### Database Connection Issues
1. **Test connectivity**: Port-forward and manual connection test
2. **Check secrets**: `kubectl get secrets -n investorcenter`
3. **Network policies**: Verify pod-to-pod communication allowed

### Image Pull Failures
1. **ECR permissions**: Ensure EKS can pull from ECR
2. **Image exists**: Check ECR console for pushed images
3. **Registry URL**: Verify image URL in CronJob YAML

## Scaling and Updates

### Update CronJob Schedule
```bash
# Change to daily at 6 AM
kubectl patch cronjob ticker-update -n investorcenter -p '{"spec":{"schedule":"0 6 * * *"}}'

# Suspend temporarily
kubectl patch cronjob ticker-update -n investorcenter -p '{"spec":{"suspend":true}}'
```

### Update Docker Image
```bash
# Build new version
./scripts/build-ticker-updater.sh

# Tag and push to ECR
docker tag investorcenter/ticker-updater:latest $ECR_REGISTRY/investorcenter/ticker-updater:latest
docker push $ECR_REGISTRY/investorcenter/ticker-updater:latest

# CronJob will use new image on next run (no restart needed)
```

## Security Considerations

- **IAM Roles**: EKS service account with minimal ECR permissions
- **Network Security**: CronJob runs in private subnets
- **Secrets Management**: Database credentials in K8s secrets
- **Image Scanning**: ECR automatically scans for vulnerabilities
- **Non-root Containers**: All containers run as non-root user

## Cost Optimization

- **CronJob Resources**: Right-sized for 2-5 minute execution
- **ECR Lifecycle**: Automatically deletes old images
- **Job Cleanup**: Completed jobs auto-delete after 1 hour
- **Spot Instances**: Consider using spot instances for worker nodes

## Monitoring and Alerts

### CloudWatch Integration
- **EKS Logs**: Automatically sent to CloudWatch
- **Metrics**: Pod CPU/memory usage tracked
- **Events**: K8s events for troubleshooting

### Future Enhancements
- **Slack Alerts**: Notify on CronJob failures
- **Metrics Dashboard**: Grafana/Prometheus monitoring
- **Dead Letter Queue**: Handle persistent failures
