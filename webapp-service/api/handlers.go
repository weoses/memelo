package api

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/labstack/echo/v4"
	"github.com/weoses/memelo/webapp/conf"
	"github.com/weoses/memelo/webapp/service"
)

var typeTagRe = regexp.MustCompile(`(?i)\s*-type:(image|video)\s*`)

type Handlers struct {
	proxy     service.StorageProxy
	upload    service.UploadService
	auth      service.AuthService
	accountId string
	log       *slog.Logger
}

func NewHandlers(proxy service.StorageProxy, upload service.UploadService, auth service.AuthService, cfg *conf.Config) *Handlers {
	return &Handlers{
		proxy:  proxy,
		upload: upload,
		auth:   auth,
		log:    slog.With("component", "api_handlers"),
	}
}

// resolveAccountId returns the user-id from the JWT context, falling back to
// the configured account ID while null-uid tokens are in use.
func (h *Handlers) resolveAccountId(c echo.Context) (string, error) {
	if uid, ok := c.Get("user_id").(string); ok && uid != "" {
		return uid, nil
	}
	return "", errors.New("no user_id")
}

func (h *Handlers) Health(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

func (h *Handlers) SearchMemes(c echo.Context) error {
	accountId, err := h.resolveAccountId(c)
	if err != nil {
		return fmt.Errorf("could not resolve account id: %w", err)
	}
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

	var metadataType *string
	if match := typeTagRe.FindStringSubmatch(query); match != nil {
		t := strings.ToLower(match[1])
		metadataType = &t
		query = strings.TrimSpace(typeTagRe.ReplaceAllString(query, " "))
	}

	result, err := h.proxy.Search(c.Request().Context(), accountId, query, metadataType, pagination, limit)
	if err != nil {
		h.log.Error("search failed", "error", err)
		return echo.NewHTTPError(http.StatusBadGateway, "upstream search failed")
	}
	return c.JSON(http.StatusOK, searchToResponse(result))
}

func (h *Handlers) GetUploadUrl(c echo.Context) error {
	var body GetUploadUrlRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if body.Mime == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing mime param")
	}
	if body.Length <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid length param")
	}

	result, err := h.upload.GetUploadUrl(c.Request().Context(), body.Mime, body.Length)
	if err != nil {
		h.log.Error("get upload url failed", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate upload url")
	}
	return c.JSON(http.StatusOK, uploadUrlToResponse(result))
}

func (h *Handlers) ParseByToken(c echo.Context) error {
	accountId, err := h.resolveAccountId(c)
	if err != nil {
		return fmt.Errorf("could not resolve account id: %w", err)
	}

	var body ParseByTokenRequest
	if err := c.Bind(&body); err != nil || body.Token == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing token")
	}

	result, err := h.upload.ParseByToken(c.Request().Context(), accountId, body.Token)
	if err != nil {
		h.log.Error("parse by token failed", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid or expired token")
	}
	return c.JSON(http.StatusCreated, memeToResponse(*result))
}

func (h *Handlers) GetMeme(c echo.Context) error {
	accountId, err := h.resolveAccountId(c)
	if err != nil {
		return fmt.Errorf("could not resolve account id: %w", err)
	}

	id := c.Param("id")
	result, err := h.proxy.GetMeme(c.Request().Context(), accountId, id)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "meme not found")
		}
		h.log.Error("get meme failed", "error", err)
		return echo.NewHTTPError(http.StatusBadGateway, "upstream error")
	}
	return c.JSON(http.StatusOK, memeToResponse(*result))
}

func (h *Handlers) UpdateMeme(c echo.Context) error {
	accountId, err := h.resolveAccountId(c)
	if err != nil {
		return fmt.Errorf("could not resolve account id: %w", err)
	}

	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing id")
	}

	var body UpdateMemeRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body")
	}

	params := service.UpdateParams{
		Caption:         body.Caption,
		OnScreenText:    body.OnScreenText,
		AudioTranscript: body.AudioTranscript,
		AudioTrack:      body.AudioTrack,
	}
	result, err := h.proxy.UpdateMeme(c.Request().Context(), accountId, id, params)
	if err != nil {
		h.log.Error("update meme failed", "error", err)
		return echo.NewHTTPError(http.StatusBadGateway, "upstream update failed")
	}
	return c.JSON(http.StatusOK, memeToResponse(*result))
}

func (h *Handlers) DeleteMeme(c echo.Context) error {
	accountId, err := h.resolveAccountId(c)
	if err != nil {
		return fmt.Errorf("could not resolve account id: %w", err)
	}

	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing id")
	}
	if err := h.proxy.DeleteMeme(c.Request().Context(), accountId, id); err != nil {
		h.log.Error("delete meme failed", "error", err)
		return echo.NewHTTPError(http.StatusBadGateway, "upstream delete failed")
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handlers) RecomputeMeme(c echo.Context) error {
	accountId, err := h.resolveAccountId(c)
	if err != nil {
		return fmt.Errorf("could not resolve account id: %w", err)
	}
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing id")
	}
	result, err := h.proxy.RecomputeById(c.Request().Context(), accountId, id)
	if err != nil {
		h.log.Error("recompute meme failed", "error", err)
		return echo.NewHTTPError(http.StatusBadGateway, "upstream recompute failed")
	}
	return c.JSON(http.StatusOK, memeToResponse(*result))
}

func (h *Handlers) UpdateMemeMedia(c echo.Context) error {
	accountId, err := h.resolveAccountId(c)
	if err != nil {
		return fmt.Errorf("could not resolve account id: %w", err)
	}
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing id")
	}

	field := c.Param("field")
	if field != "original" && field != "thumbnail" {
		return echo.NewHTTPError(http.StatusBadRequest, "field must be 'original' or 'thumbnail'")
	}

	var body UpdateMemeMediaRequest
	if err := c.Bind(&body); err != nil || body.Token == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing token")
	}

	s3path, _, err := h.upload.DecodeUploadToken(body.Token)
	if err != nil {
		h.log.Error("decode upload token failed", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid or expired token")
	}

	params := service.UpdateParams{}
	if field == "original" {
		params.OriginalS3Path = &s3path
	} else {
		params.ThumbnailS3Path = &s3path
	}

	result, err := h.proxy.UpdateMeme(c.Request().Context(), accountId, id, params)
	if err != nil {
		h.log.Error("update meme media failed", "error", err)
		return echo.NewHTTPError(http.StatusBadGateway, "upstream update failed")
	}
	return c.JSON(http.StatusOK, memeToResponse(*result))
}
