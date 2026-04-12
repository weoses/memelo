package temp

import (
	"context"
	"io"
)

const MaxInmemSize = 1 * 1024 * 1024

type Data interface {
	io.Closer
	Size() (int64, error)
	Reader() (io.ReadCloser, error)
	ReadAll() ([]byte, error)
}

type S3BackedData interface {
	Data
	IsGsSupported() bool
	GetS3Path(ctx context.Context) (string, error)
	GetS3Url(ctx context.Context) (string, error)
}
