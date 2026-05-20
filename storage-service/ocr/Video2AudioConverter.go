package ocr

import (
	"context"

	"github.com/weoses/memelo/common/temp"
)

type Video2AudioConverter interface {
	ConvertToMp3(ctx context.Context, video temp.Data) (temp.Data, error)
}
