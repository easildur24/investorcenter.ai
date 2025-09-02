# Local Development vs Production Environment

Clear separation between local development and production deployment.

## Local Development Environment

### Purpose
- **Fast iteration** and development
- **Testing** new features
- **Debugging** and troubleshooting
- **Code validation** before production

### Infrastructure
- **PostgreSQL**: Local Homebrew installation (localhost:5432)
- **Backend**: Go API running locally (localhost:8080)
- **Frontend**: Next.js dev server (localhost:3000)
- **Data Updates**: Manual only (`make db-import`)

### Setup Commands
```bash
make setup          # Complete local development setup
make dev            # Start development environment
make db-import      # Manual ticker data update
make db-status      # Check local database status
```

### Key Points
- ✅ **No automated jobs** - all updates are manual
- ✅ **Fast feedback** - immediate code changes
- ✅ **Isolated** - doesn't affect production
- ✅ **Complete control** - start/stop services as needed

## Production Environment  

### Purpose
- **Live application** serving real users
- **Automated operations** with minimal manual intervention
- **High availability** and reliability
- **Scalable** infrastructure

### Infrastructure
- **Kubernetes Cluster**: AWS EKS or other cloud provider
- **PostgreSQL**: RDS or K8s-managed with persistent storage
- **Backend**: Go API pods with auto-scaling
- **Frontend**: Static hosting or container deployment
- **Automated Updates**: Weekly CronJob for ticker data

### Deployment Commands
```bash
# ⚠️  PRODUCTION ONLY - verify cluster context first!
make prod-k8s-setup      # Deploy database infrastructure
make prod-deploy-cron    # Deploy automated ticker updates
make prod-cron-status    # Monitor CronJob status
```

### Key Points
- ✅ **Automated ticker updates** - weekly CronJob
- ✅ **High availability** - K8s manages failures and restarts
- ✅ **Monitoring** - comprehensive logging and status checks
- ✅ **Security** - proper secrets management and SSL

## Clear Separation

### What Runs Where

| Component | Local Development | Production |
|-----------|------------------|------------|
| **PostgreSQL** | Homebrew (localhost:5432) | K8s Pod or RDS |
| **Ticker Updates** | Manual (`make db-import`) | Automated CronJob |
| **API Server** | Local process | K8s Deployment |
| **Frontend** | Dev server | Static hosting/K8s |
| **Monitoring** | Manual checks | Automated alerts |

### Commands by Environment

| Operation | Local Command | Production Command |
|-----------|---------------|-------------------|
| **Setup** | `make setup` | `make prod-k8s-setup` |
| **Update Data** | `make db-import` | Automatic (CronJob) |
| **Check Status** | `make db-status` | `make prod-cron-status` |
| **View Logs** | Local files | `make prod-cron-logs` |

## Best Practices

### Local Development
- **Never run CronJobs locally** - use manual updates only
- **Use local PostgreSQL** - fast and isolated
- **Test thoroughly** before production deployment
- **Validate with `make check`** before pushing

### Production
- **Always verify cluster context** before deployment
- **Use container registry** - never rely on local images
- **Monitor CronJob status** regularly
- **Test with manual jobs** before relying on schedule

## Switching Between Environments

### To Local Development
```bash
# Ensure you're using local services
kubectl config use-context rancher-desktop  # or docker-desktop
make db-status  # Should show local PostgreSQL
```

### To Production
```bash
# Switch to production cluster
kubectl config use-context your-production-cluster
kubectl get nodes  # Verify you're on production
make prod-cron-status  # Check production services
```

## Safety Measures

### Prevent Accidental Production Changes
- Production commands have confirmation prompts
- Clear naming: `prod-*` prefix for all production commands
- Context verification in all production operations
- Separate documentation for production procedures

### Prevent Local Pollution
- CronJobs only in production configurations
- Local development uses different ports/services
- No automated jobs in local environment
- Clear separation in Makefile and documentation
