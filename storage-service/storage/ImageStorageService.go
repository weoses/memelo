package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/weoses/memelo/storage-service/conf"
)

type ImageStorageService interface {
	Save(ctx context.Context, id uuid.UUID, imgRaw []byte, imgThumbnail []byte) error
	GetImageBytes(ctx context.Context, id uuid.UUID) ([]byte, error)
	GetImageThumbBytes(ctx context.Context, id uuid.UUID) ([]byte, error)

	GetUrl(ctx context.Context, id uuid.UUID) (string, error)
	GetUrlThumb(ctx context.Context, id uuid.UUID) (string, error)
	DeleteImage(ctx context.Context, id uuid.UUID) error
}

type MinioFileStorageServiceImpl struct {
	client     minio.Client
	bucketName string
}

func (m *MinioFileStorageServiceImpl) DeleteImage(ctx context.Context, id uuid.UUID) error {
	err1 := m.client.RemoveObject(ctx, m.bucketName, getObjectNameV1(id, false), minio.RemoveObjectOptions{})
	err2 := m.client.RemoveObject(ctx, m.bucketName, getObjectNameV1(id, true), minio.RemoveObjectOptions{})

	return errors.Join(err1, err2)
}

// GetImageBytes implements ImageStorageService.
func (m *MinioFileStorageServiceImpl) GetImageBytes(ctx context.Context, id uuid.UUID) ([]byte, error) {
	return m.getObjectBytes(ctx, getObjectNameV1(id, false))
}

// GetImageThumbBytes implements ImageStorageService.
func (m *MinioFileStorageServiceImpl) GetImageThumbBytes(ctx context.Context, id uuid.UUID) ([]byte, error) {
	return m.getObjectBytes(ctx, getObjectNameV1(id, true))
}

func (m *MinioFileStorageServiceImpl) getObjectBytes(ctx context.Context, objectName string) ([]byte, error) {
	obj, err := m.client.GetObject(ctx, m.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(obj)
}

// GetUrl implements ImageStorageService.
func (m *MinioFileStorageServiceImpl) GetUrl(ctx context.Context, id uuid.UUID) (string, error) {
	u, err := m.client.PresignedGetObject(
		ctx,
		m.bucketName,
		getObjectNameV1(id, false),
		time.Hour*5,
		url.Values{},
	)

	if err != nil {
		return "", err
	}
	return u.String(), err
}

// GetUrlThumb implements ImageStorageService.
func (m *MinioFileStorageServiceImpl) GetUrlThumb(ctx context.Context, id uuid.UUID) (string, error) {
	u, err := m.client.PresignedGetObject(
		ctx,
		m.bucketName,
		getObjectNameV1(id, true),
		time.Hour*5,
		url.Values{},
	)

	if err != nil {
		return "", err
	}

	return u.String(), err
}

// Save implements ImageStorageService.
func (m *MinioFileStorageServiceImpl) Save(ctx context.Context, id uuid.UUID, image []byte, imgThumbnail []byte) error {
	_, err := m.client.PutObject(
		ctx,
		m.bucketName,
		getObjectNameV1(id, false),
		bytes.NewReader(image),
		-1,
		minio.PutObjectOptions{
			ContentType: "image/jpeg",
		},
	)

	if err != nil {
		return fmt.Errorf("PutObject failed for source doc: %w", err)
	}

	_, err = m.client.PutObject(
		ctx,
		m.bucketName,
		getObjectNameV1(id, true),
		bytes.NewReader(imgThumbnail),
		-1,
		minio.PutObjectOptions{
			ContentType: "image/jpeg",
		},
	)

	if err != nil {
		return fmt.Errorf("PutObject failed for thumb doc: %w", err)
	}

	return err
}

func NewMinioFileStorageServiceImpl(config *conf.ImageStorageConfig) (ImageStorageService, error) {

	minioClient, err := minio.New(config.S3.Endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(
			config.S3.AccessKey,
			config.S3.SecretKey,
			""),
		Secure: config.S3.Secure,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	exists, err := minioClient.BucketExists(context.Background(), config.S3.Bucket)

	if err != nil {
		return nil, fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		err = minioClient.MakeBucket(context.Background(), config.S3.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create minio bucket: %w", err)
		}
	}
	return &MinioFileStorageServiceImpl{
			bucketName: config.S3.Bucket,
			client:     *minioClient,
		},
		nil
}

func getObjectNameV1(id uuid.UUID, thumb bool) string {
	var imgName string
	if !thumb {
		imgName = "image.jpg"
	} else {
		imgName = "thumb-1.jpg"
	}

	return id.String() + "/" + imgName
}

func NewImageStorageService(config *conf.ImageStorageConfig) (ImageStorageService, error) {
	return NewMinioFileStorageServiceImpl(config)
}
