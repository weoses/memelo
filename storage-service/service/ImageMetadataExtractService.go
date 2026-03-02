package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

type PipelineContext struct {
	AccountId          uuid.UUID
	ImageHash          string
	ImageEmbedding     *entity.ElasticEmbeddingV1
	ImageOcrResult     string
	ImageThumbnail     []byte
	ImageThumbnailSize *entity.ElasticSizes
	ImageRaw           []byte
	ImageRawSize       *entity.ElasticSizes
	Duplicate          *entity.ElasticImageMetaData
}

type ImageMetadataExtractService interface {
	ProcessCreate(ctx context.Context, accountId uuid.UUID, raw []byte) (*PipelineContext, error)
}

type CreateImageServiceImpl struct {
	metadata       MetadataStorageService
	imageConverter ocr.ImageConveter
	imageEmbedder  ocr.EmbeddingExtractor
	image2text     ocr.Img2TextService
}

func (c *CreateImageServiceImpl) ProcessCreate(ctx context.Context, accountId uuid.UUID, raw []byte) (*PipelineContext, error) {
	pipelineCtx := &PipelineContext{
		AccountId: accountId,
		ImageRaw:  raw,
	}

	err := c.calcHash(ctx, pipelineCtx)
	if err != nil {
		return nil, fmt.Errorf("create pipeline: error calculating hash: %w", err)
	}

	err = c.checkDuplicateByHash(ctx, pipelineCtx)
	if err != nil {
		return nil, fmt.Errorf("create pipeline: error checking duplicate by hash: %w", err)
	}

	if pipelineCtx.Duplicate != nil {
		return pipelineCtx, nil
	}

	err = c.toJpeg(ctx, pipelineCtx)
	if err != nil {
		return nil, fmt.Errorf("create pipeline: error converting jpeg: %w", err)
	}

	err = c.calcEmbedding(ctx, pipelineCtx)
	if err != nil {
		return nil, fmt.Errorf("create pipeline: error calculating embedding: %w", err)
	}

	err = c.checkDuplicateByEmbedding(ctx, pipelineCtx)
	if err != nil {
		return nil, fmt.Errorf("create pipeline: error checking duplicate by embedding: %w", err)
	}

	if pipelineCtx.Duplicate != nil {
		return pipelineCtx, nil
	}

	err = c.ocrImage(ctx, pipelineCtx)
	if err != nil {
		return nil, fmt.Errorf("create pipeline: error ocring image: %w", err)
	}

	err = c.createThumbnail(ctx, pipelineCtx)
	if err != nil {
		return nil, fmt.Errorf("create pipeline: error creating thumbnail: %w", err)
	}

	err = c.calcSizes(ctx, pipelineCtx)
	if err != nil {
		return nil, fmt.Errorf("create pipeline: error calculating sizes: %w", err)
	}

	return pipelineCtx, nil
}

func (c *CreateImageServiceImpl) toJpeg(ctx context.Context, pipelineCtx *PipelineContext) error {
	imgJpeg, err := c.imageConverter.ProcessOriginalImage(ctx, pipelineCtx.ImageRaw)
	if err != nil {
		return fmt.Errorf("cannot process original_image: %w", err)
	}
	pipelineCtx.ImageRaw = imgJpeg
	return nil
}

func (c *CreateImageServiceImpl) createThumbnail(ctx context.Context, pipelineCtx *PipelineContext) error {
	imgThumb, err := c.imageConverter.MakeThumbnail(ctx, pipelineCtx.ImageRaw)
	if err != nil {
		return fmt.Errorf("cannot process original_image: %w", err)
	}
	pipelineCtx.ImageThumbnail = imgThumb
	return nil
}

func (c *CreateImageServiceImpl) calcHash(ctx context.Context, pipelineCtx *PipelineContext) error {
	hash, err := calcRawImageHash(pipelineCtx.ImageRaw)
	if err != nil {
		return fmt.Errorf("error calculating raw image hash: %w", err)
	}
	pipelineCtx.ImageHash = hash
	return nil
}

func (c *CreateImageServiceImpl) calcEmbedding(ctx context.Context, pipelineCtx *PipelineContext) error {
	embeddingNew, err := c.imageEmbedder.GetImageEmbeddingV1(ctx, pipelineCtx.ImageRaw)
	if err != nil {
		return fmt.Errorf("error getting embedding image: %w", err)
	}
	pipelineCtx.ImageEmbedding = embeddingNew
	return nil
}

func (c *CreateImageServiceImpl) checkDuplicateByHash(ctx context.Context, pipelineCtx *PipelineContext) error {
	items, err := c.metadata.GetByHash(ctx, pipelineCtx.AccountId, pipelineCtx.ImageHash, helper.Addr(1))
	if err != nil {
		return fmt.Errorf("error getting items by hash: %w", err)
	}

	if len(items) > 0 {
		pipelineCtx.Duplicate = items[0]
	}

	return nil
}

func (c *CreateImageServiceImpl) checkDuplicateByEmbedding(ctx context.Context, pipelineCtx *PipelineContext) error {
	items, err := c.metadata.SearchByEmbeddingV1(ctx, pipelineCtx.AccountId, pipelineCtx.ImageEmbedding, 1, true)

	if err != nil {
		return fmt.Errorf("error getting items by embedding: %w", err)
	}

	if len(items) > 0 {
		pipelineCtx.Duplicate = items[0]
	}

	return nil
}

func (c *CreateImageServiceImpl) ocrImage(ctx context.Context, pipelineCtx *PipelineContext) error {
	ocrResult, err := c.image2text.DoOcr(ctx, pipelineCtx.ImageRaw)
	if err != nil {
		return fmt.Errorf("error ocring image: %w", err)
	}
	pipelineCtx.ImageOcrResult = ocrResult
	return nil
}

func (c *CreateImageServiceImpl) calcSizes(ctx context.Context, pipelineCtx *PipelineContext) error {
	wRaw, hRaw, err := c.imageConverter.GetSize(ctx, pipelineCtx.ImageRaw)
	if err != nil {
		return fmt.Errorf("error getting size raw image: %w", err)
	}

	wThumb, hThumb, err := c.imageConverter.GetSize(ctx, pipelineCtx.ImageThumbnail)
	if err != nil {
		return fmt.Errorf("error getting size raw image: %w", err)
	}

	pipelineCtx.ImageRawSize = &entity.ElasticSizes{
		Width:  wRaw,
		Height: hRaw,
	}

	pipelineCtx.ImageThumbnailSize = &entity.ElasticSizes{
		Width:  wThumb,
		Height: hThumb,
	}

	return nil
}

func calcRawImageHash(raw []byte) (string, error) {
	base64DataBuffer := bytes.NewBuffer(make([]byte, 0))
	encoder := base64.NewEncoder(base64.RawStdEncoding, base64DataBuffer)
	_, err := encoder.Write(raw)
	if err != nil {
		return "", fmt.Errorf("failed to write raw data to base64 encode buffer: %w", err)
	}

	hash := helper.CalcHash(base64DataBuffer.String())
	return hash, nil
}

func NewImageMetadataExtractService(
	metadata MetadataStorageService,
	imageConverter ocr.ImageConveter,
	imageEmbedder ocr.EmbeddingExtractor,
	image2text ocr.Img2TextService,
) ImageMetadataExtractService {
	return &CreateImageServiceImpl{
		metadata:       metadata,
		imageConverter: imageConverter,
		imageEmbedder:  imageEmbedder,
		image2text:     image2text,
	}
}
