package tokenizer

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/events"
	"userclouds.com/idp/paths"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/multitenant"
)

// createEventTypesForTransformer creates the custom event types to track an individual policy
func (h handler) createEventTypesForTransformer(ctx context.Context, id uuid.UUID, v int) {
	if h.logServerClient == nil {
		return
	}

	ts := multitenant.MustGetTenantState(ctx)

	e := events.GetEventsForTransformer(id, v)

	if _, err := h.logServerClient.CreateEventTypesForTenant(ctx, "tokenizer", uuid.Nil, ts.ID, &e); err != nil {
		uclog.Errorf(ctx, "Failed to create event types for transformer with ID %s: %v", id, err)
	}
}

func (h handler) createEventTypesForAccessPolicy(ctx context.Context, id uuid.UUID, v int) {
	if h.logServerClient == nil {
		return
	}

	ts := multitenant.MustGetTenantState(ctx)

	e := events.GetEventsForAccessPolicy(id, v)

	if _, err := h.logServerClient.CreateEventTypesForTenant(ctx, "tokenizer", uuid.Nil, ts.ID, &e); err != nil {
		uclog.Errorf(ctx, "Failed to create event types for access policy with ID %s: %v", id, err)
	}
}

func (h handler) createEventTypesForAccessPolicyTemplate(ctx context.Context, id uuid.UUID, v int) {
	if h.logServerClient == nil {
		return
	}

	ts := multitenant.MustGetTenantState(ctx)

	e := events.GetEventsForAccessPolicyTemplate(id, v)

	if _, err := h.logServerClient.CreateEventTypesForTenant(ctx, "tokenizer", uuid.Nil, ts.ID, &e); err != nil {
		uclog.Errorf(ctx, "Failed to create event types for access policy template with ID %s: %v", id, err)
	}
}

// deleteEventsForTransformer delete custom events for transformer being removed
func (h handler) deleteEventsForTransformer(ctx context.Context, id uuid.UUID, v int) error {
	if h.logServerClient == nil {
		return nil
	}

	ts := multitenant.MustGetTenantState(ctx)

	if err := h.logServerClient.DeleteEventTypeForReferenceURLForTenant(ctx, uuid.Nil, paths.GetReferenceURLForTransformer(id, v), ts.ID); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// deleteEventsForAccessPolicy delete custom events for access policy being removed
func (h handler) deleteEventsForAccessPolicy(ctx context.Context, id uuid.UUID, v int) error {
	if h.logServerClient == nil {
		return nil
	}

	ts := multitenant.MustGetTenantState(ctx)

	if err := h.logServerClient.DeleteEventTypeForReferenceURLForTenant(ctx, uuid.Nil, paths.GetReferenceURLForAccessPolicy(id, v), ts.ID); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

func logTransformerCall(ctx context.Context, id uuid.UUID, v int) {
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryCall, events.TransformerPrefix, "", v))
}

func logTransformerError(ctx context.Context, id uuid.UUID, v int) {
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryInputError, events.TransformerPrefix, "", v))
}

func logTransformerConflict(ctx context.Context, id uuid.UUID, v int) {
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryInputError, events.TransformerPrefix, events.SubCategoryConflict, v))
}

func logAPCall(ctx context.Context, id uuid.UUID, v int) {
	// Don't log event for anonymous policies
	if id.IsNil() {
		return
	}

	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryCall, events.APPrefix, "", v))
}

func logAPError(ctx context.Context, id uuid.UUID, v int) {
	// Don't log event for anonymous policies
	if id.IsNil() {
		return
	}
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryInputError, events.APPrefix, "", v))
}

func logAPSuccess(ctx context.Context, id uuid.UUID, v int) {
	// Don't log event for anonymous policies
	if id.IsNil() {
		return
	}
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryResultSuccess, events.APPrefix, "", v))
}
func logAPFailure(ctx context.Context, id uuid.UUID, v int) {
	// Don't log event for anonymous policies
	if id.IsNil() {
		return
	}
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryResultFailure, events.APPrefix, "", v))
}

func logAPTemplateCall(ctx context.Context, id uuid.UUID, v int) {
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryCall, events.APPrefix, "", v))
}

func logAPTemplateError(ctx context.Context, id uuid.UUID, v int) {
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryInputError, events.APPrefix, "", v))
}

func logAPTemplateSuccess(ctx context.Context, id uuid.UUID, v int) {
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryResultSuccess, events.APPrefix, "", v))
}
func logAPTemplateFailure(ctx context.Context, id uuid.UUID, v int) {
	uclog.IncrementEvent(ctx, events.GetEventName(id, uclog.EventCategoryResultFailure, events.APPrefix, "", v))
}

func logPolicyDuration(ctx context.Context, eventName string, id uuid.UUID, version int, start time.Time) {
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

func logTransformerDuration(ctx context.Context, id uuid.UUID, version int, start time.Time) {
	logPolicyDuration(ctx, events.GetEventName(id, uclog.EventCategoryDuration, events.TransformerPrefix, "", version), id, version, start)
}

func logAPDuration(ctx context.Context, id uuid.UUID, version int, start time.Time) {
	// Don't log event for anonymous policies
	if id.IsNil() {
		return
	}
	logPolicyDuration(ctx, events.GetEventName(id, uclog.EventCategoryDuration, events.APPrefix, "", version), id, version, start)
}

func logAPTemplateDuration(ctx context.Context, id uuid.UUID, version int, start time.Time) {
	logPolicyDuration(ctx, events.GetEventName(id, uclog.EventCategoryDuration, events.APPrefix, "", version), id, version, start)
}

func logAPResult(ctx context.Context, id uuid.UUID, version int, rv bool) {
	// Don't log event for anonymous policies
	if id.IsNil() {
		return
	}

	if rv {
		logAPSuccess(ctx, id, version)
	} else {
		logAPFailure(ctx, id, version)
	}
}

func logAPTemplateResult(ctx context.Context, id uuid.UUID, version int, rv bool) {
	if rv {
		logAPTemplateSuccess(ctx, id, version)
	} else {
		logAPTemplateFailure(ctx, id, version)
	}
}
