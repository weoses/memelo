package app

import (
	"context"
	"errors"

	"go.uber.org/fx"
)

// Starter is implemented by components that launch background goroutines.
type Starter interface {
	Start(ctx context.Context) error
}

// Stopper is implemented by components that need graceful shutdown.
type Stopper interface {
	Stop() error
}

// registerLifecycle wires all collected Starter and Stopper implementations
// into the fx lifecycle. Add new components to the "starters" / "stoppers"
// fx groups in module.go rather than writing per-component lifecycle hooks.
func registerLifecycle(lc fx.Lifecycle, starters []Starter, stoppers []Stopper) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			for _, s := range starters {
				if err := s.Start(ctx); err != nil {
					return err
				}
			}
			return nil
		},
		OnStop: func(_ context.Context) error {
			var errs []error
			for _, s := range stoppers {
				errs = append(errs, s.Stop())
			}
			return errors.Join(errs...)
		},
	})
}
