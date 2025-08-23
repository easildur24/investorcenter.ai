# Deployment Guide

This guide walks you through deploying InvestorCenter.ai to AWS EKS step by step.

## Prerequisites Checklist

- [ ] AWS CLI installed and configured
- [ ] Docker installed and running
- [ ] kubectl installed
- [ ] Terraform installed
- [ ] Node.js 18+ installed
- [ ] Domain registered and Route53 hosted zone created

## Step-by-Step Deployment

### Step 1: Prepare Your Environment

1. **Configure AWS CLI**:
   ```bash
   aws configure
   # Enter your AWS Access Key ID, Secret Access Key, and region (us-east-1)
   ```

2. **Verify AWS access**:
   ```bash
   aws sts get-caller-identity
   ```

3. **Install dependencies**:
   ```bash
   npm install
   ```

### Step 2: Configure Your Domain

1. **Update domain in Terraform variables**:
   Edit `terraform/variables.tf` and change the default domain:
   ```hcl
   variable "domain_name" {
     description = "Domain name for the application"
     type        = string
     default     = "investorcenter.com"  # Change this to your domain
   }
   ```

2. **Ensure Route53 hosted zone exists**:
   - Go to AWS Route53 console
   - Verify your domain has a hosted zone
   - Note the hosted zone ID

### Step 3: Deploy Infrastructure

1. **Run the infrastructure setup script**:
   ```bash
   ./scripts/setup-infrastructure.sh
   ```

   This will:
   - Initialize Terraform
   - Show you the planned changes
   - Ask for confirmation before creating resources
   - Create EKS cluster, VPC, ALB, ECR, and Route53 records
   - Update your kubeconfig

2. **Wait for infrastructure to be ready** (10-15 minutes):
   ```bash
   # Check EKS cluster status
   aws eks describe-cluster --name investorcenter-eks --query cluster.status

   # Check node group status
   aws eks describe-nodegroup --cluster-name investorcenter-eks --nodegroup-name investorcenter-eks-node-group --query nodegroup.status
   ```

### Step 4: Build and Push Application

1. **Build and push Docker image**:
   ```bash
   ./scripts/build-and-push.sh
   ```

   This will:
   - Build the Docker image
   - Login to ECR
   - Tag and push the image

2. **Update deployment with correct image URI**:
   ```bash
   # Get your account ID
   AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
   
   # Update the deployment YAML
   sed -i "s|investorcenter/app:latest|${AWS_ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com/investorcenter/app:latest|g" k8s/deployment.yaml
   ```

### Step 5: Deploy Application to Kubernetes

1. **Deploy the application**:
   ```bash
   ./scripts/deploy.sh
   ```

2. **Wait for pods to be ready**:
   ```bash
   kubectl get pods -n investorcenter -w
   ```

### Step 6: Configure SSL Certificate

1. **Get the certificate ARN from Terraform output**:
   ```bash
   cd terraform
   terraform output
   ```

2. **Update ingress with certificate ARN**:
   Edit `k8s/ingress.yaml` and replace `CERTIFICATE_ID` with your actual certificate ARN:
   ```yaml
   alb.ingress.kubernetes.io/certificate-arn: arn:aws:acm:us-east-1:YOUR_ACCOUNT_ID:certificate/YOUR_CERTIFICATE_ID
   ```

3. **Apply the updated ingress**:
   ```bash
   kubectl apply -f k8s/ingress.yaml
   ```

### Step 7: Verify Deployment

1. **Check all resources are running**:
   ```bash
   kubectl get all -n investorcenter
   ```

2. **Check ingress status**:
   ```bash
   kubectl describe ingress investorcenter-ingress -n investorcenter
   ```

3. **Get the load balancer DNS name**:
   ```bash
   kubectl get ingress investorcenter-ingress -n investorcenter -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'
   ```

4. **Test the application**:
   ```bash
   # Test with curl (may take a few minutes for DNS to propagate)
   curl -I https://investorcenter.com
   ```

### Step 8: Final Verification

1. **Visit your domain**: https://investorcenter.com
2. **Check SSL certificate**: Ensure the green lock appears in your browser
3. **Test responsiveness**: Check the site works on mobile and desktop

## Troubleshooting

### Common Issues and Solutions

#### 1. Certificate Validation Stuck

**Problem**: ACM certificate stuck in "Pending validation"

**Solution**:
```bash
# Check Route53 records
aws route53 list-resource-record-sets --hosted-zone-id YOUR_ZONE_ID

# Manually add CNAME records if needed
aws acm describe-certificate --certificate-arn YOUR_CERT_ARN
```

#### 2. Pods Not Starting

**Problem**: Pods stuck in `ImagePullBackOff`

**Solution**:
```bash
# Check if image exists in ECR
aws ecr describe-images --repository-name investorcenter/app

# Update deployment with correct image URI
kubectl set image deployment/investorcenter-app investorcenter-app=YOUR_ECR_URI -n investorcenter
```

#### 3. Load Balancer Not Ready

**Problem**: Ingress shows no ADDRESS

**Solution**:
```bash
# Check ALB controller logs
kubectl logs -n kube-system deployment/aws-load-balancer-controller

# Verify ALB controller is running
kubectl get deployment -n kube-system aws-load-balancer-controller
```

#### 4. Domain Not Resolving

**Problem**: Domain doesn't point to load balancer

**Solution**:
```bash
# Check Route53 records
aws route53 list-resource-record-sets --hosted-zone-id YOUR_ZONE_ID

# Check if ALB is created
aws elbv2 describe-load-balancers --names investorcenter-eks-alb
```

### Useful Commands

```bash
# View all resources
kubectl get all -n investorcenter

# Check pod logs
kubectl logs -f deployment/investorcenter-app -n investorcenter

# Describe ingress for troubleshooting
kubectl describe ingress investorcenter-ingress -n investorcenter

# Check ALB controller
kubectl get deployment -n kube-system aws-load-balancer-controller

# Scale deployment
kubectl scale deployment investorcenter-app --replicas=3 -n investorcenter

# Update image
kubectl set image deployment/investorcenter-app investorcenter-app=NEW_IMAGE_URI -n investorcenter
```

## Cleanup

To destroy all resources:

```bash
# Delete Kubernetes resources
kubectl delete namespace investorcenter

# Destroy Terraform infrastructure
cd terraform
terraform destroy
```

**Warning**: This will delete all resources and cannot be undone!

## Next Steps

After successful deployment:

1. **Set up monitoring**: Consider adding CloudWatch, Prometheus, or Grafana
2. **Configure CI/CD**: Set up GitHub Actions or AWS CodePipeline
3. **Add database**: Integrate RDS or DynamoDB for data persistence
4. **Implement caching**: Add Redis or ElastiCache
5. **Set up backups**: Configure automated backups for critical data

## Support

If you encounter issues:

1. Check the troubleshooting section above
2. Review AWS CloudFormation events in the console
3. Check EKS cluster logs in CloudWatch
4. Verify all prerequisites are met
