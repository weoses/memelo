package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bluele/gcache"
	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/ffmpeg-service/conf"
	"github.com/weoses/memelo/ffmpeg-service/entity"
	"github.com/weoses/memelo/ffmpeg-service/ffmpeg"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// tracer creates spans for ffmpeg jobs -- background work that outlives
// the synchronous SubmitJob RPC call that kicks it off.
var tracer = otel.Tracer("ffmpeg-service")

type FfmpegJobService interface {
	CreateJob(ctx context.Context, req entity.JobRequest) (string, error)
	GetJob(ctx context.Context, jobId string) (*entity.Job, error)
}

type ffmpegJobServiceImpl struct {
	tmpDataService commonservice.TmpDataService
	ffmpegCfg      *conf.FfmpegConfig
	cache          gcache.Cache
	semaphore      chan struct{}
	jobTTL         time.Duration
	log            *slog.Logger
}

func (s *ffmpegJobServiceImpl) CreateJob(ctx context.Context, req entity.JobRequest) (string, error) {
	jobId := uuid.NewString()
	job := &entity.Job{
		JobId:        jobId,
		Action:       req.Action,
		State:        entity.JobStatePending,
		CreatedAt:    time.Now(),
		EffectiveTTL: resolveEffectiveTTL(req.RetentionSeconds, s.jobTTL),
	}
	_ = s.cache.Set(jobId, job)

	// Detach from SubmitJob's own RPC lifecycle (ctx is cancelled the
	// instant this call returns) but keep it as the parent span --
	// WithoutCancel first, then Start, so the job span's context is
	// itself uncancelable too. A child span outliving its already-ended
	// parent is expected/valid for this fire-and-forget-a-job-id shape.
	// cleanupCtx is the same detached context but *without* the span --
	// scheduleCleanup's TTL cleanup fires well after this span has already
	// ended (see scheduleCleanup), so nesting under it would just look like
	// a child span appearing hours after its "ended" parent. Reusing this
	// non-fabricated context (rather than a fresh context.Background()) at
	// least keeps it a real continuation of the request's context lineage.
	cleanupCtx := context.WithoutCancel(ctx)
	jobCtx, span := tracer.Start(cleanupCtx, "ffmpeg.job",
		trace.WithAttributes(attribute.String("job.id", jobId), attribute.Int("action", int(req.Action))))

	go func() {
		defer span.End()
		s.semaphore <- struct{}{}
		defer func() { <-s.semaphore }()

		job.Mu.Lock()
		job.State = entity.JobStateRunning
		job.Mu.Unlock()

		result, err := s.runAction(jobCtx, req)

		job.Mu.Lock()
		if err != nil {
			job.State = entity.JobStateFailed
			job.Error = err
			s.log.ErrorContext(jobCtx, "ffmpeg job failed", "jobId", jobId, "error", err)
		} else {
			job.State = entity.JobStateDone
			job.Result = result
		}
		job.Mu.Unlock()

		if result != nil {
			s.scheduleCleanup(cleanupCtx, job)
		}
	}()

	return jobId, nil
}

func (s *ffmpegJobServiceImpl) GetJob(_ context.Context, jobId string) (*entity.Job, error) {
	val, err := s.cache.Get(jobId)
	if err != nil {
		return nil, fmt.Errorf("job %q not found", jobId)
	}
	return val.(*entity.Job), nil
}

// runAction downloads the input from S3, dispatches to the requested ffmpeg
// action, and uploads the produced output(s) back to S3. The caller (the job
// goroutine) owns the returned JobResult's S3Data and is responsible for
// eventually closing it.
func (s *ffmpegJobServiceImpl) runAction(ctx context.Context, req entity.JobRequest) (*entity.JobResult, error) {
	// Read the input via WrapExternalS3Path (not WrapInternalS3Path): this job didn't
	// create the input object, so Close() must not delete it — only release
	// the locally-downloaded copy. Cleanup of the input is the responsibility
	// of whoever created it.
	input, err := s.tmpDataService.WrapExternalS3Path(ctx, req.InputS3Path)
	if err != nil {
		return nil, fmt.Errorf("resolve input: %w", err)
	}
	defer helper.QuietCloseCtx(ctx, input, s.log)

	switch req.Action {
	case entity.ActionExtractThumbnail:
		frame, err := ffmpeg.ExtractOneFrame(ctx, s.ffmpegCfg, input, s.log)
		if err != nil {
			return nil, fmt.Errorf("extract thumbnail: %w", err)
		}
		s3Frame, err := s.tmpDataService.WrapData(ctx, "image/jpeg", frame)
		if err != nil {
			helper.QuietCloseCtx(ctx, frame, s.log)
			return nil, fmt.Errorf("upload thumbnail: %w", err)
		}
		s3Path, err := s3Frame.GetS3Path(ctx)
		if err != nil {
			helper.QuietCloseCtx(ctx, s3Frame, s.log)
			return nil, fmt.Errorf("get thumbnail s3 path: %w", err)
		}
		return &entity.JobResult{OutputS3Path: s3Path, S3Data: []temp.S3BackedData{s3Frame}}, nil

	case entity.ActionConvertToMp4:
		mp4, err := ffmpeg.ConvertToMp4(ctx, s.ffmpegCfg, input, s.log)
		if err != nil {
			return nil, fmt.Errorf("convert to mp4: %w", err)
		}
		s3Mp4, err := s.tmpDataService.WrapData(ctx, "video/mp4", mp4)
		if err != nil {
			helper.QuietCloseCtx(ctx, mp4, s.log)
			return nil, fmt.Errorf("upload mp4: %w", err)
		}
		s3Path, err := s3Mp4.GetS3Path(ctx)
		if err != nil {
			helper.QuietCloseCtx(ctx, s3Mp4, s.log)
			return nil, fmt.Errorf("get mp4 s3 path: %w", err)
		}
		return &entity.JobResult{OutputS3Path: s3Path, S3Data: []temp.S3BackedData{s3Mp4}}, nil

	case entity.ActionSliceVideo:
		interval := time.Duration(req.IntervalSeconds) * time.Second
		overlap := time.Duration(req.OverlapSeconds) * time.Second
		slices, err := ffmpeg.SliceVideoWithOverlap(ctx, s.ffmpegCfg, input, interval, overlap, s.log)
		if err != nil {
			return nil, fmt.Errorf("slice video: %w", err)
		}
		result := &entity.JobResult{
			Slices: make([]entity.SliceResult, 0, len(slices)),
			S3Data: make([]temp.S3BackedData, 0, len(slices)),
		}
		for i, slice := range slices {
			s3Slice, err := s.tmpDataService.WrapData(ctx, "video/mp4", slice.Data)
			if err != nil {
				helper.QuietCloseCtx(ctx, slice.Data, s.log)
				for _, remaining := range slices[i+1:] {
					helper.QuietCloseCtx(ctx, remaining.Data, s.log)
				}
				helper.QuietCloseAllCtx(ctx, result.S3Data, s.log)
				return nil, fmt.Errorf("upload slice %d: %w", i, err)
			}
			s3Path, err := s3Slice.GetS3Path(ctx)
			if err != nil {
				helper.QuietCloseCtx(ctx, s3Slice, s.log)
				for _, remaining := range slices[i+1:] {
					helper.QuietCloseCtx(ctx, remaining.Data, s.log)
				}
				helper.QuietCloseAllCtx(ctx, result.S3Data, s.log)
				return nil, fmt.Errorf("get slice %d s3 path: %w", i, err)
			}
			result.Slices = append(result.Slices, entity.SliceResult{
				S3Path:    s3Path,
				StartTime: slice.StartTime,
				EndTime:   slice.EndTime,
			})
			result.S3Data = append(result.S3Data, s3Slice)
		}
		return result, nil

	default:
		return nil, fmt.Errorf("unknown action %d", req.Action)
	}
}

func (s *ffmpegJobServiceImpl) scheduleCleanup(ctx context.Context, job *entity.Job) {
	job.Mu.RLock()
	ttl := job.EffectiveTTL
	result := job.Result
	job.Mu.RUnlock()

	if ttl < 0 || result == nil {
		return
	}

	go func() {
		time.Sleep(ttl)
		s.log.Info("cleaning up expired ffmpeg job output", "jobId", job.JobId)
		helper.QuietCloseAllCtx(ctx, result.S3Data, s.log)
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

func NewFfmpegJobService(
	cfg *conf.Config,
	tmpDataService commonservice.TmpDataService,
) (FfmpegJobService, error) {
	jobTTL := time.Duration(cfg.Job.JobTtlSeconds) * time.Second
	return &ffmpegJobServiceImpl{
		tmpDataService: tmpDataService,
		ffmpegCfg:      cfg.Ffmpeg,
		cache:          gcache.New(256).LFU().Expiration(jobTTL).Build(),
		semaphore:      make(chan struct{}, cfg.Job.MaxConcurrentJobs),
		jobTTL:         jobTTL,
		log:            slog.With("service", "FfmpegJobService"),
	}, nil
}
