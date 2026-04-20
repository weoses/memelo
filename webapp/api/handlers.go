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
	upload    service.UploadService
	accountId string
	log       *slog.Logger
}

func NewHandlers(proxy service.StorageProxy, upload service.UploadService, cfg *conf.Config) *Handlers {
	return &Handlers{
		proxy:     proxy,
		upload:    upload,
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
	return c.JSON(http.StatusOK, searchToResponse(result))
}

func (h *Handlers) GetUploadUrl(c echo.Context) error {
	mime := c.QueryParam("mime")
	if mime == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing mime param")
	}
	lengthStr := c.QueryParam("length")
	if lengthStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing length param")
	}
	size, err := strconv.ParseInt(lengthStr, 10, 64)
	if err != nil || size <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid length param")
	}

	result, err := h.upload.GetUploadUrl(c.Request().Context(), mime, size)
	if err != nil {
		h.log.Error("get upload url failed", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate upload url")
	}
	return c.JSON(http.StatusOK, uploadUrlToResponse(result))
}

func (h *Handlers) ParseByToken(c echo.Context) error {
	var body struct {
		Token string `json:"token"`
	}
	if err := c.Bind(&body); err != nil || body.Token == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing token")
	}

	result, err := h.upload.ParseByToken(c.Request().Context(), h.accountId, body.Token)
	if err != nil {
		h.log.Error("parse by token failed", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid or expired token")
	}
	return c.JSON(http.StatusCreated, memeToResponse(*result))
}
