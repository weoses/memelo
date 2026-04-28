package service

import (
	"context"
	"fmt"

	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

// MediaEncoder encodes media for a specific type (image or video).
type MediaEncoder interface {
	// EncodeOriginal re-encodes raw input to the canonical format (JPEG or MP4) and returns the result with its dimensions.
	EncodeOriginal(ctx context.Context, raw temp.S3BackedData) (temp.S3BackedData, entity.Sizes, error)
	// EncodeThumbnail produces a resized JPEG thumbnail from the raw input and returns the result with its dimensions.
	EncodeThumbnail(ctx context.Context, raw temp.S3BackedData) (temp.S3BackedData, entity.Sizes, error)
}

// MediaEncoderService is a factory that returns the right encoder for a given media type.
type MediaEncoderService interface {
	GetEncoderByType(typ entity.MetadataType) MediaEncoder
}

// --- factory ---

type MediaEncoderServiceImpl struct {
	imageEncoder MediaEncoder
	videoEncoder MediaEncoder
}

func (s *MediaEncoderServiceImpl) GetEncoderByType(typ entity.MetadataType) MediaEncoder {
	if typ == entity.VideoMetadataType {
		return s.videoEncoder
	}
	return s.imageEncoder
}

func NewMediaEncoderService(
	imageConverter ocr.ImageConveter,
	videoConverter ocr.Video2Mp4Converter,
	frameExtractor ocr.Video2FrameExtractor,
	tmpDataService commonservice.TmpDataService,
) MediaEncoderService {
	return &MediaEncoderServiceImpl{
		imageEncoder: &ImageEncoderImpl{imageConverter: imageConverter, tmpDataService: tmpDataService},
		videoEncoder: &VideoEncoderImpl{videoConverter: videoConverter, frameExtractor: frameExtractor, imageConverter: imageConverter, tmpDataService: tmpDataService},
	}
}

// --- image encoder ---

type ImageEncoderImpl struct {
	imageConverter ocr.ImageConveter
	tmpDataService commonservice.TmpDataService
}

func (e *ImageEncoderImpl) EncodeOriginal(ctx context.Context, raw temp.S3BackedData) (temp.S3BackedData, entity.Sizes, error) {
	jpeg, err := e.imageConverter.Convert2Jpeg(ctx, raw)
	if err != nil {
		return nil, entity.Sizes{}, fmt.Errorf("ImageEncoder.EncodeOriginal: convert to jpeg: %w", err)
	}
	s3Jpeg, err := e.tmpDataService.WrapData(ctx, "image/jpeg", jpeg)
	if err != nil {
		return nil, entity.Sizes{}, fmt.Errorf("ImageEncoder.EncodeOriginal: wrap data: %w", err)
	}
	w, h, err := e.imageConverter.GetSize(ctx, s3Jpeg)
	if err != nil {
		return nil, entity.Sizes{}, fmt.Errorf("ImageEncoder.EncodeOriginal: get size: %w", err)
	}
	return s3Jpeg, entity.Sizes{Width: w, Height: h}, nil
}

func (e *ImageEncoderImpl) EncodeThumbnail(ctx context.Context, raw temp.S3BackedData) (temp.S3BackedData, entity.Sizes, error) {
	jpeg, err := e.imageConverter.Convert2Jpeg(ctx, raw)
	if err != nil {
		return nil, entity.Sizes{}, fmt.Errorf("ImageEncoder.EncodeThumbnail: convert to jpeg: %w", err)
	}
	thumb, err := e.imageConverter.MakeThumbnail(ctx, jpeg)
	if err != nil {
		return nil, entity.Sizes{}, fmt.Errorf("ImageEncoder.EncodeThumbnail: make thumbnail: %w", err)
	}
	s3Thumb, err := e.tmpDataService.WrapData(ctx, "image/jpeg", thumb)
	if err != nil {
		return nil, entity.Sizes{}, fmt.Errorf("ImageEncoder.EncodeThumbnail: wrap data: %w", err)
	}
	w, h, err := e.imageConverter.GetSize(ctx, s3Thumb)
	if err != nil {
		return nil, entity.Sizes{}, fmt.Errorf("ImageEncoder.EncodeThumbnail: get size: %w", err)
	}
	return s3Thumb, entity.Sizes{Width: w, Height: h}, nil
}

// --- video encoder ---

type VideoEncoderImpl struct {
	videoConverter ocr.Video2Mp4Converter
	frameExtractor ocr.Video2FrameExtractor
	imageConverter ocr.ImageConveter
	tmpDataService commonservice.TmpDataService
}

func (e *VideoEncoderImpl) EncodeOriginal(ctx context.Context, raw temp.S3BackedData) (temp.S3BackedData, entity.Sizes, error) {
	mp4, err := e.videoConverter.ConvertToMp4(ctx, raw)
	if err != nil {
		return nil, entity.Sizes{}, fmt.Errorf("VideoEncoder.EncodeOriginal: convert to mp4: %w", err)
	}
	s3Mp4, err := e.tmpDataService.WrapData(ctx, "video/mp4", mp4)
	if err != nil {
		return nil, entity.Sizes{}, fmt.Errorf("VideoEncoder.EncodeOriginal: wrap data: %w", err)
	}
	frame, err := e.frameExtractor.ExtractOneFrame(ctx, s3Mp4)
	if err != nil {
		return nil, entity.Sizes{}, fmt.Errorf("VideoEncoder.EncodeOriginal: extract frame for size: %w", err)
	}
	w, h, err := e.imageConverter.GetSize(ctx, frame)
	if err != nil {
		return nil, entity.Sizes{}, fmt.Errorf("VideoEncoder.EncodeOriginal: get frame size: %w", err)
	}
	return s3Mp4, entity.Sizes{Width: w, Height: h}, nil
}

func (e *VideoEncoderImpl) EncodeThumbnail(ctx context.Context, raw temp.S3BackedData) (temp.S3BackedData, entity.Sizes, error) {
	frame, err := e.frameExtractor.ExtractOneFrame(ctx, raw)
	if err != nil {
		return nil, entity.Sizes{}, fmt.Errorf("VideoEncoder.EncodeThumbnail: extract frame: %w", err)
	}
	thumb, err := e.imageConverter.MakeThumbnail(ctx, frame)
	if err != nil {
		return nil, entity.Sizes{}, fmt.Errorf("VideoEncoder.EncodeThumbnail: make thumbnail: %w", err)
	}
	s3Thumb, err := e.tmpDataService.WrapData(ctx, "image/jpeg", thumb)
	if err != nil {
		return nil, entity.Sizes{}, fmt.Errorf("VideoEncoder.EncodeThumbnail: wrap data: %w", err)
	}
	w, h, err := e.imageConverter.GetSize(ctx, s3Thumb)
	if err != nil {
		return nil, entity.Sizes{}, fmt.Errorf("VideoEncoder.EncodeThumbnail: get size: %w", err)
	}
	return s3Thumb, entity.Sizes{Width: w, Height: h}, nil
}
