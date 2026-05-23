package ocr

import (
	"context"

	"github.com/weoses/memelo/common/temp"
)

type Video2AudioConverter interface {
	ExtractAudio(ctx context.Context, video temp.Data) (temp.Data, error)
	CutAudio(ctx context.Context, video temp.Data) (temp.Data, error)
}
