package service

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/weoses/memelo/common/storage"
	"github.com/weoses/memelo/common/temp"
)

type TmpDataService interface {
	ByBytes(ctx context.Context, data []byte) (temp.S3BackedData, error)
	ByReader(ctx context.Context, reader io.Reader) (temp.S3BackedData, error)

	WrapData(context.Context, temp.Data) (temp.S3BackedData, error)
	WrapS3Path(context.Context, string) (temp.S3BackedData, error)
}

type TmpDataServiceImpl struct {
	ops storage.S3OperationsAdapter
}

func (s *TmpDataServiceImpl) ByBytes(ctx context.Context, data []byte) (temp.S3BackedData, error) {
	return s.WrapData(ctx, temp.DataBytes(data))
}

func (s *TmpDataServiceImpl) ByReader(ctx context.Context, reader io.Reader) (temp.S3BackedData, error) {
	data, err := temp.DataTemp(reader)
	if err != nil {
		return nil, err
	}
	return s.WrapData(ctx, data)
}

func (s *TmpDataServiceImpl) WrapData(ctx context.Context, data temp.Data) (temp.S3BackedData, error) {
	return temp.NewS3BackedDataFromLocal(
		data,
		s.ops.IsGs(),
		func(ctx context.Context, d temp.Data) (string, error) {
			path := uuid.NewString()
			if err := s.ops.Save(ctx, path, d, storage.WithContentType("application/octet-stream")); err != nil {
				return "", fmt.Errorf("upload failed: %w", err)
			}
			return path, nil
		},
		s.ops.GetUrl,
		s.ops.GetPresignedUrl,
		s.ops.Delete,
	), nil
}

func (s *TmpDataServiceImpl) WrapS3Path(ctx context.Context, path string) (temp.S3BackedData, error) {
	return temp.NewS3BackedDataFromPath(
		path,
		s.ops.IsGs(),
		s.ops.Read,
		s.ops.GetUrl,
		s.ops.GetPresignedUrl,
		s.ops.Delete,
	), nil
}

func NewTmpDataS3Service(ops storage.S3OperationsAdapter) (TmpDataService, error) {
	return &TmpDataServiceImpl{ops: ops}, nil
}
