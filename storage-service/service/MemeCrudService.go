package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"

	"github.com/weoses/memelo/storage-service/entity"
	storage2 "github.com/weoses/memelo/storage-service/storage"
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

type SearchResult struct {
	Result  []*MetadataWithUrls
	AfterID *PipelineAfterID
}

type MemeCrudService interface {
	SearchMeme(ctx context.Context, accountId uuid.UUID, query string, afterId *PipelineAfterID, size int) (*SearchResult, error)
	CreateMeme(ctx context.Context, accountId uuid.UUID, typ entity.MetadataType, raw temp.S3BackedData) (*CreateResult, error)
	DeleteMeme(ctx context.Context, accountId uuid.UUID, id uuid.UUID) error
	DeleteAll(ctx context.Context, accountId uuid.UUID) error
}

type MemeCrudServiceImpl struct {
	imageStorageService    storage2.MediaStorageService
	metadataStorageService storage2.MetadataStorageService
	metadataExtractService MetadataExtractService
	searchService          SearchService
	slogger                *slog.Logger
}

func (m *MemeCrudServiceImpl) CreateMeme(ctx context.Context, accountId uuid.UUID, mediaType entity.MetadataType, raw temp.S3BackedData) (*CreateResult, error) {
	pipelineResult, err := m.metadataExtractService.Extract(ctx, MetadataInputContext{
		AccountId:        accountId,
		Type:             mediaType,
		RawInput:         raw,
		CheckDuplicates:  true,
		ComputeHash:      true,
		ComputeExtractor: true,
		ComputeEmbedding: true,
	})
	if err != nil {
		return nil, fmt.Errorf("metadataService extract pipeline failed: %w", err)
	}
	defer helper.QuietClose(pipelineResult, m.slogger)

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
	for i := range pipelineResult.StorageArtifacts {
		artifact := pipelineResult.StorageArtifacts[i]
		err = m.imageStorageService.Save(ctx, s3id, storageMediaType(mediaType, artifact.Type), artifact.Data)
		if err != nil {
			return nil, fmt.Errorf("save artifact with type %s failed: %w", artifact.Type, err)
		}
	}

	joinedResult := fmt.Sprintf("%s %s %s",
		pipelineResult.Result.OnScreenText,
		pipelineResult.Result.AudioTranscript,
		pipelineResult.Result.AudioTrack)

	metadataEntity := &entity.ElasticImageMetaData{
		ImageId:              imgId,
		Type:                 mediaType,
		S3Id:                 s3id,
		AccountId:            accountId,
		Result:               joinedResult,
		ResultData:           pipelineResult.Result,
		Hash:                 pipelineResult.Hash,
		EmbeddingList:        pipelineResult.Embedding,
		ImageSize:            &pipelineResult.OriginalSize,
		ThumbSize:            &pipelineResult.ThumbnailSize,
		Created:              time.Now().UnixMicro(),
		Updated:              time.Now().UnixMicro(),
		ResultPerVideoSlices: pipelineResult.ResultPerVideoSlices,
		Tags: helper.TransformSlice(
			pipelineResult.Tags,
			make([]string, len(pipelineResult.Tags)),
			func(tag entity.ElasticTag) string {
				return tag.Tag
			}),
	}
	err = m.metadataStorageService.Save(ctx, metadataEntity)
	if err != nil {
		return nil, fmt.Errorf("save metadataService failed: %w", err)
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

func (m *MemeCrudServiceImpl) SearchMeme(ctx context.Context, accountId uuid.UUID, query string, afterId *PipelineAfterID, size int) (*SearchResult, error) {
	elasticData, err := m.searchService.Search(ctx, accountId, query, afterId, size)
	if err != nil {
		return nil, fmt.Errorf("search_pipeline service failed: %w", err)
	}

	results, err := m.constructMetadataWithUrls(ctx, elasticData.Result)
	if err != nil {
		return nil, err
	}

	return &SearchResult{
		Result:  results,
		AfterID: elasticData.AfterID,
	}, nil
}

func (m *MemeCrudServiceImpl) DeleteMeme(ctx context.Context, accountId uuid.UUID, id uuid.UUID) error {
	metadata, err := m.metadataStorageService.GetById(ctx, accountId, id)
	if err != nil {
		return fmt.Errorf("get metadataService failed: %w", err)
	}
	if metadata == nil {
		return fmt.Errorf("meme not found: %s", id)
	}

	err1 := m.imageStorageService.Delete(ctx, metadata.S3Id, storageMediaType(metadata.Type, SavedOriginal))
	err2 := m.imageStorageService.Delete(ctx, metadata.S3Id, storageMediaType(metadata.Type, SavedThumb))
	if err := errors.Join(err1, err2); err != nil {
		return fmt.Errorf("delete image failed: %w", err)
	}

	if err = m.metadataStorageService.DeleteById(ctx, accountId, id); err != nil {
		return fmt.Errorf("delete metadataService failed: %w", err)
	}

	return nil
}

func (m *MemeCrudServiceImpl) DeleteAll(ctx context.Context, accountId uuid.UUID) error {
	const pageSize = 100
	var sortKey entity.ElasticSortKey

	for {
		results, nextKey, err := m.metadataStorageService.GetByAccountIdOrderByCreated(ctx, accountId, sortKey, pageSize)
		if err != nil {
			return fmt.Errorf("list memes failed: %w", err)
		}

		for _, meta := range results {
			errMinioOrig := m.imageStorageService.Delete(ctx, meta.S3Id, storageMediaType(meta.Type, SavedOriginal))
			errMinioThumb := m.imageStorageService.Delete(ctx, meta.S3Id, storageMediaType(meta.Type, SavedThumb))

			if errMinioOrig != nil {
				m.slogger.WarnContext(ctx, "delete s3 image failed", "s3Id", meta.S3Id, "error", errMinioOrig)
			}

			if errMinioThumb != nil {
				m.slogger.WarnContext(ctx, "delete s3 image failed", "s3Id", meta.S3Id, "error", errMinioThumb)
			}

			errElastic := m.metadataStorageService.DeleteById(ctx, accountId, meta.ImageId)
			if errElastic != nil {
				m.slogger.WarnContext(ctx, "delete elastic failed", "imageId", meta.ImageId, "error", errElastic)
			}
		}

		if len(results) < pageSize {
			break
		}

		sortKey = nextKey
	}
	return nil
}

func (m *MemeCrudServiceImpl) constructMetadataWithUrls(ctx context.Context, elasticData []*entity.ElasticImageMetaData) ([]*MetadataWithUrls, error) {
	results := make([]*MetadataWithUrls, len(elasticData))

	for i, elasticDataObject := range elasticData {

		urlOriginal, err := m.imageStorageService.GetUrl(ctx, elasticDataObject.S3Id, storageMediaType(elasticDataObject.Type, SavedOriginal))
		if err != nil {
			return nil, fmt.Errorf("get original url by %s failed: %w", elasticDataObject.ImageId, err)
		}

		urlThumb, err := m.imageStorageService.GetUrl(ctx, elasticDataObject.S3Id, storageMediaType(elasticDataObject.Type, SavedThumb))
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
	imageStore storage2.MediaStorageService,
	metadataStore storage2.MetadataStorageService,
	imageMetadataExtract MetadataExtractService,
	searchService SearchService,
) MemeCrudService {
	return &MemeCrudServiceImpl{
		imageStorageService:    imageStore,
		metadataStorageService: metadataStore,
		metadataExtractService: imageMetadataExtract,
		searchService:          searchService,
		slogger:                slog.With("service", "MemeCrudService"),
	}
}
