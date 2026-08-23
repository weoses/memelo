package helper

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
)

type ErrLogger interface {
	Error(msg string, args ...any)
}

func QuietClose(c io.Closer, logger ErrLogger) {
	if c == nil {
		return
	}
	if err := c.Close(); err != nil {
		logger.Error("failed to close", "error", err)
	}
}

func QuietCloseAll[T io.Closer](c []T, logger ErrLogger) {
	if c == nil {
		return
	}
	for _, c := range c {
		QuietClose(c, logger)
	}
}

// CtxCloser is Close(ctx) error's counterpart to io.Closer, for closers that
// need to propagate a real request/trace context (e.g. temp.Data, backed by
// an S3 delete call) rather than fabricating their own.
type CtxCloser interface {
	Close(ctx context.Context) error
}

func QuietCloseCtx(ctx context.Context, c CtxCloser, logger ErrLogger) {
	if c == nil {
		return
	}
	if err := c.Close(ctx); err != nil {
		logger.Error("failed to close", "error", err)
	}
}

func QuietCloseAllCtx[T CtxCloser](ctx context.Context, c []T, logger ErrLogger) {
	if c == nil {
		return
	}
	for _, item := range c {
		QuietCloseCtx(ctx, item, logger)
	}
}

func Addr[T any](v T) *T { return &v }

func DefaultString(item *string) string {
	if item == nil {
		return "<nil>"
	}
	return *item
}

func TransformSlice[F any, T any](from []F, to []T, transformer func(F) T) []T {
	for i, f := range from {
		to[i] = transformer(f)
	}
	return to
}

func TransformSliceErr[F any, T any](from []F, to []T, transformer func(F) (T, error)) ([]T, error) {
	var err error
	for i, f := range from {
		to[i], err = transformer(f)
		if err != nil {
			return nil, err
		}
	}
	return to, nil
}

func CalcHash(base64Image string) string {
	hasher := md5.New()
	hasher.Write([]byte(base64Image))
	byteHash := hasher.Sum(nil)
	return hex.EncodeToString(byteHash)
}
