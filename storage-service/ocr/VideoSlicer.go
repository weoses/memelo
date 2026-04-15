package ocr

import (
	"context"
	"time"

	"github.com/weoses/memelo/common/temp"
)

type VideoSlice struct {
	Data      temp.Data
	StartTime int // seconds
	EndTime   int // seconds
}

type VideoSlicer interface {
	SliceVideoWithOverlap(ctx context.Context, video temp.Data, interval time.Duration, overlap time.Duration) ([]VideoSlice, error)
}
