package e2e_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/weoses/memelo/gen/proto/v1"
)

func uniqueTagName() string {
	return fmt.Sprintf("e2e-tag-%d", time.Now().UnixNano())
}

func TestCreateTag(t *testing.T) {
	resp, err := tagsClient.CreateTag(context.Background(), &v1.CreateTagRequest{
		AccountId:   testAccountId,
		Tag:         uniqueTagName(),
		Description: "e2e test tag",
	})
	if err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}
	if resp.GetResult().GetId() == "" {
		t.Fatal("expected non-empty ID in CreateTag response")
	}
}

func TestListTags(t *testing.T) {
	resp, err := tagsClient.ListTags(context.Background(), &v1.ListTagRequest{
		AccountId: testAccountId,
	})
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}
	if resp.GetResult() == nil {
		t.Fatal("expected non-nil result slice from ListTags")
	}
}

func TestDeleteTag(t *testing.T) {
	name := uniqueTagName()

	created, err := tagsClient.CreateTag(context.Background(), &v1.CreateTagRequest{
		AccountId:   testAccountId,
		Tag:         name,
		Description: "e2e delete test",
	})
	if err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}
	id := created.GetResult().GetId()

	_, err = tagsClient.DeleteTag(context.Background(), &v1.DeleteTagRequest{
		AccountId: testAccountId,
		Id:        id,
	})
	if err != nil {
		t.Fatalf("DeleteTag failed: %v", err)
	}

	list, err := tagsClient.ListTags(context.Background(), &v1.ListTagRequest{
		AccountId: testAccountId,
	})
	if err != nil {
		t.Fatalf("ListTags after delete failed: %v", err)
	}
	for _, tag := range list.GetResult() {
		if tag.GetId() == id {
			t.Fatalf("deleted tag %s still present in ListTags", id)
		}
	}
}

func TestCreateTag_Duplicate(t *testing.T) {
	name := uniqueTagName()
	req := &v1.CreateTagRequest{
		AccountId:   testAccountId,
		Tag:         name,
		Description: "dup test",
	}

	_, err := tagsClient.CreateTag(context.Background(), req)
	if err != nil {
		t.Fatalf("first CreateTag failed: %v", err)
	}

	_, err = tagsClient.CreateTag(context.Background(), req)
	if err == nil {
		t.Fatal("expected error on duplicate CreateTag, got nil")
	}

	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("expected connect.Error, got: %T: %v", err, err)
	}
	if connectErr.Code() != connect.CodeAlreadyExists {
		t.Fatalf("expected CodeAlreadyExists, got: %v", connectErr.Code())
	}
}
