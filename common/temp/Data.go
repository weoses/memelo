package temp

import (
	"context"
	"io"
)

const MaxInmemSize = 1 * 1024 * 1024

type Data interface {
	Size(ctx context.Context) (int64, error)
	Reader(ctx context.Context) (io.ReadCloser, error)
	ReadAll(ctx context.Context) ([]byte, error)
	Close(ctx context.Context) error
}

type S3BackedData interface {
	Data
	IsGsSupported() bool
	GetS3Path(ctx context.Context) (string, error)
	GetS3Url(ctx context.Context) (string, error)
	GetPresignedUrl(ctx context.Context) (string, error)
}
