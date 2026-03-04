package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
)

const (
	UNSPECIFIED = iota
	NEW
	DUPLICATED
)

type MetadataWithUrls struct {
	Metadata    *entity.ElasticImageMetaData
	UrlThumb    string
	UrlOriginal string
}

type CreateResult struct {
	Metadata *MetadataWithUrls
	Status   int
}

type MemeCrudService interface {
	SearchMeme(ctx context.Context, accountId uuid.UUID, query string, afterId *uuid.UUID, size *int) ([]*MetadataWithUrls, error)
	CreateMeme(ctx context.Context, accountId uuid.UUID, imgRaw []byte) (*CreateResult, error)
	DeleteMeme(ctx context.Context, accountId uuid.UUID, id uuid.UUID) error
}

type MemeCrudServiceImpl struct {
	imageStorageService    ImageStorageService
	metadataStorageService MetadataStorageService
	imageExtractService    ImageMetadataExtractService
	searchService          SearchService
	slogger                *slog.Logger
}

func (m *MemeCrudServiceImpl) CreateMeme(ctx context.Context, accountId uuid.UUID, imgRaw []byte) (*CreateResult, error) {
	pipelineResult, err := m.imageExtractService.ProcessCreate(ctx, accountId, &StepsToDo{
		DuplicateSearch: true,
		Ocr:             true,
		CreateThumbnail: true,
		CalcSize:        true,
		CreateEmbedding: true,
		CalcHash:        true,
	}, imgRaw)
	if err != nil {
		return nil, fmt.Errorf("metadata extract pipeline failed: %w", err)
	}

	if pipelineResult.Duplicate != nil {
		results, err := m.constructMetadataWithUrls(ctx, []*entity.ElasticImageMetaData{pipelineResult.Duplicate})
		if err != nil {
			return nil, fmt.Errorf("add urls to elastic entities failed: %w", err)
		}

		return &CreateResult{
			Metadata: results[0],
			Status:   DUPLICATED,
		}, nil
	}

	s3id := uuid.New()
	imgId := uuid.New()

	err = m.imageStorageService.Save(ctx, s3id, pipelineResult.ImageRaw, *pipelineResult.ImageThumbnail)
	if err != nil {
		return nil, fmt.Errorf("save image files failed: %w", err)
	}

	metadataEntity := &entity.ElasticImageMetaData{
		ImageId:     imgId,
		S3Id:        s3id,
		AccountId:   accountId,
		Result:      *pipelineResult.ImageOcrResult,
		Hash:        *pipelineResult.ImageHash,
		EmbeddingV1: pipelineResult.ImageEmbedding,
		ImageSize:   pipelineResult.ImageRawSize,
		ThumbSize:   pipelineResult.ImageThumbnailSize,
		Created:     time.Now().UnixMicro(),
		Updated:     time.Now().UnixMicro(),
	}
	err = m.metadataStorageService.Save(ctx, metadataEntity)
	if err != nil {
		return nil, fmt.Errorf("save metadata failed: %w", err)
	}

	entities, err := m.constructMetadataWithUrls(ctx, []*entity.ElasticImageMetaData{metadataEntity})
	if err != nil {
		return nil, fmt.Errorf("add urls to elastic entities failed: %w", err)
	}

	return &CreateResult{
		Metadata: entities[0],
		Status:   NEW,
	}, nil
}

func (m *MemeCrudServiceImpl) SearchMeme(ctx context.Context, accountId uuid.UUID, query string, afterId *uuid.UUID, size *int) ([]*MetadataWithUrls, error) {
	elasticData, err := m.searchService.Search(ctx, accountId, query, afterId, size)
	if err != nil {
		return nil, fmt.Errorf("search service failed: %w", err)
	}

	results, err := m.constructMetadataWithUrls(ctx, elasticData)
	if err != nil {
		return results, err
	}

	return results, nil
}

func (m *MemeCrudServiceImpl) DeleteMeme(ctx context.Context, accountId uuid.UUID, id uuid.UUID) error {
	metadata, err := m.metadataStorageService.GetById(ctx, accountId, id)
	if err != nil {
		return fmt.Errorf("get metadata failed: %w", err)
	}
	if metadata == nil {
		return fmt.Errorf("meme not found: %s", id)
	}

	if err = m.imageStorageService.DeleteImage(ctx, metadata.S3Id); err != nil {
		return fmt.Errorf("delete image failed: %w", err)
	}

	if err = m.metadataStorageService.DeleteById(ctx, accountId, id); err != nil {
		return fmt.Errorf("delete metadata failed: %w", err)
	}

	return nil
}

func (m *MemeCrudServiceImpl) constructMetadataWithUrls(ctx context.Context, elasticData []*entity.ElasticImageMetaData) ([]*MetadataWithUrls, error) {
	results := make([]*MetadataWithUrls, len(elasticData))

	for i, elasticDataObject := range elasticData {

		urlOriginal, err := m.imageStorageService.GetUrl(ctx, elasticDataObject.ImageId)
		if err != nil {
			return nil, fmt.Errorf("get original url by %s failed: %w", elasticDataObject.ImageId, err)
		}

		urlThumb, err := m.imageStorageService.GetUrlThumb(ctx, elasticDataObject.ImageId)
		if err != nil {
			return nil, fmt.Errorf("get thumbnail url by %s failed: %w", elasticDataObject.ImageId, err)
		}

		results[i] = &MetadataWithUrls{
			Metadata:    elasticDataObject,
			UrlThumb:    urlThumb,
			UrlOriginal: urlOriginal,
		}
	}
	return results, nil
}

func NewMemeCrudService(
	imageStore ImageStorageService,
	metadataStore MetadataStorageService,
	imageMetadataExtract ImageMetadataExtractService,
	searchService SearchService,
) MemeCrudService {
	return &MemeCrudServiceImpl{
		imageStorageService:    imageStore,
		metadataStorageService: metadataStore,
		imageExtractService:    imageMetadataExtract,
		searchService:          searchService,
		slogger:                slog.With("service", "MemeCrudService"),
	}
}
