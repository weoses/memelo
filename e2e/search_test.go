package e2e_test

import (
	"context"
	"os"
	"testing"

	v1 "github.com/weoses/memelo/gen/proto/v1"
)

func TestCreateMeme(t *testing.T) {
	accountId := genAccountId()
	imageData, err := os.ReadFile("images/test-pic-cat.jpeg")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}

	resp, err := searchClient.CreateMeme(context.Background(), &v1.CreateMemeRequest{
		AccountId: accountId,
		RawImage:  imageData,
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
	accountId := genAccountId()
	imageData, err := os.ReadFile("images/test-pic-cat.jpeg")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}

	resp, err := searchClient.CreateMeme(context.Background(), &v1.CreateMemeRequest{
		AccountId: accountId,
		RawImage:  imageData,
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

	resp2, err := searchClient.CreateMeme(context.Background(), &v1.CreateMemeRequest{
		AccountId: accountId,
		RawImage:  imageData,
	})
	if err != nil {
		t.Fatalf("CreateMeme failed: %v", err)
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
	accountId := genAccountId()

	imageData, err := os.ReadFile("images/test-pic-cat.jpeg")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}

	respCreate, err := searchClient.CreateMeme(context.Background(), &v1.CreateMemeRequest{
		AccountId: accountId,
		RawImage:  imageData,
	})
	if err != nil {
		t.Fatalf("CreateMeme failed: %v", err)
	}
	if respCreate.GetResult().GetId() == "" {
		t.Fatal("expected non-empty ID in CreateMeme response")
	}

	if respCreate.GetStatus() != v1.CreateMemeStatus_STATUS_NEW {
		t.Fatal("expected CreateMeme status STATUS_NEW")
	}

	respSearch, err := searchClient.SearchMeme(context.Background(), &v1.SearchMemeRequest{
		AccountId: accountId,
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
	if respCreate.Result.Id != respSearch.Results[0].Id {
		t.Fatal("expected the result to be the same as the search result")
	}
	if respSearch.Results[0].OcrResult != "ok guys this is my face reveal..." {
		t.Fatal("expected the result to be 'ok guys this is my face reveal'")
	}
}

func TestSearchMeme_All(t *testing.T) {
	accountId := genAccountId()

	imageData, err := os.ReadFile("images/test-pic-cat.jpeg")
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}

	respCreate, err := searchClient.CreateMeme(context.Background(), &v1.CreateMemeRequest{
		AccountId: accountId,
		RawImage:  imageData,
	})
	if err != nil {
		t.Fatalf("CreateMeme failed: %v", err)
	}
	if respCreate.GetResult().GetId() == "" {
		t.Fatal("expected non-empty ID in CreateMeme response")
	}

	if respCreate.GetStatus() != v1.CreateMemeStatus_STATUS_NEW {
		t.Fatal("expected CreateMeme status STATUS_NEW")
	}

	respSearch, err := searchClient.SearchMeme(context.Background(), &v1.SearchMemeRequest{
		AccountId: accountId,
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
	if respCreate.Result.Id != respSearch.Results[0].Id {
		t.Fatal("expected the result to be the same as the search result")
	}
	if respSearch.Results[0].OcrResult != "ok guys this is my face reveal..." {
		t.Fatal("expected the result to be 'ok guys this is my face reveal'")
	}
}

func TestSearchMeme_Empty(t *testing.T) {
	accountId := genAccountId()
	resp, err := searchClient.SearchMeme(context.Background(), &v1.SearchMemeRequest{
		AccountId: accountId,
		Query:     "test",
	})
	if err != nil {
		t.Fatalf("SearchMeme failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response from SearchMeme")
	}
}
