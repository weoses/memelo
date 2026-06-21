package service

import (
	"context"
	"errors"

	"github.com/weoses/memelo/telegram-service/conf"
)

const (
	PermissionCreate    = "create"
	PermissionDelete    = "delete"
	PermissionRecompute = "recompute"
	PermissionSearch    = "search"
)

var ErrForbidden = errors.New("forbidden")

type PermissionService interface {
	IsAllowed(userId int64, permission string) bool
}

// InvokeWithPermission checks the named permission for userId before calling fn.
// Returns ErrForbidden if the user is not allowed.
func InvokeWithPermission[T any](
	_ context.Context,
	svc PermissionService,
	userId int64,
	permission string,
	fn func() (T, error),
) (T, error) {
	if !svc.IsAllowed(userId, permission) {
		var zero T
		return zero, ErrForbidden
	}
	return fn()
}

type PermissionServiceImpl struct {
	permissions map[string]map[int64]struct{}
}

func (p *PermissionServiceImpl) IsAllowed(userId int64, permission string) bool {
	allowed, ok := p.permissions[permission]
	if !ok || len(allowed) == 0 {
		return true
	}
	_, ok = allowed[userId]
	return ok
}

func NewPermissionService(cfg *conf.Config) PermissionService {
	perms := make(map[string]map[int64]struct{})

	type entry struct {
		name string
		cfg  *conf.PermissionEntryConfig
	}

	if cfg.Permissions != nil {
		entries := []entry{
			{PermissionCreate, cfg.Permissions.Create},
			{PermissionDelete, cfg.Permissions.Delete},
			{PermissionRecompute, cfg.Permissions.Recompute},
			{PermissionSearch, cfg.Permissions.Search},
		}
		for _, e := range entries {
			if e.cfg == nil {
				continue
			}
			set := make(map[int64]struct{}, len(e.cfg.AllowedUserIds))
			for _, id := range e.cfg.AllowedUserIds {
				set[id] = struct{}{}
			}
			perms[e.name] = set
		}
	}

	return &PermissionServiceImpl{permissions: perms}
}
