package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/youtube-service/conf"
	"github.com/weoses/memelo/youtube-service/entity"
)

type VideoJobService interface {
	RunSync(ctx context.Context, req entity.DownloadRequest) (*entity.VideoDownloadResult, error)
	CreateJob(ctx context.Context, req entity.DownloadRequest) (string, error)
	GetJob(ctx context.Context, jobId string) (*entity.DownloadJob, error)
}

type videoJobServiceImpl struct {
	downloader     YouTubeDownloader[temp.S3BackedData]
	tmpDataService commonservice.TmpDataService
	semaphore      chan struct{}
	jobs           sync.Map
	jobTTL         time.Duration
	log            *slog.Logger
}

func (s *videoJobServiceImpl) RunSync(ctx context.Context, req entity.DownloadRequest) (*entity.VideoDownloadResult, error) {
	s.semaphore <- struct{}{}
	defer func() { <-s.semaphore }()
	store, err := s.downloadAndStore(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("downloadAndStore: %w", err)
	}

	ttl := resolveEffectiveTTL(req.RetentionSeconds, s.jobTTL)
	if ttl >= 0 {
		go func() {
			time.Sleep(ttl)
			s.log.Info("Cleaning up sync s3 object", "s3Path", store.S3Path)
			helper.QuietClose(store.S3Data, s.log)
		}()
	}

	return store, nil
}

func (s *videoJobServiceImpl) CreateJob(_ context.Context, req entity.DownloadRequest) (string, error) {
	jobId := uuid.NewString()
	job := &entity.DownloadJob{
		JobId:        jobId,
		State:        entity.JobStatePending,
		CreatedAt:    time.Now(),
		EffectiveTTL: resolveEffectiveTTL(req.RetentionSeconds, s.jobTTL),
	}
	s.jobs.Store(jobId, job)

	go func() {
		s.semaphore <- struct{}{}
		defer func() { <-s.semaphore }()

		job.Mu.Lock()
		job.State = entity.JobStateRunning
		job.Mu.Unlock()

		result, err := s.downloadAndStore(context.Background(), req)

		job.Mu.Lock()
		if err != nil {
			job.State = entity.JobStateFailed
			job.Error = err
			s.log.Error("async job failed", "jobId", jobId, "error", err)
		} else {
			job.State = entity.JobStateDone
			job.Result = result
		}
		job.Mu.Unlock()
	}()

	return jobId, nil
}

func (s *videoJobServiceImpl) GetJob(_ context.Context, jobId string) (*entity.DownloadJob, error) {
	val, ok := s.jobs.Load(jobId)
	if !ok {
		return nil, fmt.Errorf("job %q not found", jobId)
	}
	return val.(*entity.DownloadJob), nil
}

// downloadAndStore downloads the video, uploads to temp S3, and returns the result.
// The caller owns the S3Data in the result and is responsible for closing it.
func (s *videoJobServiceImpl) downloadAndStore(ctx context.Context, req entity.DownloadRequest) (*entity.VideoDownloadResult, error) {
	var mimeType string
	s3data, err := s.downloader.Download(
		ctx,
		req.YoutubeURL,
		func(ctx context.Context, reader io.Reader, mt string) (temp.S3BackedData, error) {
			mimeType = mt
			return s.tmpDataService.ByReaderUpload(ctx, mt, reader)
		})
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}

	s3path, err := s3data.GetS3Path(ctx)
	if err != nil {
		helper.QuietClose(s3data, s.log)
		return nil, fmt.Errorf("failed to get s3 path: %w", err)
	}

	return &entity.VideoDownloadResult{
		S3Path:   s3path,
		MimeType: mimeType,
		S3Data:   s3data,
	}, nil
}

func (s *videoJobServiceImpl) startTTLCleaner() {
	go func() {
		ticker := time.NewTicker(s.jobTTL / 2)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			s.jobs.Range(func(key, value any) bool {
				job := value.(*entity.DownloadJob)
				job.Mu.RLock()
				ttl := job.EffectiveTTL
				elapsed := now.Sub(job.CreatedAt)
				job.Mu.RUnlock()
				if ttl < 0 || elapsed <= ttl {
					return true
				}
				job.Mu.Lock()
				s3Path := ""
				if job.Result != nil {
					s3Path = job.Result.S3Path
					if job.Result.S3Data != nil {
						helper.QuietClose(job.Result.S3Data, s.log)
						job.Result.S3Data = nil
					}
				}
				job.Mu.Unlock()
				s.log.Info("Cleaned up expired job", "jobId", job.JobId, "s3Path", s3Path)
				s.jobs.Delete(key)
				return true
			})
		}
	}()
}

// resolveEffectiveTTL maps the per-request retention_seconds value to a duration.
// < 0 → never expire (returns -1); > 0 → that many seconds; = 0 → defaultTTL.
func resolveEffectiveTTL(retentionSeconds int32, defaultTTL time.Duration) time.Duration {
	switch {
	case retentionSeconds < 0:
		return -1
	case retentionSeconds > 0:
		return time.Duration(retentionSeconds) * time.Second
	default:
		return defaultTTL
	}
}

func NewVideoJobService(
	cfg *conf.Config,
	downloader YouTubeDownloader[temp.S3BackedData],
	tmpDataService commonservice.TmpDataService,
) (VideoJobService, error) {
	svc := &videoJobServiceImpl{
		downloader:     downloader,
		tmpDataService: tmpDataService,
		semaphore:      make(chan struct{}, cfg.YouTube.MaxConcurrentDownloads),
		jobTTL:         time.Duration(cfg.YouTube.JobTtlSeconds) * time.Second,
		log:            slog.With("service", "VideoJobService"),
	}
	svc.startTTLCleaner()
	return svc, nil
}
