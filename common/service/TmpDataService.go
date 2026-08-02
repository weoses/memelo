package service

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/weoses/memelo/common/storage"
	"github.com/weoses/memelo/common/temp"
)

type UploadUrlResult struct {
	S3Path     string
	URL        string
	FormFields map[string]string
}

type TmpDataService interface {
	ByBytes(ctx context.Context, mime string, data []byte) (temp.S3BackedData, error)
	ByReader(ctx context.Context, mime string, reader io.Reader) (temp.S3BackedData, error)
	// ByReaderUpload uploads reader straight to S3 without buffering to disk
	// (or memory) first. The returned S3BackedData has no local physical
	// copy — reads later resolve by re-downloading from S3.
	ByReaderUpload(ctx context.Context, mime string, reader io.Reader) (temp.S3BackedData, error)

	WrapData(context.Context, string, temp.Data) (temp.S3BackedData, error)
	WrapInternalS3Path(context.Context, string) (temp.S3BackedData, error)
	// WrapExternalS3Path wraps an existing S3 path this service does not own:
	// Close() releases any locally-downloaded copy but never deletes the
	// remote object. Use for inputs whose lifecycle belongs to the caller.
	WrapExternalS3Path(context.Context, string) (temp.S3BackedData, error)

	GetUploadUrl(ctx context.Context, mime string, size int64) (UploadUrlResult, error)
}

type TmpDataServiceImpl struct {
	ops storage.S3OperationsAdapter
}

func (s *TmpDataServiceImpl) ByBytes(ctx context.Context, mime string, data []byte) (temp.S3BackedData, error) {
	return s.WrapData(ctx, mime, temp.DataBytes(data))
}

func (s *TmpDataServiceImpl) ByReader(ctx context.Context, mime string, reader io.Reader) (temp.S3BackedData, error) {
	data, err := temp.DataTemp(reader)
	if err != nil {
		return nil, err
	}
	return s.WrapData(ctx, mime, data)
}

func (s *TmpDataServiceImpl) ByReaderUpload(ctx context.Context, mime string, reader io.Reader) (temp.S3BackedData, error) {
	path := newTmpS3Path(mime)
	if err := s.ops.Save(ctx, path, temp.DataStream(reader), storage.WithContentType(mime)); err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	return s.WrapInternalS3Path(ctx, path)
}

func (s *TmpDataServiceImpl) WrapData(ctx context.Context, mime string, data temp.Data) (temp.S3BackedData, error) {
	return temp.NewS3BackedDataFromLocal(
		data,
		s.ops.IsGs(),
		func(ctx context.Context, d temp.Data) (string, error) {
			path := newTmpS3Path(mime)
			if err := s.ops.Save(ctx, path, d, storage.WithContentType(mime)); err != nil {
				return "", fmt.Errorf("upload failed: %w", err)
			}
			return path, nil
		},
		s.ops.GetUrl,
		s.ops.GetPresignedUrl,
		s.ops.Delete,
	), nil
}

func newTmpS3Path(mime string) string {
	return uuid.NewString() + "." + strings.Split(mime, "/")[1]
}

func (s *TmpDataServiceImpl) WrapInternalS3Path(ctx context.Context, path string) (temp.S3BackedData, error) {
	return temp.NewS3BackedDataFromPath(
		path,
		s.ops.IsGs(),
		s.ops.Read,
		s.ops.GetUrl,
		s.ops.GetPresignedUrl,
		s.ops.Delete,
	), nil
}

func (s *TmpDataServiceImpl) WrapExternalS3Path(ctx context.Context, path string) (temp.S3BackedData, error) {
	return temp.NewS3BackedDataFromPath(
		path,
		s.ops.IsGs(),
		s.ops.Read,
		s.ops.GetUrl,
		s.ops.GetPresignedUrl,
		func(context.Context, string) error { return nil }, // not owned: never delete
	), nil
}

func (s *TmpDataServiceImpl) GetUploadUrl(ctx context.Context, mime string, size int64) (UploadUrlResult, error) {
	path := uuid.NewString() + "." + strings.Split(mime, "/")[1]
	data, err := s.ops.GetPresignedPostUrl(ctx, path, mime, size)
	if err != nil {
		return UploadUrlResult{}, fmt.Errorf("presigned post failed: %w", err)
	}
	return UploadUrlResult{S3Path: path, URL: data.URL, FormFields: data.FormFields}, nil
}

func NewTmpDataS3Service(ops storage.S3OperationsAdapter) (TmpDataService, error) {
	return &TmpDataServiceImpl{ops: ops}, nil
}
