package temp

import (
	"bytes"
	"context"
	"io"
	"slices"
)

type byteBasedData struct {
	data []byte
}

func (m *byteBasedData) Reader(ctx context.Context) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(m.data)), nil
}

func (m *byteBasedData) Size(ctx context.Context) (int64, error) {
	return int64(len(m.data)), nil
}

func (m *byteBasedData) ReadAll(ctx context.Context) ([]byte, error) {
	return slices.Clone(m.data), nil
}

func (m *byteBasedData) Close(ctx context.Context) error { return nil }

func DataBytes(data []byte) Data {
	return &byteBasedData{data: data}
}
