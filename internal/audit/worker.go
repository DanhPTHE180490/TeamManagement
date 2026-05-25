package audit

import (
	"context"
	"encoding/json"
	"log"

	"team-management/internal/models"

	"github.com/redis/go-redis/v9"
)

func StartAuditWorker(ctx context.Context, redisClient *redis.Client, auditRepo AuditRepository) {

	if redisClient == nil {
		return
	}

	pubsub := redisClient.Subscribe(ctx, "audit_events")
	defer pubsub.Close()

	ch := pubsub.Channel()
	log.Println("Audit Worker: Listening for system events...")

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				log.Println("Audit Worker: Channel closed.")
				return
			}
			var logEntry models.AuditLog
			if err := json.Unmarshal([]byte(msg.Payload), &logEntry); err != nil {
				log.Printf("Audit Worker Error: Failed to parse message: %v", err)
				continue
			}

			err := auditRepo.CreateLog(ctx, &logEntry)
			if err != nil {
				log.Printf("Audit Worker Error: Failed to save log to DB: %v", err)
			} else {
				if logEntry.UserID != nil {
					log.Printf("Audit Logged: User %d -> %s", *logEntry.UserID, logEntry.Action)
				} else {
					log.Printf("Audit Logged: System/Anonymous -> %s", logEntry.Action)
				}
			}

		case <-ctx.Done():
			log.Println("Audit Worker: Shutting down.")
			return
		}
	}
}

func PublishEvent(ctx context.Context, redisClient *redis.Client, userID *int64, action string, entityType string, entityID *int64, details map[string]any) {

	if redisClient == nil {
		return
	}

	var detailsBytes json.RawMessage
	if details != nil {
		bytes, err := json.Marshal(details)
		if err == nil {
			detailsBytes = bytes
		} else {
			log.Printf("Warning: Failed to marshal audit details: %v", err)
		}
	}

	logEntry := models.AuditLog{
		UserID:     userID,
		Action:     action,
		EntityType: &entityType,
		EntityID:   entityID,
		Details:    detailsBytes,
	}

	auditBytes, err := json.Marshal(logEntry)
	if err != nil {
		log.Printf("Warning: Failed to marshal audit log: %v", err)
		return
	}

	err = redisClient.Publish(ctx, "audit_events", auditBytes).Err()
	if err != nil {
		log.Printf("Warning: Failed to publish audit event to Redis: %v", err)
	}
}
