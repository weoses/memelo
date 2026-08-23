package auth

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
)

// NewIDTokenTransport returns an http.RoundTripper that attaches a
// Google-signed ID token, audienced to `audience`, to every outgoing
// request — the same mechanism as GoogleIDTokenInterceptor, for callers
// that speak plain HTTP rather than Connect RPC (e.g. a reverse proxy).
// Like GoogleIDTokenInterceptor it has no server-side verification
// counterpart; Cloud Run's own IAM check enforces the token.
//
// When enabled is false (local/docker-compose, mirroring the
// RequireGoogleIDToken config convention), base is returned unchanged.
func NewIDTokenTransport(ctx context.Context, audience string, enabled bool, base http.RoundTripper) (http.RoundTripper, error) {
	if !enabled {
		return base, nil
	}
	ts, err := idtoken.NewTokenSource(ctx, audience)
	if err != nil {
		return nil, fmt.Errorf("auth.NewIDTokenTransport: %w", err)
	}
	return &oauth2.Transport{Source: ts, Base: base}, nil
}
