package ocr

import (
	"context"
	"time"

	"github.com/weoses/memelo/common/temp"
)

type VideoSlicer interface {
	SliceVideoWithOverlap(ctx context.Context, video temp.Data, interval time.Duration, overlap time.Duration) ([]temp.Data, error)
}
