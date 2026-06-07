package api

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/common/temp"
	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/kafka"
	"github.com/weoses/memelo/storage-service/key"
	"github.com/weoses/memelo/storage-service/service"
	"github.com/weoses/memelo/storage-service/utils"
)

// CreateMemeKafkaController handles CreateMeme requests arriving via Kafka.
// It owns all proto↔service mapping, mirroring the role of SearchServiceApi for gRPC.
type CreateMemeKafkaController struct {
	crud        service.MemeCrudService
	dataService commonservice.TmpDataService

	consumer *kafka.KafkaReceiver[v1.KafkaCreateMemeRequest]
	producer *kafka.KafkaProducer[*v1.KafkaCreateMemeResponse]
	slogger  *slog.Logger
	wg       sync.WaitGroup

	ctxController       context.Context
	cancelCtxController context.CancelFunc
}

func NewCreateMemeKafkaController(
	crud service.MemeCrudService,
	dataService commonservice.TmpDataService,
	consumer *kafka.KafkaReceiver[v1.KafkaCreateMemeRequest],
	producer *kafka.KafkaProducer[*v1.KafkaCreateMemeResponse],
) *CreateMemeKafkaController {
	return &CreateMemeKafkaController{
		crud:        crud,
		dataService: dataService,
		consumer:    consumer,
		producer:    producer,
		slogger:     slog.With("service", "CreateMemeKafkaController"),
	}
}

func (c *CreateMemeKafkaController) Start(ctx context.Context) error {
	c.ctxController, c.cancelCtxController = context.WithCancel(ctx)

	for i := 0; i < runtime.NumCPU(); i++ {
		c.wg.Go(func() {
			c.loop(c.ctxController)
		})
	}
	return nil
}

func (c *CreateMemeKafkaController) Stop() error {
	c.cancelCtxController()
	c.wg.Wait()
	return nil
}

func (c *CreateMemeKafkaController) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.slogger.InfoContext(ctx, "CreateMemeKafkaController stopped")
			return
		default:
		}

		one, err := c.consumer.FetchOne(ctx)
		if err != nil {
			c.slogger.ErrorContext(ctx, "failed to fetch one record from kafka", "error", err)
			time.Sleep(time.Second)
			continue
		}

		response, err := c.processRequest(ctx, one.Data)
		if err != nil {
			c.slogger.ErrorContext(ctx, "failed to process request", "error", err)
			continue
		}

		err = c.producer.Produce(ctx, one.Data.GetKafkaRequestId(), response)
		if err != nil {
			c.slogger.ErrorContext(ctx, "failed to produce response", "error", err)
			continue
		}

		err = c.consumer.Commit(ctx, *one)
		if err != nil {
			c.slogger.ErrorContext(ctx, "failed to commit input topic", "error", err)
		}
	}

}

// Handle is the MessageHandler callback for KafkaReceiver.
// Validation errors are absorbed into the response envelope (message is committed).
// Infrastructure errors are returned (message is not committed and will be retried).
func (c *CreateMemeKafkaController) processRequest(ctx context.Context, req *v1.KafkaCreateMemeRequest) (*v1.KafkaCreateMemeResponse, error) {
	requestId := req.KafkaRequestId
	c.slogger.InfoContext(ctx, "kafka CreateMeme request", "kafka_request_id", requestId)

	ctx = context.WithValue(ctx, key.AccountId, req.Payload.GetAccountId())

	resp := &v1.KafkaCreateMemeResponse{KafkaRequestId: requestId}

	accountIdUuid, err := uuid.Parse(req.Payload.GetAccountId())
	if err != nil {
		c.slogger.ErrorContext(ctx, "process request error", "error", err)
		resp.ErrorMessage = fmt.Sprintf("invalid account_id: %s", req.Payload.GetAccountId())
		return resp, nil
	}

	var data temp.S3BackedData
	var closeable bool
	var metadataType entity.MetadataType

	if req.Payload.GetImage() != nil {
		metadataType = entity.ImageMetadataType
		data, closeable, err = utils.ToData(ctx, c.dataService, req.Payload.GetImage())
	} else if req.Payload.GetVideo() != nil {
		metadataType = entity.VideoMetadataType
		data, closeable, err = utils.ToData(ctx, c.dataService, req.Payload.GetVideo())
	} else {
		resp.ErrorMessage = "neither image nor video provided"
		c.slogger.ErrorContext(ctx, "process request error", "error", resp.ErrorMessage)
		return resp, nil
	}

	if err != nil {
		resp.ErrorMessage = fmt.Sprintf("kafka controller: toData: %s", err)
		c.slogger.ErrorContext(ctx, "process request error", "error", resp.ErrorMessage)
		return resp, nil
	}
	if closeable {
		defer helper.QuietClose(data, c.slogger)
	}

	meme, err := c.crud.CreateMeme(ctx, accountIdUuid, metadataType, data)
	if err != nil || meme == nil {
		resp.ErrorMessage = fmt.Sprintf("kafka controller: CreateMeme service call: %s", err)
		c.slogger.ErrorContext(ctx, "process request error", "error", resp.ErrorMessage)
		return resp, nil
	}

	c.slogger.InfoContext(ctx, "kafka CreateMeme response",
		"kafka_request_id", requestId,
		"id", meme.Metadata.Metadata.ImageId.String(),
		"status", meme.Status,
	)

	resp.Payload = &v1.CreateMemeResponse{
		Result: utils.MetadataToMemeDto(meme.Metadata),
		Status: v1.CreateMemeStatus(meme.Status),
	}
	return resp, nil
}
