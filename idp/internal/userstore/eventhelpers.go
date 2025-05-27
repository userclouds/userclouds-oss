package userstore

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/events"
	"userclouds.com/idp/paths"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/multitenant"
)

// createEventTypesForAccessor creates the custom event types to track an individual accessor's execution
func (h handler) createEventTypesForAccessor(ctx context.Context, id uuid.UUID, v int) {
	if h.logServerClient == nil {
		return
	}

	ts := multitenant.MustGetTenantState(ctx)

	e := events.GetEventsForAccessor(id, v)

	if _, err := h.logServerClient.CreateEventTypesForTenant(ctx, "idp", uuid.Nil, ts.ID, &e); err != nil {
		uclog.Errorf(ctx, "failed to create event types for accessor %v: %v", id, err)
	}
}

// createEventTypesForMutator creates the custom event types to track an individual mutator's execution
func (h handler) createEventTypesForMutator(ctx context.Context, id uuid.UUID, v int) {
	if h.logServerClient == nil {
		return
	}

	ts := multitenant.MustGetTenantState(ctx)

	e := events.GetEventsForMutator(id, v)

	if _, err := h.logServerClient.CreateEventTypesForTenant(ctx, "idp", uuid.Nil, ts.ID, &e); err != nil {
		uclog.Errorf(ctx, "failed to create event types for mutator %v: %v", id, err)
	}
}

// deleteEventTypesForAccessor delete custom events for accessor being removed
func (h handler) deleteEventTypesForAccessor(ctx context.Context, id uuid.UUID, v int) error {
	if h.logServerClient == nil {
		return nil
	}

	ts := multitenant.MustGetTenantState(ctx)

	if err := h.logServerClient.DeleteEventTypeForReferenceURLForTenant(ctx, uuid.Nil, paths.GetReferenceURLForAccessor(id, v), ts.ID); err != nil {
		uclog.Errorf(ctx, "failed to delete event types for accessor %v: %v", id, err)
	}
	return nil
}

// deleteEventTypesForMutator delete custom events for mutator being removed
func (h handler) deleteEventTypesForMutator(ctx context.Context, id uuid.UUID, v int) error {
	if h.logServerClient == nil {
		return nil
	}

	ts := multitenant.MustGetTenantState(ctx)

	if err := h.logServerClient.DeleteEventTypeForReferenceURLForTenant(ctx, uuid.Nil, paths.GetReferenceURLForMutator(id, v), ts.ID); err != nil {
		uclog.Errorf(ctx, "failed to delete event types for mutator %v: %v", id, err)
	}
	return nil
}

func logAccessorCall(ctx context.Context, id uuid.UUID, version int) {
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryCall, events.AccessorPrefix, "", version))
}

func logAccessorConfigError(ctx context.Context, id uuid.UUID, version int) {
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryInputError, events.AccessorPrefix, events.SubCategoryConfig, version))
}

func logAccessorNotFoundError(ctx context.Context, id uuid.UUID, version int) {
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryInputError, events.AccessorPrefix, events.SubCategoryNotFound, version))
}

func logAccessorTransformerError(ctx context.Context, id uuid.UUID, version int) {
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryInputError, events.AccessorPrefix, events.SubCategoryTransformError, version))
}

func logAccessorSuccess(ctx context.Context, id uuid.UUID, version int) {
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryResultSuccess, events.AccessorPrefix, "", version))
}

func logMutatorCall(ctx context.Context, id uuid.UUID, version int) {
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryCall, events.MutatorPrefix, "", version))
}

func logMutatorConfigError(ctx context.Context, id uuid.UUID, version int) {
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryInputError, events.MutatorPrefix, events.SubCategoryConfig, version))
}

func logMutatorNotFoundError(ctx context.Context, id uuid.UUID, version int) {
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryInputError, events.MutatorPrefix, events.SubCategoryNotFound, version))
}

func logMutatorSuccess(ctx context.Context, id uuid.UUID, version int) {
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryResultSuccess, events.MutatorPrefix, "", version))
}

func logExecutionDuration(ctx context.Context, eventName string, id uuid.UUID, version int, start time.Time) {
	duration := time.Since(start)

	ms := int(duration.Milliseconds())

	// Don't log duration event if the execution took 0ms
	if ms > 0 {
		ev := uclog.LogEvent{
			LogLevel: uclog.LogLevelNonMessage,
			Name:     eventName,
			Count:    ms,
		}
		uclog.Log(ctx, ev)

		// TODO: remove this once we have dashboards etc plumbed through
		uclog.Debugf(ctx, "%s (%v, %v) took %dms", eventName, id, version, duration.Milliseconds())
	}
}

func logAccessorDuration(ctx context.Context, id uuid.UUID, version int, start time.Time) {
	logExecutionDuration(ctx, events.GetEventName(id, uclog.EventCategoryDuration, events.AccessorPrefix, "", version), id, version, start)
}

func logMutatorDuration(ctx context.Context, id uuid.UUID, version int, start time.Time) {
	logExecutionDuration(ctx, events.GetEventName(id, uclog.EventCategoryDuration, events.MutatorPrefix, "", version), id, version, start)
}
