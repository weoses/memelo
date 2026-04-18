package functional_test //nolint

import (
	"context"
	"testing"
	"time"

	v1 "github.com/weoses/memelo/gen/proto/v1"
)

func waitForRecompute(t *testing.T, ctx context.Context, c *TestClient, jobID string, timeout time.Duration) *v1.RecomputeJobStatus {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := c.GetRecomputeStatus(ctx, jobID)
		if err != nil {
			t.Fatalf("get recompute status: %v", err)
		}
		switch status.GetState() {
		case v1.RecomputeJobStatus_STATE_DONE, v1.RecomputeJobStatus_STATE_FAILED:
			return status
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("recompute job %s did not finish within %s", jobID, timeout)
	return nil
}

func TestRecomputeBasic(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("ec001")

	if _, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1)); err != nil {
		t.Fatalf("create meme 1: %v", err)
	}
	if _, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(2)); err != nil {
		t.Fatalf("create meme 2: %v", err)
	}

	job, err := c.StartRecompute(ctx, &v1.RecomputeRequest{
		ComputeExtractor: true,
		ComputeEmbedding: true,
	})
	if err != nil {
		t.Fatalf("start recompute: %v", err)
	}
	if job.GetJobId() == "" {
		t.Fatal("expected non-empty job id")
	}

	status := waitForRecompute(t, ctx, c, job.GetJobId(), 30*time.Second)

	if status.GetState() != v1.RecomputeJobStatus_STATE_DONE {
		t.Fatalf("expected STATE_DONE, got %s", status.GetState())
	}
	// TODO
	//if status.GetTotal() != 2 {
	//	t.Errorf("expected total=2, got %d", status.GetTotal())
	//}
	if status.GetProcessed() != 2 {
		t.Errorf("expected processed=2, got %d", status.GetProcessed())
	}
	if len(status.GetErrors()) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(status.GetErrors()), status.GetErrors())
	}
	if testMockEx.CallCount() != 4 {
		t.Errorf("expected exactly 4 extractor calls, got %d", testMockEx.CallCount())
	}
	if testMockEmb.CallCount() != 4 {
		t.Errorf("expected exactly 4 embedder calls, got %d", testMockEx.CallCount())
	}
}

func TestRecomputeNoExtractor(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("ec002")

	if _, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1)); err != nil {
		t.Fatalf("create meme 1: %v", err)
	}
	if _, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(2)); err != nil {
		t.Fatalf("create meme 2: %v", err)
	}

	// Reset call counters before recompute so creation calls don't interfere.
	testMockEx.Reset()
	testMockEmb.Reset()

	job, err := c.StartRecompute(ctx, &v1.RecomputeRequest{
		ComputeExtractor: false,
		ComputeEmbedding: true,
	})
	if err != nil {
		t.Fatalf("start recompute: %v", err)
	}

	status := waitForRecompute(t, ctx, c, job.GetJobId(), 30*time.Second)

	if status.GetState() != v1.RecomputeJobStatus_STATE_DONE {
		t.Fatalf("expected STATE_DONE, got %s", status.GetState())
	}
	if status.GetProcessed() != 2 {
		t.Errorf("expected processed=2, got %d", status.GetProcessed())
	}
	if len(status.GetErrors()) != 0 {
		t.Errorf("expected no errors, got %v", status.GetErrors())
	}
	if n := testMockEx.CallCount(); n != 0 {
		t.Errorf("extractor should not be called when ComputeExtractor=false, got %d calls", n)
	}
	if n := testMockEmb.CallCount(); n == 0 {
		t.Errorf("embedder should be called when ComputeEmbedding=true, got 0 calls")
	}
}

func TestRecomputeNoEmbedding(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("ec003")

	if _, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1)); err != nil {
		t.Fatalf("create meme 1: %v", err)
	}
	if _, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(2)); err != nil {
		t.Fatalf("create meme 2: %v", err)
	}

	testMockEx.Reset()
	testMockEmb.Reset()

	job, err := c.StartRecompute(ctx, &v1.RecomputeRequest{
		ComputeExtractor: true,
		ComputeEmbedding: false,
	})
	if err != nil {
		t.Fatalf("start recompute: %v", err)
	}

	status := waitForRecompute(t, ctx, c, job.GetJobId(), 30*time.Second)

	if status.GetState() != v1.RecomputeJobStatus_STATE_DONE {
		t.Fatalf("expected STATE_DONE, got %s", status.GetState())
	}
	if status.GetProcessed() != 2 {
		t.Errorf("expected processed=2, got %d", status.GetProcessed())
	}
	if len(status.GetErrors()) != 0 {
		t.Errorf("expected no errors, got %v", status.GetErrors())
	}
	if n := testMockEmb.CallCount(); n != 0 {
		t.Errorf("embedder should not be called when ComputeEmbedding=false, got %d calls", n)
	}
	if n := testMockEx.CallCount(); n == 0 {
		t.Errorf("extractor should be called when ComputeExtractor=true, got 0 calls")
	}
}
