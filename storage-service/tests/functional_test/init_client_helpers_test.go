package functional_test

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"golang.org/x/net/http2"
)

// h2cClient creates an HTTP client that speaks HTTP/2 cleartext (h2c).
func h2cClient() *http.Client {
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}
}

type TestClient struct {
	search    v1connect.SearchServiceClient
	tags      v1connect.TagsServiceClient
	recompute v1connect.RecomputeServiceClient
}

func NewTestClient(addr string) *TestClient {
	c := h2cClient()
	return &TestClient{
		search:    v1connect.NewSearchServiceClient(c, addr),
		tags:      v1connect.NewTagsServiceClient(c, addr),
		recompute: v1connect.NewRecomputeServiceClient(c, addr),
	}
}

func (c *TestClient) CreateMemeFromBytes(ctx context.Context, accountID string, imageData []byte) (*v1.CreateMemeResponse, error) {
	return c.search.CreateMeme(ctx, &v1.CreateMemeRequest{
		AccountId: accountID,
		Image:     &v1.MediaDataDto{Data: imageData},
	})
}

func (c *TestClient) CreateVideoFromBytes(ctx context.Context, accountID string, videoData []byte) (*v1.CreateMemeResponse, error) {
	return c.search.CreateMeme(ctx, &v1.CreateMemeRequest{
		AccountId: accountID,
		Video:     &v1.MediaDataDto{Data: videoData},
	})
}

func (c *TestClient) SearchMeme(ctx context.Context, accountID, query string, pageSize int32) (*v1.SearchMemeResponse, error) {
	return c.SearchMemePage(ctx, accountID, query, pageSize, nil)
}

func (c *TestClient) SearchMemePage(ctx context.Context, accountID, query string, pageSize int32, cursor *v1.PipelinePagination) (*v1.SearchMemeResponse, error) {
	return c.search.SearchMeme(ctx, &v1.SearchMemeRequest{
		AccountId: accountID,
		Query:     query,
		PageSize:  pageSize,
		AfterId:   cursor,
	})
}

func (c *TestClient) SearchMemeWithType(ctx context.Context, accountID, query string, pageSize int32, mediaType *string) (*v1.SearchMemeResponse, error) {
	return c.search.SearchMeme(ctx, &v1.SearchMemeRequest{
		AccountId: accountID,
		Query:     query,
		PageSize:  pageSize,
		Type:      mediaType,
	})
}

func (c *TestClient) DeleteAll(ctx context.Context, accountID string) error {
	_, err := c.search.DeleteAll(ctx, &v1.DeleteAllRequest{AccountId: accountID})
	return err
}

func (c *TestClient) DeleteMeme(ctx context.Context, accountID, memeID string) error {
	_, err := c.search.DeleteMeme(ctx, &v1.DeleteMemeRequest{AccountId: accountID, Id: memeID})
	return err
}

func (c *TestClient) StartRecompute(ctx context.Context, req *v1.RecomputeRequest) (*v1.RecomputeJob, error) {
	return c.recompute.StartRecomputeJob(ctx, req)
}

func (c *TestClient) GetRecomputeStatus(ctx context.Context, jobID string) (*v1.RecomputeJobStatus, error) {
	return c.recompute.GetRecomputeJobStatus(ctx, &v1.RecomputeJob{JobId: jobID})
}

func (c *TestClient) UpdateMeme(ctx context.Context, req *v1.UpdateMemeRequest) (*v1.UpdateMemeResponse, error) {
	return c.search.UpdateMeme(ctx, req)
}

func (c *TestClient) GetRandomMeme(ctx context.Context, accountID string, mediaType string) (*v1.GetRandomMemeResponse, error) {
	req := &v1.GetRandomMemeRequest{AccountId: accountID}
	if mediaType != "" {
		req.Type = &mediaType
	}
	return c.search.GetRandomMeme(ctx, req)
}

func (c *TestClient) Tags() v1connect.TagsServiceClient           { return c.tags }
func (c *TestClient) Recompute() v1connect.RecomputeServiceClient { return c.recompute }
