package uchttp

import (
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
)

// ErrUnauthorized is returned by default whenever a request is unauthorized.
var ErrUnauthorized = ucerr.New("unauthorized")

// AuthorizeFunc is used to authorize access to generic APIs without IDs.
// Implementations should return an error if a request is forbidden, or nil otherwise.
type AuthorizeFunc func(r *http.Request) error

// AuthorizeWithIDFunc is used to authorize access to generic APIs with IDs.
// Implementations should return an error if a request is forbidden, or nil otherwise.
type AuthorizeWithIDFunc func(r *http.Request, id uuid.UUID) error

// AuthorizeWithNestedIDFunc is used to authorize access to generic APIs with parent & child IDs.
// Implementations should return an error if a request is forbidden, or nil otherwise.
type AuthorizeWithNestedIDFunc func(r *http.Request, parentID, nestedID uuid.UUID) error

// CollectionAuthorizer provides an interface to control authorization for all methods
// of a CollectionHandler-based REST API.
// NOTE: a nested handler may in turn have its own Authorizer; the Nested
// method will be called first on the parent handler before any nested
// handler authorizers are invoked.
type CollectionAuthorizer interface {
	GetAll(r *http.Request) error
	GetOne(r *http.Request, id uuid.UUID) error
	Post(r *http.Request) error
	Put(r *http.Request, id uuid.UUID) error
	Delete(r *http.Request, id uuid.UUID) error
	DeleteAll(r *http.Request) error
	Nested(r *http.Request, id uuid.UUID) error
}

// NestedCollectionAuthorizer provides an interface to control authorization for NestedCollectionHandlers.
// NOTE: the `Nested` method on `CollectionAuthorizer` will be checked first before any of these methods.
type NestedCollectionAuthorizer interface {
	GetAll(r *http.Request, parentID uuid.UUID) error
	GetOne(r *http.Request, parentID, nestedID uuid.UUID) error
	Post(r *http.Request, parentID uuid.UUID) error
	Put(r *http.Request, parentID, nestedID uuid.UUID) error
	DeleteAll(r *http.Request, parentID uuid.UUID) error
	Delete(r *http.Request, parentID, nestedID uuid.UUID) error
}

// NewAllowAllAuthorizer returns a CollectionAuthorizer that allows all requests by default.
func NewAllowAllAuthorizer() CollectionAuthorizer {
	allowFunc := func(_ *http.Request) error { return nil }
	allowWithIDFunc := func(_ *http.Request, _ uuid.UUID) error { return nil }
	return &MethodAuthorizer{
		GetAllF:    allowFunc,
		GetOneF:    allowWithIDFunc,
		PostF:      allowFunc,
		PutF:       allowWithIDFunc,
		DeleteF:    allowWithIDFunc,
		DeleteAllF: allowFunc,
		NestedF:    allowWithIDFunc,
	}
}

// NewNestedAllowAllAuthorizer returns a NestedCollectionAuthorizer that allows all requests by default.
func NewNestedAllowAllAuthorizer() NestedCollectionAuthorizer {
	allowWithIDFunc := func(_ *http.Request, _ uuid.UUID) error { return nil }
	allowWithNestedIDFunc := func(_ *http.Request, _, _ uuid.UUID) error { return nil }

	return &NestedMethodAuthorizer{
		GetAllF:    allowWithIDFunc,
		GetOneF:    allowWithNestedIDFunc,
		PostF:      allowWithIDFunc,
		PutF:       allowWithNestedIDFunc,
		DeleteF:    allowWithNestedIDFunc,
		DeleteAllF: allowWithIDFunc,
	}
}

// MethodAuthorizer implements CollectionAuthorizer and allows for separate
// authorization functions for each HTTP method.
type MethodAuthorizer struct {
	GetAllF    AuthorizeFunc
	GetOneF    AuthorizeWithIDFunc
	PostF      AuthorizeFunc
	PutF       AuthorizeWithIDFunc
	DeleteF    AuthorizeWithIDFunc
	DeleteAllF AuthorizeFunc
	NestedF    AuthorizeWithIDFunc
}

// GetAll calls the underlying function to authorize a request.
func (m *MethodAuthorizer) GetAll(r *http.Request) error {
	if m.GetAllF != nil {
		return ucerr.Wrap(m.GetAllF(r))
	}
	return ucerr.Wrap(ErrUnauthorized)
}

// DeleteAll calls the underlying function to authorize a request.
func (m *MethodAuthorizer) DeleteAll(r *http.Request) error {
	if m.DeleteAllF != nil {
		return ucerr.Wrap(m.DeleteAllF(r))
	}
	return ucerr.Wrap(ErrUnauthorized)
}

// GetOne calls the underlying function to authorize a request.
func (m *MethodAuthorizer) GetOne(r *http.Request, id uuid.UUID) error {
	if m.GetOneF != nil {
		return ucerr.Wrap(m.GetOneF(r, id))
	}
	return ucerr.Wrap(ErrUnauthorized)
}

// Post calls the underlying function to authorize a request.
func (m *MethodAuthorizer) Post(r *http.Request) error {
	if m.PostF != nil {
		return ucerr.Wrap(m.PostF(r))
	}
	return ucerr.Wrap(ErrUnauthorized)
}

// Put calls the underlying function to authorize a request.
func (m *MethodAuthorizer) Put(r *http.Request, id uuid.UUID) error {
	if m.PutF != nil {
		return ucerr.Wrap(m.PutF(r, id))
	}
	return ucerr.Wrap(ErrUnauthorized)
}

// Delete calls the underlying function to authorize a request.
func (m *MethodAuthorizer) Delete(r *http.Request, id uuid.UUID) error {
	if m.DeleteF != nil {
		return ucerr.Wrap(m.DeleteF(r, id))
	}
	return ucerr.Wrap(ErrUnauthorized)
}

// Nested calls the underlying function to authorize a request.
func (m *MethodAuthorizer) Nested(r *http.Request, id uuid.UUID) error {
	if m.NestedF != nil {
		return ucerr.Wrap(m.NestedF(r, id))
	}
	return ucerr.Wrap(ErrUnauthorized)
}

// NestedMethodAuthorizer implements CollectionAuthorizer and allows for separate
// authorization functions for each HTTP method.
type NestedMethodAuthorizer struct {
	GetAllF    AuthorizeWithIDFunc
	GetOneF    AuthorizeWithNestedIDFunc
	PostF      AuthorizeWithIDFunc
	PutF       AuthorizeWithNestedIDFunc
	DeleteF    AuthorizeWithNestedIDFunc
	DeleteAllF AuthorizeWithIDFunc
}

// GetAll calls the underlying function to authorize a request.
func (m *NestedMethodAuthorizer) GetAll(r *http.Request, parentID uuid.UUID) error {
	if m.GetAllF != nil {
		return ucerr.Wrap(m.GetAllF(r, parentID))
	}
	return ucerr.Wrap(ErrUnauthorized)
}

// DeleteAll calls the underlying function to authorize a request.
func (m *NestedMethodAuthorizer) DeleteAll(r *http.Request, parentID uuid.UUID) error {
	if m.DeleteAllF != nil {
		return ucerr.Wrap(m.DeleteAllF(r, parentID))
	}
	return ucerr.Wrap(ErrUnauthorized)
}

// GetOne calls the underlying function to authorize a request.
func (m *NestedMethodAuthorizer) GetOne(r *http.Request, parentID, nestedID uuid.UUID) error {
	if m.GetOneF != nil {
		return ucerr.Wrap(m.GetOneF(r, parentID, nestedID))
	}
	return ucerr.Wrap(ErrUnauthorized)
}

// Post calls the underlying function to authorize a request.
func (m *NestedMethodAuthorizer) Post(r *http.Request, parentID uuid.UUID) error {
	if m.PostF != nil {
		return ucerr.Wrap(m.PostF(r, parentID))
	}
	return ucerr.Wrap(ErrUnauthorized)
}

// Put calls the underlying function to authorize a request.
func (m *NestedMethodAuthorizer) Put(r *http.Request, parentID, nestedID uuid.UUID) error {
	if m.PutF != nil {
		return ucerr.Wrap(m.PutF(r, parentID, nestedID))
	}
	return ucerr.Wrap(ErrUnauthorized)
}

// Delete calls the underlying function to authorize a request.
func (m *NestedMethodAuthorizer) Delete(r *http.Request, parentID, nestedID uuid.UUID) error {
	if m.DeleteF != nil {
		return ucerr.Wrap(m.DeleteF(r, parentID, nestedID))
	}
	return ucerr.Wrap(ErrUnauthorized)
}
