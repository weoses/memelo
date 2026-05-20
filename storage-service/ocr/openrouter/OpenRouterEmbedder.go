package openrouter

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"

	orsdk "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/operations"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

type OpenRouterEmbedder struct {
	client       *orsdk.OpenRouter
	model        string
	frameExtract ocr.Video2FrameExtractor
	wrapper      *ocr.ConnectorWrapper
	slogger      *slog.Logger
}

func NewOpenRouterEmbedder(cfg *conf.Config, fe ocr.Video2FrameExtractor) (ocr.LlmEmbeddingExtractor, error) {
	c := cfg.OpenRouterEmbedding
	if c == nil {
		return nil, fmt.Errorf("openrouter-embedding config is missing")
	}
	client := orsdk.New(orsdk.WithSecurity(c.ApiKey))
	return &OpenRouterEmbedder{
		client:       client,
		model:        c.Model,
		frameExtract: fe,
		wrapper:      ocr.NewConnectorWrapper(10, 3),
		slogger:      slog.With("service", "OpenRouterEmbedder"),
	}, nil
}

func (e *OpenRouterEmbedder) GetTextEmbedding(ctx context.Context, text string) (*entity.EmbeddingItem, error) {
	e.slogger.InfoContext(ctx, "GetTextEmbedding start")

	var result *entity.EmbeddingItem
	if err := e.wrapper.Do(ctx, func() error {
		input := operations.CreateInputUnionArrayOfStr([]string{text})
		resp, err := e.client.Embeddings.Generate(ctx, operations.CreateEmbeddingsRequest{
			Model:          e.model,
			Input:          input,
			EncodingFormat: operations.EncodingFormatFloat.ToPointer(),
		})
		if err != nil {
			return fmt.Errorf("GetTextEmbedding: %w", err)
		}
		vec, err := extractEmbedding(resp)
		if err != nil {
			return fmt.Errorf("GetTextEmbedding: %w", err)
		}
		result = &entity.EmbeddingItem{
			Data:  vec,
			Model: e.model,
			Type:  entity.EmbeddingTypeText,
		}
		return nil
	}); err != nil {
		return nil, err
	}

	e.slogger.InfoContext(ctx, "GetTextEmbedding done")
	return result, nil
}

func (e *OpenRouterEmbedder) GetImageEmbedding(ctx context.Context, rawImage temp.Data) (*entity.EmbeddingItem, error) {
	e.slogger.InfoContext(ctx, "GetImageEmbedding start")

	var result *entity.EmbeddingItem
	if err := e.wrapper.Do(ctx, func() error {
		dataURL, err := toDataURL(rawImage, "image/jpeg", e.slogger)
		if err != nil {
			return fmt.Errorf("GetImageEmbedding: %w", err)
		}
		input := operations.CreateInputUnionArrayOfInput([]operations.Input{{
			Content: []operations.Content{
				operations.CreateContentImageURL(operations.ContentImageURL{
					ImageURL: operations.ImageURL{URL: dataURL},
					Type:     operations.TypeImageURLImageURL,
				}),
			},
		}})
		resp, err := e.client.Embeddings.Generate(ctx, operations.CreateEmbeddingsRequest{
			Model:          e.model,
			Input:          input,
			EncodingFormat: operations.EncodingFormatFloat.ToPointer(),
		})
		if err != nil {
			return fmt.Errorf("GetImageEmbedding: %w", err)
		}
		vec, err := extractEmbedding(resp)
		if err != nil {
			return fmt.Errorf("GetImageEmbedding: %w", err)
		}
		result = &entity.EmbeddingItem{
			Data:  vec,
			Model: e.model,
			Type:  entity.EmbeddingTypeImage,
		}
		return nil
	}); err != nil {
		return nil, err
	}

	e.slogger.InfoContext(ctx, "GetImageEmbedding done")
	return result, nil
}

func (e *OpenRouterEmbedder) GetVideoEmbedding(ctx context.Context, video temp.Data) ([]entity.EmbeddingItem, error) {
	e.slogger.InfoContext(ctx, "GetVideoEmbedding start")

	frame, err := e.frameExtract.ExtractOneFrame(ctx, video)
	if err != nil {
		return nil, fmt.Errorf("GetVideoEmbedding: extract frame: %w", err)
	}
	defer helper.QuietClose(frame, e.slogger)

	item, err := e.GetImageEmbedding(ctx, frame)
	if err != nil {
		return nil, fmt.Errorf("GetVideoEmbedding: %w", err)
	}
	item.Type = entity.EmbeddingTypeVideo

	e.slogger.InfoContext(ctx, "GetVideoEmbedding done")
	return []entity.EmbeddingItem{*item}, nil
}

func extractEmbedding(resp *operations.CreateEmbeddingsResponse) ([]float32, error) {
	if resp == nil || resp.CreateEmbeddingsResponseBody == nil {
		return nil, fmt.Errorf("empty embeddings response")
	}
	data := resp.CreateEmbeddingsResponseBody.Data
	if len(data) == 0 {
		return nil, fmt.Errorf("embeddings response has no data")
	}
	f64 := data[0].Embedding.ArrayOfNumber
	if f64 == nil {
		return nil, fmt.Errorf("embeddings response has no float vector")
	}
	vec := make([]float32, len(f64))
	for i, v := range f64 {
		vec[i] = float32(v)
	}
	return vec, nil
}

func toDataURL(data temp.Data, mimeType string, logger *slog.Logger) (string, error) {
	r, err := data.Reader()
	if err != nil {
		return "", fmt.Errorf("read data: %w", err)
	}
	defer helper.QuietClose(r, logger)
	raw, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read all: %w", err)
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}
