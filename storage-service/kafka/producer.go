package kafka

import (
	"context"
	"fmt"
	"log/slog"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/weoses/memelo/storage-service/conf"
	"google.golang.org/protobuf/proto"
)

// KafkaProducer writes proto messages to a fixed Kafka topic.
// The topic is bound at construction time; create separate instances for different topics.
type KafkaProducer[T proto.Message] struct {
	topic   string
	writer  *kafkago.Writer
	slogger *slog.Logger
}

func NewKafkaProducer[T proto.Message](cfg *conf.Config) (*KafkaProducer[T], error) {
	if cfg.Kafka == nil {
		return nil, fmt.Errorf("kafka config is nil")
	}
	w := &kafkago.Writer{
		Addr:         kafkago.TCP(cfg.Kafka.Brokers...),
		Balancer:     &kafkago.LeastBytes{},
		RequiredAcks: kafkago.RequireOne,
		Async:        false,
	}
	return &KafkaProducer[T]{
		topic:   cfg.Kafka.ParseResponseTopic,
		writer:  w,
		slogger: slog.With("service", fmt.Sprintf("KafkaProducer[%T]", *new(T))),
	}, nil
}

// Produce marshals msg as proto bytes and writes to the producer's topic.
// key controls partition assignment; pass nil for round-robin.
func (p *KafkaProducer[T]) Produce(ctx context.Context, id string, msg T) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("kafka producer: marshal: %w", err)
	}
	if err := p.writer.WriteMessages(ctx, kafkago.Message{
		Topic: p.topic,
		Key:   []byte(id),
		Value: data,
	}); err != nil {
		return fmt.Errorf("kafka producer: write: %w", err)
	}
	return nil
}

func (p *KafkaProducer[T]) Close() error {
	return p.writer.Close()
}
