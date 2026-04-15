package e2e_test

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	v1 "github.com/weoses/memelo/gen/proto/v1"
)

func TestCreateMeme(t *testing.T) {
	defer cleanup()
	imageData, err := os.ReadFile("images/test-pic-cat.jpeg")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}

	resp, err := searchClient.CreateMeme(context.Background(), &v1.CreateMemeRequest{
		AccountId: testAccountId,
		Image: &v1.MediaDataDto{
			Data: imageData,
		},
	})
	if err != nil {
		t.Fatalf("CreateMeme failed: %v", err)
	}
	if resp.GetResult().GetId() == "" {
		t.Fatal("expected non-empty ID in CreateMeme response")
	}

	if resp.GetStatus() != v1.CreateMemeStatus_STATUS_NEW {
		t.Fatal("expected CreateMeme status STATUS_NEW")
	}
}

func TestCreateDuplicate(t *testing.T) {
	defer cleanup()
	imageData, err := os.ReadFile("images/test-pic-cat.jpeg")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}

	resp, err := searchClient.CreateMeme(context.Background(), &v1.CreateMemeRequest{
		AccountId: testAccountId,
		Image: &v1.MediaDataDto{
			Data: imageData,
		},
	})
	if err != nil {
		t.Fatalf("CreateMeme failed: %v", err)
	}
	if resp.GetResult().GetId() == "" {
		t.Fatal("expected non-empty ID in CreateMeme response")
	}

	resp2, err := searchClient.CreateMeme(context.Background(), &v1.CreateMemeRequest{
		AccountId: testAccountId,
		Image: &v1.MediaDataDto{
			Data: imageData,
		},
	})
	if err != nil {
		t.Fatalf("second CreateMeme failed: %v", err)
	}
	if resp2.GetResult().GetId() == "" {
		t.Fatal("expected non-empty ID in CreateMeme response")
	}

	if resp2.GetStatus() != v1.CreateMemeStatus_STATUS_DUPLICATE {
		t.Fatal("expected CreateMeme status STATUS_DUPLICATE")
	}

	if resp.GetResult().GetId() != resp2.GetResult().GetId() {
		t.Fatal("expected CreateMeme response same ids if duplicate")
	}
}

func TestSearchMeme_Simple(t *testing.T) {
	defer cleanup()
	imageData, err := os.ReadFile("images/test-pic-cat.jpeg")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}

	respCreate, err := searchClient.CreateMeme(context.Background(), &v1.CreateMemeRequest{
		AccountId: testAccountId,
		Image: &v1.MediaDataDto{
			Data: imageData,
		},
	})
	if err != nil {
		t.Fatalf("CreateMeme failed: %v", err)
	}
	if respCreate.GetResult().GetId() == "" {
		t.Fatal("expected non-empty ID in CreateMeme response")
	}

	if respCreate.GetStatus() != v1.CreateMemeStatus_STATUS_NEW && respCreate.GetStatus() != v1.CreateMemeStatus_STATUS_DUPLICATE {
		t.Fatal("expected CreateMeme status STATUS_NEW or STATUS_DUPLICATE")
	}

	respSearch, err := searchClient.SearchMeme(context.Background(), &v1.SearchMemeRequest{
		AccountId: testAccountId,
		Query:     "guys",
	})
	if err != nil {
		t.Fatalf("SearchMeme failed: %v", err)
	}
	if respSearch == nil {
		t.Fatal("expected non-nil response from SearchMeme")
	}
	if len(respSearch.Results) == 0 {
		t.Fatal("expected at least one result")
	}
	if respSearch.SearcherName != "simple_searcher" {
		t.Fatal("expected simple_searcher")
	}
	if !strings.Contains(respSearch.Results[0].OcrResult, "ok guys this is my face reveal") {
		t.Fatalf("expected the result to be 'ok guys this is my face reveal' to be contained in the result, got %s", respSearch.Results[0].OcrResult)
	}
}

func TestSearchMeme_All(t *testing.T) {
	defer cleanup()
	imageData, err := os.ReadFile("images/test-pic-cat.jpeg")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}

	respCreate, err := searchClient.CreateMeme(context.Background(), &v1.CreateMemeRequest{
		AccountId: testAccountId,
		Image: &v1.MediaDataDto{
			Data: imageData,
		},
	})
	if err != nil {
		t.Fatalf("CreateMeme failed: %v", err)
	}
	if respCreate.GetResult().GetId() == "" {
		t.Fatal("expected non-empty ID in CreateMeme response")
	}

	respSearch, err := searchClient.SearchMeme(context.Background(), &v1.SearchMemeRequest{
		AccountId: testAccountId,
		Query:     "",
	})
	if err != nil {
		t.Fatalf("SearchMeme failed: %v", err)
	}
	if respSearch == nil {
		t.Fatal("expected non-nil response from SearchMeme")
	}
	if len(respSearch.Results) == 0 {
		t.Fatal("expected at least one result")
	}
	if respSearch.SearcherName != "all_searcher" {
		t.Fatal("expected all_searcher")
	}
	if !strings.Contains(respSearch.Results[0].OcrResult, "ok guys this is my face reveal") {
		t.Fatalf("expected the result to be 'ok guys this is my face reveal' to be contained in the result, got %s", respSearch.Results[0].OcrResult)
	}
}

func TestSearchMeme_ById(t *testing.T) {
	defer cleanup()
	imageData, err := os.ReadFile("images/test-pic-cat.jpeg")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}

	respCreate, err := searchClient.CreateMeme(context.Background(), &v1.CreateMemeRequest{
		AccountId: testAccountId,
		Image: &v1.MediaDataDto{
			Data: imageData,
		},
	})
	if err != nil {
		t.Fatalf("CreateMeme failed: %v", err)
	}
	id := respCreate.GetResult().GetId()
	if id == "" {
		t.Fatal("expected non-empty ID in CreateMeme response")
	}

	respSearch, err := searchClient.SearchMeme(context.Background(), &v1.SearchMemeRequest{
		AccountId: testAccountId,
		Query:     id,
	})
	if err != nil {
		t.Fatalf("SearchMeme by ID failed: %v", err)
	}
	if len(respSearch.Results) != 1 {
		t.Fatalf("expected exactly 1 result, got %d", len(respSearch.Results))
	}
	if respSearch.Results[0].Id != id {
		t.Fatalf("expected result ID %s, got %s", id, respSearch.Results[0].Id)
	}
	if respSearch.SearcherName != "id_searcher" {
		t.Fatalf("expected id_searcher, got %s", respSearch.SearcherName)
	}
}

func TestSearchMeme_ById_ValidImageLinks(t *testing.T) {
	defer cleanup()
	imageData, err := os.ReadFile("images/test-pic-cat.jpeg")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}

	respCreate, err := searchClient.CreateMeme(context.Background(), &v1.CreateMemeRequest{
		AccountId: testAccountId,
		Image: &v1.MediaDataDto{
			Data: imageData,
		},
	})
	if err != nil {
		t.Fatalf("CreateMeme failed: %v", err)
	}
	id := respCreate.GetResult().GetId()
	if id == "" {
		t.Fatal("expected non-empty ID in CreateMeme response")
	}

	respSearch, err := searchClient.SearchMeme(context.Background(), &v1.SearchMemeRequest{
		AccountId: testAccountId,
		Query:     id,
	})
	if err != nil {
		t.Fatalf("SearchMeme by ID failed: %v", err)
	}
	if len(respSearch.Results) != 1 {
		t.Fatalf("expected exactly 1 result, got %d", len(respSearch.Results))
	}

	meme := respSearch.Results[0]

	original := meme.GetMediaOriginal()
	if original == nil {
		t.Fatal("expected non-nil ImageOriginal")
	}
	if original.GetUrl() == "" {
		t.Fatal("expected non-empty ImageOriginal URL")
	}

	thumbnail := meme.GetImageThumbnail()
	if thumbnail == nil {
		t.Fatal("expected non-nil ImageThumbnail")
	}
	if thumbnail.GetUrl() == "" {
		t.Fatal("expected non-empty ImageThumbnail URL")
	}

	if original.GetUrl() == thumbnail.GetUrl() {
		t.Fatalf("expected original and thumbnail URLs to differ, both are %q", original.GetUrl())
	}

	for _, tc := range []struct {
		name string
		url  string
	}{
		{"original", original.GetUrl()},
		{"thumbnail", thumbnail.GetUrl()},
	} {
		resp, err := http.Get(tc.url)
		if err != nil {
			t.Fatalf("%s URL %q is not accessible: %v", tc.name, tc.url, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s URL %q returned status %d", tc.name, tc.url, resp.StatusCode)
		}
	}
}

func TestSearchMeme_EmbeddingSearch(t *testing.T) {
	defer cleanup()
	imageData, err := os.ReadFile("images/test-pic-cat.jpeg")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}

	respCreate, err := searchClient.CreateMeme(context.Background(), &v1.CreateMemeRequest{
		AccountId: testAccountId,
		Image: &v1.MediaDataDto{
			Data: imageData,
		},
	})
	if err != nil {
		t.Fatalf("CreateMeme failed: %v", err)
	}
	if respCreate.GetResult().GetId() == "" {
		t.Fatal("expected non-empty ID in CreateMeme response")
	}

	respSearch, err := searchClient.SearchMeme(context.Background(), &v1.SearchMemeRequest{
		AccountId: testAccountId,
		Query:     "cat",
	})
	if err != nil {
		t.Fatalf("SearchMeme by embedding failed: %v", err)
	}
	if len(respSearch.Results) == 0 {
		t.Fatal("expected at least one result")
	}
	if respSearch.SearcherName != "text_embedding_searcher" {
		t.Fatalf("expected text_embedding_searcher, got %s", respSearch.SearcherName)
	}
}

func TestCreateVideoMemeV1(t *testing.T) {
	createVideoTestInternal(t, "videos/video-test-v1-steven.mp4")
}

func TestCreateVideoMemeV2(t *testing.T) {
	createVideoTestInternal(t, "videos/video-test-v2-static-text.mp4")
}

func TestCreateVideoMemeV3(t *testing.T) {
	createVideoTestInternal(t, "videos/video-test-v3-language.mp4")
}

func createVideoTestInternal(t *testing.T, file string) {
	defer cleanup()
	videoData, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("failed to read test video: %v", err)
	}

	resp, err := searchClient.CreateMeme(context.Background(), &v1.CreateMemeRequest{
		AccountId: testAccountId,
		Video: &v1.MediaDataDto{
			Data: videoData,
		},
	})
	if err != nil {
		t.Fatalf("CreateMeme (video) failed: %v", err)
	}
	if resp.GetResult().GetId() == "" {
		t.Fatal("expected non-empty ID in CreateMeme response")
	}
	if resp.GetStatus() != v1.CreateMemeStatus_STATUS_NEW {
		t.Fatalf("expected STATUS_NEW, got %v", resp.GetStatus())
	}

	println(resp.GetResult().GetOcrResult())

	id := resp.GetResult().GetId()
	respSearch, err := searchClient.SearchMeme(context.Background(), &v1.SearchMemeRequest{
		AccountId: testAccountId,
		Query:     id,
	})
	if err != nil {
		t.Fatalf("SearchMeme by ID failed: %v", err)
	}
	if len(respSearch.Results) != 1 {
		t.Fatalf("expected exactly 1 result, got %d", len(respSearch.Results))
	}

	meme := respSearch.Results[0]
	if meme.Id != id {
		t.Fatalf("expected result ID %s, got %s", id, meme.Id)
	}

	original := meme.GetMediaOriginal()
	if original == nil {
		t.Fatal("expected non-nil MediaOriginal")
	}
	if original.GetUrl() == "" {
		t.Fatal("expected non-empty MediaOriginal URL")
	}

	thumbnail := meme.GetImageThumbnail()
	if thumbnail == nil {
		t.Fatal("expected non-nil ImageThumbnail")
	}
	if thumbnail.GetUrl() == "" {
		t.Fatal("expected non-empty ImageThumbnail URL")
	}

	for _, tc := range []struct {
		name string
		url  string
	}{
		{"original", original.GetUrl()},
		{"thumbnail", thumbnail.GetUrl()},
	} {
		resp, err := http.Get(tc.url)
		if err != nil {
			t.Fatalf("%s URL %q is not accessible: %v", tc.name, tc.url, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s URL %q returned status %d", tc.name, tc.url, resp.StatusCode)
		}
	}
}

func TestSearchMeme_Empty(t *testing.T) {
	resp, err := searchClient.SearchMeme(context.Background(), &v1.SearchMemeRequest{
		AccountId: testAccountId,
		Query:     "test",
	})
	if err != nil {
		t.Fatalf("SearchMeme failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response from SearchMeme")
	}
	if len(resp.Results) > 0 {
		t.Fatal("expected zero results")
	}
}
