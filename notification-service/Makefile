.PHONY: build docker-build push deploy test clean validate-k8s

ECR_REPO = 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/notification-service
AWS_PROFILE = investorcenter

# Build the Go binary locally
build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .
	@echo "✅ Built notification-service binary"

# Build Docker image (linux/amd64 for EKS)
docker-build:
	docker build --platform linux/amd64 -t $(ECR_REPO):latest .
	@echo "✅ Docker image built"

# Push Docker image to ECR
push: docker-build
	aws ecr get-login-password --region us-east-1 --profile $(AWS_PROFILE) \
		| docker login --username AWS --password-stdin 360358043271.dkr.ecr.us-east-1.amazonaws.com
	docker push $(ECR_REPO):latest
	@echo "✅ Pushed to ECR"

# Validate K8s manifests don't contain REPLACE_ME placeholders
validate-k8s:
	@if grep -r "REPLACE_ME" k8s/ 2>/dev/null; then \
		echo "❌ ERROR: Found REPLACE_ME placeholders in k8s/ manifests."; \
		echo "   Set SQS_QUEUE_URL (from terraform output sqs_notification_alerts_url)"; \
		echo "   Set eks.amazonaws.com/role-arn (from terraform output notification_sqs_consumer_role_arn)"; \
		exit 1; \
	fi
	@echo "✅ K8s manifests validated — no REPLACE_ME placeholders found"

# Deploy to Kubernetes (validates manifests first)
deploy: push validate-k8s
	kubectl set image deployment/notification-service \
		notification-service=$(ECR_REPO):latest \
		-n investorcenter
	kubectl rollout restart deployment/notification-service -n investorcenter
	kubectl rollout status deployment/notification-service -n investorcenter --timeout=120s
	@echo "✅ Deployed to Kubernetes"

# Run tests
test:
	go test ./... -v

# Clean build artifacts
clean:
	rm -f main
