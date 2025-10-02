package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sbalaji09/LogBuilder/log_analytics_engine/internal/models"
	"github.com/sirupsen/logrus"
)

type RedisClient struct {
	client *redis.Client
	logger *logrus.Logger
}

// creates a new Redis Client for the server to connect to
func NewRedisClient(addr string, password string, db int) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger := logrus.New()
	logger.Info("Connected to Redis successfully")

	return &RedisClient{
		client: client,
		logger: logger,
	}, nil
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

// publishes a log entry to Redis Stream
func (r *RedisClient) PublishLog(ctx context.Context, log *models.LogEntry) error {
	// Serialize log to JSON
	logJSON, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	// Add to Redis Stream
	streamName := "logs:incoming"
	result := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		Values: map[string]interface{}{
			"log": string(logJSON),
		},
	})

	if err := result.Err(); err != nil {
		return fmt.Errorf("failed to add log to stream: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"stream":  streamName,
		"log_id":  result.Val(),
		"user_id": log.UserID,
		"level":   log.Level,
	}).Debug("Log published to stream")

	return nil
}

// publishes multiple log entries to Redis Stream
func (r *RedisClient) PublishLogs(ctx context.Context, logs []*models.LogEntry) error {
	if len(logs) == 0 {
		return nil
	}

	streamName := "logs:incoming"
	pipe := r.client.Pipeline()

	for _, log := range logs {
		logJSON, err := json.Marshal(log)
		if err != nil {
			r.logger.WithError(err).Error("Failed to marshal log in batch")
			continue
		}

		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: streamName,
			Values: map[string]interface{}{
				"log": string(logJSON),
			},
		})
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to publish batch logs: %w", err)
	}

	r.logger.WithField("count", len(logs)).Info("Batch logs published to stream")
	return nil
}

// consumes logs from Redis Stream
func (r *RedisClient) ConsumeLogStream(ctx context.Context, consumerGroup, consumerName string, handler func(*models.LogEntry) error) error {
	streamName := "logs:incoming"

	// Create consumer group if it doesn't exist
	err := r.client.XGroupCreateMkStream(ctx, streamName, consumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"stream":   streamName,
		"group":    consumerGroup,
		"consumer": consumerName,
	}).Info("Starting to consume from stream")

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Consumer context cancelled, stopping...")
			return ctx.Err()
		default:
			// Read from stream
			streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    consumerGroup,
				Consumer: consumerName,
				Streams:  []string{streamName, ">"},
				Count:    10,              // Process 10 messages at a time
				Block:    1 * time.Second, // Block for 1 second if no messages
			}).Result()

			if err != nil {
				if err == redis.Nil {
					// No new messages, continue
					continue
				}
				r.logger.WithError(err).Error("Failed to read from stream")
				time.Sleep(1 * time.Second)
				continue
			}

			// Process messages
			for _, stream := range streams {
				for _, message := range stream.Messages {
					if err := r.processMessage(ctx, streamName, consumerGroup, message, handler); err != nil {
						r.logger.WithError(err).Error("Failed to process message")
					}
				}
			}
		}
	}
}

// consumes a single Redis stream message, deserialize its contents into a structured log entry, pass it to a handler function, and acknowledge the message in Redis if processing succeeded
func (r *RedisClient) processMessage(ctx context.Context, streamName, consumerGroup string, message redis.XMessage, handler func(*models.LogEntry) error) error {
	// Extract log JSON from message
	logJSON, ok := message.Values["log"].(string)
	if !ok {
		r.logger.Error("Invalid message format: missing log field")
		// Acknowledge bad message to remove it from pending
		r.client.XAck(ctx, streamName, consumerGroup, message.ID)
		return fmt.Errorf("invalid message format")
	}

	// Deserialize log
	var log models.LogEntry
	if err := json.Unmarshal([]byte(logJSON), &log); err != nil {
		r.logger.WithError(err).Error("Failed to unmarshal log")
		// Acknowledge bad message
		r.client.XAck(ctx, streamName, consumerGroup, message.ID)
		return fmt.Errorf("failed to unmarshal log: %w", err)
	}

	// Call handler function
	if err := handler(&log); err != nil {
		r.logger.WithError(err).WithField("log_id", log.ID).Error("Handler failed to process log")
		// Don't acknowledge - message will be retried
		return fmt.Errorf("handler failed: %w", err)
	}

	// Acknowledge successful processing
	if err := r.client.XAck(ctx, streamName, consumerGroup, message.ID).Err(); err != nil {
		r.logger.WithError(err).Error("Failed to acknowledge message")
		return fmt.Errorf("failed to acknowledge: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"message_id": message.ID,
		"user_id":    log.UserID,
		"level":      log.Level,
	}).Debug("Message processed and acknowledged")

	return nil
}

// returns information about the stream
func (r *RedisClient) GetStreamInfo(ctx context.Context) (map[string]interface{}, error) {
	streamName := "logs:incoming"

	// Get stream length
	length, err := r.client.XLen(ctx, streamName).Result()
	if err != nil {
		return nil, err
	}

	// Get consumer group info
	groups, err := r.client.XInfoGroups(ctx, streamName).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	info := map[string]interface{}{
		"stream_name":   streamName,
		"stream_length": length,
		"groups":        groups,
	}

	return info, nil
}

// returns the underlying Redis client (for advanced usage)
func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}

// CacheAPIKey stores an API key with associated user ID in Redis with TTL
func (r *RedisClient) CacheAPIKey(ctx context.Context, apiKey string, userID int, ttl time.Duration) error {
	key := fmt.Sprintf("apikey:%s", apiKey)
	err := r.client.Set(ctx, key, userID, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to cache API key: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"ttl":     ttl,
	}).Debug("API key cached in Redis")

	return nil
}

// GetCachedAPIKey retrieves the user ID associated with an API key from cache
func (r *RedisClient) GetCachedAPIKey(ctx context.Context, apiKey string) (int, error) {
	key := fmt.Sprintf("apikey:%s", apiKey)
	result, err := r.client.Get(ctx, key).Int()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("API key not in cache")
		}
		return 0, fmt.Errorf("failed to get cached API key: %w", err)
	}

	return result, nil
}

// InvalidateCachedAPIKey removes an API key from the cache
func (r *RedisClient) InvalidateCachedAPIKey(ctx context.Context, apiKey string) error {
	key := fmt.Sprintf("apikey:%s", apiKey)
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to invalidate cached API key: %w", err)
	}

	r.logger.Debug("API key invalidated from cache")
	return nil
}
