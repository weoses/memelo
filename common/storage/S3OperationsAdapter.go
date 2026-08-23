package storage

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/weoses/memelo/common/temp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/weoses/memelo/common/config"
	"github.com/weoses/memelo/common/helper"
)

// tracer is deliberately named after the package, not a specific service --
// this adapter is shared by every service (see grep for "common/storage"),
// so each span picks up the caller's own service.name from that service's
// own TracerProvider resource rather than a hardcoded one here.
var tracer = otel.Tracer("common/storage")

type SaveOptions func(options *SaveOptionsParameter)

type PresignedPostData struct {
	URL        string
	FormFields map[string]string
}

type S3OperationsAdapter interface {
	Save(ctx context.Context, path string, data temp.Data, options ...SaveOptions) error
	Read(ctx context.Context, path string) (temp.Data, error)
	GetUrl(ctx context.Context, path string) (string, error)
	GetPresignedUrl(ctx context.Context, path string) (string, error)
	GetPresignedPostUrl(ctx context.Context, path string, mime string, size int64) (PresignedPostData, error)
	Delete(ctx context.Context, path string) error
	IsGs() bool
}

type SaveOptionsParameter struct {
	Expires     *time.Time
	ContentType *string
}

func WithExpires(expires time.Time) SaveOptions {
	return func(options *SaveOptionsParameter) {
		options.Expires = &expires
	}
}

func WithContentType(contentType string) SaveOptions {
	return func(options *SaveOptionsParameter) {
		options.ContentType = &contentType
	}
}

type S3OperationsAdapterService struct {
	client  minio.Client
	slogger *slog.Logger

	BucketName string
	Endpoint   string
	Secure     bool
}

func NewS3OperationsAdapter(cfg *config.MediaStorageConfig) (S3OperationsAdapter, error) {
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.Secure,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create service client: %w", err)
	}

	exists, err := minioClient.BucketExists(context.Background(), cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !exists {
		err = minioClient.MakeBucket(context.Background(), cfg.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create service bucket: %w", err)
		}
	}

	return &S3OperationsAdapterService{
		client:     *minioClient,
		BucketName: cfg.Bucket,
		Endpoint:   cfg.Endpoint,
		Secure:     cfg.Secure,
		slogger:    slog.With("service", "S3OperationsAdapterService", "endpoint", cfg.Endpoint, "bucket", cfg.Bucket),
	}, nil
}

func (m *S3OperationsAdapterService) IsGs() bool {
	return strings.Contains(m.Endpoint, "googleapis.com")
}

func (m *S3OperationsAdapterService) Save(ctx context.Context, path string, data temp.Data, options ...SaveOptions) error {
	size, err := data.Size()
	if err != nil {
		return fmt.Errorf("failed to get size of data: %w", err)
	}

	dataReader, err := data.Reader()
	if err != nil {
		return fmt.Errorf("failed to get temp reader: %w", err)
	}
	defer helper.QuietClose(dataReader, m.slogger)

	parameters := SaveOptionsParameter{}
	for _, option := range options {
		option(&parameters)
	}

	putOptions := minio.PutObjectOptions{}
	if parameters.Expires != nil {
		putOptions.Expires = *parameters.Expires
	}

	if parameters.ContentType != nil {
		putOptions.ContentType = *parameters.ContentType
	}

	m.slogger.InfoContext(ctx, "service PutObject", "object", path)
	ctx, span := tracer.Start(ctx, "s3.put_object", trace.WithAttributes(
		attribute.String("s3.bucket", m.BucketName), attribute.String("s3.key", path),
	))
	defer span.End()

	_, err = m.client.PutObject(
		ctx,
		m.BucketName,
		path,
		dataReader,
		size,
		putOptions,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		m.slogger.ErrorContext(ctx, "service PutObject failed", "object", path, "error", err)
		return fmt.Errorf("PutObject failed for %s: %w", path, err)
	}
	m.slogger.DebugContext(ctx, "service PutObject ok", "object", path)
	return nil
}

func (m *S3OperationsAdapterService) Read(ctx context.Context, path string) (temp.Data, error) {
	m.slogger.InfoContext(ctx, "service GetObject", "object", path)
	ctx, span := tracer.Start(ctx, "s3.get_object", trace.WithAttributes(
		attribute.String("s3.bucket", m.BucketName), attribute.String("s3.key", path),
	))
	defer span.End()

	obj, err := m.client.GetObject(ctx, m.BucketName, path, minio.GetObjectOptions{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		m.slogger.ErrorContext(ctx, "service GetObject failed", "object", path, "error", err)
		return nil, err
	}
	defer helper.QuietClose(obj, m.slogger)

	d, err := temp.DataTemp(obj)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		m.slogger.ErrorContext(ctx, "service GetObject DataTemp failed", "object", path, "error", err)
		return nil, err
	}

	m.slogger.DebugContext(ctx, "service GetObject ok", "object", path)
	return d, nil
}

func (m *S3OperationsAdapterService) GetPresignedUrl(ctx context.Context, path string) (string, error) {
	m.slogger.InfoContext(ctx, "service PresignedGetObject", "object", path)
	ctx, span := tracer.Start(ctx, "s3.presigned_get_url", trace.WithAttributes(
		attribute.String("s3.bucket", m.BucketName), attribute.String("s3.key", path),
	))
	defer span.End()

	u, err := m.client.PresignedGetObject(
		ctx,
		m.BucketName,
		path,
		time.Hour*5,
		url.Values{},
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		m.slogger.ErrorContext(ctx, "service PresignedGetObject failed", "object", path, "error", err)
		return "", err
	}
	m.slogger.DebugContext(ctx, "service PresignedGetObject ok", "object", path)
	return u.String(), nil
}

func (m *S3OperationsAdapterService) GetUrl(ctx context.Context, path string) (string, error) {
	scheme := "http"
	if m.Secure {
		scheme = "https"
	}
	u := fmt.Sprintf("%s://%s/%s/%s", scheme, m.Endpoint, m.BucketName, path)
	m.slogger.DebugContext(ctx, "service GetUrl", "object", path, "url", u)
	return u, nil
}

func (m *S3OperationsAdapterService) GetPresignedPostUrl(ctx context.Context, path string, mime string, size int64) (PresignedPostData, error) {
	m.slogger.InfoContext(ctx, "service PresignedPostPolicy", "object", path)
	policy := minio.NewPostPolicy()
	_ = policy.SetBucket(m.BucketName)
	_ = policy.SetKey(path)
	_ = policy.SetExpires(time.Now().Add(time.Hour * 5))
	_ = policy.SetContentType(mime)
	_ = policy.SetContentLengthRange(size, size)

	ctx, span := tracer.Start(ctx, "s3.presigned_post_url", trace.WithAttributes(
		attribute.String("s3.bucket", m.BucketName), attribute.String("s3.key", path),
	))
	defer span.End()

	u, formData, err := m.client.PresignedPostPolicy(ctx, policy)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		m.slogger.ErrorContext(ctx, "service PresignedPostPolicy failed", "object", path, "error", err)
		return PresignedPostData{}, err
	}
	m.slogger.DebugContext(ctx, "service PresignedPostPolicy ok", "object", path)
	return PresignedPostData{URL: u.String(), FormFields: formData}, nil
}

func (m *S3OperationsAdapterService) Delete(ctx context.Context, path string) error {
	m.slogger.InfoContext(ctx, "service RemoveObject", "object", path)
	ctx, span := tracer.Start(ctx, "s3.delete_object", trace.WithAttributes(
		attribute.String("s3.bucket", m.BucketName), attribute.String("s3.key", path),
	))
	defer span.End()

	err := m.client.RemoveObject(ctx, m.BucketName, path, minio.RemoveObjectOptions{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		m.slogger.ErrorContext(ctx, "service RemoveObject failed", "object", path, "error", err)
		return err
	}
	m.slogger.DebugContext(ctx, "service RemoveObject ok", "object", path)
	return nil
}
