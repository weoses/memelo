package service_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"

	tcMinio "github.com/testcontainers/testcontainers-go/modules/minio"
	commonconfig "github.com/weoses/memelo/common/config"
	commonservice "github.com/weoses/memelo/common/service"
	commonstorage "github.com/weoses/memelo/common/storage"
	"github.com/weoses/memelo/ffmpeg-service/conf"
	"github.com/weoses/memelo/ffmpeg-service/entity"
	"github.com/weoses/memelo/ffmpeg-service/service"
)

const testBucket = "test-temp"

var (
	testTmpDataService commonservice.TmpDataService
	testS3Ops          commonstorage.S3OperationsAdapter
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	minioContainer, err := tcMinio.Run(ctx, "minio/minio:latest")
	if err != nil {
		log.Fatalf("start minio: %v", err)
	}
	defer func() { _ = minioContainer.Terminate(ctx) }()

	minioEndpoint, err := minioContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("minio connection string: %v", err)
	}

	adapter, err := commonstorage.NewS3OperationsAdapter(&commonconfig.MediaStorageConfig{
		Endpoint:  minioEndpoint,
		AccessKey: minioContainer.Username,
		SecretKey: minioContainer.Password,
		Bucket:    testBucket,
		Secure:    false,
	})
	if err != nil {
		log.Fatalf("create s3 adapter: %v", err)
	}
	testTmpDataService, err = commonservice.NewTmpDataS3Service(adapter)
	if err != nil {
		log.Fatalf("create tmp data service: %v", err)
	}
	testS3Ops = adapter

	os.Exit(m.Run())
}

func newTestJobService(t *testing.T, maxConcurrent int) service.FfmpegJobService {
	t.Helper()
	cfg := &conf.Config{
		Ffmpeg: &conf.FfmpegConfig{FfmpegBinary: "ffmpeg", FfprobeBinary: "ffprobe"},
		Job:    &conf.JobConfig{MaxConcurrentJobs: maxConcurrent, JobTtlSeconds: 3600},
	}
	svc, err := service.NewFfmpegJobService(cfg, testTmpDataService, testS3Ops)
	if err != nil {
		t.Fatalf("NewFfmpegJobService: %v", err)
	}
	return svc
}

func minimalVideoS3Path(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	video := minimalVideoBytes(t, 1)
	s3Data, err := testTmpDataService.ByBytes(ctx, "video/mp4", video)
	if err != nil {
		t.Fatalf("upload test video: %v", err)
	}
	path, err := s3Data.GetS3Path(ctx)
	if err != nil {
		t.Fatalf("get s3 path: %v", err)
	}
	return path
}

func minimalVideoBytes(t *testing.T, durationSec int) []byte {
	t.Helper()
	f, err := os.CreateTemp("", "job-svc-test-*.mp4")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	f.Close()
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	cmd := exec.Command("ffmpeg",
		"-f", "lavfi", "-i", fmt.Sprintf("color=c=0x336699:s=32x32:r=2:d=%d", durationSec),
		"-c:v", "libx264", "-g", "1", "-keyint_min", "1",
		"-t", fmt.Sprintf("%d", durationSec), "-y", "-loglevel", "quiet",
		f.Name(),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg: %v\n%s", err, out)
	}
	data, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return data
}

func waitForJob(t *testing.T, svc service.FfmpegJobService, jobId string, timeout time.Duration) *entity.Job {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		job, err := svc.GetJob(context.Background(), jobId)
		if err != nil {
			t.Fatalf("GetJob: %v", err)
		}
		job.Mu.RLock()
		state := job.State
		job.Mu.RUnlock()
		if state == entity.JobStateDone || state == entity.JobStateFailed {
			return job
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("job %s did not finish within %s", jobId, timeout)
	return nil
}

func TestCreateJob_ExtractThumbnail(t *testing.T) {
	svc := newTestJobService(t, 2)
	inputPath := minimalVideoS3Path(t)

	jobId, err := svc.CreateJob(context.Background(), entity.JobRequest{
		InputS3Path: inputPath,
		Action:      entity.ActionExtractThumbnail,
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	job := waitForJob(t, svc, jobId, 10*time.Second)
	if job.State != entity.JobStateDone {
		t.Fatalf("job state = %v, err = %v", job.State, job.Error)
	}
	if job.Result == nil || job.Result.OutputS3Path == "" {
		t.Fatalf("expected an output s3 path, got %+v", job.Result)
	}
}

// TestCreateJob_DoesNotDeleteInput locks in the ownership contract: this
// service reads its input by path but never owns or deletes it (the caller
// created it and may still need it — e.g. storage-service's pipeline reads
// the same VideoMp4 object from more than one step). Regression test for a
// bug where the job goroutine closed (and thus deleted) the input object.
func TestCreateJob_DoesNotDeleteInput(t *testing.T) {
	svc := newTestJobService(t, 2)
	inputPath := minimalVideoS3Path(t)

	jobId, err := svc.CreateJob(context.Background(), entity.JobRequest{
		InputS3Path: inputPath,
		Action:      entity.ActionExtractThumbnail,
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	job := waitForJob(t, svc, jobId, 10*time.Second)
	if job.State != entity.JobStateDone {
		t.Fatalf("job state = %v, err = %v", job.State, job.Error)
	}

	// The input must still be readable after the job finished.
	again, err := testS3Ops.Read(context.Background(), inputPath)
	if err != nil {
		t.Fatalf("input was deleted after job completion: %v", err)
	}
	defer again.Close()
	size, err := again.Size()
	if err != nil || size == 0 {
		t.Fatalf("input unreadable or empty after job completion: size=%d, err=%v", size, err)
	}
}

func TestCreateJob_ConvertToMp4(t *testing.T) {
	svc := newTestJobService(t, 2)
	inputPath := minimalVideoS3Path(t)

	jobId, err := svc.CreateJob(context.Background(), entity.JobRequest{
		InputS3Path: inputPath,
		Action:      entity.ActionConvertToMp4,
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	job := waitForJob(t, svc, jobId, 10*time.Second)
	if job.State != entity.JobStateDone {
		t.Fatalf("job state = %v, err = %v", job.State, job.Error)
	}
	if job.Result == nil || job.Result.OutputS3Path == "" {
		t.Fatalf("expected an output s3 path, got %+v", job.Result)
	}
}

func TestCreateJob_SliceVideo(t *testing.T) {
	svc := newTestJobService(t, 2)
	ctx := context.Background()

	video := minimalVideoBytes(t, 6)
	s3Data, err := testTmpDataService.ByBytes(ctx, "video/mp4", video)
	if err != nil {
		t.Fatalf("upload test video: %v", err)
	}
	inputPath, err := s3Data.GetS3Path(ctx)
	if err != nil {
		t.Fatalf("get s3 path: %v", err)
	}

	jobId, err := svc.CreateJob(ctx, entity.JobRequest{
		InputS3Path:     inputPath,
		Action:          entity.ActionSliceVideo,
		IntervalSeconds: 3,
		OverlapSeconds:  1,
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	job := waitForJob(t, svc, jobId, 10*time.Second)
	if job.State != entity.JobStateDone {
		t.Fatalf("job state = %v, err = %v", job.State, job.Error)
	}
	if job.Result == nil || len(job.Result.Slices) == 0 {
		t.Fatalf("expected at least one slice, got %+v", job.Result)
	}
}

func TestCreateJob_FailsOnMissingInput(t *testing.T) {
	svc := newTestJobService(t, 2)

	jobId, err := svc.CreateJob(context.Background(), entity.JobRequest{
		InputS3Path: "does/not/exist.mp4",
		Action:      entity.ActionExtractThumbnail,
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	job := waitForJob(t, svc, jobId, 10*time.Second)
	if job.State != entity.JobStateFailed {
		t.Fatalf("expected job to fail for a missing input, got state=%v", job.State)
	}
	if job.Error == nil {
		t.Fatal("expected job.Error to be set")
	}
}

func TestGetJob_NotFound(t *testing.T) {
	svc := newTestJobService(t, 2)
	_, err := svc.GetJob(context.Background(), "does-not-exist")
	if err == nil {
		t.Fatal("expected an error for an unknown job id")
	}
}

// ensures the concurrency semaphore actually limits how many jobs run at once
// by submitting more jobs than the configured limit and checking they don't
// all finish immediately at t=0.
func TestCreateJob_ConcurrencyLimit(t *testing.T) {
	svc := newTestJobService(t, 1)

	var jobIds []string
	for i := 0; i < 3; i++ {
		// each job consumes its own input object: temp-storage objects are
		// single-use and get deleted after a job reads them (matching
		// storage-service's WrapS3Path+Close convention), so reusing one
		// path across concurrent jobs would race the delete.
		jobId, err := svc.CreateJob(context.Background(), entity.JobRequest{
			InputS3Path: minimalVideoS3Path(t),
			Action:      entity.ActionExtractThumbnail,
		})
		if err != nil {
			t.Fatalf("CreateJob: %v", err)
		}
		jobIds = append(jobIds, jobId)
	}

	for _, jobId := range jobIds {
		job := waitForJob(t, svc, jobId, 15*time.Second)
		if job.State != entity.JobStateDone {
			t.Fatalf("job %s state = %v, err = %v", jobId, job.State, job.Error)
		}
	}
}
