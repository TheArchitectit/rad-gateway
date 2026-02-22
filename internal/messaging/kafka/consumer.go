package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
)

// HandlerFunc processes consumed messages
type HandlerFunc func(ctx context.Context, msg *sarama.ConsumerMessage) error

// Consumer wraps Sarama consumer group
type Consumer struct {
	group   sarama.ConsumerGroup
	topics  []string
	handler HandlerFunc
}

// NewConsumer creates a new consumer group
func NewConsumer(brokers []string, groupID string, topics []string) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Group.Session.Timeout = 30 * time.Second
	config.Consumer.Group.Heartbeat.Interval = 10 * time.Second

	group, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, fmt.Errorf("creating consumer group: %w", err)
	}

	return &Consumer{
		group:  group,
		topics: topics,
	}, nil
}

// Start begins consuming messages
func (c *Consumer) Start(ctx context.Context, handler HandlerFunc) error {
	c.handler = handler

	for {
		if err := c.group.Consume(ctx, c.topics, c); err != nil {
			return fmt.Errorf("consume error: %w", err)
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

// Close shuts down the consumer
func (c *Consumer) Close() error {
	return c.group.Close()
}

// Setup is called at the start of a new session
func (c *Consumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is called at the end of a session
func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim processes messages from a consumer group claim
func (c *Consumer) ConsumeClaim(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	ctx := context.Background()

	for msg := range claim.Messages() {
		slog.Debug("received message",
			"topic", msg.Topic,
			"partition", msg.Partition,
			"offset", msg.Offset,
		)

		if err := c.handler(ctx, msg); err != nil {
			slog.Error("handler error",
				"topic", msg.Topic,
				"error", err,
			)
			// Don't commit on error - will retry
			continue
		}

		session.MarkMessage(msg, "")
	}

	return nil
}

// ProcessWebhookCallback handles webhook delivery from the queue
func ProcessWebhookCallback(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var callback WebhookCallback
	if err := json.Unmarshal(msg.Value, &callback); err != nil {
		return fmt.Errorf("unmarshaling callback: %w", err)
	}

	slog.Info("processing webhook",
		"callback_id", callback.CallbackID,
		"task_id", callback.TaskID,
		"url", callback.WebhookURL,
		"retry", callback.RetryCount,
	)

	// TODO: Implement actual HTTP POST to webhookURL
	// For now, just log the callback

	return nil
}

// ProcessTaskEvent handles task lifecycle events
func ProcessTaskEvent(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var event TaskEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("unmarshaling event: %w", err)
	}

	slog.Info("task event",
		"event_id", event.EventID,
		"task_id", event.TaskID,
		"event_type", event.EventType,
		"status", event.Status,
	)

	return nil
}
