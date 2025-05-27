package uclog

import (
	"context"

	"github.com/gofrs/uuid"
)

type contextKey int

const (
	ctxHandlerData contextKey = 1 // name for the handler mapped to the request
)

// handlerData maintains the current client side status of security check for a request
type handlerData struct {
	Name      string
	ErrorName string
	TenantID  uuid.UUID
}

// InitHandlerData initializes the uclog middleware data for the request
func InitHandlerData(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxHandlerData, &handlerData{Name: "", ErrorName: ""})
}

// GetHandlerName returns the name of the handler that processed the request
func GetHandlerName(ctx context.Context) string {
	val := ctx.Value(ctxHandlerData)

	hData, ok := val.(*handlerData)
	if !ok {
		return ""
	}
	return hData.Name
}

// SetHandlerName stores the handler name in the context
func SetHandlerName(ctx context.Context, name string) context.Context {
	val := ctx.Value(ctxHandlerData)
	hData, ok := val.(*handlerData)
	if !ok {
		return context.WithValue(ctx, ctxHandlerData, &handlerData{Name: name, ErrorName: ""})
	}
	hData.Name = name
	return ctx
}

// GetHandlerErrorName returns the name of the error that the handler hit if any
func GetHandlerErrorName(ctx context.Context) string {
	val := ctx.Value(ctxHandlerData)
	hData, ok := val.(*handlerData)
	if !ok {
		return ""
	}
	return hData.ErrorName
}

// SetHandlerErrorName stores the error name in the context
func SetHandlerErrorName(ctx context.Context, name string) context.Context {
	val := ctx.Value(ctxHandlerData)
	hData, ok := val.(*handlerData)
	if !ok {
		return context.WithValue(ctx, ctxHandlerData, &handlerData{Name: "", ErrorName: name})
	}
	hData.ErrorName = name
	return ctx
}

// GetTenantID returns the tenant id for the handler
func GetTenantID(ctx context.Context) uuid.UUID {
	val := ctx.Value(ctxHandlerData)
	hData, ok := val.(*handlerData)
	if !ok {
		return uuid.Nil
	}
	return hData.TenantID
}

// SetTenantID sets tenant ID for the handler
func SetTenantID(ctx context.Context, tenantID uuid.UUID) context.Context {
	val := ctx.Value(ctxHandlerData)

	hData, ok := val.(*handlerData)
	if !ok {
		return context.WithValue(ctx, ctxHandlerData, &handlerData{TenantID: tenantID})
	}
	hData.TenantID = tenantID
	return ctx
}
