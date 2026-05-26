package app

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
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
		fx.Provide(service.NewAuthService),
		fx.Provide(api.NewHandlers),
		fx.Provide(func(c *conf.Config) (net.Listener, error) {
			return net.Listen("tcp", c.Server.ListenAddress)
		}),
		fx.Invoke(startup),
	)
}

func startup(lc fx.Lifecycle, ln net.Listener, h *api.Handlers, authSvc service.AuthService, dist embed.FS, cfg *conf.Config) {
	distFS, err := fs.Sub(dist, "frontend/dist")
	if err != nil {
		panic(err)
	}

	baseUrl := ""
	if cfg.Frontend != nil {
		baseUrl = cfg.Frontend.BaseUrl
	}
	cfgData, _ := json.Marshal(map[string]string{"baseUrl": baseUrl})
	injection := fmt.Sprintf(`<script>window.__MEMELO_CONFIG__=%s;</script>`, cfgData)
	rawHTML, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		panic(err)
	}
	indexHTML := []byte(strings.Replace(string(rawHTML), "</head>", injection+"</head>", 1))

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.POST("/api/auth/login", h.Login)
	e.POST("/api/auth/refresh", h.Refresh)
	e.GET("/api/health", h.Health)

	jwtMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing or invalid authorization header")
			}
			userID, err := authSvc.ExtractUserID(authHeader[7:])
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}
			c.Set("user_id", userID)
			return next(c)
		}
	}

	e.POST("/api/auth/logout", h.Logout, jwtMiddleware)

	memes := e.Group("/api/memes", jwtMiddleware)
	memes.GET("", h.SearchMemes)
	memes.POST("/get-upload-url", h.GetUploadUrl)
	memes.POST("/parse-by-token", h.ParseByToken)
	memes.GET("/:id", h.GetMeme)
	memes.PATCH("/:id", h.UpdateMeme)
	memes.DELETE("/:id", h.DeleteMeme)
	memes.POST("/:id/recompute", h.RecomputeMeme)
	memes.POST("/:id/update-media/:field", h.UpdateMemeMedia)

	e.GET("/*", func(c echo.Context) error {
		return c.HTMLBlob(http.StatusOK, indexHTML)
	})

	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:       "/",
		Index:      "",
		HTML5:      false,
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
