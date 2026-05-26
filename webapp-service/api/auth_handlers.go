package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (h *Handlers) Login(c echo.Context) error {
	var body LoginRequest
	if err := c.Bind(&body); err != nil || body.Username == "" || body.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing username or password")
	}

	access, refresh, err := h.auth.Login(c.Request().Context(), body.Username, body.Password)
	if err != nil {
		h.log.Error("login failed", "error", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	return c.JSON(http.StatusOK, LoginResponse{AccessToken: access, RefreshToken: refresh})
}

func (h *Handlers) Logout(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (h *Handlers) Refresh(c echo.Context) error {
	var body RefreshRequest
	if err := c.Bind(&body); err != nil || body.RefreshToken == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing refresh_token")
	}

	access, err := h.auth.RefreshAccessToken(c.Request().Context(), body.RefreshToken)
	if err != nil {
		h.log.Error("refresh failed", "error", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired refresh token")
	}

	return c.JSON(http.StatusOK, RefreshResponse{AccessToken: access})
}
