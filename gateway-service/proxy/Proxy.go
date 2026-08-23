// Package proxy builds the gateway's http.ServeMux: a thin reverse proxy in
// front of telegram-service (/webhook) and webapp-service (everything else,
// gated by Basic Auth), attaching a Google ID token to every proxied
// request so the (now IAM-protected) backends accept the call.
package proxy

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/weoses/memelo/common/auth"
	"github.com/weoses/memelo/common/tracing"
	"github.com/weoses/memelo/gateway-service/conf"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const tracerName = "gateway-service"

// NewMux builds the gateway's full route table.
func NewMux(ctx context.Context, cfg *conf.Config) (*http.ServeMux, error) {
	telegramProxy, err := newBackendProxy(ctx, "telegram-service", cfg.TelegramService)
	if err != nil {
		return nil, err
	}
	webappProxy, err := newBackendProxy(ctx, "webapp-service", cfg.WebappService)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.Handle("/webhook", tracingMiddleware(telegramProxy))
	mux.Handle("/", basicAuthMiddleware(cfg.BasicAuth, tracingMiddleware(webappProxy)))
	return mux, nil
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// newBackendProxy builds a reverse proxy to cfg.Uri, with a Google ID token
// (audienced to cfg.Uri, matching the audience convention every other
// inter-service caller in this repo uses) attached to every request via
// auth.NewIDTokenTransport, wrapped in otelhttp.NewTransport so the current
// span's trace context (including the X-App-Traceparent dual header) is
// propagated downstream -- httputil.ReverseProxy does nothing like that on
// its own, unlike the otelconnect interceptors every Connect RPC call in
// this repo already gets for free.
func newBackendProxy(ctx context.Context, name string, target *conf.ProxyTargetConfig) (*httputil.ReverseProxy, error) {
	targetURL, err := url.Parse(target.Uri)
	if err != nil {
		return nil, fmt.Errorf("gateway: parse %s uri: %w", name, err)
	}

	transport, err := auth.NewIDTokenTransport(ctx, target.Uri, target.RequireGoogleIDToken, http.DefaultTransport)
	if err != nil {
		return nil, fmt.Errorf("gateway: %s id token transport: %w", name, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = otelhttp.NewTransport(transport)
	return proxy, nil
}

// tracingMiddleware starts the gateway's own entry span for every proxied
// request -- the request is handled synchronously end to end within the
// handler, so unlike telegram-service's async webhook dispatch there's no
// need to build on a detached base context here, r.Context() is fine.
func tracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.StartHTTP(r.Context(), tracerName, r.Method+" "+r.URL.Path, r.Header)
		defer span.End()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func basicAuthMiddleware(cfg *conf.BasicAuthConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		slog.Info("DEBUG basicAuth", "ok", ok, "u", u, "p", p, "cfgU", cfg.Username, "cfgP", cfg.Password, "authHeader", r.Header.Get("Authorization"))
		if !ok ||
			subtle.ConstantTimeCompare([]byte(u), []byte(cfg.Username)) != 1 ||
			subtle.ConstantTimeCompare([]byte(p), []byte(cfg.Password)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="webapp"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
