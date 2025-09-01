# InvestorCenter.ai

A professional financial data and analytics platform similar to YCharts, built with Next.js and deployed on AWS EKS.

## Features

- **Modern Frontend**: Next.js 14 with React 18, TypeScript, and Tailwind CSS
- **High-Performance Backend**: Go API server with Gin framework
- **Real-time Data**: Live market data and financial analytics
- **Database**: PostgreSQL for persistent data, Redis for caching
- **Professional UI**: Clean, responsive design similar to YCharts
- **Cloud-Native**: Containerized with Docker and deployed on AWS EKS
- **Scalable Infrastructure**: Auto-scaling Kubernetes deployment
- **SSL/TLS**: Secure HTTPS with AWS Certificate Manager
- **Domain Integration**: Custom domain routing with Route53
- **Automated Data**: Direct ticker import from exchanges with periodic updates

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────────────────┐
│   Route53       │    │  Application     │    │         EKS Cluster             │
│   DNS           │───▶│  Load Balancer   │───▶│  ┌─────────────┐ ┌─────────────┐ │
│                 │    │  (ALB)           │    │  │  Next.js    │ │   Go API    │ │
└─────────────────┘    └──────────────────┘    │  │  Frontend   │ │   Backend   │ │
                                               │  └─────────────┘ └─────────────┘ │
                                               │  ┌─────────────┐ ┌─────────────┐ │
                                               │  │ PostgreSQL  │ │   Redis     │ │
                                               │  │  Database   │ │   Cache     │ │
                                               │  └─────────────┘ └─────────────┘ │
                                               └─────────────────────────────────┘
                                                         │
                                                         ▼
                                               ┌─────────────────┐
                                               │  ECR Registry   │
                                               │  (Docker Images)│
                                               └─────────────────┘
```

## 📋 Prerequisites

Before you begin, ensure you have the following installed:

- [Node.js](https://nodejs.org/) (v18 or later)
- [Docker](https://www.docker.com/)
- [AWS CLI](https://aws.amazon.com/cli/) (configured with appropriate permissions)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Terraform](https://www.terraform.io/) (v1.0 or later)

### AWS Permissions Required

Your AWS user/role needs the following permissions:
- EKS full access
- EC2 full access
- VPC full access
- IAM full access
- Route53 full access
- Certificate Manager full access
- Elastic Load Balancing full access
- ECR full access

## Quick Start

### 1. Clone and Setup

```bash
git clone <your-repo>
cd investorcenter.ai
npm install
```

### 2. Configure Domain

Update the domain configuration in `terraform/variables.tf`:

```hcl
variable "domain_name" {
  description = "Domain name for the application"
  type        = string
  default     = "investorcenter.com"  # Change to your domain
}
```

### 3. Setup AWS Infrastructure

```bash
# Make scripts executable
chmod +x scripts/*.sh

# Setup infrastructure
./scripts/setup-infrastructure.sh
```

This will create:
- EKS cluster with worker nodes
- VPC with public/private subnets
- Application Load Balancer
- ECR repository
- Route53 records
- SSL certificate

### 4. Build and Deploy Application

```bash
# Build and push Docker image
./scripts/build-and-push.sh

# Deploy to Kubernetes
./scripts/deploy.sh
```

### 5. Update Kubernetes Deployment

After the ECR image is pushed, update the deployment to use your specific image:

```bash
# Get your ECR URI
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ECR_URI="${AWS_ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com/investorcenter/app:latest"

# Update the deployment
kubectl set image deployment/investorcenter-app investorcenter-app=${ECR_URI} -n investorcenter
```

## 📁 Project Structure

```
investorcenter.ai/
├── app/                    # Next.js app directory
│   ├── globals.css        # Global styles
│   ├── layout.tsx         # Root layout
│   └── page.tsx           # Home page
├── backend/               # Go API backend
│   ├── main.go           # Main API server
│   ├── go.mod            # Go dependencies
│   ├── Dockerfile        # Backend container
│   └── env.example       # Environment variables
├── components/            # React components
│   └── MarketOverview.tsx # Market data component
├── lib/                   # Utility libraries
│   └── api.ts            # API client
├── k8s/                   # Kubernetes manifests
│   ├── namespace.yaml     # Namespace definition
│   ├── deployment.yaml    # Frontend deployment
│   ├── service.yaml       # Frontend service
│   ├── backend-deployment.yaml # Backend deployment
│   ├── backend-service.yaml    # Backend service
│   ├── postgres-deployment.yaml # Database
│   ├── redis-deployment.yaml   # Cache
│   ├── secrets.yaml       # Kubernetes secrets
│   └── ingress.yaml       # ALB ingress
├── scripts/               # Deployment scripts
│   ├── setup-infrastructure.sh
│   ├── build-and-push.sh
│   ├── build-and-push-backend.sh
│   ├── deploy.sh
│   └── complete-setup.sh
├── terraform/             # Infrastructure as Code
│   ├── main.tf           # Main configuration
│   ├── variables.tf      # Variables
│   ├── vpc.tf           # VPC resources
│   ├── eks.tf           # EKS cluster
│   ├── alb-controller.tf # Load balancer controller
│   ├── ecr.tf           # Container registry
│   ├── route53.tf       # DNS and certificates
│   └── outputs.tf       # Output values
├── Dockerfile            # Frontend container
├── next.config.js       # Next.js configuration
└── package.json         # Frontend dependencies
```

## 🔧 Configuration

### Environment Variables

The application supports the following environment variables:

- `NODE_ENV`: Environment (production/development)
- `PORT`: Application port (default: 3000)

### Kubernetes Configuration

Key configurations in `k8s/deployment.yaml`:

```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "250m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### Terraform Variables

Customize your infrastructure in `terraform/variables.tf`:

- `aws_region`: AWS region (default: us-east-1)
- `cluster_version`: Kubernetes version (default: 1.28)
- `node_instance_type`: EC2 instance type (default: t3.medium)
- `min_size`: Minimum nodes (default: 1)
- `max_size`: Maximum nodes (default: 3)
- `desired_size`: Desired nodes (default: 2)

## 🚀 Deployment Process

### Manual Deployment Steps

1. **Infrastructure Setup**:
   ```bash
   cd terraform
   terraform init
   terraform plan
   terraform apply
   ```

2. **Build Application**:
   ```bash
   docker build -t investorcenter/app:latest .
   ```

3. **Push to ECR**:
   ```bash
   aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin <account-id>.dkr.ecr.us-east-1.amazonaws.com
   docker tag investorcenter/app:latest <account-id>.dkr.ecr.us-east-1.amazonaws.com/investorcenter/app:latest
   docker push <account-id>.dkr.ecr.us-east-1.amazonaws.com/investorcenter/app:latest
   ```

4. **Deploy to Kubernetes**:
   ```bash
   kubectl apply -f k8s/
   ```

### Automated Deployment

Use the provided scripts for automated deployment:

```bash
./scripts/setup-infrastructure.sh  # One-time setup
./scripts/build-and-push.sh       # Build and push image
./scripts/deploy.sh               # Deploy to Kubernetes
```

## 🔍 Monitoring and Troubleshooting

### Check Deployment Status

```bash
# Check pods
kubectl get pods -n investorcenter

# Check services
kubectl get services -n investorcenter

# Check ingress
kubectl get ingress -n investorcenter

# View logs
kubectl logs -f deployment/investorcenter-app -n investorcenter
```

### Common Issues

1. **Domain not resolving**: Check Route53 records and DNS propagation
2. **SSL certificate issues**: Verify ACM certificate validation
3. **Pod startup issues**: Check ECR image URI in deployment
4. **Load balancer not ready**: Wait for ALB controller to provision resources

## 🔒 Security Considerations

- SSL/TLS encryption with AWS Certificate Manager
- Private subnets for worker nodes
- Security groups with minimal required access
- ECR image scanning enabled
- Kubernetes RBAC (can be enhanced)

## 💰 Cost Optimization

Current setup costs approximately:
- EKS cluster: ~$73/month
- EC2 instances (2x t3.medium): ~$60/month
- ALB: ~$20/month
- NAT Gateway: ~$45/month
- **Total: ~$200/month**

To reduce costs:
- Use t3.small instances for development
- Implement cluster autoscaling
- Use Spot instances for non-production workloads

## 🔄 Updates and Maintenance

### Updating the Application

1. Make changes to your code
2. Run `./scripts/build-and-push.sh`
3. Run `./scripts/deploy.sh`

### Updating Infrastructure

1. Modify Terraform files
2. Run `terraform plan` to review changes
3. Run `terraform apply` to apply changes

### Scaling

To scale the application:

```bash
# Scale deployment
kubectl scale deployment investorcenter-app --replicas=5 -n investorcenter

# Or update the deployment YAML and apply
kubectl apply -f k8s/deployment.yaml
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License.

## 🆘 Support

For support and questions:
- Create an issue in the repository
- Check the troubleshooting section above
- Review AWS EKS documentation

---

**Note**: Remember to update the certificate ARN in `k8s/ingress.yaml` after the ACM certificate is created, and ensure your domain is properly configured in Route53.
