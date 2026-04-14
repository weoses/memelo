package gapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
	"golang.org/x/time/rate"
	"google.golang.org/genai"
)

type GcloudImageEmbeddingExtractorGenaiImpl struct {
	client    *genai.Client
	model     string
	dimension int32
	limiter   *rate.Limiter
	slogger   *slog.Logger
}

func (i *GcloudImageEmbeddingExtractorGenaiImpl) GetImageEmbedding(ctx context.Context, rawImage temp.Data) (*entity.EmbeddingItem, error) {
	i.slogger.InfoContext(ctx, "GetImageEmbedding start")
	if err := i.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	var part *genai.Part
	if s3data, ok := rawImage.(temp.S3BackedData); ok && s3data.IsGsSupported() {
		if uri, pathErr := s3data.GetPresignedUrl(ctx); pathErr == nil {
			i.slogger.InfoContext(ctx, "GetImageEmbedding using gcsUri", "uri", uri)
			part = genai.NewPartFromURI(uri, "image/jpeg")
		}
	}
	if part == nil {
		i.slogger.InfoContext(ctx, "GetImageEmbedding using inline bytes")
		reader, err := rawImage.Reader()
		if err != nil {
			return nil, fmt.Errorf("GetImageEmbedding: read image: %w", err)
		}
		defer helper.QuietClose(reader, i.slogger)
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("GetImageEmbedding: read all: %w", err)
		}
		part = genai.NewPartFromBytes(data, "image/jpeg")
	}

	resp, err := i.embedContent(ctx, part)
	if err != nil {
		return nil, fmt.Errorf("GetImageEmbedding: %w", err)
	}

	i.slogger.InfoContext(ctx, "GetImageEmbedding done")
	return &entity.EmbeddingItem{
		Data:  resp.Embeddings[0].Values,
		Model: i.model,
		Type:  entity.EmbeddingTypeImage,
	}, nil
}

func (i *GcloudImageEmbeddingExtractorGenaiImpl) GetVideoEmbedding(ctx context.Context, video temp.Data) ([]*entity.EmbeddingItem, error) {
	i.slogger.InfoContext(ctx, "GetVideoEmbedding start")
	if err := i.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	var part *genai.Part
	if s3data, ok := video.(temp.S3BackedData); ok && s3data.IsGsSupported() {
		if uri, pathErr := s3data.GetPresignedUrl(ctx); pathErr == nil {
			i.slogger.InfoContext(ctx, "GetImageEmbedding using gcsUri", "uri", uri)
			part = genai.NewPartFromURI(uri, "video/mp4")
		}
	}
	if part == nil {
		i.slogger.InfoContext(ctx, "GetVideoEmbedding using inline bytes")
		reader, err := video.Reader()
		if err != nil {
			return nil, fmt.Errorf("GetVideoEmbedding: read video: %w", err)
		}
		defer helper.QuietClose(reader, i.slogger)
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("GetVideoEmbedding: read all: %w", err)
		}
		part = genai.NewPartFromBytes(data, "video/mp4")
	}

	resp, err := i.embedContent(ctx, part)
	if err != nil {
		return nil, fmt.Errorf("GetVideoEmbedding: %w", err)
	}

	items := make([]*entity.EmbeddingItem, len(resp.Embeddings))
	for idx, emb := range resp.Embeddings {
		items[idx] = &entity.EmbeddingItem{
			Data:  emb.Values,
			Model: i.model,
			Type:  entity.EmbeddingTypeVideo,
		}
	}

	i.slogger.InfoContext(ctx, "GetVideoEmbedding done", "segments", len(items))
	return items, nil
}

func (i *GcloudImageEmbeddingExtractorGenaiImpl) GetTextEmbedding(ctx context.Context, text string) (*entity.EmbeddingItem, error) {
	i.slogger.InfoContext(ctx, "GetTextEmbedding start")
	if err := i.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	resp, err := i.embedContent(ctx, genai.NewPartFromText(text))
	if err != nil {
		return nil, fmt.Errorf("GetTextEmbedding: %w", err)
	}

	i.slogger.InfoContext(ctx, "GetTextEmbedding done")
	return &entity.EmbeddingItem{
		Data:  resp.Embeddings[0].Values,
		Model: i.model,
		Type:  entity.EmbeddingTypeText,
	}, nil
}

func (i *GcloudImageEmbeddingExtractorGenaiImpl) embedContent(ctx context.Context, part *genai.Part) (*genai.EmbedContentResponse, error) {
	content := genai.NewContentFromParts([]*genai.Part{part}, genai.RoleUser)
	resp, err := i.client.Models.EmbedContent(ctx, i.model, []*genai.Content{content}, &genai.EmbedContentConfig{
		OutputDimensionality: &i.dimension,
		TaskType:             "SEMANTIC_SIMILARITY",
	})
	if err != nil {
		return nil, fmt.Errorf("EmbedContent: %w", err)
	}
	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("EmbedContent: empty response")
	}
	return resp, nil
}

func NewImageEmbeddingExtractorGenai(cfg *conf.Config) (ocr.LlmEmbeddingExtractor, error) {
	geminiConfig := cfg.GeminiEmbedding
	embeddingConfig := cfg.Embeddings

	clientCfg := &genai.ClientConfig{}

	if geminiConfig.ApiKey != "" {
		clientCfg.APIKey = geminiConfig.ApiKey
	}
	if geminiConfig.ApiEndpoint != "" {
		clientCfg.HTTPOptions = genai.HTTPOptions{BaseURL: geminiConfig.ApiEndpoint}
	}
	client, err := genai.NewClient(context.Background(), clientCfg)
	if err != nil {
		return nil, fmt.Errorf("NewImageEmbeddingExtractor: create genai client: %w", err)
	}

	return &GcloudImageEmbeddingExtractorGenaiImpl{
		client:    client,
		model:     geminiConfig.Model,
		dimension: int32(embeddingConfig.Dimensions),
		limiter:   rate.NewLimiter(2, 4),
		slogger:   slog.With("service", "LlmEmbeddingExtractor"),
	}, nil
}
