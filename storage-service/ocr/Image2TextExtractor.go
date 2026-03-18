package ocr

import (
	"context"

	"github.com/weoses/memelo/common/temp"
)

type Image2TextExtractor interface {
	GetName() string
	DoOcr(ctx context.Context, image temp.Data) (string, error)
}
