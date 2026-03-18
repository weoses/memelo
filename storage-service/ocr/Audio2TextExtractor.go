package ocr

import (
	"context"

	"github.com/weoses/memelo/common/temp"
)

type Audio2TextExtractor interface {
	Transcript(ctx context.Context, audio temp.Data) (string, error)
}
