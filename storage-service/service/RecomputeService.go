package service

import (
	"context"
	"log/slog"
)

type ProgressDataRecompute struct {
	processed int
}

type RecomputeService interface {
	RecomputeOcrData(ctx context.Context, callback func(ctx context.Context, recompute ProgressDataRecompute) error) error
}

type RecomputeServiceImpl struct {
	slogger         *slog.Logger
	extractService  ImageMetadataExtractService
	metadataService MetadataStorageService
	imageService    ImageStorageService
}

func (r RecomputeServiceImpl) RecomputeOcrData(ctx context.Context, callback func(ctx context.Context, recompute ProgressDataRecompute) error) error {
	panic("implement me")
}

func NewRecomputeService(extractService ImageMetadataExtractService) RecomputeService {
	return &RecomputeServiceImpl{
		slogger:        slog.With("service", "RecomputeService"),
		extractService: extractService,
	}
}
