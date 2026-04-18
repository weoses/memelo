package functional_test

import (
	"bytes"
	"context"
	"fmt"

	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
)

func clearElastic(ctx context.Context, esClient *elasticsearch8.Client, indices ...string) error {
	body := []byte(`{"query":{"match_all":{}}}`)
	for _, index := range indices {
		res, err := esClient.DeleteByQuery(
			[]string{index},
			bytes.NewReader(body),
			esClient.DeleteByQuery.WithContext(ctx),
			esClient.DeleteByQuery.WithRefresh(true),
		)
		if err != nil {
			return fmt.Errorf("delete_by_query %s: %w", index, err)
		}
		_ = res.Body.Close()
		if res.IsError() {
			return fmt.Errorf("delete_by_query %s: status %s", index, res.Status())
		}
	}
	return nil
}
