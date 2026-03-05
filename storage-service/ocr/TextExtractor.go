package ocr

import "context"

type TextExtractor interface {
	GetName() string
	DoOcr(ctx context.Context, image []byte) (string, error)
}
