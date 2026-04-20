package app

import (
	"context"
	"embed"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	commonconfig "github.com/weoses/memelo/common/config"
	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/common/storage"
	"github.com/weoses/memelo/webapp/api"
	"github.com/weoses/memelo/webapp/conf"
	"github.com/weoses/memelo/webapp/service"
	"go.uber.org/fx"
)

func Module() fx.Option {
	return fx.Options(
		fx.Provide(service.NewStorageProxy),
		fx.Provide(func(cfg *conf.Config) (*commonconfig.MediaStorageConfig, error) {
			return cfg.TempStorage, nil
		}),
		fx.Provide(storage.NewS3OperationsAdapter),
		fx.Provide(commonservice.NewTmpDataS3Service),
		fx.Provide(service.NewUploadService),
		fx.Provide(api.NewHandlers),
		fx.Provide(func(c *conf.Config) (net.Listener, error) {
			return net.Listen("tcp", c.Server.ListenAddress)
		}),
		fx.Invoke(startup),
	)
}

func startup(lc fx.Lifecycle, ln net.Listener, h *api.Handlers, dist embed.FS) {
	distFS, err := fs.Sub(dist, "frontend/dist")
	if err != nil {
		panic(err)
	}

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/api/memes", h.SearchMemes)
	e.GET("/api/memes/get-upload-url", h.GetUploadUrl)
	e.POST("/api/memes/parse-by-token", h.ParseByToken)
	e.GET("/api/health", h.Health)

	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:       "/",
		Index:      "index.html",
		HTML5:      true,
		Filesystem: http.FS(distFS),
		Skipper: func(c echo.Context) bool {
			return strings.HasPrefix(c.Request().URL.Path, "/api")
		},
	}))

	srv := &http.Server{
		Handler:      e,
		WriteTimeout: 60 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() { _ = srv.Serve(ln) }()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
}
