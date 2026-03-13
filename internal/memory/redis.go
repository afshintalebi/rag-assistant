package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sashabaranov/go-openai"
)

type RedisMemory struct {
	client *redis.Client
}

// NewRedisMemory initializes the Redis client for chat history.
func NewRedisMemory(addr string) (*RedisMemory, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisMemory{
		client: client,
	}, nil
}

// AddMessage appends a new OpenAI message to the session's history.
func (r *RedisMemory) AddMessage(ctx context.Context, sessionID string, message openai.ChatCompletionMessage) error {
	key := fmt.Sprintf("chat_history:%s", sessionID)
	
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Push message to the right side of the list
	err = r.client.RPush(ctx, key, data).Err()
	if err != nil {
		return fmt.Errorf("failed to save message to redis: %w", err)
	}

	// Keep history for 24 hours
	r.client.Expire(ctx, key, 24*time.Hour)

	return nil
}

// GetHistory retrieves the conversation history for a specific session.
func (r *RedisMemory) GetHistory(ctx context.Context, sessionID string) ([]openai.ChatCompletionMessage, error) {
	key := fmt.Sprintf("chat_history:%s", sessionID)

	// Fetch all messages from the list
	data, err := r.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve history from redis: %w", err)
	}

	var history []openai.ChatCompletionMessage
	for _, item := range data {
		var msg openai.ChatCompletionMessage
		if err := json.Unmarshal([]byte(item), &msg); err == nil {
			history = append(history, msg)
		}
	}

	return history, nil
}