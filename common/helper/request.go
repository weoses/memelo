package helper

import (
	"fmt"

	"github.com/google/uuid"
)

type IdAware interface {
	GetId() string
}

type AccountIdAware interface {
	GetAccountId() string
}

func AccountIdUuid[T AccountIdAware](request T) (*uuid.UUID, error) {
	if request.GetAccountId() != "" {
		u, err := uuid.Parse(request.GetAccountId())
		if err != nil {
			return nil, fmt.Errorf("account id is not a valid uuid: %w", err)
		}
		return &u, nil
	}
	return nil, nil
}

func IdUuid[T IdAware](request T) (*uuid.UUID, error) {
	if request.GetId() != "" {
		u, err := uuid.Parse(request.GetId())
		if err != nil {
			return nil, fmt.Errorf("id is not a valid uuid: %w", err)
		}
		return &u, nil
	}
	return nil, nil
}
