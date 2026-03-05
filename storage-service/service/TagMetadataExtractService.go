package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

type TagMetadataPipelineContext struct {
	Name        string
	Description string
	Embedding   *entity.ElasticEmbeddingV1
}

type TagMetadataExtractService interface {
	ProcessTagMetadata(ctx context.Context, name string, description string) (*TagMetadataPipelineContext, error)
}

type TagMetadataExtractServiceImpl struct {
	slogger  *slog.Logger
	embedder ocr.EmbeddingExtractor
}

func (s *TagMetadataExtractServiceImpl) ProcessTagMetadata(ctx context.Context, name string, description string) (*TagMetadataPipelineContext, error) {
	text := name + " " + description

	s.slogger.InfoContext(ctx, "ProcessTagMetadata: computing embedding", "name", name)

	embedding, err := s.embedder.GetTextEmbeddingV1(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("get text embedding failed: %w", err)
	}

	s.slogger.InfoContext(ctx, "ProcessTagMetadata: done", "name", name)

	return &TagMetadataPipelineContext{
		Name:        name,
		Description: description,
		Embedding:   embedding,
	}, nil
}

func NewTagMetadataExtractService(embedder ocr.EmbeddingExtractor) TagMetadataExtractService {
	return &TagMetadataExtractServiceImpl{
		slogger:  slog.With("service", "TagMetadataExtractService"),
		embedder: embedder,
	}
}
