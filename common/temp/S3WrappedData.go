package temp

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"time"
)

type UploadResult struct {
	Url  string
	Path string
}

type s3wrappedData struct {
	physical     Data
	once         sync.Once
	s3path       string
	upload       func(ctx context.Context, data Data) (string, error)
	download     func(ctx context.Context, s3path string) (Data, error)
	delete       func(ctx context.Context, s3path string) error
	url          func(ctx context.Context, s3path string) (string, error)
	presignedUrl func(ctx context.Context, s3path string) (string, error)
	gsSupported  bool
	closed       bool
	slogger      *slog.Logger
	mu           sync.Mutex
}

func (m *s3wrappedData) GetPresignedUrl(ctx context.Context) (string, error) {
	err := m.resolveUpload(ctx)
	if err != nil {
		return "", err
	}
	return m.presignedUrl(ctx, m.s3path)
}

func (m *s3wrappedData) IsGsSupported() bool {
	return m.gsSupported
}

func (m *s3wrappedData) Size(ctx context.Context) (int64, error) {
	err := m.resolveDownload(ctx)
	if err != nil {
		return 0, err
	}

	return m.physical.Size(ctx)
}

func (m *s3wrappedData) Reader(ctx context.Context) (io.ReadCloser, error) {
	err := m.resolveDownload(ctx)
	if err != nil {
		return nil, err
	}

	return m.physical.Reader(ctx)
}

func (m *s3wrappedData) ReadAll(ctx context.Context) ([]byte, error) {
	reader, err := m.Reader(ctx)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(reader)
}

func (m *s3wrappedData) GetS3Path(ctx context.Context) (string, error) {
	err := m.resolveUpload(ctx)
	if err != nil {
		return "", err
	}
	return m.s3path, err
}

func (m *s3wrappedData) GetS3Url(ctx context.Context) (string, error) {
	err := m.resolveUpload(ctx)
	if err != nil {
		return "", err
	}
	return m.url(ctx, m.s3path)
}

func (m *s3wrappedData) Close(ctx context.Context) error {
	if m.closed {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)

	var errS3Delete error
	var errDataClose error
	defer cancel()

	if m.s3path != "" {
		errS3Delete = m.delete(ctx, m.s3path)
	}
	if m.physical != nil {
		errDataClose = m.physical.Close(ctx)
	}
	m.closed = true
	return errors.Join(errDataClose, errS3Delete)
}

func (m *s3wrappedData) resolveDownload(ctx context.Context) error {
	if m.physical == nil {
		m.mu.Lock()
		defer m.mu.Unlock()

		var err error
		m.once.Do(func() {
			m.slogger.InfoContext(ctx, "s3wrappedData: downloaded object from s3", "uri", m.s3path)
			m.physical, err = m.download(ctx, m.s3path)
		})
		if err != nil {
			return err
		}
	}
	if m.physical == nil {
		return errors.New("s3wrappedData: downloaded object from s3 doesn't exist")
	}

	return nil
}

func (m *s3wrappedData) resolveUpload(ctx context.Context) error {
	var err error
	if m.s3path == "" {
		m.mu.Lock()
		defer m.mu.Unlock()

		m.once.Do(func() {
			m.s3path, err = m.upload(ctx, m.physical)
			m.slogger.InfoContext(ctx, "s3wrappedData: uploaded object to s3", "uri", m.s3path)
		})
		if err != nil {
			return err
		}
	}

	if m.s3path == "" {
		return errors.New("s3wrappedData: uploaded object from s3 doesn't exist")
	}
	return nil
}

func NewS3BackedDataFromLocal(
	physical Data,
	gsSupported bool,
	upload func(ctx context.Context, data Data) (string, error),
	url func(ctx context.Context, s3path string) (string, error),
	presignedUrl func(ctx context.Context, s3path string) (string, error),
	delete func(ctx context.Context, s3path string) error,
) S3BackedData {
	return &s3wrappedData{
		physical:     physical,
		gsSupported:  gsSupported,
		upload:       upload,
		delete:       delete,
		url:          url,
		presignedUrl: presignedUrl,
		slogger:      slog.With(slog.String("component", "s3wrappedData")),
	}
}

func NewS3BackedDataFromPath(
	s3path string,
	gsSupported bool,
	download func(ctx context.Context, s3path string) (Data, error),
	url func(ctx context.Context, s3path string) (string, error),
	presignedUrl func(ctx context.Context, s3path string) (string, error),
	delete func(ctx context.Context, s3path string) error,
) S3BackedData {
	return &s3wrappedData{
		s3path:       s3path,
		gsSupported:  gsSupported,
		download:     download,
		url:          url,
		presignedUrl: presignedUrl,
		delete:       delete,
		slogger:      slog.With(slog.String("component", "s3wrappedData")),
	}
}
