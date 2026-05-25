package audit_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"team-management/internal/audit"
	"team-management/internal/models"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type mockAuditRepo struct {
	mu   sync.Mutex
	logs []*models.AuditLog
}

func (m *mockAuditRepo) CreateLog(ctx context.Context, logEntry *models.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, logEntry)
	return nil
}

func (m *mockAuditRepo) getLogs() []*models.AuditLog {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.logs
}

func TestStartAuditWorker(t *testing.T) {
	// 1. Start in-memory Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	// 2. Setup Mock Repo and Context
	mockRepo := &mockAuditRepo{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. Start the worker in a goroutine
	workerDone := make(chan struct{})
	go func() {
		audit.StartAuditWorker(ctx, client, mockRepo)
		close(workerDone)
	}()

	// Give the worker a tiny fraction of a second to subscribe
	time.Sleep(50 * time.Millisecond)

	// 4. Publish a raw JSON string to simulate an incoming event
	validJSON := `{"user_id": 99, "action": "DELETE", "entityType": "POST"}`
	err = client.Publish(context.Background(), "audit_events", validJSON).Err()
	require.NoError(t, err)

	// 5. Verify Persistence: wait until the repo has 1 log
	// We use require.Eventually to avoid flaky tests with static time.Sleep
	require.Eventually(t, func() bool {
		return len(mockRepo.getLogs()) == 1
	}, 2*time.Second, 10*time.Millisecond, "Worker failed to persist the message to DB")

	logs := mockRepo.getLogs()
	require.Equal(t, "DELETE", logs[0].Action)
	require.Equal(t, int64(99), *logs[0].UserID)

	// 6. Verify Closure: Close the client to drop the pubsub channel
	client.Close()

	// 7. Verify the worker exits cleanly instead of panicking
	select {
	case <-workerDone:
		// Success! The worker exited gracefully.
	case <-time.After(2 * time.Second):
		t.Fatal("Worker did not exit cleanly upon Redis channel closure (possible goroutine leak)")
	}
}

func TestPublishEvent(t *testing.T) {
	// 1. Start in-memory Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()

	// 2. Setup a subscriber to catch the published event
	pubsub := client.Subscribe(context.Background(), "audit_events")
	defer pubsub.Close()

	// Wait for subscription to be confirmed before publishing
	_, err = pubsub.Receive(context.Background())
	require.NoError(t, err)

	// 3. Call the function under test
	userID := int64(42)
	entityID := int64(100)
	details := map[string]any{"ip": "192.168.1.1"}

	audit.PublishEvent(context.Background(), client, &userID, "UPDATE", "USER", &entityID, details)

	// 4. Receive the message from Redis and assert the payload shape
	msg, err := pubsub.ReceiveMessage(context.Background())
	require.NoError(t, err)

	var payload models.AuditLog
	err = json.Unmarshal([]byte(msg.Payload), &payload)
	require.NoError(t, err)

	// 5. Verify the fields
	require.Equal(t, int64(42), *payload.UserID)
	require.Equal(t, "UPDATE", payload.Action)
	require.Equal(t, "USER", *payload.EntityType)
	require.Equal(t, int64(100), *payload.EntityID)

	// Verify details map was marshaled correctly
	var parsedDetails map[string]any
	err = json.Unmarshal(payload.Details, &parsedDetails)
	require.NoError(t, err)
	require.Equal(t, "192.168.1.1", parsedDetails["ip"])
}
