package ocr

import (
	"context"

	retry "github.com/avast/retry-go/v4"
	"golang.org/x/time/rate"
)

type ConnectorWrapper struct {
	limiter      *rate.Limiter
	retryAttempt uint
}

func NewConnectorWrapper(rateLimit float64, retryAttempt uint) *ConnectorWrapper {
	burst := int(rateLimit)
	if burst < 1 {
		burst = 1
	}
	return &ConnectorWrapper{
		limiter:      rate.NewLimiter(rate.Limit(rateLimit), burst),
		retryAttempt: retryAttempt,
	}
}

func (w *ConnectorWrapper) Do(ctx context.Context, fn func() error) error {
	return retry.Do(
		func() error {
			if err := w.limiter.Wait(ctx); err != nil {
				return retry.Unrecoverable(err)
			}
			return fn()
		},
		retry.Attempts(w.retryAttempt),
		retry.Context(ctx),
	)
}
