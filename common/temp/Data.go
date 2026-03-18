package temp

import (
	"context"
	"io"
)

const MaxInmemSize = 1 * 1024 * 1024

type Data interface {
	io.Closer
	Reader() (io.ReadCloser, error)
	ReadAll() ([]byte, error)
}

type S3BackedData interface {
	Data
	GetS3Path(ctx context.Context) (string, error)
}
