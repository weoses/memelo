package service

import "context"

type StorageConnector interface {
	ProcessSearchQuery(ctx context.Context, query string)
}
