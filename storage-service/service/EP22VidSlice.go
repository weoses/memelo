package service

import (
	"context"
	"fmt"
	"time"

	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

const videoSliceOverlap = 2 * time.Second

type VidSlicePipelineStep struct {
	BasePipelineStep

	slicer         ocr.VideoSlicer
	tmpDataService commonservice.TmpDataService
	cfg            *conf.CommonExtractingConfig
}

func (s *VidSlicePipelineStep) Do(ctx context.Context, _ MetadataInputContext, pCtx *MetadataPipelineContext) error {
	if pCtx.VideoMp4 == nil {
		return nil
	}

	interval := time.Duration(s.cfg.VideoSliceIntervalSec) * time.Second
	slices, err := s.slicer.SliceVideoWithOverlap(ctx, pCtx.VideoMp4, interval, videoSliceOverlap)
	if err != nil {
		return fmt.Errorf("VidSlicePipelineStep: cannot slice video: %w", err)
	}

	for i, slice := range slices {
		wrapped, errWrap := s.tmpDataService.WrapData(ctx, "video/mp4", slice.Data)
		if errWrap != nil {
			for _, remaining := range slices[i:] {
				_ = remaining.Data.Close()
			}
			return fmt.Errorf("VidSlicePipelineStep: cannot wrap slice %d: %w", i, errWrap)
		}
		pCtx.VideoSlices = append(pCtx.VideoSlices, VideoSlice{
			SliceNumber:    i,
			SliceStartTime: slice.StartTime,
			SliceEndTime:   slice.EndTime,
			Slice:          wrapped,
		})
	}
	return nil
}

func NewVidSlicePipelineStep(slicer ocr.VideoSlicer, tmpDataService commonservice.TmpDataService, cfg *conf.CommonExtractingConfig) ExtractPipelineStep {
	return &VidSlicePipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 22,
			typ: []entity.MetadataType{entity.VideoMetadataType},
		},
		slicer:         slicer,
		tmpDataService: tmpDataService,
		cfg:            cfg,
	}
}
