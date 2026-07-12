package order_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/magicvr/allinme.core-api/internal/auth"
	"github.com/magicvr/allinme.core-api/internal/order"
)

func TestTransitionForDefinesFulfillmentStateMachine(t *testing.T) {
	tests := []struct {
		action  order.Action
		sources []order.Status
		target  order.Status
	}{
		{order.ActionConfirm, []order.Status{order.StatusDraft}, order.StatusConfirmed},
		{order.ActionFulfill, []order.Status{order.StatusConfirmed}, order.StatusFulfilling},
		{order.ActionShip, []order.Status{order.StatusFulfilling}, order.StatusShipped},
		{order.ActionComplete, []order.Status{order.StatusShipped}, order.StatusCompleted},
		{order.ActionCancel, []order.Status{order.StatusDraft, order.StatusConfirmed, order.StatusFulfilling}, order.StatusCancelled},
	}
	for _, test := range tests {
		sources, target, ok := order.TransitionFor(test.action)
		if !ok || target != test.target || len(sources) != len(test.sources) {
			t.Fatalf("TransitionFor(%q) = %v %q %v", test.action, sources, target, ok)
		}
		for index := range sources {
			if sources[index] != test.sources[index] {
				t.Fatalf("TransitionFor(%q) sources = %v", test.action, sources)
			}
		}
	}
	if _, _, ok := order.TransitionFor("unknown"); ok {
		t.Fatal("unknown action accepted")
	}
}

func TestTransitionValidatesRoleVersionAndPersistence(t *testing.T) {
	repository := &writeRepository{}
	service, err := order.NewServiceWithDependencies(repository, func() time.Time { return time.Date(2026, 7, 13, 1, 2, 3, 0, time.FixedZone("offset", 8*60*60)) }, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []auth.Role{auth.RoleViewer, auth.RoleApprover} {
		if _, err := service.Transition(context.Background(), auth.Principal{Role: role}, "ord_00000000000000000000000000000001", order.ActionConfirm, order.TransitionCommand{Version: 1}); !errors.Is(err, order.ErrForbidden) {
			t.Fatalf("%s transition error = %v", role, err)
		}
	}
	if _, err := service.Transition(context.Background(), auth.Principal{Role: auth.RoleOperator}, "ord_00000000000000000000000000000001", order.ActionConfirm, order.TransitionCommand{}); err == nil {
		t.Fatal("zero version error = nil")
	}
	result, err := service.Transition(context.Background(), auth.Principal{Role: auth.RoleAdmin}, "ord_00000000000000000000000000000001", order.ActionCancel, order.TransitionCommand{Version: 7})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != order.StatusCancelled || result.Version != 8 || repository.transitioned.Version != 7 || repository.transitioned.Target != order.StatusCancelled || repository.transitioned.UpdatedAt != "2026-07-12T17:02:03Z" {
		t.Fatalf("transition result=%+v persistence=%+v", result, repository.transitioned)
	}
}
