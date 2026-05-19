package functional_test

import (
	"context"
	"testing"

	v1 "github.com/weoses/memelo/gen/proto/v1"
)

// TestImageDeduplicationLlmConfirmed verifies that when the embedding search
// finds a candidate and the LLM confirms it is a duplicate, the second image
// is reported as STATUS_DUPLICATE pointing to the first.
func TestImageDeduplicationLlmConfirmed(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("d01")

	staticVec := make([]float32, mockEmbeddingDimensions)
	staticVec[0] = 1.0
	testMockEmb.UseStaticEmbedding(staticVec)
	testMockEx.SetCheckDuplicateResult(true)

	first, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1))
	if err != nil {
		t.Fatalf("create first image: %v", err)
	}
	if first.GetStatus() != v1.CreateMemeStatus_STATUS_NEW {
		t.Fatalf("expected STATUS_NEW for first image, got %s", first.GetStatus())
	}
	originalID := first.GetResult().GetId()

	// Second image has different bytes (different hash) but the same embedding.
	second, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(2))
	if err != nil {
		t.Fatalf("create second image: %v", err)
	}
	if second.GetStatus() != v1.CreateMemeStatus_STATUS_DUPLICATE {
		t.Fatalf("expected STATUS_DUPLICATE when LLM confirms duplicate, got %s", second.GetStatus())
	}
	if second.GetResult().GetId() != originalID {
		t.Errorf("duplicate result id %s != original id %s", second.GetResult().GetId(), originalID)
	}
	if testMockEx.CheckDuplicateCallCount() == 0 {
		t.Error("expected CheckDuplicate to be called on the extractor, but call count is 0")
	}
}

// TestImageDeduplicationLlmDenied verifies that when the embedding search finds
// a candidate but the LLM says it is NOT a duplicate, the second image is stored
// as a new entry (STATUS_NEW).
func TestImageDeduplicationLlmDenied(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("d02")

	staticVec := make([]float32, mockEmbeddingDimensions)
	staticVec[0] = 1.0
	testMockEmb.UseStaticEmbedding(staticVec)
	testMockEx.SetCheckDuplicateResult(false)

	if _, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1)); err != nil {
		t.Fatalf("create first image: %v", err)
	}

	second, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(2))
	if err != nil {
		t.Fatalf("create second image: %v", err)
	}
	if second.GetStatus() != v1.CreateMemeStatus_STATUS_NEW {
		t.Fatalf("expected STATUS_NEW when LLM denies duplicate, got %s", second.GetStatus())
	}
	if testMockEx.CheckDuplicateCallCount() == 0 {
		t.Error("expected CheckDuplicate to be called on the extractor, but call count is 0")
	}
}
