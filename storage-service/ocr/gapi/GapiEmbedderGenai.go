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
	"google.golang.org/genai"
)

type GcloudImageEmbeddingExtractorGenaiImpl struct {
	client    *genai.Client
	model     string
	dimension int32
	wrapper   *ocr.ConnectorWrapper
	slogger   *slog.Logger
}

func (i *GcloudImageEmbeddingExtractorGenaiImpl) GetImageEmbedding(ctx context.Context, rawImage temp.Data) (*entity.EmbeddingItem, error) {
	i.slogger.InfoContext(ctx, "GetImageEmbedding start")

	var result *entity.EmbeddingItem
	err := i.wrapper.Do(ctx, func() error {
		var part *genai.Part
		if s3data, ok := rawImage.(temp.S3BackedData); ok {
			if uri, pathErr := s3data.GetPresignedUrl(ctx); pathErr == nil {
				i.slogger.InfoContext(ctx, "GetImageEmbedding using signedUri", "uri", uri)
				part = genai.NewPartFromURI(uri, "image/jpeg")
			}
		}
		if part == nil {
			i.slogger.InfoContext(ctx, "GetImageEmbedding using inline bytes")
			reader, err := rawImage.Reader()
			if err != nil {
				return fmt.Errorf("GetImageEmbedding: read image: %w", err)
			}
			defer helper.QuietClose(reader, i.slogger)
			data, err := io.ReadAll(reader)
			if err != nil {
				return fmt.Errorf("GetImageEmbedding: read all: %w", err)
			}
			part = genai.NewPartFromBytes(data, "image/jpeg")
		}

		resp, err := i.embedContent(ctx, part)
		if err != nil {
			return fmt.Errorf("GetImageEmbedding: %w", err)
		}

		result = &entity.EmbeddingItem{
			Data:  resp.Embeddings[0].Values,
			Model: i.model,
			Type:  entity.EmbeddingTypeImage,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	i.slogger.InfoContext(ctx, "GetImageEmbedding done")
	return result, nil
}

func (i *GcloudImageEmbeddingExtractorGenaiImpl) GetVideoEmbedding(ctx context.Context, video temp.Data) ([]entity.EmbeddingItem, error) {
	i.slogger.InfoContext(ctx, "GetVideoEmbedding start")

	var items []entity.EmbeddingItem
	err := i.wrapper.Do(ctx, func() error {
		var part *genai.Part
		if s3data, ok := video.(temp.S3BackedData); ok {
			if uri, pathErr := s3data.GetPresignedUrl(ctx); pathErr == nil {
				i.slogger.InfoContext(ctx, "GetVideoEmbedding using signedUri", "uri", uri)
				part = genai.NewPartFromURI(uri, "video/mp4")
			}
		}
		if part == nil {
			i.slogger.InfoContext(ctx, "GetVideoEmbedding using inline bytes")
			reader, err := video.Reader()
			if err != nil {
				return fmt.Errorf("GetVideoEmbedding: read video: %w", err)
			}
			defer helper.QuietClose(reader, i.slogger)
			data, err := io.ReadAll(reader)
			if err != nil {
				return fmt.Errorf("GetVideoEmbedding: read all: %w", err)
			}
			part = genai.NewPartFromBytes(data, "video/mp4")
		}

		resp, err := i.embedContent(ctx, part)
		if err != nil {
			return fmt.Errorf("GetVideoEmbedding: %w", err)
		}

		items = make([]entity.EmbeddingItem, len(resp.Embeddings))
		for idx, emb := range resp.Embeddings {
			items[idx] = entity.EmbeddingItem{
				Data:  emb.Values,
				Model: i.model,
				Type:  entity.EmbeddingTypeVideo,
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	i.slogger.InfoContext(ctx, "GetVideoEmbedding done", "segments", len(items))
	return items, nil
}

func (i *GcloudImageEmbeddingExtractorGenaiImpl) GetTextEmbedding(ctx context.Context, text string) (*entity.EmbeddingItem, error) {
	i.slogger.InfoContext(ctx, "GetTextEmbedding start")

	var result *entity.EmbeddingItem
	err := i.wrapper.Do(ctx, func() error {
		resp, err := i.embedContent(ctx, genai.NewPartFromText(text))
		if err != nil {
			return fmt.Errorf("GetTextEmbedding: %w", err)
		}
		result = &entity.EmbeddingItem{
			Data:  resp.Embeddings[0].Values,
			Model: i.model,
			Type:  entity.EmbeddingTypeText,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	i.slogger.InfoContext(ctx, "GetTextEmbedding done")
	return result, nil
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
	embeddingConfig := cfg.Extracting

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
		dimension: int32(embeddingConfig.EmbeddingDimensions),
		wrapper:   ocr.NewConnectorWrapper(40, 3),
		slogger:   slog.With("service", "LlmEmbeddingExtractor"),
	}, nil
}
