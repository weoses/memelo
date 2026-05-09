package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/storage-service/entity"
	storage2 "github.com/weoses/memelo/storage-service/storage"
)

type PipelineSaveConfig struct {
	SaveHash      bool
	SaveExtractor bool
	SaveArtifacts bool
	SaveEmbedding bool
}

type PipelineSaveService interface {
	Save(ctx context.Context, target *entity.ElasticImageMetaData, pipelineResult *MetadataPipelineContext, config PipelineSaveConfig) error
}

type PipelineSaveServiceImpl struct {
	imageStorageService    storage2.MediaStorageService
	metadataStorageService storage2.MetadataStorageService
	slogger                *slog.Logger
}

func (p *PipelineSaveServiceImpl) Save(ctx context.Context, target *entity.ElasticImageMetaData, pipelineResult *MetadataPipelineContext, config PipelineSaveConfig) error {
	if config.SaveHash {
		target.Hash = pipelineResult.Hash
	}

	if config.SaveExtractor && pipelineResult.Result != nil {
		target.Result = fmt.Sprintf("%s %s %s",
			pipelineResult.Result.OnScreenText,
			pipelineResult.Result.AudioTranscript,
			pipelineResult.Result.AudioTrack)
		target.ResultData = pipelineResult.Result
		target.ResultPerVideoSlices = pipelineResult.ResultPerVideoSlices
	}

	if config.SaveArtifacts {
		for i := range pipelineResult.StorageArtifacts {
			artifact := pipelineResult.StorageArtifacts[i]
			err := p.imageStorageService.Save(ctx, target.S3Id, storageMediaType(target.Type, artifact.Type), artifact.Data)
			if err != nil {
				return fmt.Errorf("save artifact with type %s failed: %w", artifact.Type, err)
			}
		}
		if pipelineResult.OriginalSize.Width > 0 {
			target.ImageSize = &entity.Sizes{
				Width:  pipelineResult.OriginalSize.Width,
				Height: pipelineResult.OriginalSize.Height,
			}
		}
		if pipelineResult.ThumbnailSize.Width > 0 {
			target.ThumbSize = &entity.Sizes{
				Width:  pipelineResult.ThumbnailSize.Width,
				Height: pipelineResult.ThumbnailSize.Height,
			}
		}
	}

	if config.SaveEmbedding && len(pipelineResult.Embedding) > 0 {
		target.EmbeddingList = []entity.EmbeddingItem{}
		for _, v := range pipelineResult.Embedding {
			if len(v.Data) >= 1 {
				target.EmbeddingList = append(target.EmbeddingList, v)
			}
		}
	}

	if len(pipelineResult.Tags) > 0 {
		target.Tags = helper.TransformSlice(
			pipelineResult.Tags,
			make([]string, len(pipelineResult.Tags)),
			func(tag entity.ElasticTag) string { return tag.Tag })
	}

	if err := p.metadataStorageService.Save(ctx, target); err != nil {
		return fmt.Errorf("save metadata failed: %w", err)
	}

	return nil
}

func NewPipelineSaveService(
	imageStorageService storage2.MediaStorageService,
	metadataStorageService storage2.MetadataStorageService,
) PipelineSaveService {
	return &PipelineSaveServiceImpl{
		imageStorageService:    imageStorageService,
		metadataStorageService: metadataStorageService,
		slogger:                slog.With("service", "PipelineSaveService"),
	}
}
