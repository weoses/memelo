package functional_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
)

func TestGetRandomMeme_ReturnsResult(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("010")

	created, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1))
	if err != nil {
		t.Fatalf("create meme: %v", err)
	}
	createdID := created.GetResult().GetId()

	resp, err := c.GetRandomMeme(ctx, acct, "")
	if err != nil {
		t.Fatalf("GetRandomMeme: %v", err)
	}
	if resp.GetResult() == nil {
		t.Fatal("expected non-nil result")
	}
	if resp.GetResult().GetId() != createdID {
		t.Errorf("expected id %s, got %s", createdID, resp.GetResult().GetId())
	}
	if resp.GetResult().GetType() == "" {
		t.Error("expected non-empty type in result")
	}
}

func TestGetRandomMeme_EmptyStore_ReturnsNotFound(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("011")

	_, err := c.GetRandomMeme(ctx, acct, "")
	if err == nil {
		t.Fatal("expected error for empty store, got nil")
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", err)
	}
}

func TestGetRandomMeme_TypeFilter_MatchesOnlyImages(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("012")

	if _, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1)); err != nil {
		t.Fatalf("create meme: %v", err)
	}

	resp, err := c.GetRandomMeme(ctx, acct, "image")
	if err != nil {
		t.Fatalf("GetRandomMeme(image): %v", err)
	}
	if resp.GetResult().GetType() != "image" {
		t.Errorf("expected type 'image', got %q", resp.GetResult().GetType())
	}
}

func TestGetRandomMeme_TypeFilter_VideoOnImageStore_ReturnsNotFound(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("013")

	if _, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1)); err != nil {
		t.Fatalf("create meme: %v", err)
	}

	_, err := c.GetRandomMeme(ctx, acct, "video")
	if err == nil {
		t.Fatal("expected not-found error when requesting video from image-only store")
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", err)
	}
}

func TestGetRandomMeme_Distribution(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("014")

	const numMemes = 3
	for i := range numMemes {
		if _, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(i+1)); err != nil {
			t.Fatalf("create meme %d: %v", i, err)
		}
	}

	// Call GetRandomMeme 10 times. With 3 memes the probability of all 10
	// returning the same ID is (1/3)^9 ≈ 0.0005%, so a uniform failure here
	// almost certainly means the endpoint is not random.
	const calls = 10
	seen := make(map[string]struct{}, numMemes)
	for range calls {
		resp, err := c.GetRandomMeme(ctx, acct, "")
		if err != nil {
			t.Fatalf("GetRandomMeme: %v", err)
		}
		seen[resp.GetResult().GetId()] = struct{}{}
	}
	if len(seen) < 2 {
		t.Errorf("expected at least 2 distinct IDs across %d calls, got %d — endpoint may not be random", calls, len(seen))
	}
}

func TestGetRandomMeme_AccountIsolation(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acctA := accountID("015")
	acctB := accountID("016")

	created, err := c.CreateMemeFromBytes(ctx, acctA, minimalJPEG(1))
	if err != nil {
		t.Fatalf("create meme for acctA: %v", err)
	}
	acctAID := created.GetResult().GetId()

	// acctB has no memes — must get not-found
	_, err = c.GetRandomMeme(ctx, acctB, "")
	if err == nil {
		t.Fatal("expected not-found for acctB, got nil")
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound for empty account, got %v", err)
	}

	// acctA gets its own meme
	resp, err := c.GetRandomMeme(ctx, acctA, "")
	if err != nil {
		t.Fatalf("GetRandomMeme for acctA: %v", err)
	}
	if resp.GetResult().GetId() != acctAID {
		t.Errorf("acctA: expected %s, got %s", acctAID, resp.GetResult().GetId())
	}
}
