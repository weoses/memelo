package ocr

import (
	"context"

	"github.com/weoses/memelo/common/temp"
)

type Video2FrameExtractor interface {
	ExtractFrames(ctx context.Context, video temp.Data) ([]temp.Data, error)
	ExtractThumbnail(ctx context.Context, video temp.Data) (temp.Data, error)
}
