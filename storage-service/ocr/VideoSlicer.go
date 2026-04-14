package ocr

import (
	"context"
	"time"

	"github.com/weoses/memelo/common/temp"
)

type VideoSlicer interface {
	SliceVideo(ctx context.Context, video temp.Data, interval time.Duration) ([]temp.Data, error)
}
