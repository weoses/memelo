package functional_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/storage-service/entity"
)

// minimalMP4 generates a tiny 1-second H.264 MP4 with a seed-based colour so
// different seeds produce different file hashes, preventing EP10 hash dedup
// from short-circuiting the embedding check.
func minimalMP4(t *testing.T, seed int) []byte {
	t.Helper()

	f, err := os.CreateTemp("", "test-video-*.mp4")
	if err != nil {
		t.Fatalf("minimalMP4: create temp file: %v", err)
	}
	f.Close()
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	r := uint8(seed * 25 % 255)
	g := uint8(seed * 37 % 255)
	b := uint8(seed * 59 % 255)
	colorStr := fmt.Sprintf("0x%02X%02X%02X", r, g, b)

	cmd := exec.Command("ffmpeg",
		"-f", "lavfi", "-i", fmt.Sprintf("color=c=%s:s=4x4:r=1:d=1", colorStr),
		"-c:v", "libx264", "-t", "1", "-y", "-loglevel", "quiet",
		f.Name(),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("minimalMP4: ffmpeg: %v\n%s", err, out)
	}

	data, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatalf("minimalMP4: read: %v", err)
	}
	return data
}

// TestVideoDeduplicationAllEmbeddingsSame verifies that a video whose embeddings
// all match an already-stored video is reported as a duplicate.
func TestVideoDeduplicationAllEmbeddingsSame(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("e01")

	// Return 3 identical embeddings per GetVideoEmbedding call.
	staticVec := make([]float32, mockEmbeddingDimensions)
	staticVec[0] = 1.0
	testMockEmb.SetVideoEmbeddingsPerCall(3)
	testMockEmb.UseStaticEmbedding(staticVec)

	first, err := c.CreateVideoFromBytes(ctx, acct, minimalMP4(t, 1))
	if err != nil {
		t.Fatalf("create first video: %v", err)
	}
	if first.GetStatus() != v1.CreateMemeStatus_STATUS_NEW {
		t.Fatalf("expected STATUS_NEW for first video, got %s", first.GetStatus())
	}
	originalID := first.GetResult().GetId()

	// Second video has different bytes (different hash) but the same embeddings.
	second, err := c.CreateVideoFromBytes(ctx, acct, minimalMP4(t, 2))
	if err != nil {
		t.Fatalf("create second video: %v", err)
	}
	if second.GetStatus() != v1.CreateMemeStatus_STATUS_DUPLICATE {
		t.Fatalf("expected STATUS_DUPLICATE when all embeddings match, got %s", second.GetStatus())
	}
	if second.GetResult().GetId() != originalID {
		t.Errorf("duplicate result id %s != original id %s", second.GetResult().GetId(), originalID)
	}
}

// TestVideoDeduplicationOnlyOneEmbeddingSame verifies that a video where only
// one of several embeddings matches an existing video is NOT a duplicate
// (below the PercentageDuplicatePartsThreshold of 70%).
func TestVideoDeduplicationOnlyOneEmbeddingSame(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("e02")

	staticVec := make([]float32, mockEmbeddingDimensions)
	staticVec[0] = 1.0

	// Video A: 3 identical embeddings stored in ES.
	testMockEmb.SetVideoEmbeddingsPerCall(3)
	testMockEmb.UseStaticEmbedding(staticVec)

	first, err := c.CreateVideoFromBytes(ctx, acct, minimalMP4(t, 3))
	if err != nil {
		t.Fatalf("create first video: %v", err)
	}
	if first.GetStatus() != v1.CreateMemeStatus_STATUS_NEW {
		t.Fatalf("expected STATUS_NEW for first video, got %s", first.GetStatus())
	}

	// Video B: 3 embeddings, only the first matches A — 1/3 ≈ 33% < 70% threshold.
	unique1 := make([]float32, mockEmbeddingDimensions)
	unique1[5] = 1.0
	unique2 := make([]float32, mockEmbeddingDimensions)
	unique2[6] = 1.0
	testMockEmb.QueueVideoEmbeddings([]entity.EmbeddingItem{
		{Data: staticVec, Model: "mock-model", Type: entity.EmbeddingTypeVideo},
		{Data: unique1, Model: "mock-model", Type: entity.EmbeddingTypeVideo},
		{Data: unique2, Model: "mock-model", Type: entity.EmbeddingTypeVideo},
	})

	second, err := c.CreateVideoFromBytes(ctx, acct, minimalMP4(t, 4))
	if err != nil {
		t.Fatalf("create second video: %v", err)
	}
	if second.GetStatus() != v1.CreateMemeStatus_STATUS_NEW {
		t.Fatalf("expected STATUS_NEW when only 1 of 3 embeddings matches, got %s", second.GetStatus())
	}
}
