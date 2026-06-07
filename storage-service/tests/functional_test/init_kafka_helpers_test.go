package functional_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	v1 "github.com/weoses/memelo/gen/proto/v1"
	"google.golang.org/protobuf/proto"
)

const (
	testKafkaRequestTopic  = "test-parse-request"
	testKafkaResponseTopic = "test-parse-response"
	testKafkaConsumerGroup = "test-controller-group"
)

// KafkaTestClient sends JSON-encoded requests to the request topic and reads
// proto-encoded responses from the response topic, correlating by kafka_request_id.
type KafkaTestClient struct {
	brokers       []string
	requestTopic  string
	responseTopic string
	writer        *kafkago.Writer
	reader        *kafkago.Reader
}

func NewKafkaTestClient(brokers []string, requestTopic, responseTopic string) *KafkaTestClient {
	c := &KafkaTestClient{
		brokers:       brokers,
		requestTopic:  requestTopic,
		responseTopic: responseTopic,
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(brokers...),
			Balancer:     &kafkago.LeastBytes{},
			RequiredAcks: kafkago.RequireOne,
			Async:        false,
		},
	}
	c.resetReader()
	return c
}

// resetReader repositions the response reader to the current end of the topic,
// so each test only sees responses produced after its own request.
func (c *KafkaTestClient) resetReader() {
	if c.reader != nil {
		_ = c.reader.Close()
	}
	c.reader = kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:     c.brokers,
		Topic:       c.responseTopic,
		StartOffset: kafkago.LastOffset,
	})
}

func (c *KafkaTestClient) CreateImageMeme(ctx context.Context, accountID string, data []byte) (*v1.KafkaCreateMemeResponse, error) {
	req := &v1.KafkaCreateMemeRequest{
		KafkaRequestId: uuid.New().String(),
		Payload: &v1.CreateMemeRequest{
			AccountId: accountID,
			Image:     &v1.MediaDataDto{Data: data},
		},
	}
	return c.sendRequest(ctx, req)
}

func (c *KafkaTestClient) CreateVideoMeme(ctx context.Context, accountID string, data []byte) (*v1.KafkaCreateMemeResponse, error) {
	req := &v1.KafkaCreateMemeRequest{
		KafkaRequestId: uuid.New().String(),
		Payload: &v1.CreateMemeRequest{
			AccountId: accountID,
			Video:     &v1.MediaDataDto{Data: data},
		},
	}
	return c.sendRequest(ctx, req)
}

// SendRawRequest publishes a request directly, allowing tests to craft malformed payloads.
func (c *KafkaTestClient) SendRawRequest(ctx context.Context, req *v1.KafkaCreateMemeRequest) (*v1.KafkaCreateMemeResponse, error) {
	return c.sendRequest(ctx, req)
}

func (c *KafkaTestClient) sendRequest(ctx context.Context, req *v1.KafkaCreateMemeRequest) (*v1.KafkaCreateMemeResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("kafka test client: marshal request: %w", err)
	}
	err = c.writer.WriteMessages(ctx, kafkago.Message{
		Topic: c.requestTopic,
		Key:   []byte(req.KafkaRequestId),
		Value: body,
	})
	if err != nil {
		return nil, fmt.Errorf("kafka test client: write message: %w", err)
	}
	return c.waitForResponse(ctx, req.KafkaRequestId)
}

func (c *KafkaTestClient) waitForResponse(ctx context.Context, requestID string) (*v1.KafkaCreateMemeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			return nil, fmt.Errorf("kafka test client: fetch response for %s: %w", requestID, err)
		}
		_ = c.reader.CommitMessages(ctx, msg)

		resp := &v1.KafkaCreateMemeResponse{}
		if err := proto.Unmarshal(msg.Value, resp); err != nil {
			continue
		}
		if resp.KafkaRequestId == requestID {
			return resp, nil
		}
	}
}
