package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// Handler processes a raw message payload (the inner SNS Message body).
type Handler func(msg []byte) error

// Consumer long-polls an SQS queue and dispatches messages to a handler.
type Consumer struct {
	client   *sqs.Client
	queueURL string
	healthy  atomic.Bool
}

// New creates an SQS consumer for the given queue URL and AWS region.
func New(queueURL, region string) (*Consumer, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	c := &Consumer{
		client:   sqs.NewFromConfig(cfg),
		queueURL: queueURL,
	}
	c.healthy.Store(true)
	return c, nil
}

// IsHealthy returns whether the consumer is actively polling.
func (c *Consumer) IsHealthy() bool {
	return c.healthy.Load()
}

// Start begins long-polling the SQS queue. Blocks until ctx is cancelled.
// Each message is passed to handler; on success the message is deleted.
// On handler error the message stays in the queue and will be retried
// after the visibility timeout (30s), up to maxReceiveCount (3) times
// before being sent to the DLQ.
func (c *Consumer) Start(ctx context.Context, handler Handler) {
	log.Printf("SQS consumer started — polling %s", c.queueURL)

	for {
		select {
		case <-ctx.Done():
			log.Println("SQS consumer stopped")
			c.healthy.Store(false)
			return
		default:
			c.poll(ctx, handler)
		}
	}
}

// poll performs a single long-poll receive and processes messages.
func (c *Consumer) poll(ctx context.Context, handler Handler) {
	output, err := c.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(c.queueURL),
		MaxNumberOfMessages: 1,  // One bulk price update at a time
		WaitTimeSeconds:     20, // Long polling — blocks up to 20s
		VisibilityTimeout:   30,
	})
	if err != nil {
		// Context cancelled is expected during shutdown
		if ctx.Err() != nil {
			return
		}
		log.Printf("SQS receive error: %v — retrying in 5s", err)
		c.healthy.Store(false)
		time.Sleep(5 * time.Second)
		c.healthy.Store(true)
		return
	}

	for _, msg := range output.Messages {
		// SNS wraps the original message in an envelope.
		// Extract the actual payload from the "Message" field.
		payload, err := extractSNSPayload(msg.Body)
		if err != nil {
			log.Printf("Failed to extract SNS payload: %v — skipping message", err)
			c.deleteMessage(ctx, msg.ReceiptHandle)
			continue
		}

		if err := handler(payload); err != nil {
			log.Printf("Handler error: %v — message will be retried", err)
			// Don't delete — message returns to queue after visibility timeout
			continue
		}

		// Success — delete the message
		c.deleteMessage(ctx, msg.ReceiptHandle)
	}
}

// deleteMessage removes a processed message from the queue.
func (c *Consumer) deleteMessage(ctx context.Context, receiptHandle *string) {
	_, err := c.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.queueURL),
		ReceiptHandle: receiptHandle,
	})
	if err != nil {
		log.Printf("Failed to delete SQS message: %v", err)
	}
}

// extractSNSPayload unwraps the SNS envelope to get the original message body.
// SNS wraps messages in a JSON envelope with Type, MessageId, Message, etc.
func extractSNSPayload(body *string) ([]byte, error) {
	if body == nil {
		return nil, fmt.Errorf("nil message body")
	}

	var envelope struct {
		Message string `json:"Message"`
		Type    string `json:"Type"`
	}
	if err := json.Unmarshal([]byte(*body), &envelope); err != nil {
		// If it's not an SNS envelope, treat the body as the raw message
		return []byte(*body), nil
	}

	if envelope.Message == "" {
		return []byte(*body), nil
	}

	return []byte(envelope.Message), nil
}
