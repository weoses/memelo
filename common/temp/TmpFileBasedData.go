package temp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

type tempBasedData struct {
	path   string
	closed bool
}

func (m *tempBasedData) Size(ctx context.Context) (int64, error) {
	info, err := os.Stat(m.path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (m *tempBasedData) Reader(ctx context.Context) (io.ReadCloser, error) {
	return os.Open(m.path)
}

func (m *tempBasedData) ReadAll(ctx context.Context) ([]byte, error) {
	return os.ReadFile(m.path)
}

func (m *tempBasedData) Close(ctx context.Context) error {
	if m.closed {
		return nil
	}
	m.closed = true
	return os.Remove(m.path)
}

func DataTemp(r io.Reader) (Data, error) {
	buf := make([]byte, MaxInmemSize)
	n, err := io.ReadFull(r, buf)
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return &byteBasedData{data: buf[:n]}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error reading temp: %w", err)
	}

	// Hit the limit — spill to temp file
	temp, err := os.CreateTemp("", "melo-tmp-file-*")
	if err != nil {
		return nil, fmt.Errorf("error creating temp file: %w", err)
	}
	name := temp.Name()

	if _, err = temp.Write(buf); err != nil {
		_ = temp.Close()
		_ = os.Remove(name)
		return nil, fmt.Errorf("error writing buffer to temp file: %w", err)
	}
	if _, err = io.Copy(temp, r); err != nil {
		_ = temp.Close()
		_ = os.Remove(name)
		return nil, fmt.Errorf("error writing rest to temp file: %w", err)
	}
	if err = temp.Close(); err != nil {
		_ = os.Remove(name)
		return nil, fmt.Errorf("error closing temp file: %w", err)
	}

	return &tempBasedData{path: name}, nil
}
