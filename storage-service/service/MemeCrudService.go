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

var ErrMemeNotFound = errors.New("meme not found")

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

type UpdateMemeInput struct {
	Id              uuid.UUID
	AccountId       uuid.UUID
	OnScreenText    *string
	AudioTranscript *string
	Caption         *string
	AudioTrack      *string
	Original        temp.S3BackedData
	Thumbnail       temp.S3BackedData
}

type MemeCrudService interface {
	SearchMeme(ctx context.Context, accountId uuid.UUID, query string, metadataType *entity.MetadataType, afterId *PipelineAfterID, size int) (*SearchResult, error)
	CreateMeme(ctx context.Context, accountId uuid.UUID, typ entity.MetadataType, raw temp.S3BackedData) (*CreateResult, error)
	GetMeme(ctx context.Context, accountId uuid.UUID, id uuid.UUID) (*MetadataWithUrls, error)
	UpdateMeme(ctx context.Context, input UpdateMemeInput) (*MetadataWithUrls, error)
	DeleteMeme(ctx context.Context, accountId uuid.UUID, id uuid.UUID) error
	DeleteAll(ctx context.Context, accountId uuid.UUID) error
	GetRandomMeme(ctx context.Context, accountId uuid.UUID, mediaType entity.MetadataType) (*MetadataWithUrls, error)
}

type MemeCrudServiceImpl struct {
	imageStorageService    storage2.MediaStorageService
	metadataStorageService storage2.MetadataStorageService
	metadataExtractService MetadataExtractService
	pipelineSaveService    PipelineSaveService
	searchService          SearchService
	encoder                MediaEncoderService
	slogger                *slog.Logger
}

func (m *MemeCrudServiceImpl) CreateMeme(ctx context.Context, accountId uuid.UUID, mediaType entity.MetadataType, raw temp.S3BackedData) (*CreateResult, error) {
	pipelineResult, err := m.metadataExtractService.Extract(ctx, MetadataInputContext{
		AccountId: accountId,
		Type:      mediaType,
		RawInput:  raw,
		StepCallback: func(ctx context.Context, stepName string, pipelineContext *MetadataPipelineContext) bool {
			return pipelineContext.Duplicate == nil
		},
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
	metadataEntity := &entity.ElasticImageMetaData{
		ImageId:   imgId,
		Type:      mediaType,
		S3Id:      s3id,
		AccountId: accountId,
		Created:   time.Now().UnixMicro(),
		Updated:   time.Now().UnixMicro(),
	}
	err = m.pipelineSaveService.Save(ctx, metadataEntity, pipelineResult, PipelineSaveConfig{
		SaveHash: true, SaveExtractor: true, SaveArtifacts: true, SaveEmbedding: true,
	})
	if err != nil {
		return nil, fmt.Errorf("pipeline save failed: %w", err)
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

func (m *MemeCrudServiceImpl) SearchMeme(ctx context.Context, accountId uuid.UUID, query string, metadataType *entity.MetadataType, afterId *PipelineAfterID, size int) (*SearchResult, error) {
	elasticData, err := m.searchService.Search(ctx, accountId, query, metadataType, afterId, size)
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

func (m *MemeCrudServiceImpl) UpdateMeme(ctx context.Context, input UpdateMemeInput) (*MetadataWithUrls, error) {
	existing, err := m.metadataStorageService.GetById(ctx, input.AccountId, input.Id)
	if err != nil {
		return nil, fmt.Errorf("get metadata failed: %w", err)
	}
	if existing == nil {
		return nil, ErrMemeNotFound
	}

	if existing.ResultData == nil {
		existing.ResultData = &entity.Result{}
	}

	llmChanged := false
	if input.OnScreenText != nil {
		existing.ResultData.OnScreenText = *input.OnScreenText
		llmChanged = true
	}
	if input.AudioTranscript != nil {
		existing.ResultData.AudioTranscript = *input.AudioTranscript
		llmChanged = true
	}
	if input.Caption != nil {
		existing.ResultData.Caption = *input.Caption
	}
	if input.AudioTrack != nil {
		existing.ResultData.AudioTrack = *input.AudioTrack
		llmChanged = true
	}

	if input.Original != nil {
		encoded, size, err := m.encoder.GetEncoderByType(existing.Type).EncodeOriginal(ctx, input.Original)
		if err != nil {
			return nil, fmt.Errorf("encode original failed: %w", err)
		}
		defer helper.QuietClose(encoded, m.slogger)

		err = m.imageStorageService.Save(ctx, existing.S3Id, storageMediaType(existing.Type, SavedOriginal), encoded)
		if err != nil {
			return nil, fmt.Errorf("save encoded original failed: %w", err)
		}
		existing.ImageSize = &size
	}

	if input.Thumbnail != nil {
		// thumbnails are always images regardless of meme type
		thumb, size, err := m.encoder.GetEncoderByType(entity.ImageMetadataType).EncodeThumbnail(ctx, input.Thumbnail)
		if err != nil {
			return nil, fmt.Errorf("encode thumbnail failed: %w", err)
		}
		defer helper.QuietClose(thumb, m.slogger)

		err = m.imageStorageService.Save(ctx, existing.S3Id, storageMediaType(existing.Type, SavedThumb), thumb)
		if err != nil {
			return nil, fmt.Errorf("save thumbnail failed: %w", err)
		}
		existing.ThumbSize = &size
	}

	if llmChanged {
		existing.Result = fmt.Sprintf("%s %s %s",
			existing.ResultData.OnScreenText,
			existing.ResultData.AudioTranscript,
			existing.ResultData.AudioTrack)
	}

	existing.Edited = true

	if err = m.metadataStorageService.Save(ctx, existing); err != nil {
		return nil, fmt.Errorf("save metadata failed: %w", err)
	}

	results, err := m.constructMetadataWithUrls(ctx, []*entity.ElasticImageMetaData{existing})
	if err != nil {
		return nil, fmt.Errorf("add urls to elastic entities failed: %w", err)
	}
	return results[0], nil
}

func (m *MemeCrudServiceImpl) GetMeme(ctx context.Context, accountId uuid.UUID, id uuid.UUID) (*MetadataWithUrls, error) {
	existing, err := m.metadataStorageService.GetById(ctx, accountId, id)
	if err != nil {
		return nil, fmt.Errorf("get metadata failed: %w", err)
	}
	if existing == nil {
		return nil, ErrMemeNotFound
	}
	results, err := m.constructMetadataWithUrls(ctx, []*entity.ElasticImageMetaData{existing})
	if err != nil {
		return nil, fmt.Errorf("add urls failed: %w", err)
	}
	return results[0], nil
}

func (m *MemeCrudServiceImpl) GetRandomMeme(ctx context.Context, accountId uuid.UUID, mediaType entity.MetadataType) (*MetadataWithUrls, error) {
	var metadataType *entity.MetadataType
	if mediaType != "" {
		metadataType = &mediaType
	}
	result, err := m.metadataStorageService.GetRandom(ctx, accountId, metadataType)
	if err != nil {
		return nil, fmt.Errorf("GetRandomMeme: %w", err)
	}
	if result == nil {
		return nil, ErrMemeNotFound
	}
	results, err := m.constructMetadataWithUrls(ctx, []*entity.ElasticImageMetaData{result})
	if err != nil {
		return nil, fmt.Errorf("GetRandomMeme: add urls failed: %w", err)
	}
	return results[0], nil
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
		results, nextKey, err := m.metadataStorageService.GetByAccountIdOrderByCreated(ctx, accountId, nil, sortKey, pageSize)
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
	pipelineSaveService PipelineSaveService,
	searchService SearchService,
	encoder MediaEncoderService,
) MemeCrudService {
	return &MemeCrudServiceImpl{
		imageStorageService:    imageStore,
		metadataStorageService: metadataStore,
		metadataExtractService: imageMetadataExtract,
		pipelineSaveService:    pipelineSaveService,
		searchService:          searchService,
		encoder:                encoder,
		slogger:                slog.With("service", "MemeCrudService"),
	}
}
