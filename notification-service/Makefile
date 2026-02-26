.PHONY: build docker-build push deploy test clean

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

# Deploy to Kubernetes
deploy: push
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
