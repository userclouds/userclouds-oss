package workerclient

import (
	"context"
	"sync"
	"testing"
	"time"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/ucerr"
	"userclouds.com/worker"
)

// TypeTest defines the test client type
const TypeTest Type = "test"

// TestClient is a no-op client for testing (functionality can be added later)
type TestClient struct {
	messagesLock sync.RWMutex
	messages     []worker.Message
}

// GetMessages returns the messages queued into the test client
func (tc *TestClient) GetMessages() []worker.Message {
	tc.messagesLock.RLock()
	msgs := make([]worker.Message, len(tc.messages))
	copy(msgs, tc.messages)
	defer tc.messagesLock.RUnlock()
	return msgs
}

// Send implements the workerclient.Client interface
func (tc *TestClient) Send(ctx context.Context, msg worker.Message) error {
	msg.SetSourceRegionIfNotSet()
	if err := msg.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	tc.messagesLock.Lock()
	defer tc.messagesLock.Unlock()
	tc.messages = append(tc.messages, msg)
	return nil
}

// NewTestClient returns a new TestClient
func NewTestClient() *TestClient {
	return &TestClient{messages: make([]worker.Message, 0)}
}

// WaitForMessages waits (up to a timeout, if specified) for the expected number of messages to be received
func (tc *TestClient) WaitForMessages(t *testing.T, expectedMessages int, timeout time.Duration) []worker.Message {
	if timeout > 0 {
		timeoutExpiration := time.Now().UTC().Add(timeout)
		for time.Now().UTC().Before(timeoutExpiration) {
			if len(tc.GetMessages()) >= expectedMessages {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	messages := tc.GetMessages()
	assert.Equal(t, len(messages), expectedMessages, assert.Must())
	return messages
}

func (tc *TestClient) String() string {
	return "test"
}
