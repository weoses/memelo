package storage

import (
	"context"

	"github.com/google/uuid"
	commonstorage "github.com/weoses/memelo/common/storage"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/conf"
)

type MediaType string
type MediaStorageServiceS3Adapter commonstorage.S3OperationsAdapter

const (
	ImageJpegOriginal MediaType = "image-jpeg-original"
	ImageJpegThumbV1  MediaType = "image-jpeg-thumbv1"
	VideoMp4Original  MediaType = "video-mp4-original"
	VideoMp4ThumbV1   MediaType = "video-mp4-thumb"
)

func objectName(id uuid.UUID, t MediaType) string {
	switch t {
	case ImageJpegOriginal:
		return id.String() + "/image.jpg"
	case ImageJpegThumbV1:
		return id.String() + "/thumb-1.jpg"
	case VideoMp4Original:
		return id.String() + "/video.mp4"
	case VideoMp4ThumbV1:
		return id.String() + "/thumb-1.mp4"
	default:
		return id.String() + "/" + string(t)
	}
}

func contentType(t MediaType) string {
	switch t {
	case ImageJpegOriginal, ImageJpegThumbV1:
		return "image/jpeg"
	case VideoMp4Original, VideoMp4ThumbV1:
		return "video/mp4"
	default:
		return "application/octet-stream"
	}
}

type MediaStorageService interface {
	Save(ctx context.Context, id uuid.UUID, mediaType MediaType, data temp.Data) error
	Read(ctx context.Context, id uuid.UUID, mediaType MediaType) (temp.Data, error)
	GetUrl(ctx context.Context, id uuid.UUID, mediaType MediaType) (string, error)
	Delete(ctx context.Context, id uuid.UUID, mediaType MediaType) error
}

type MediaStorageServiceImpl struct {
	storage MediaStorageServiceS3Adapter
}

func (m *MediaStorageServiceImpl) Save(ctx context.Context, id uuid.UUID, mediaType MediaType, data temp.Data) error {
	return m.storage.Save(ctx, objectName(id, mediaType), data, commonstorage.WithContentType(contentType(mediaType)))
}

func (m *MediaStorageServiceImpl) Read(ctx context.Context, id uuid.UUID, mediaType MediaType) (temp.Data, error) {
	return m.storage.Read(ctx, objectName(id, mediaType))
}

func (m *MediaStorageServiceImpl) GetUrl(ctx context.Context, id uuid.UUID, mediaType MediaType) (string, error) {
	return m.storage.GetPresignedUrl(ctx, objectName(id, mediaType))
}

func (m *MediaStorageServiceImpl) Delete(ctx context.Context, id uuid.UUID, mediaType MediaType) error {
	return m.storage.Delete(ctx, objectName(id, mediaType))
}

func NewMediaStorageServiceS3Adapter(cfg *conf.Config) (MediaStorageServiceS3Adapter, error) {
	return commonstorage.NewS3OperationsAdapter(cfg.MediaStorage)
}
func NewMediaStorageService(imageStorage MediaStorageServiceS3Adapter) (MediaStorageService, error) {
	return &MediaStorageServiceImpl{storage: imageStorage}, nil
}
