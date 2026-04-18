package functional_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"testing"

	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/storage-service/ocr"
)

// minimalJPEG creates a small solid-color JPEG. Pass a seed to get distinct
// bytes (and thus a distinct MD5 hash) for each image.
func minimalJPEG(seed int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	r := uint8(seed*25) % 255
	g := uint8(seed*37) % 255
	b := uint8(seed*59) % 255
	for y := range 10 {
		for x := range 10 {
			img.Set(x, y, color.RGBA{R: r + uint8(x), G: g + uint8(y), B: b, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

func TestCreateAndSearchMeme(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("001")

	resp, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1))
	if err != nil {
		t.Fatalf("create meme: %v", err)
	}
	if resp.GetResult() == nil || resp.GetResult().GetId() == "" {
		t.Fatal("expected non-empty result id in response")
	}

	searchResp, err := c.SearchMeme(ctx, acct, "", 10)
	if err != nil {
		t.Fatalf("search meme: %v", err)
	}
	if len(searchResp.GetResults()) == 0 {
		t.Fatal("expected at least one result after create")
	}
}

func TestDeleteAll(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("002")

	if _, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1)); err != nil {
		t.Fatalf("create meme: %v", err)
	}

	if err := c.DeleteAll(ctx, acct); err != nil {
		t.Fatalf("delete all: %v", err)
	}

	searchResp, err := c.SearchMeme(ctx, acct, "", 10)
	if err != nil {
		t.Fatalf("search after delete: %v", err)
	}
	if len(searchResp.GetResults()) != 0 {
		t.Fatalf("expected empty results after delete all, got %d", len(searchResp.GetResults()))
	}
}

func TestPagination(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("003")

	const total = 10
	createdIDs := make(map[string]struct{}, total)
	for i := range total {
		resp, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(i+1))
		if err != nil {
			t.Fatalf("create meme %d: %v", i, err)
		}
		id := resp.GetResult().GetId()
		if id == "" {
			t.Fatalf("create meme %d: empty result id (duplicate?)", i)
		}
		createdIDs[id] = struct{}{}
	}
	if len(createdIDs) != total {
		t.Fatalf("expected %d unique memes, got %d (some detected as duplicates)", total, len(createdIDs))
	}

	// Collect all results via cursor pagination with page_size=3.
	const pageSize = 3
	seen := make(map[string]struct{}, total)
	var cursor *v1.PipelinePagination
	for {
		resp, err := c.SearchMemePage(ctx, acct, "", pageSize, cursor)
		if err != nil {
			t.Fatalf("search page (cursor=%v): %v", cursor, err)
		}
		for _, r := range resp.GetResults() {
			id := r.GetId()
			if _, dup := seen[id]; dup {
				t.Errorf("duplicate id %s in paginated results", id)
			}
			seen[id] = struct{}{}
		}
		if len(resp.GetResults()) < pageSize || resp.GetLastId() == nil {
			break
		}
		cursor = resp.GetLastId()
	}

	if len(seen) != total {
		t.Fatalf("expected %d results across all pages, got %d", total, len(seen))
	}
}

func TestS3LinksAreValid(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("004")

	resp, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1))
	if err != nil {
		t.Fatalf("create meme: %v", err)
	}
	meme := resp.GetResult()
	if meme == nil {
		t.Fatal("nil result")
	}

	for _, name := range []string{"thumbnail", "original"} {
		var url string
		switch name {
		case "thumbnail":
			url = meme.GetImageThumbnail().GetUrl()
		case "original":
			url = meme.GetMediaOriginal().GetUrl()
		}
		if url == "" {
			t.Errorf("%s url is empty", name)
			continue
		}
		httpResp, err := http.Get(url) //nolint:noctx
		if err != nil {
			t.Errorf("%s GET %s: %v", name, url, err)
			continue
		}
		_ = httpResp.Body.Close()
		if httpResp.StatusCode != http.StatusOK {
			t.Errorf("%s: expected 200 got %d (url=%s)", name, httpResp.StatusCode, url)
		}
	}
}

func TestIDSearcherReturnsSingleResult(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("005")

	resp1, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1))
	if err != nil {
		t.Fatalf("create meme 1: %v", err)
	}
	if _, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(2)); err != nil {
		t.Fatalf("create meme 2: %v", err)
	}

	targetID := resp1.GetResult().GetId()
	if targetID == "" {
		t.Fatal("empty id for first meme")
	}

	searchResp, err := c.SearchMeme(ctx, acct, targetID, 10)
	if err != nil {
		t.Fatalf("search by id: %v", err)
	}
	results := searchResp.GetResults()
	if len(results) != 1 {
		t.Fatalf("expected exactly 1 result, got %d", len(results))
	}
	if results[0].GetId() != targetID {
		t.Errorf("expected id %s, got %s", targetID, results[0].GetId())
	}
}

func TestDeduplicationByHash(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("006")

	img := minimalJPEG(1)

	first, err := c.CreateMemeFromBytes(ctx, acct, img)
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	if first.GetStatus() != v1.CreateMemeStatus_STATUS_NEW {
		t.Fatalf("expected STATUS_NEW for first upload, got %s", first.GetStatus())
	}
	originalID := first.GetResult().GetId()

	second, err := c.CreateMemeFromBytes(ctx, acct, img)
	if err != nil {
		t.Fatalf("create second (duplicate): %v", err)
	}
	if second.GetStatus() != v1.CreateMemeStatus_STATUS_DUPLICATE {
		t.Fatalf("expected STATUS_DUPLICATE for identical bytes, got %s", second.GetStatus())
	}
	if second.GetResult().GetId() != originalID {
		t.Errorf("duplicate result id %s != original id %s", second.GetResult().GetId(), originalID)
	}

	// Only one entry should exist in search results.
	searchResp, err := c.SearchMeme(ctx, acct, "", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(searchResp.GetResults()) != 1 {
		t.Fatalf("expected 1 stored meme after duplicate upload, got %d", len(searchResp.GetResults()))
	}
}

func TestDeduplicationByEmbedding(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("007")

	// Force the same embedding for every call — triggers EP40 semantic deduplication.
	staticVec := make([]float32, mockEmbeddingDimensions)
	staticVec[0] = 1.0
	testMockEmb.UseStaticEmbedding(staticVec)

	// Two images with different bytes (different hash) but same embedding.
	first, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1))
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	if first.GetStatus() != v1.CreateMemeStatus_STATUS_NEW {
		t.Fatalf("expected STATUS_NEW for first upload, got %s", first.GetStatus())
	}
	originalID := first.GetResult().GetId()

	second, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(2))
	if err != nil {
		t.Fatalf("create second (semantic duplicate): %v", err)
	}
	if second.GetStatus() != v1.CreateMemeStatus_STATUS_DUPLICATE {
		t.Fatalf("expected STATUS_DUPLICATE for same-embedding image, got %s", second.GetStatus())
	}
	if second.GetResult().GetId() != originalID {
		t.Errorf("duplicate result id %s != original id %s", second.GetResult().GetId(), originalID)
	}

	// Only one entry should exist in search results.
	searchResp, err := c.SearchMeme(ctx, acct, "", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(searchResp.GetResults()) != 1 {
		t.Fatalf("expected 1 stored meme after semantic duplicate upload, got %d", len(searchResp.GetResults()))
	}
}

func TestTextSearch(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("008")

	testMockEx.SetResult(&ocr.MediaExtractResult{OnScreenText: "orange tabby cat sitting on a mat"})
	resp1, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1))
	if err != nil {
		t.Fatalf("create meme 1: %v", err)
	}
	catID := resp1.GetResult().GetId()

	testMockEx.SetResult(&ocr.MediaExtractResult{OnScreenText: "golden retriever dog running in park"})
	if _, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(2)); err != nil {
		t.Fatalf("create meme 2: %v", err)
	}
	testMockEx.Reset()

	searchResp, err := c.SearchMeme(ctx, acct, "orange tabby cat", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	results := searchResp.GetResults()
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if results[0].GetId() != catID {
		t.Errorf("expected cat image (%s) as first result, got %s (ocr=%q)", catID, results[0].GetId(), results[0].GetOcrResult())
	}
}
