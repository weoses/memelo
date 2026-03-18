package temp

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"
)

type s3wrappedData struct {
	physical Data
	once     sync.Once
	s3path   string
	upload   func(ctx context.Context, data Data) (string, error)
	download func(ctx context.Context, s3path string) (Data, error)
	delete   func(ctx context.Context, s3path string) error
	closed   bool
}

func (m *s3wrappedData) Reader() (io.ReadCloser, error) {
	if m.physical == nil {
		var err error
		m.once.Do(func() {
			m.physical, err = m.download(context.Background(), m.s3path)
		})

		if err != nil {
			return nil, err
		}
	}
	return m.physical.Reader()
}

func (m *s3wrappedData) ReadAll() ([]byte, error) {
	reader, err := m.Reader()
	if err != nil {
		return nil, err
	}
	return io.ReadAll(reader)
}

func (m *s3wrappedData) GetS3Path(ctx context.Context) (string, error) {
	var err error

	if m.s3path == "" {
		m.once.Do(func() {
			m.s3path, err = m.upload(ctx, m.physical)
		})
	}
	return m.s3path, err
}

func (m *s3wrappedData) Close() error {
	if m.closed {
		return nil
	}

	closeCtx := context.Background()
	ctx, cancel := context.WithTimeout(closeCtx, 10*time.Second)

	var errS3Delete error
	var errDataClose error
	defer cancel()

	if m.s3path != "" {
		errS3Delete = m.delete(ctx, m.s3path)
	}
	if m.physical != nil {
		errDataClose = m.physical.Close()
	}
	m.closed = true
	return errors.Join(errDataClose, errS3Delete)
}

func NewS3BackedDataFromLocal(
	physical Data,
	upload func(ctx context.Context, data Data) (string, error),
	delete func(ctx context.Context, s3path string) error,
) S3BackedData {
	return &s3wrappedData{
		physical: physical,
		upload:   upload,
		delete:   delete,
	}
}

func NewS3BackedDataFromPath(
	s3path string,
	download func(ctx context.Context, s3path string) (Data, error),
	delete func(ctx context.Context, s3path string) error,
) S3BackedData {
	return &s3wrappedData{
		s3path:   s3path,
		download: download,
		delete:   delete,
	}
}
