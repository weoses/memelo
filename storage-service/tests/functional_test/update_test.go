package functional_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	v1 "github.com/weoses/memelo/gen/proto/v1"
)

func fetchURL(t *testing.T, url string) []byte {
	t.Helper()
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: expected 200, got %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body %s: %v", url, err)
	}
	return data
}

// TestUpdateMemeTextField verifies that updating a single LLM-extracted text
// field persists the new value, recomputes OcrResult, and does not trigger LLM
// re-extraction.
func TestUpdateMemeTextField(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("ab001")

	createResp, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1))
	if err != nil {
		t.Fatalf("create meme: %v", err)
	}
	id := createResp.GetResult().GetId()

	testMockEx.Reset()

	newText := "hello updated world"
	updateResp, err := c.UpdateMeme(ctx, &v1.UpdateMemeRequest{
		AccountId:    acct,
		Id:           id,
		OnScreenText: &newText,
	})
	if err != nil {
		t.Fatalf("update meme: %v", err)
	}

	result := updateResp.GetResult()
	if got := result.GetOnScreenText(); got != newText {
		t.Errorf("OnScreenText: got %q, want %q", got, newText)
	}
	if ocr := result.GetOcrResult(); !strings.Contains(ocr, newText) {
		t.Errorf("OcrResult %q does not contain updated text %q", ocr, newText)
	}
	if n := testMockEx.CallCount(); n != 0 {
		t.Errorf("LLM extractor must not be called during text-only update, got %d calls", n)
	}
}

// TestUpdateMemeOriginal verifies that replacing the original media re-encodes
// the file, stores new content at the same S3 path, and does not trigger LLM
// re-extraction.
func TestUpdateMemeOriginal(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("ab02")

	createResp, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1))
	if err != nil {
		t.Fatalf("create meme: %v", err)
	}
	id := createResp.GetResult().GetId()
	beforeOriginal := fetchURL(t, createResp.GetResult().GetMediaOriginal().GetUrl())
	beforeThumb := fetchURL(t, createResp.GetResult().GetImageThumbnail().GetUrl())

	testMockEx.Reset()

	updateResp, err := c.UpdateMeme(ctx, &v1.UpdateMemeRequest{
		AccountId: acct,
		Id:        id,
		Original:  &v1.MediaDataDto{Data: minimalJPEG(2)},
	})
	if err != nil {
		t.Fatalf("update meme original: %v", err)
	}

	afterOriginal := fetchURL(t, updateResp.GetResult().GetMediaOriginal().GetUrl())
	afterThumb := fetchURL(t, updateResp.GetResult().GetImageThumbnail().GetUrl())

	if bytes.Equal(beforeOriginal, afterOriginal) {
		t.Error("expected original content to change after media update")
	}
	if !bytes.Equal(beforeThumb, afterThumb) {
		t.Error("expected thumbnail to be unchanged when only original is updated")
	}
	if n := testMockEx.CallCount(); n != 0 {
		t.Errorf("LLM extractor must not be called during media update, got %d calls", n)
	}
}

// TestUpdateMemeThumbnail verifies that providing a new thumbnail re-encodes it
// through MakeThumbnail and stores different content at the thumbnail S3 path.
func TestUpdateMemeThumbnail(t *testing.T) {
	resetState(t)
	ctx := context.Background()
	c := testClient()
	acct := accountID("ab003")

	createResp, err := c.CreateMemeFromBytes(ctx, acct, minimalJPEG(1))
	if err != nil {
		t.Fatalf("create meme: %v", err)
	}
	id := createResp.GetResult().GetId()
	beforeThumb := fetchURL(t, createResp.GetResult().GetImageThumbnail().GetUrl())
	beforeOriginal := fetchURL(t, createResp.GetResult().GetMediaOriginal().GetUrl())

	updateResp, err := c.UpdateMeme(ctx, &v1.UpdateMemeRequest{
		AccountId: acct,
		Id:        id,
		Thumbnail: &v1.MediaDataDto{Data: minimalJPEG(3)},
	})
	if err != nil {
		t.Fatalf("update meme thumbnail: %v", err)
	}

	afterThumb := fetchURL(t, updateResp.GetResult().GetImageThumbnail().GetUrl())
	afterOriginal := fetchURL(t, updateResp.GetResult().GetMediaOriginal().GetUrl())

	if bytes.Equal(beforeThumb, afterThumb) {
		t.Error("expected thumbnail content to change after thumbnail update")
	}
	if !bytes.Equal(beforeOriginal, afterOriginal) {
		t.Error("expected original to be unchanged when only thumbnail is updated")
	}
}
