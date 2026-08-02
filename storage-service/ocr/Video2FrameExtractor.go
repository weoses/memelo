package ocr

import (
	"context"

	"github.com/weoses/memelo/common/temp"
)

type Video2FrameExtractor interface {
	ExtractOneFrame(ctx context.Context, video temp.Data) (temp.Data, error)
}
