package kafka

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/goccy/go-json"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/weoses/memelo/storage-service/conf"
)

type ConsumerMessageItem[T any] struct {
	Data *T
	raw  kafkago.Message
}

// KafkaReceiver consumes messages from a single Kafka topic and calls a MessageHandler.
// It is topic-agnostic; the topic is fixed at construction time via config.
type KafkaReceiver[T any] struct {
	reader  *kafkago.Reader
	slogger *slog.Logger
}

func (r *KafkaReceiver[T]) FetchOne(ctx context.Context) (*ConsumerMessageItem[T], error) {
	msg, err := r.reader.FetchMessage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch message: %w", err)
	}

	data, err := r.unmarshallMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &ConsumerMessageItem[T]{
		Data: data,
		raw:  msg,
	}, nil
}

func (r *KafkaReceiver[T]) Commit(ctx context.Context, msg ConsumerMessageItem[T]) error {
	return r.commitRaw(ctx, msg.raw)
}

func (r *KafkaReceiver[T]) commitRaw(ctx context.Context, msg kafkago.Message) error {
	err := r.reader.CommitMessages(ctx, msg)
	return err
}

func (r *KafkaReceiver[T]) unmarshallMessage(msg kafkago.Message) (*T, error) {
	data := new(T)
	err := json.Unmarshal(msg.Value, data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal kafka message: %w", err)
	}

	return data, nil
}

func NewKafkaReceiver[T any](cfg *conf.Config) (*KafkaReceiver[T], error) {
	if cfg.Kafka == nil {
		return nil, fmt.Errorf("kafka config is nil")
	}
	r := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  cfg.Kafka.Brokers,
		GroupID:  cfg.Kafka.ParseConsumerGroup,
		Topic:    cfg.Kafka.ParseRequestTopic,
		MinBytes: 1,
		MaxBytes: 10 * 1024 * 1024, // 10 MB
	})
	return &KafkaReceiver[T]{
		reader:  r,
		slogger: slog.With("service", "KafkaReceiver[%T]", *new(T)),
	}, nil
}
