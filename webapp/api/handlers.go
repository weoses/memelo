package api

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/weoses/memelo/webapp/conf"
	"github.com/weoses/memelo/webapp/service"
)

type Handlers struct {
	proxy     service.StorageProxy
	accountId string
	log       *slog.Logger
}

func NewHandlers(proxy service.StorageProxy, cfg *conf.Config) *Handlers {
	return &Handlers{
		proxy:     proxy,
		accountId: cfg.Account.Id,
		log:       slog.With("component", "api_handlers"),
	}
}

func (h *Handlers) Health(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

func (h *Handlers) SearchMemes(c echo.Context) error {
	query := c.QueryParam("q")
	limitStr := c.QueryParam("limit")
	limit := int32(20)
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 100 {
			limit = int32(v)
		}
	}

	var pagination *service.Pagination
	if searcher := c.QueryParam("after_searcher"); searcher != "" {
		pagination = &service.Pagination{
			Searcher:     searcher,
			SortingAfter: c.QueryParams()["after_sorting"],
		}
	}

	result, err := h.proxy.Search(c.Request().Context(), h.accountId, query, pagination, limit)
	if err != nil {
		h.log.Error("search failed", "error", err)
		return echo.NewHTTPError(http.StatusBadGateway, "upstream search failed")
	}
	return c.JSON(http.StatusOK, result)
}

func (h *Handlers) UploadMeme(c echo.Context) error {
	if err := c.Request().ParseMultipartForm(50 << 20); err != nil {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "file too large (max 50MB)")
	}

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "missing file field")
	}
	defer file.Close()

	mime := header.Header.Get("Content-Type")
	if mime == "" {
		mime = "application/octet-stream"
	}

	result, err := h.proxy.Upload(c.Request().Context(), h.accountId, header.Filename, file, mime)
	if err != nil {
		h.log.Error("upload failed", "error", err)
		return echo.NewHTTPError(http.StatusBadGateway, "upstream upload failed")
	}
	return c.JSON(http.StatusCreated, result)
}
