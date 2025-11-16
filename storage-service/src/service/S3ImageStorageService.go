package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"mine.local/ocr-gallery/storage-service/conf"
)

type ImageStorageService interface {
	Save(ctx context.Context, id uuid.UUID, imageBase64 string, thumbBase64 string) error
	GetImage(ctx context.Context, id uuid.UUID) (*string, error)

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

// GetImage implements ImageStorageService.
func (m *MinioFileStorageServiceImpl) GetImage(ctx context.Context, id uuid.UUID) (*string, error) {
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

	dataBase64 := buf.String()

	return &dataBase64, nil
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
func (m *MinioFileStorageServiceImpl) Save(ctx context.Context, id uuid.UUID, image string, thumb string) error {
	_, err := m.client.PutObject(
		ctx,
		m.bucketName,
		getObjectNameV1(id, false),
		base64.NewDecoder(base64.RawStdEncoding, strings.NewReader(image)),
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
		base64.NewDecoder(base64.RawStdEncoding, strings.NewReader(thumb)),
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

func NewImageStorageService(config *conf.ImageStorageConfig) (ImageStorageService, error) {
	return NewMinioFileStorageServiceImpl(config)
}
