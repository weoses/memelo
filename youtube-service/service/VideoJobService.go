package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/youtube-service/conf"
	"github.com/weoses/memelo/youtube-service/entity"
)

type VideoJobService interface {
	RunSync(ctx context.Context, req entity.DownloadRequest) (*entity.VideoDownloadResult, error)
	CreateJob(ctx context.Context, req entity.DownloadRequest) (string, error)
	GetJob(ctx context.Context, jobId string) (*entity.DownloadJob, error)
}

type videoJobServiceImpl struct {
	downloader     YouTubeDownloader
	tmpDataService commonservice.TmpDataService
	semaphore      chan struct{}
	jobs           sync.Map
	jobTTL         time.Duration
	log            *slog.Logger
}

func (s *videoJobServiceImpl) RunSync(ctx context.Context, req entity.DownloadRequest) (*entity.VideoDownloadResult, error) {
	s.semaphore <- struct{}{}
	defer func() { <-s.semaphore }()
	return s.downloadAndStore(ctx, req)
}

func (s *videoJobServiceImpl) CreateJob(ctx context.Context, req entity.DownloadRequest) (string, error) {
	jobId := uuid.NewString()
	job := &entity.DownloadJob{
		JobId:     jobId,
		State:     entity.JobStatePending,
		CreatedAt: time.Now(),
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
	filePath, mimeType, err := s.downloader.Download(ctx, req.YoutubeURL)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer func() {
		if removeErr := os.Remove(filePath); removeErr != nil {
			s.log.Warn("failed to remove temp file", "path", filePath, "error", removeErr)
		}
	}()

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open temp file: %w", err)
	}
	defer f.Close()

	s3data, err := s.tmpDataService.ByReader(ctx, mimeType, f)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to temp storage: %w", err)
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
				expired := now.Sub(job.CreatedAt) > s.jobTTL
				job.Mu.RUnlock()
				if expired {
					job.Mu.Lock()
					if job.Result != nil && job.Result.S3Data != nil {
						helper.QuietClose(job.Result.S3Data, s.log)
						job.Result.S3Data = nil
					}
					job.Mu.Unlock()
					s.jobs.Delete(key)
				}
				return true
			})
		}
	}()
}

func NewVideoJobService(
	cfg *conf.Config,
	downloader YouTubeDownloader,
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
