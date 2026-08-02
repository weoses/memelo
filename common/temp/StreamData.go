package temp

import "io"

type streamData struct {
	reader io.Reader
}

func (s *streamData) Size() (int64, error) { return -1, nil }

func (s *streamData) Reader() (io.ReadCloser, error) {
	return io.NopCloser(s.reader), nil
}

func (s *streamData) ReadAll() ([]byte, error) {
	return io.ReadAll(s.reader)
}

func (s *streamData) Close() error { return nil }

// DataStream wraps reader as a Data with unknown size, for streaming
// straight to S3 (via PutObject's unknown-length multipart path) without
// buffering to memory or disk first. Reader() may only be called once.
func DataStream(r io.Reader) Data {
	return &streamData{reader: r}
}
