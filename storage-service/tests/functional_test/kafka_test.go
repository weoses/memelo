package functional_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/weoses/memelo/gen/proto/v1"
)

func TestKafkaCreateAndSearchMeme(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("k01")

	resp, err := testKafkaClient.CreateImageMeme(ctx, acct, minimalJPEG(1))
	if err != nil {
		t.Fatalf("kafka create image: %v", err)
	}
	if resp.GetErrorMessage() != "" {
		t.Fatalf("unexpected error in response: %s", resp.GetErrorMessage())
	}
	if resp.GetPayload().GetResult().GetId() == "" {
		t.Fatal("expected non-empty result id in kafka response")
	}

	searchResp, err := c.SearchMeme(ctx, acct, "", 10)
	if err != nil {
		t.Fatalf("search after kafka create: %v", err)
	}
	if len(searchResp.GetResults()) == 0 {
		t.Fatal("expected at least one result in search after kafka create")
	}
}

func TestKafkaDeduplicationByHash(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("k02")

	img := minimalJPEG(1)

	first, err := testKafkaClient.CreateImageMeme(ctx, acct, img)
	if err != nil {
		t.Fatalf("kafka create first: %v", err)
	}
	if first.GetErrorMessage() != "" {
		t.Fatalf("first upload error: %s", first.GetErrorMessage())
	}
	if first.GetPayload().GetStatus() != v1.CreateMemeStatus_STATUS_NEW {
		t.Fatalf("expected STATUS_NEW for first upload, got %s", first.GetPayload().GetStatus())
	}
	originalID := first.GetPayload().GetResult().GetId()

	second, err := testKafkaClient.CreateImageMeme(ctx, acct, img)
	if err != nil {
		t.Fatalf("kafka create second (duplicate): %v", err)
	}
	if second.GetErrorMessage() != "" {
		t.Fatalf("second upload error: %s", second.GetErrorMessage())
	}
	if second.GetPayload().GetStatus() != v1.CreateMemeStatus_STATUS_DUPLICATE {
		t.Fatalf("expected STATUS_DUPLICATE for identical bytes, got %s", second.GetPayload().GetStatus())
	}
	if second.GetPayload().GetResult().GetId() != originalID {
		t.Errorf("duplicate result id %s != original id %s", second.GetPayload().GetResult().GetId(), originalID)
	}

	searchResp, err := c.SearchMeme(ctx, acct, "", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(searchResp.GetResults()) != 1 {
		t.Fatalf("expected 1 stored meme after duplicate upload, got %d", len(searchResp.GetResults()))
	}
}

func TestKafkaDeduplicationByEmbedding(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("k03")

	staticVec := make([]float32, mockEmbeddingDimensions)
	staticVec[0] = 1.0
	testMockEmb.UseStaticEmbedding(staticVec)
	testMockEx.SetCheckDuplicateResult(true)

	first, err := testKafkaClient.CreateImageMeme(ctx, acct, minimalJPEG(1))
	if err != nil {
		t.Fatalf("kafka create first: %v", err)
	}
	if first.GetPayload().GetStatus() != v1.CreateMemeStatus_STATUS_NEW {
		t.Fatalf("expected STATUS_NEW for first upload, got %s", first.GetPayload().GetStatus())
	}
	originalID := first.GetPayload().GetResult().GetId()

	second, err := testKafkaClient.CreateImageMeme(ctx, acct, minimalJPEG(2))
	if err != nil {
		t.Fatalf("kafka create second (semantic duplicate): %v", err)
	}
	if second.GetPayload().GetStatus() != v1.CreateMemeStatus_STATUS_DUPLICATE {
		t.Fatalf("expected STATUS_DUPLICATE for same-embedding image, got %s", second.GetPayload().GetStatus())
	}
	if second.GetPayload().GetResult().GetId() != originalID {
		t.Errorf("duplicate result id %s != original id %s", second.GetPayload().GetResult().GetId(), originalID)
	}

	searchResp, err := c.SearchMeme(ctx, acct, "", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(searchResp.GetResults()) != 1 {
		t.Fatalf("expected 1 stored meme after semantic duplicate upload, got %d", len(searchResp.GetResults()))
	}
}

func TestKafkaCreateVideo(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("k04")

	resp, err := testKafkaClient.CreateVideoMeme(ctx, acct, minimalMP4(t, 1))
	if err != nil {
		t.Fatalf("kafka create video: %v", err)
	}
	if resp.GetErrorMessage() != "" {
		t.Fatalf("unexpected error in response: %s", resp.GetErrorMessage())
	}
	if resp.GetPayload().GetResult().GetId() == "" {
		t.Fatal("expected non-empty result id in kafka video response")
	}

	searchResp, err := c.SearchMeme(ctx, acct, "", 10)
	if err != nil {
		t.Fatalf("search after kafka video create: %v", err)
	}
	if len(searchResp.GetResults()) == 0 {
		t.Fatal("expected at least one result in search after kafka video create")
	}
}

func TestKafkaInvalidAccountId(t *testing.T) {
	resetState(t)
	ctx := context.Background()

	req := &v1.KafkaCreateMemeRequest{
		Payload: &v1.CreateMemeRequest{
			AccountId: "not-a-uuid",
			Image:     &v1.MediaDataDto{Data: minimalJPEG(1)},
		},
	}
	resp, err := testKafkaClient.SendRawRequest(ctx, req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	if resp.GetErrorMessage() == "" {
		t.Fatal("expected non-empty error_message for invalid account_id")
	}
	if resp.GetPayload() != nil {
		t.Errorf("expected nil payload on error, got %v", resp.GetPayload())
	}
}

func TestKafkaNoPayload(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	acct := accountID("k06")

	req := &v1.KafkaCreateMemeRequest{
		Payload: &v1.CreateMemeRequest{
			AccountId: acct,
			// neither Image nor Video set
		},
	}
	resp, err := testKafkaClient.SendRawRequest(ctx, req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	if resp.GetErrorMessage() == "" {
		t.Fatal("expected non-empty error_message when neither image nor video provided")
	}
	if resp.GetPayload() != nil {
		t.Errorf("expected nil payload on error, got %v", resp.GetPayload())
	}
}

func TestKafkaCorrelationId(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	acct := accountID("k07")

	requestID := uuid.New().String()
	req := &v1.KafkaCreateMemeRequest{
		KafkaRequestId: requestID,
		Payload: &v1.CreateMemeRequest{
			AccountId: acct,
			Image:     &v1.MediaDataDto{Data: minimalJPEG(1)},
		},
	}
	resp, err := testKafkaClient.SendRawRequest(ctx, req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	if resp.KafkaRequestId != requestID {
		t.Errorf("correlation id mismatch: sent %q, got %q", requestID, resp.KafkaRequestId)
	}
}
