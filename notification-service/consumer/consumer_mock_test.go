package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// ---------------------------------------------------------------------------
// Mock SQS client
// ---------------------------------------------------------------------------

type mockSQSClient struct {
	receiveFn func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	deleteFn  func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

func (m *mockSQSClient) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	if m.receiveFn != nil {
		return m.receiveFn(ctx, params, optFns...)
	}
	return &sqs.ReceiveMessageOutput{}, nil
}

func (m *mockSQSClient) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, params, optFns...)
	}
	return &sqs.DeleteMessageOutput{}, nil
}

// ---------------------------------------------------------------------------
// Helper: build a Consumer wired to a mock
// ---------------------------------------------------------------------------

const testQueueURL = "https://sqs.us-east-1.amazonaws.com/123456789/test-queue"

func newTestConsumer(mock *mockSQSClient) *Consumer {
	c := &Consumer{
		client:      mock,
		queueURL:    testQueueURL,
		maxMessages: 1,
		sleepFn:     func(d time.Duration) {}, // no-op
	}
	c.healthy.Store(true)
	return c
}

// snsEnvelope returns a valid SNS-wrapped JSON body string.
func snsEnvelope(innerPayload string) *string {
	env := struct {
		Type    string `json:"Type"`
		Message string `json:"Message"`
	}{
		Type:    "Notification",
		Message: innerPayload,
	}
	b, _ := json.Marshal(env)
	s := string(b)
	return &s
}

// ---------------------------------------------------------------------------
// IsHealthy tests
// ---------------------------------------------------------------------------

func TestIsHealthy_InitiallyTrue(t *testing.T) {
	c := newTestConsumer(&mockSQSClient{})
	if !c.IsHealthy() {
		t.Fatal("expected consumer to be healthy after construction")
	}
}

func TestIsHealthy_ReturnsFalseAfterSettingFalse(t *testing.T) {
	c := newTestConsumer(&mockSQSClient{})
	c.healthy.Store(false)
	if c.IsHealthy() {
		t.Fatal("expected consumer to be unhealthy after setting healthy to false")
	}
}

// ---------------------------------------------------------------------------
// poll tests
// ---------------------------------------------------------------------------

func TestPoll_SuccessfulReceiveAndDelete(t *testing.T) {
	innerPayload := `{"event":"alert","ticker":"AAPL"}`
	var deletedHandle *string
	handlerCalled := false

	mock := &mockSQSClient{
		receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
			return &sqs.ReceiveMessageOutput{
				Messages: []sqstypes.Message{
					{
						Body:          snsEnvelope(innerPayload),
						ReceiptHandle: aws.String("receipt-1"),
					},
				},
			}, nil
		},
		deleteFn: func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
			deletedHandle = params.ReceiptHandle
			return &sqs.DeleteMessageOutput{}, nil
		},
	}

	c := newTestConsumer(mock)

	handler := func(msg []byte) error {
		handlerCalled = true
		if string(msg) != innerPayload {
			t.Errorf("handler got %q, want %q", string(msg), innerPayload)
		}
		return nil
	}

	c.poll(context.Background(), handler)

	if !handlerCalled {
		t.Fatal("expected handler to be called")
	}
	if deletedHandle == nil || *deletedHandle != "receipt-1" {
		t.Fatalf("expected receipt handle 'receipt-1' to be deleted, got %v", deletedHandle)
	}
}

func TestPoll_ZeroMessages_NoHandlerCall(t *testing.T) {
	mock := &mockSQSClient{
		receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
			return &sqs.ReceiveMessageOutput{
				Messages: []sqstypes.Message{},
			}, nil
		},
	}

	c := newTestConsumer(mock)
	handlerCalled := false

	handler := func(msg []byte) error {
		handlerCalled = true
		return nil
	}

	c.poll(context.Background(), handler)

	if handlerCalled {
		t.Fatal("handler should not be called when there are zero messages")
	}
}

func TestPoll_ReceiveError_IncrementsFailsStaysHealthy(t *testing.T) {
	mock := &mockSQSClient{
		receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
			return nil, errors.New("network timeout")
		},
	}

	c := newTestConsumer(mock)

	c.poll(context.Background(), func(msg []byte) error { return nil })

	fails := atomic.LoadInt32(&c.consecutiveFails)
	if fails != 1 {
		t.Fatalf("expected consecutiveFails=1, got %d", fails)
	}
	if !c.IsHealthy() {
		t.Fatal("expected consumer to remain healthy after 1 failure")
	}
}

func TestPoll_ReceiveError_ThreeTimesMarksUnhealthy(t *testing.T) {
	mock := &mockSQSClient{
		receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
			return nil, errors.New("persistent error")
		},
	}

	c := newTestConsumer(mock)
	handler := func(msg []byte) error { return nil }

	// Fail maxConsecutiveFailures times (3)
	for i := 0; i < int(maxConsecutiveFailures); i++ {
		c.poll(context.Background(), handler)
	}

	fails := atomic.LoadInt32(&c.consecutiveFails)
	if fails != maxConsecutiveFailures {
		t.Fatalf("expected consecutiveFails=%d, got %d", maxConsecutiveFailures, fails)
	}
	if c.IsHealthy() {
		t.Fatal("expected consumer to be unhealthy after 3 consecutive failures")
	}
}

func TestPoll_SuccessAfterFailures_ResetsAndMarksHealthy(t *testing.T) {
	callCount := 0

	mock := &mockSQSClient{
		receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
			callCount++
			if callCount <= 2 {
				return nil, errors.New("transient error")
			}
			// Third call succeeds with no messages
			return &sqs.ReceiveMessageOutput{Messages: []sqstypes.Message{}}, nil
		},
	}

	c := newTestConsumer(mock)
	handler := func(msg []byte) error { return nil }

	// Two failures
	c.poll(context.Background(), handler)
	c.poll(context.Background(), handler)

	if atomic.LoadInt32(&c.consecutiveFails) != 2 {
		t.Fatalf("expected 2 consecutive failures, got %d", atomic.LoadInt32(&c.consecutiveFails))
	}

	// Third call succeeds
	c.poll(context.Background(), handler)

	if atomic.LoadInt32(&c.consecutiveFails) != 0 {
		t.Fatalf("expected consecutiveFails reset to 0, got %d", atomic.LoadInt32(&c.consecutiveFails))
	}
	if !c.IsHealthy() {
		t.Fatal("expected consumer to be healthy after successful poll")
	}
}

func TestPoll_HandlerError_MessageNotDeleted(t *testing.T) {
	deleteWasCalled := false

	mock := &mockSQSClient{
		receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
			return &sqs.ReceiveMessageOutput{
				Messages: []sqstypes.Message{
					{
						Body:          snsEnvelope(`{"event":"test"}`),
						ReceiptHandle: aws.String("receipt-2"),
					},
				},
			}, nil
		},
		deleteFn: func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
			deleteWasCalled = true
			return &sqs.DeleteMessageOutput{}, nil
		},
	}

	c := newTestConsumer(mock)

	handler := func(msg []byte) error {
		return errors.New("processing failed")
	}

	c.poll(context.Background(), handler)

	if deleteWasCalled {
		t.Fatal("message should NOT be deleted when handler returns an error")
	}
}

func TestPoll_NilBody_DeletesMessageAnyway(t *testing.T) {
	deleteWasCalled := false

	mock := &mockSQSClient{
		receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
			return &sqs.ReceiveMessageOutput{
				Messages: []sqstypes.Message{
					{
						Body:          nil, // nil body triggers extractSNSPayload error
						ReceiptHandle: aws.String("receipt-bad"),
					},
				},
			}, nil
		},
		deleteFn: func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
			deleteWasCalled = true
			if params.ReceiptHandle == nil || *params.ReceiptHandle != "receipt-bad" {
				t.Errorf("expected receipt handle 'receipt-bad', got %v", params.ReceiptHandle)
			}
			return &sqs.DeleteMessageOutput{}, nil
		},
	}

	c := newTestConsumer(mock)
	handlerCalled := false

	handler := func(msg []byte) error {
		handlerCalled = true
		return nil
	}

	c.poll(context.Background(), handler)

	if handlerCalled {
		t.Fatal("handler should not be called when extractSNSPayload fails")
	}
	if !deleteWasCalled {
		t.Fatal("message with nil body should still be deleted (skip bad message)")
	}
}

func TestPoll_ContextCancelled_ReturnsWithoutSleep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	sleepCalled := false

	mock := &mockSQSClient{
		receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
			return nil, errors.New("some error")
		},
	}

	c := &Consumer{
		client:      mock,
		queueURL:    testQueueURL,
		maxMessages: 1,
		sleepFn: func(d time.Duration) {
			sleepCalled = true
		},
	}
	c.healthy.Store(true)

	c.poll(ctx, func(msg []byte) error { return nil })

	if sleepCalled {
		t.Fatal("sleep should not be called when context is cancelled")
	}
	// consecutiveFails should NOT be incremented on context cancellation
	if atomic.LoadInt32(&c.consecutiveFails) != 0 {
		t.Fatalf("expected consecutiveFails=0 on context cancel, got %d", atomic.LoadInt32(&c.consecutiveFails))
	}
}

// ---------------------------------------------------------------------------
// Start tests
// ---------------------------------------------------------------------------

func TestStart_ContextCancelled_StopsAndMarksUnhealthy(t *testing.T) {
	mock := &mockSQSClient{
		receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
			// Should not be called because context is already cancelled
			return &sqs.ReceiveMessageOutput{}, nil
		},
	}

	c := newTestConsumer(mock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before Start

	done := make(chan struct{})
	go func() {
		c.Start(ctx, func(msg []byte) error { return nil })
		close(done)
	}()

	select {
	case <-done:
		// Start returned as expected
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}

	if c.IsHealthy() {
		t.Fatal("expected consumer to be unhealthy after Start exits")
	}
}

// ---------------------------------------------------------------------------
// deleteMessage tests
// ---------------------------------------------------------------------------

func TestDeleteMessage_Success(t *testing.T) {
	deleteCalled := false

	mock := &mockSQSClient{
		deleteFn: func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
			deleteCalled = true
			if params.QueueUrl == nil || *params.QueueUrl != testQueueURL {
				t.Errorf("expected queue URL %q, got %v", testQueueURL, params.QueueUrl)
			}
			if params.ReceiptHandle == nil || *params.ReceiptHandle != "receipt-ok" {
				t.Errorf("expected receipt handle 'receipt-ok', got %v", params.ReceiptHandle)
			}
			return &sqs.DeleteMessageOutput{}, nil
		},
	}

	c := newTestConsumer(mock)
	c.deleteMessage(context.Background(), aws.String("receipt-ok"))

	if !deleteCalled {
		t.Fatal("expected DeleteMessage to be called")
	}
}

func TestDeleteMessage_Error_NoPanic(t *testing.T) {
	mock := &mockSQSClient{
		deleteFn: func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
			return nil, errors.New("access denied")
		},
	}

	c := newTestConsumer(mock)

	// Should log the error but not panic
	c.deleteMessage(context.Background(), aws.String("receipt-fail"))
	// If we reach here without panicking, the test passes.
}
