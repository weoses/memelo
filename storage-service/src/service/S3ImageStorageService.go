package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"mine.local/ocr-gallery/storage-service/conf"
	"mine.local/ocr-gallery/storage-service/entity"
)

type MinioFileStorageServiceImpl struct {
	client     minio.Client
	bucketName string
}

// GetImage implements ImageStorageService.
func (m *MinioFileStorageServiceImpl) GetImage(ctx context.Context, id uuid.UUID) (*entity.Image, error) {
	obj, err := m.client.GetObject(
		ctx,
		m.bucketName,
		getObjectNameV1(id, false),
		minio.GetObjectOptions{},
	)

	if err != nil {
		return nil, err
	}

	buf := bytes.NewBufferString("")
	base64Encoder := base64.NewEncoder(base64.RawStdEncoding, buf)
	_, err = io.Copy(base64Encoder, obj)
	base64Encoder.Close()

	if err != nil {
		return nil, err
	}

	stat, err := obj.Stat()
	if err != nil {
		return nil, err
	}

	dataBase64 := buf.String()
	contentType := stat.ContentType

	return &entity.Image{
		ImageBase64: &dataBase64,
		MimeType:    contentType,
	}, nil
}

// GetUrl implements ImageStorageService.
func (m *MinioFileStorageServiceImpl) GetUrl(ctx context.Context, id uuid.UUID) (string, error) {
	url, err := m.client.PresignedGetObject(
		ctx,
		m.bucketName,
		getObjectNameV1(id, false),
		time.Hour*5,
		url.Values{},
	)

	if err != nil {
		return "", err
	}
	return url.String(), err
}

// GetUrlThumb implements ImageStorageService.
func (m *MinioFileStorageServiceImpl) GetUrlThumb(ctx context.Context, id uuid.UUID) (string, error) {
	url, err := m.client.PresignedGetObject(
		ctx,
		m.bucketName,
		getObjectNameV1(id, true),
		time.Hour*5,
		url.Values{},
	)

	if err != nil {
		return "", err
	}

	return url.String(), err
}

// Save implements ImageStorageService.
func (m *MinioFileStorageServiceImpl) Save(ctx context.Context, id uuid.UUID, image *entity.Image, thumb *entity.Image) error {
	_, err := m.client.PutObject(
		ctx,
		m.bucketName,
		getObjectNameV1(id, false),
		base64.NewDecoder(base64.RawStdEncoding, strings.NewReader(*image.ImageBase64)),
		-1,
		minio.PutObjectOptions{
			ContentType: image.MimeType,
		},
	)

	if err != nil {
		return err
	}

	_, err = m.client.PutObject(
		ctx,
		m.bucketName,
		getObjectNameV1(id, true),
		base64.NewDecoder(base64.RawStdEncoding, strings.NewReader(*thumb.ImageBase64)),
		-1,
		minio.PutObjectOptions{
			ContentType: thumb.MimeType,
		},
	)
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
		return nil, err
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
