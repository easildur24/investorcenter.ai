package services

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

var (
	snsClient     *sns.Client
	snsClientOnce sync.Once
)

// GetSNSClient returns a singleton AWS SNS client.
// Initializes on first call using default AWS credentials (IRSA in K8s, env vars locally).
func GetSNSClient() *sns.Client {
	snsClientOnce.Do(func() {
		region := os.Getenv("AWS_REGION")
		if region == "" {
			region = "us-east-1"
		}

		cfg, err := config.LoadDefaultConfig(context.Background(),
			config.WithRegion(region),
		)
		if err != nil {
			log.Printf("⚠️ Failed to load AWS config for SNS: %v (SNS publishing disabled)", err)
			return
		}

		snsClient = sns.NewFromConfig(cfg)
		log.Println("✅ SNS client initialized")
	})
	return snsClient
}
