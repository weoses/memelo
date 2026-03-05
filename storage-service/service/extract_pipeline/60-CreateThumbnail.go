package extract_pipeline

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/ocr"
	"github.com/weoses/memelo/storage-service/service"
)

type CreateThumbnailPipelineStep struct {
	imageConverter ocr.ImageConveter
}

func (s *CreateThumbnailPipelineStep) GetPos() int {
	return 60
}

func (s *CreateThumbnailPipelineStep) Check(_ context.Context, steps *service.StepsToDo) (bool, error) {
	return steps.CreateThumbnail, nil
}

func (s *CreateThumbnailPipelineStep) Do(ctx context.Context, pCtx *service.PipelineContext) error {
	imgThumb, err := s.imageConverter.MakeThumbnail(ctx, pCtx.ImageRaw)
	if err != nil {
		return fmt.Errorf("cannot create thumbnail: %w", err)
	}
	pCtx.ImageThumbnail = &imgThumb
	return nil
}

func NewCreateThumbnailPipelineStep(imageConverter ocr.ImageConveter) service.PipelineStep {
	return &CreateThumbnailPipelineStep{imageConverter: imageConverter}
}
