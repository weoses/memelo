package ocr

import (
	"context"

	"github.com/weoses/memelo/common/temp"
)

type Video2Mp4Converter interface {
	ConvertToMp4(ctx context.Context, video temp.Data) (temp.Data, error)
}
