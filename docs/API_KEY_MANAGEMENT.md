# API Key Management Guide

## 🔒 Security Best Practices

This document outlines how to securely manage API keys for the InvestorCenter.ai application.

## Current API Keys Required

1. **Alpha Vantage API** - For real-time stock quotes
2. **Finnhub API** - For additional market data (future implementation)

## ⚠️ IMPORTANT: Never Commit API Keys

The API key `6MVGMJ4FCAGF2ONU` or any other API key should **NEVER** be committed to the repository.

## Setup Instructions

### 1. Local Development

1. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

2. Get your free API keys:
   - **Alpha Vantage**: https://www.alphavantage.co/support/#api-key
   - **Finnhub**: https://finnhub.io/register

3. Edit `.env` and add your keys:
   ```bash
   ALPHA_VANTAGE_API_KEY=your_actual_api_key_here
   FINNHUB_API_KEY=your_finnhub_key_here
   ```

4. The `.env` file is gitignored and will never be committed.

### 2. Production (AWS EKS)

For production deployments, use Kubernetes secrets:

```bash
# Create the secret
kubectl create secret generic api-keys \
  --from-literal=alpha-vantage-api-key='your_api_key' \
  --from-literal=finnhub-api-key='your_finnhub_key' \
  -n production

# Verify the secret was created
kubectl get secrets -n production
```

### 3. CI/CD (GitHub Actions)

Add secrets to your GitHub repository:

1. Go to Settings → Secrets and variables → Actions
2. Add the following secrets:
   - `ALPHA_VANTAGE_API_KEY`
   - `FINNHUB_API_KEY`

## Security Checklist

- [ ] Never hardcode API keys in source code
- [ ] Always use environment variables or secrets management
- [ ] Rotate API keys regularly
- [ ] Use different keys for development and production
- [ ] Monitor API key usage for anomalies
- [ ] Run the security check script before commits:
  ```bash
  ./scripts/check_secrets.sh
  ```

## Code Examples

### Go (Backend)

```go
import "os"

// Get API key from environment
apiKey := os.Getenv("ALPHA_VANTAGE_API_KEY")
if apiKey == "" {
    log.Fatal("ALPHA_VANTAGE_API_KEY not set")
}

// Use the client with config
client := alphavantage.NewClient(apiKey)
```

### Docker

```dockerfile
# Never put API keys in Dockerfile
# Pass them at runtime instead:
docker run -e ALPHA_VANTAGE_API_KEY=$ALPHA_VANTAGE_API_KEY myapp
```

### Kubernetes

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: api
    env:
    - name: ALPHA_VANTAGE_API_KEY
      valueFrom:
        secretKeyRef:
          name: api-keys
          key: alpha-vantage-api-key
```

## Rate Limits

### Alpha Vantage Free Tier
- 5 API requests per minute
- 500 requests per day

### Managing Rate Limits

The Alpha Vantage client includes built-in rate limiting:
- Automatic request throttling
- Circuit breaker for API failures
- Graceful degradation

## Monitoring & Alerts

1. **Track API usage**: Monitor your daily/minute quotas
2. **Set up alerts**: Get notified before hitting limits
3. **Use caching**: Redis cache reduces API calls
4. **Implement fallbacks**: Use cached data when API is unavailable

## Emergency Procedures

If an API key is exposed:

1. **Immediately rotate the key**:
   - Generate a new key from the provider
   - Update all environments with the new key
   - Revoke the old key

2. **Check for unauthorized usage**:
   - Review API logs for suspicious activity
   - Check your billing/usage dashboard

3. **Update secrets**:
   ```bash
   # Local
   Update .env file
   
   # Kubernetes
   kubectl delete secret api-keys -n production
   kubectl create secret generic api-keys --from-literal=alpha-vantage-api-key='new_key'
   
   # GitHub Actions
   Update repository secrets
   ```

4. **Run security scan**:
   ```bash
   ./scripts/check_secrets.sh
   git log -p | grep -E "6MVGMJ4FCAGF2ONU|api[_-]?key"
   ```

## Additional Resources

- [Alpha Vantage Documentation](https://www.alphavantage.co/documentation/)
- [Kubernetes Secrets Best Practices](https://kubernetes.io/docs/concepts/security/secrets-good-practices/)
- [12 Factor App - Config](https://12factor.net/config)
- [OWASP API Security](https://owasp.org/www-project-api-security/)