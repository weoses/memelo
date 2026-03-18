package ocr

import (
	"context"

	"github.com/weoses/memelo/common/temp"
)

type Video2AudioExtractor interface {
	ExtractAudio(ctx context.Context, video temp.Data) (temp.Data, error)
}
