package functional_test

import (
	"context"
	"testing"
)

func ptr(s string) *string { return &s }

// TestSearchMeme_TypeFilter_ImageOnly verifies that type="image" returns image
// results when the account contains only images (AllSearcher path, empty query).
func TestSearchMeme_TypeFilter_ImageOnly(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("ca001")

	if _, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1)); err != nil {
		t.Fatalf("create meme: %v", err)
	}

	resp, err := c.SearchMemeWithType(ctx, acct, "", 10, ptr("image"))
	if err != nil {
		t.Fatalf("SearchMeme(type=image): %v", err)
	}
	if len(resp.GetResults()) == 0 {
		t.Fatal("expected at least one image result")
	}
	for _, r := range resp.GetResults() {
		if r.GetType() != "image" {
			t.Errorf("expected type 'image', got %q (id=%s)", r.GetType(), r.GetId())
		}
	}
}

// TestSearchMeme_TypeFilter_VideoOnImageStore_ReturnsEmpty verifies that
// type="video" returns no results when the account contains only images.
func TestSearchMeme_TypeFilter_VideoOnImageStore_ReturnsEmpty(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("ca002")

	if _, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1)); err != nil {
		t.Fatalf("create meme: %v", err)
	}

	resp, err := c.SearchMemeWithType(ctx, acct, "", 10, ptr("video"))
	if err != nil {
		t.Fatalf("SearchMeme(type=video): %v", err)
	}
	if len(resp.GetResults()) != 0 {
		t.Fatalf("expected 0 results for video filter on image-only store, got %d", len(resp.GetResults()))
	}
}

// TestSearchMeme_NoTypeFilter_ReturnsAll verifies that omitting the type filter
// returns all results regardless of type.
func TestSearchMeme_NoTypeFilter_ReturnsAll(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("ca003")

	const total = 3
	for i := range total {
		if _, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(i+1)); err != nil {
			t.Fatalf("create meme %d: %v", i, err)
		}
	}

	resp, err := c.SearchMemeWithType(ctx, acct, "", 10, nil)
	if err != nil {
		t.Fatalf("SearchMeme(no type): %v", err)
	}
	if len(resp.GetResults()) != total {
		t.Fatalf("expected %d results without type filter, got %d", total, len(resp.GetResults()))
	}
}

// TestSearchMeme_TypeFilter_AccountIsolation verifies that the type filter is
// applied per-account and does not bleed across accounts.
func TestSearchMeme_TypeFilter_AccountIsolation(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acctA := accountID("ca004")
	acctB := accountID("ca005")

	if _, err := c.CreateMemeFromBytes(ctx, acctA, minimalJPEG(1)); err != nil {
		t.Fatalf("create meme for acctA: %v", err)
	}

	// acctB has no memes — even without the type filter it's empty
	respB, err := c.SearchMemeWithType(ctx, acctB, "", 10, ptr("image"))
	if err != nil {
		t.Fatalf("SearchMeme acctB: %v", err)
	}
	if len(respB.GetResults()) != 0 {
		t.Fatalf("expected 0 results for acctB, got %d", len(respB.GetResults()))
	}

	// acctA with type=image should return its own meme
	respA, err := c.SearchMemeWithType(ctx, acctA, "", 10, ptr("image"))
	if err != nil {
		t.Fatalf("SearchMeme acctA: %v", err)
	}
	if len(respA.GetResults()) == 0 {
		t.Fatal("expected at least one result for acctA")
	}
}
