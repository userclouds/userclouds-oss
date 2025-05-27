package auditlog

import (
	"context"

	"userclouds.com/infra/uclog"
	"userclouds.com/internal/security"
)

// EventOption defines a way to pass optional configuration parameters.
type EventOption interface {
	apply(*EventConfig)
}

type optFunc func(*EventConfig)

func (o optFunc) apply(po *EventConfig) {
	o(po)
}

// Async indicates that the call should be async
func Async() EventOption {
	return optFunc(func(po *EventConfig) {
		po.async = true
	})
}

// EventConfig describes optional parameters for configuring logging for a tool
type EventConfig struct {
	async bool
}

func postOrLogError(ctx context.Context, s *Storage, entry *Entry) {
	if err := s.SaveEntry(ctx, entry); err != nil {
		uclog.IncrementEvent(ctx, "FailedToInsertIntoAuditLog")
		uclog.Errorf(ctx, "failed to post %v audit log entry: %v", entry.Type, err)
		return
	}
	uclog.Debugf(ctx, "Wrote %v: %v into audit log. sync.", entry.Type, entry.ID)
}

// Post tries adds an entry to the audit log for a given type and payload. if there is a failure, it will not propagate
func Post(ctx context.Context, entry *Entry, opts ...EventOption) {
	// Get the optional parameters if any
	ec := &EventConfig{
		async: false,
	}
	for _, v := range opts {
		v.apply(ec)
	}

	// Grab the IP for the request from the security context and add it to the payload
	if sc := security.GetSecurityStatus(ctx); sc != nil && entry.Payload != nil {
		entry.Payload["IPs"] = sc.IPs
	}

	s := mustGetAuditLogStorage(ctx)

	if !ec.async {
		postOrLogError(ctx, s, entry)
		return
	}

	// Using go's threadpool, if this proves to be a performance bottleneck we will a single dedicate worker thread
	// but for now it seems like better exchange of complexity
	go func(ctx context.Context, e *Entry) {
		postOrLogError(ctx, s, e)
	}(context.Background(), entry)
}

// PostMultipleAsync tries adds an array of entries to the audit log for a given type and payload. if there is a failure, it will not propagate
func PostMultipleAsync(ctx context.Context, entries []Entry) {
	if len(entries) == 0 {
		return
	}

	// Grab the IP for the request from the security context and add it to the payload
	if sc := security.GetSecurityStatus(ctx); sc != nil {
		for i := range entries {
			if entries[i].Payload != nil {
				entries[i].Payload["IPs"] = sc.IPs
			}
		}
	}

	s := mustGetAuditLogStorage(ctx)

	// Using go's threadpool, if this proves to be a performance bottleneck we will a single dedicate worker thread
	// but for now it seems like better exchange of complexity
	go func(ctx context.Context, entries []Entry) {
		for _, e := range entries {
			postOrLogError(ctx, s, &e)
		}
	}(context.Background(), entries)
}
