package temp

import (
	"bytes"
	"io"
	"slices"
)

type byteBasedData struct {
	data []byte
}

func (m *byteBasedData) Reader() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(m.data)), nil
}

func (m *byteBasedData) ReadAll() ([]byte, error) {
	return slices.Clone(m.data), nil
}

func (m *byteBasedData) Close() error { return nil }

func DataBytes(data []byte) Data {
	return &byteBasedData{data: data}
}
