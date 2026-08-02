package api_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/weoses/memelo/ffmpeg-service/api"
	"github.com/weoses/memelo/ffmpeg-service/entity"
	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
)

// fakeJobService is an in-memory FfmpegJobService double so this test exercises
// only the api layer's request validation and response mapping.
type fakeJobService struct {
	mu   sync.Mutex
	jobs map[string]*entity.Job
}

func newFakeJobService() *fakeJobService {
	return &fakeJobService{jobs: map[string]*entity.Job{}}
}

func (f *fakeJobService) CreateJob(_ context.Context, req entity.JobRequest) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	jobId := fmt.Sprintf("job-%d", len(f.jobs)+1)
	f.jobs[jobId] = &entity.Job{
		JobId:  jobId,
		Action: req.Action,
		State:  entity.JobStateDone,
		Result: &entity.JobResult{OutputS3Path: "out.jpg"},
	}
	return jobId, nil
}

func (f *fakeJobService) GetJob(_ context.Context, jobId string) (*entity.Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	job, ok := f.jobs[jobId]
	if !ok {
		return nil, fmt.Errorf("job %q not found", jobId)
	}
	return job, nil
}

func startTestApi(t *testing.T) v1connect.FfmpegServiceClient {
	t.Helper()
	apiHandler, err := api.NewFfmpegServiceApi(newFakeJobService())
	if err != nil {
		t.Fatalf("NewFfmpegServiceApi: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle(v1connect.NewFfmpegServiceHandler(apiHandler))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return v1connect.NewFfmpegServiceClient(http.DefaultClient, srv.URL)
}

func TestSubmitJob_RequiresInputPath(t *testing.T) {
	cl := startTestApi(t)
	_, err := cl.SubmitJob(context.Background(), &v1.SubmitFfmpegJobRequest{
		Action: &v1.SubmitFfmpegJobRequest_ExtractThumbnail{},
	})
	assertInvalidArgument(t, err)
}

func TestSubmitJob_RequiresAction(t *testing.T) {
	cl := startTestApi(t)
	_, err := cl.SubmitJob(context.Background(), &v1.SubmitFfmpegJobRequest{
		InputS3Path: "in.mp4",
	})
	assertInvalidArgument(t, err)
}

func TestSubmitJob_RejectsInvalidSliceInterval(t *testing.T) {
	cl := startTestApi(t)
	_, err := cl.SubmitJob(context.Background(), &v1.SubmitFfmpegJobRequest{
		InputS3Path: "in.mp4",
		Action: &v1.SubmitFfmpegJobRequest_SliceVideo{
			SliceVideo: &v1.SliceVideoAction{IntervalSeconds: 1, OverlapSeconds: 1},
		},
	})
	assertInvalidArgument(t, err)
}

func TestSubmitJob_AndGetJobStatus(t *testing.T) {
	cl := startTestApi(t)
	ctx := context.Background()

	submitResp, err := cl.SubmitJob(ctx, &v1.SubmitFfmpegJobRequest{
		InputS3Path: "in.mp4",
		Action:      &v1.SubmitFfmpegJobRequest_ExtractThumbnail{},
	})
	if err != nil {
		t.Fatalf("SubmitJob: %v", err)
	}
	if submitResp.JobId == "" {
		t.Fatal("expected a non-empty job id")
	}

	statusResp, err := cl.GetJobStatus(ctx, &v1.GetFfmpegJobStatusRequest{JobId: submitResp.JobId})
	if err != nil {
		t.Fatalf("GetJobStatus: %v", err)
	}
	if statusResp.State != v1.FfmpegJobState_FFMPEG_JOB_STATE_DONE {
		t.Fatalf("state = %v, want DONE", statusResp.State)
	}
	if statusResp.GetOutputS3Path() != "out.jpg" {
		t.Fatalf("output_s3_path = %q, want out.jpg", statusResp.GetOutputS3Path())
	}
}

func TestGetJobStatus_RequiresJobId(t *testing.T) {
	cl := startTestApi(t)
	_, err := cl.GetJobStatus(context.Background(), &v1.GetFfmpegJobStatusRequest{})
	assertInvalidArgument(t, err)
}

func TestGetJobStatus_NotFound(t *testing.T) {
	cl := startTestApi(t)
	_, err := cl.GetJobStatus(context.Background(), &v1.GetFfmpegJobStatusRequest{JobId: "missing"})
	var connectErr *connect.Error
	if err == nil {
		t.Fatal("expected an error for an unknown job id")
	}
	if !asConnectError(err, &connectErr) || connectErr.Code() != connect.CodeNotFound {
		t.Fatalf("expected CodeNotFound, got %v", err)
	}
}

func assertInvalidArgument(t *testing.T, err error) {
	t.Helper()
	var connectErr *connect.Error
	if err == nil {
		t.Fatal("expected an error")
	}
	if !asConnectError(err, &connectErr) || connectErr.Code() != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", err)
	}
}

func asConnectError(err error, target **connect.Error) bool {
	ce, ok := err.(*connect.Error)
	if ok {
		*target = ce
	}
	return ok
}
