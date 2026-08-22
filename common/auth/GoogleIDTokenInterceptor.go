package auth

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
)

// GoogleIDTokenInterceptor attaches a Google-signed ID token, audienced to
// the callee's own URL, to every outgoing unary and streaming-client
// request. It has no server-side verification counterpart: Cloud Run's
// platform IAM check (roles/run.invoker) enforces the token before the
// request reaches the container when unauthenticated invocations are
// disabled, so callees never need to verify it themselves.
type GoogleIDTokenInterceptor struct {
	ts oauth2.TokenSource
}

var _ connect.Interceptor = (*GoogleIDTokenInterceptor)(nil)

// NewGoogleIDTokenInterceptor mints tokens audienced to audience (the
// callee's exact base URL, e.g. its Cloud Run URI) using Application
// Default Credentials — the Cloud Run metadata server in prod.
func NewGoogleIDTokenInterceptor(ctx context.Context, audience string) (*GoogleIDTokenInterceptor, error) {
	ts, err := idtoken.NewTokenSource(ctx, audience)
	if err != nil {
		return nil, fmt.Errorf("NewGoogleIDTokenInterceptor: %w", err)
	}
	return &GoogleIDTokenInterceptor{ts: ts}, nil
}

func (i *GoogleIDTokenInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		tok, err := i.ts.Token()
		if err != nil {
			return nil, fmt.Errorf("google id token: %w", err)
		}
		req.Header().Set("Authorization", "Bearer "+tok.AccessToken)
		return next(ctx, req)
	}
}

func (i *GoogleIDTokenInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		if tok, err := i.ts.Token(); err == nil {
			conn.RequestHeader().Set("Authorization", "Bearer "+tok.AccessToken)
		}
		return conn
	}
}

// WrapStreamingHandler is a no-op: this interceptor is client-side only.
func (i *GoogleIDTokenInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// ClientInterceptorOptions returns connect.ClientOptions carrying a
// GoogleIDTokenInterceptor audienced to `audience` when enabled is true, or
// none when false — the default for local/docker-compose, where inter
// -service calls stay unauthenticated.
func ClientInterceptorOptions(ctx context.Context, audience string, enabled bool) ([]connect.ClientOption, error) {
	if !enabled {
		return nil, nil
	}
	interceptor, err := NewGoogleIDTokenInterceptor(ctx, audience)
	if err != nil {
		return nil, err
	}
	return []connect.ClientOption{connect.WithInterceptors(interceptor)}, nil
}
