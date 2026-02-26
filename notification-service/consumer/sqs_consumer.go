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
	client          *sqs.Client
	queueURL        string
	maxMessages     int32
	healthy         atomic.Bool
	consecutiveFails int32 // tracks consecutive SQS receive failures
}

// maxConsecutiveFailures is the number of consecutive SQS receive errors before
// the consumer marks itself unhealthy. This prevents transient errors from
// causing unnecessary K8s pod restarts.
const maxConsecutiveFailures = 3

// New creates an SQS consumer for the given queue URL and AWS region.
// maxMessages controls how many messages to receive per poll (1-10).
func New(queueURL, region string, maxMessages int32) (*Consumer, error) {
	if maxMessages < 1 || maxMessages > 10 {
		maxMessages = 1
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	c := &Consumer{
		client:      sqs.NewFromConfig(cfg),
		queueURL:    queueURL,
		maxMessages: maxMessages,
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
	log.Printf("SQS consumer started — polling %s (max %d messages/poll)", c.queueURL, c.maxMessages)

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
		MaxNumberOfMessages: c.maxMessages,
		WaitTimeSeconds:     20, // Long polling — blocks up to 20s
		VisibilityTimeout:   30,
	})
	if err != nil {
		// Context cancelled is expected during shutdown
		if ctx.Err() != nil {
			return
		}
		fails := atomic.AddInt32(&c.consecutiveFails, 1)
		log.Printf("SQS receive error (consecutive: %d): %v — retrying in 5s", fails, err)

		// Only mark unhealthy after multiple consecutive failures to avoid
		// K8s readiness probes restarting the pod on transient errors.
		if fails >= maxConsecutiveFailures {
			c.healthy.Store(false)
		}

		time.Sleep(5 * time.Second)
		return
	}

	// Reset consecutive failure counter on successful receive
	if atomic.LoadInt32(&c.consecutiveFails) > 0 {
		atomic.StoreInt32(&c.consecutiveFails, 0)
		c.healthy.Store(true)
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
		// If it's not valid JSON, treat the body as the raw message
		log.Printf("Warning: SQS message body is not valid JSON — treating as raw payload")
		return []byte(*body), nil
	}

	if envelope.Message == "" {
		// Valid JSON but no Message field — unexpected format.
		// Log a warning and fall back to raw body.
		if envelope.Type != "" {
			log.Printf("Warning: SNS envelope has Type=%q but empty Message field — using raw body", envelope.Type)
		}
		return []byte(*body), nil
	}

	return []byte(envelope.Message), nil
}
