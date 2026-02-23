package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/IBM/sarama"
)

// Producer wraps Sarama async producer for A2A events
type Producer struct {
	producer sarama.AsyncProducer
	brokers  []string
}

// NewProducer creates a new Kafka producer
func NewProducer(brokers []string) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 3
	config.Producer.Compression = sarama.CompressionSnappy

	producer, err := sarama.NewAsyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("creating producer: %w", err)
	}

	return &Producer{
		producer: producer,
		brokers:  brokers,
	}, nil
}

// SendTaskEvent sends a task event to Kafka
func (p *Producer) SendTaskEvent(ctx context.Context, event TaskEvent) error {
	msg, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	p.producer.Input() <- &sarama.ProducerMessage{
		Topic: "a2a-task-events",
		Key:   sarama.StringEncoder(event.TaskID),
		Value: sarama.ByteEncoder(msg),
		Headers: []sarama.RecordHeader{
			{Key: []byte("event-type"), Value: []byte(event.EventType)},
			{Key: []byte("agent-id"), Value: []byte(event.AgentID)},
		},
	}

	slog.Debug("sent task event",
		"task_id", event.TaskID,
		"event_type", event.EventType,
	)

	return nil
}

// SendWebhookCallback sends a webhook callback to the queue
func (p *Producer) SendWebhookCallback(ctx context.Context, callback WebhookCallback) error {
	msg, err := json.Marshal(callback)
	if err != nil {
		return fmt.Errorf("marshaling callback: %w", err)
	}

	p.producer.Input() <- &sarama.ProducerMessage{
		Topic: "a2a-webhook-callbacks",
		Key:   sarama.StringEncoder(callback.CallbackID),
		Value: sarama.ByteEncoder(msg),
	}

	slog.Debug("queued webhook callback",
		"callback_id", callback.CallbackID,
		"task_id", callback.TaskID,
	)

	return nil
}

// SendAgentDiscovery sends an agent discovery event
func (p *Producer) SendAgentDiscovery(ctx context.Context, event AgentDiscoveryEvent) error {
	msg, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling discovery event: %w", err)
	}

	p.producer.Input() <- &sarama.ProducerMessage{
		Topic: "a2a-agent-discovery",
		Key:   sarama.StringEncoder(event.AgentID),
		Value: sarama.ByteEncoder(msg),
	}

	return nil
}

// SendMetrics sends protocol metrics
func (p *Producer) SendMetrics(ctx context.Context, metrics ProtocolMetrics) error {
	msg, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshaling metrics: %w", err)
	}

	p.producer.Input() <- &sarama.ProducerMessage{
		Topic: "a2a-protocol-metrics",
		Key:   sarama.StringEncoder(metrics.TaskID),
		Value: sarama.ByteEncoder(msg),
	}

	return nil
}

// Close shuts down the producer
func (p *Producer) Close() error {
	return p.producer.Close()
}

// Successes returns the success channel
func (p *Producer) Successes() <-chan *sarama.ProducerMessage {
	return p.producer.Successes()
}

// Errors returns the error channel
func (p *Producer) Errors() <-chan *sarama.ProducerError {
	return p.producer.Errors()
}
