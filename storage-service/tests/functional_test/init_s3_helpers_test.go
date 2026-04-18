package functional_test

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
)

func clearBucket(ctx context.Context, client *minio.Client, bucket string) error {
	objectsCh := client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Recursive: true})

	var errCh <-chan minio.RemoveObjectError
	removeObjectsCh := make(chan minio.ObjectInfo)
	errCh = client.RemoveObjects(ctx, bucket, removeObjectsCh, minio.RemoveObjectsOptions{})

	go func() {
		defer close(removeObjectsCh)
		for obj := range objectsCh {
			if obj.Err != nil {
				return
			}
			removeObjectsCh <- minio.ObjectInfo{Key: obj.Key}
		}
	}()

	for removeErr := range errCh {
		if removeErr.Err != nil {
			return fmt.Errorf("remove object %s: %w", removeErr.ObjectName, removeErr.Err)
		}
	}
	return nil
}
