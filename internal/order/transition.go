package order

import (
	"context"
	"fmt"

	"github.com/magicvr/allinme.core-api/internal/auth"
)

type Action string

const (
	ActionConfirm  Action = "confirm"
	ActionFulfill  Action = "fulfill"
	ActionShip     Action = "ship"
	ActionComplete Action = "complete"
	ActionCancel   Action = "cancel"
)

type TransitionCommand struct {
	Version int64
}

type TransitionPersistence struct {
	ID             string
	Version        int64
	AllowedSources []Status
	Target         Status
	UpdatedAt      string
}

func TransitionFor(action Action) ([]Status, Status, bool) {
	switch action {
	case ActionConfirm:
		return []Status{StatusDraft}, StatusConfirmed, true
	case ActionFulfill:
		return []Status{StatusConfirmed}, StatusFulfilling, true
	case ActionShip:
		return []Status{StatusFulfilling}, StatusShipped, true
	case ActionComplete:
		return []Status{StatusShipped}, StatusCompleted, true
	case ActionCancel:
		return []Status{StatusDraft, StatusConfirmed, StatusFulfilling}, StatusCancelled, true
	default:
		return nil, "", false
	}
}

func (service *Service) Transition(ctx context.Context, principal auth.Principal, id string, action Action, command TransitionCommand) (Order, error) {
	if !CanWrite(principal) {
		return Order{}, ErrForbidden
	}
	if command.Version <= 0 {
		return Order{}, &ValidationError{Details: []FieldError{{Field: "version", Message: "must be greater than 0"}}}
	}
	sources, target, ok := TransitionFor(action)
	if !ok {
		return Order{}, Internal(fmt.Errorf("unknown order action %q", action))
	}
	result, err := service.repository.TransitionOrder(ctx, TransitionPersistence{
		ID: id, Version: command.Version, AllowedSources: sources, Target: target,
		UpdatedAt: FormatTime(UTCNow(service.clock)),
	})
	if err != nil {
		return Order{}, fmt.Errorf("transition order: %w", err)
	}
	return result, nil
}
