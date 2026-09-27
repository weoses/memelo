package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"github.com/weoses/memelo/telegram-service/conf"
)

type AuthStatus int

const (
	AuthStatusNotFound AuthStatus = iota
	AuthStatusOk
)

type AuthResult struct {
	Status      AuthStatus
	UserId      string
	AccountId   string
	Permissions []string
}

// HasPermission reports whether an OK result carries the given permission
// code (e.g. "CREATE", "DELETE"). A not-found (anonymous) result never has
// any permission — callers fall back to search-only access for it.
func (r *AuthResult) HasPermission(code string) bool {
	if r.Status != AuthStatusOk {
		return false
	}
	return slices.Contains(r.Permissions, code)
}

type AuthConnector interface {
	AuthorizeTelegram(ctx context.Context, telegramId int64) (*AuthResult, error)
}

type AuthConnectorImpl struct {
	cl  v1connect.IntegrationTelegramServiceClient
	log *slog.Logger
}

func NewAuthConnector(config *conf.Config) (AuthConnector, error) {
	cl := v1connect.NewIntegrationTelegramServiceClient(http.DefaultClient, config.AuthService.Uri)
	return &AuthConnectorImpl{
		cl:  cl,
		log: slog.With("service", "AuthConnectorService"),
	}, nil
}

func (a *AuthConnectorImpl) AuthorizeTelegram(ctx context.Context, telegramId int64) (*AuthResult, error) {
	resp, err := a.cl.Authorize(ctx, &v1.TelegramAuthorizeRequest{TelegramId: telegramId})
	if err != nil {
		return nil, fmt.Errorf("authorize telegram user failed: %w", err)
	}

	return &AuthResult{
		Status:      fromProtoStatus(resp.Result.Status),
		UserId:      resp.Result.UserId,
		AccountId:   resp.Result.AccountId,
		Permissions: resp.Result.Permissions,
	}, nil
}

func fromProtoStatus(s v1.AuthStatus) AuthStatus {
	if s == v1.AuthStatus_AUTH_STATUS_OK {
		return AuthStatusOk
	}
	return AuthStatusNotFound
}
