package uchttp

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctrace"
)

var tracer = uctrace.NewTracer("uchttp")

// methodHandler implements `http.Handler` and allows for specifying different
// functions to handle different HTTP verbs in a compact syntax.
// Note: If we want to support `http.Handler` later, we can add other members
// like `PostHandler` and ensure only 1 of `Post` or `PostHandler` is set.
// We use HandlerFunc in 90%+ of places, so avoiding the boilerplate of typecasting
// handler methods to `http.HandlerFunc` seemed simplest for now.
type methodHandler struct {
	Get    http.HandlerFunc
	Post   http.HandlerFunc
	Put    http.HandlerFunc
	Delete http.HandlerFunc
}

// ServeHTTP implements `http.Handler`.
func (h *methodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check the escaped path which is the raw path the user agent sent
	// or a simple/canonical escaping if the raw path was not escaped.
	// This ensures that %2f and other nonsense in the path are left as octets.
	// Most websites do NOT treat escaped slashes as actual path separators,
	// but some Go-hosted sites do? e.g. https://pkg.go.dev/net%2furl will
	// take you to the same page as https://pkg.go.dev/net/url (which is odd).
	if strings.HasSuffix(r.URL.EscapedPath(), "/") {
		Error(ctx, w, ucerr.Errorf("trailing slashes not allowed in path: '%s'", r.URL.EscapedPath()), http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		if h.Get != nil {
			h.Get(w, r)
			return
		}
	case http.MethodPost:
		if h.Post != nil {
			h.Post(w, r)
			return
		}
	case http.MethodPut:
		if h.Put != nil {
			h.Put(w, r)
			return
		}
	case http.MethodDelete:
		if h.Delete != nil {
			h.Delete(w, r)
			return
		}
	}
	Error(ctx, w, ucerr.Errorf("unsupported http method '%s'", r.Method), http.StatusMethodNotAllowed)
}

// HandlerFuncWithID is a strongly typed handler func that requires a
// well-formed UUID to have been parsed from the route.
type HandlerFuncWithID func(w http.ResponseWriter, r *http.Request, id uuid.UUID)

// HandlerFuncWithNestedID is a strongly typed handler func that requires a
// well-formed UUID and nested/sub UUID to have been parsed from the route.
// e.g. /objects/<id>/subhandler/<nestedID>
type HandlerFuncWithNestedID func(w http.ResponseWriter, r *http.Request, id uuid.UUID, nestedID uuid.UUID)

// CollectionHandler implements `http.Handler` and allows for specifying different
// functions to handle different HTTP verbs on a RESTful collection in a compact syntax.
// It is a more opinionated version of MethodHandler, but both are useful.
type CollectionHandler struct {
	// GetAll is used to fetch the whole collection.
	GetAll http.HandlerFunc

	// DeleteAll is used to delete the whole collection or a filtered subset.
	DeleteAll http.HandlerFunc

	// GetOne is used to fetch a single element, only if a UUID is specified
	// in the path.
	GetOne HandlerFuncWithID

	// Post is used to create an element in the collection.
	Post http.HandlerFunc

	// Put is used to upsert an element in the collection, only if a UUID
	// is specified in the path.
	Put HandlerFuncWithID

	// Delete is used to delete an element in the collection, only if a UUID
	// is specified in the path.
	Delete HandlerFuncWithID

	// NestedItemHandler is invoked if an ID is specified and there are
	// remaining path elements, e.g. "/objects/<uuid>/details"
	NestedItemHandler http.Handler

	// Authorizer enforces basic entry-level permissioning on methods.
	Authorizer CollectionAuthorizer
}

// NewCollectionHandler returns a new CollectionsHandler with all the handleFuncs wrappered for logging
func NewCollectionHandler(GetAll http.HandlerFunc, GetOne HandlerFuncWithID, Post http.HandlerFunc, Put HandlerFuncWithID,
	DeleteAll http.HandlerFunc, Delete HandlerFuncWithID, NestedItemHandler http.Handler, Authorizer CollectionAuthorizer) http.Handler {

	GetAllWrapped := GetWrappedHandlerFunc(GetAll)
	GetOneWrapped := GetWrappedHandlerFuncWithID(GetOne)
	PostWrapped := GetWrappedHandlerFunc(Post)
	PutWrapped := GetWrappedHandlerFuncWithID(Put)
	DeleteWrapped := GetWrappedHandlerFuncWithID(Delete)
	DeleteAllWrapped := GetWrappedHandlerFunc(DeleteAll)

	return &CollectionHandler{GetAll: GetAllWrapped, GetOne: GetOneWrapped, Post: PostWrapped, Put: PutWrapped, Delete: DeleteWrapped,
		DeleteAll: DeleteAllWrapped, NestedItemHandler: NestedItemHandler, Authorizer: Authorizer}
}

// NewNestedMethodHandler returns a new NestedMethodHandler with all the handleFuncs wrapped for logging
func NewNestedMethodHandler(Get HandlerFuncWithID, Post HandlerFuncWithID, Put HandlerFuncWithID, Delete HandlerFuncWithID) http.Handler {

	GetWrapped := GetWrappedHandlerFuncWithID(Get)
	PostWrapped := GetWrappedHandlerFuncWithID(Post)
	PutWrapped := GetWrappedHandlerFuncWithID(Put)
	DeleteWrapped := GetWrappedHandlerFuncWithID(Delete)

	return &nestedMethodHandler{Get: GetWrapped, Post: PostWrapped, Put: PutWrapped, Delete: DeleteWrapped}
}

// NewNestedCollectionHandler returns a new NestedCollectionHandler with all the handleFuncs wrapped for logging
func NewNestedCollectionHandler(GetAll HandlerFuncWithID, GetOne HandlerFuncWithNestedID, Post HandlerFuncWithID, Put HandlerFuncWithNestedID,
	DeleteAll HandlerFuncWithID, Delete HandlerFuncWithNestedID, Authorizer NestedCollectionAuthorizer) http.Handler {

	GetAllWrapped := GetWrappedHandlerFuncWithID(GetAll)
	GetOneWrapped := GetWrappedHandlerFuncWithNestedID(GetOne)
	PostWrapped := GetWrappedHandlerFuncWithID(Post)
	PutWrapped := GetWrappedHandlerFuncWithNestedID(Put)
	DeleteWrapped := GetWrappedHandlerFuncWithNestedID(Delete)
	DeleteAllWrapped := GetWrappedHandlerFuncWithID(DeleteAll)

	return &nestedCollectionHandler{GetAll: GetAllWrapped, GetOne: GetOneWrapped, Post: PostWrapped, Put: PutWrapped, Delete: DeleteWrapped,
		DeleteAll: DeleteAllWrapped, Authorizer: Authorizer}
}

// NewMethodHandler returns a new MethodHandler with all the handleFuncs wrapped for logging
func NewMethodHandler(Get http.HandlerFunc, Post http.HandlerFunc, Put http.HandlerFunc, Delete http.HandlerFunc) http.Handler {

	GetWrapped := GetWrappedHandlerFunc(Get)
	PostWrapped := GetWrappedHandlerFunc(Post)
	PutWrapped := GetWrappedHandlerFunc(Put)
	DeleteWrapped := GetWrappedHandlerFunc(Delete)

	return &methodHandler{Get: GetWrapped, Post: PostWrapped, Put: PutWrapped, Delete: DeleteWrapped}
}

// GetWrappedHandlerFunc wraps a handler func with logging infra
func GetWrappedHandlerFunc(handler func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	// Don't wrap unused handlers
	if handler == nil {
		return nil
	}

	handlerName := computeNameFromReflection(runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name())
	handlerName = strings.ReplaceAll(handlerName, "Generated-fm", "-fm")

	// TODO Remove once the events are mapped on the server side
	uclog.AddHandlerForValidation(context.Background(), handlerName)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := traceHandler(r, handlerName)
		defer span.End()
		handler(w, r.WithContext(ctx))
	}
}

// GetWrappedHandlerFuncWithID wraps a handler func with logging infra
func GetWrappedHandlerFuncWithID(handler HandlerFuncWithID) HandlerFuncWithID {
	// Don't wrap unused handlers
	if handler == nil {
		return nil
	}

	handlerName := computeNameFromReflection(runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name())
	handlerName = strings.ReplaceAll(handlerName, "Generated-fm", "-fm")

	// TODO Remove once the events are mapped on the server side
	uclog.AddHandlerForValidation(context.Background(), handlerName)
	return func(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
		ctx, span := traceHandler(r, handlerName)
		defer span.End()
		handler(w, r.WithContext(ctx), id)
	}
}

// GetWrappedHandlerFuncWithNestedID wraps a handler func with logging infra
func GetWrappedHandlerFuncWithNestedID(handler HandlerFuncWithNestedID) HandlerFuncWithNestedID {
	// Don't wrap unused handlers
	if handler == nil {
		return nil
	}

	handlerName := computeNameFromReflection(runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name())
	handlerName = strings.ReplaceAll(handlerName, "Generated-fm", "-fm")

	// TODO Remove once the events are mapped on the server side
	uclog.AddHandlerForValidation(context.Background(), handlerName)
	return func(w http.ResponseWriter, r *http.Request, id uuid.UUID, nestedID uuid.UUID) {
		ctx, span := traceHandler(r, handlerName)
		defer span.End()
		handler(w, r.WithContext(ctx), id, nestedID)
	}
}

func traceHandler(r *http.Request, handlerName string) (context.Context, uctrace.Span) {
	ctx, span := tracer.StartSpan(r.Context(), handlerName, true)
	span.SetStringAttribute(uctrace.AttributeHandlerName, handlerName)
	// Set the handler name in context
	ctx = uclog.SetHandlerName(ctx, handlerName)
	// Increment handler count event
	uclog.IncrementEvent(ctx, handlerName+".Count")
	return ctx, span
}

// ServeHTTP implements `http.Handler`.
func (h *CollectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Fail closed by default
	authorizer := h.Authorizer
	if authorizer == nil {
		Error(ctx, w, ucerr.Wrap(ErrUnauthorized), http.StatusForbidden)
		return
	}

	// Escaped path is the raw path the user agent sent us OR a valid
	// escaped full path. This prevents us from unescaping most octets and treats
	// the path nearly literally.
	rawPath := r.URL.EscapedPath()
	id, prefixToStrip, err := parseRoute(rawPath)
	if err != nil {
		// TODO: allow overriding error handler to account for JSON vs YAML, etc?
		// Can use 'Accept' header to decide.
		// Return 404 because it means we can't recognize the route.
		Error(ctx, w, ucerr.Friendlyf(nil, "unable to parse non-nil ID from path '%s'", r.URL.Path), http.StatusNotFound)
		return
	}

	// You can't (currently) nest collection handlers. If needed we can add support
	// for named IDs and put them in a map, or add a "ctxSubUUID", or an array of IDs.
	val := ctx.Value(ctxUUID)
	if _, ok := val.(uuid.UUID); ok {
		// This is not really a runtime error, but a basic bug in the code.
		// We could panic or just error.
		Error(ctx, w, ucerr.New("can't nest CollectionHandler handlers"), http.StatusInternalServerError)
		return
	}
	r = r.WithContext(context.WithValue(ctx, ctxUUID, id))

	// There is a path remainder after the UUID and slash, trim it and
	// let the nested handler deal with it.
	if len(prefixToStrip) < len(rawPath) {
		if err := authorizer.Nested(r, id); err != nil {
			Error(ctx, w, err, http.StatusForbidden)
			return
		}
		http.StripPrefix(prefixToStrip, h.NestedItemHandler).ServeHTTP(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Handle GETs with an ID vs. no ID.
		if !id.IsNil() {
			if h.GetOne != nil {
				if err := authorizer.GetOne(r, id); err != nil {
					Error(ctx, w, err, http.StatusForbidden)
					return
				}
				h.GetOne(w, r, id)
				return
			}
		} else {
			if h.GetAll != nil {
				if err := authorizer.GetAll(r); err != nil {
					Error(ctx, w, err, http.StatusForbidden)
					return
				}
				h.GetAll(w, r)
				return
			}
		}
	case http.MethodPost:
		if h.Post != nil {
			if err := authorizer.Post(r); err != nil {
				Error(ctx, w, err, http.StatusForbidden)
				return
			}
			h.Post(w, r)
			return
		}
	case http.MethodPut:
		if !id.IsNil() && h.Put != nil {
			if err := authorizer.Put(r, id); err != nil {
				Error(ctx, w, err, http.StatusForbidden)
				return
			}
			h.Put(w, r, id)
			return
		}
	case http.MethodDelete:
		if !id.IsNil() {
			if h.Delete != nil {
				if err := authorizer.Delete(r, id); err != nil {
					Error(ctx, w, err, http.StatusForbidden)
					return
				}
				h.Delete(w, r, id)
				return
			}
		} else {
			if h.DeleteAll != nil {
				if err := authorizer.DeleteAll(r); err != nil {
					Error(ctx, w, err, http.StatusForbidden)
					return
				}
				h.DeleteAll(w, r)
				return
			}
		}
	}

	// TODO: return 404 if method is supported but ID doesn't parse? (for PUT and DELETE)
	Error(ctx, w, ucerr.Errorf("unsupported http method '%s'", r.Method), http.StatusMethodNotAllowed)
}

// nestedMethodHandler is a MethodHandler that can be nested safely inside
// a CollectionHandler with strong typing for the object ID.
type nestedMethodHandler struct {
	Get    HandlerFuncWithID
	Post   HandlerFuncWithID
	Put    HandlerFuncWithID
	Delete HandlerFuncWithID
}

// ServeHTTP implements `http.Handler`.
func (h *nestedMethodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if strings.HasSuffix(r.URL.EscapedPath(), "/") {
		Error(ctx, w, ucerr.Errorf("trailing slashes not allowed in path: '%s'", r.URL.EscapedPath()), http.StatusNotFound)
		return
	}

	id := mustGetUUID(ctx)

	switch r.Method {
	case http.MethodGet:
		if h.Get != nil {
			h.Get(w, r, id)
			return
		}
	case http.MethodPost:
		if h.Post != nil {
			h.Post(w, r, id)
			return
		}
	case http.MethodPut:
		if h.Put != nil {
			h.Put(w, r, id)
			return
		}
	case http.MethodDelete:
		if h.Delete != nil {
			h.Delete(w, r, id)
			return
		}
	}
	Error(ctx, w, ucerr.Errorf("unsupported http method '%s'", r.Method), http.StatusMethodNotAllowed)
}

// nestedCollectionHandler is a version of CollectionHandler that can be
// nested inside of a CollectionHandler.
type nestedCollectionHandler struct {
	// GetAll is used to fetch the whole nested collection.
	GetAll HandlerFuncWithID

	// DeleteAll is used to delete the whole nested collection.
	DeleteAll HandlerFuncWithID

	// GetOne is used to fetch a single element, only if a nested UUID is
	// specified in the path.
	GetOne HandlerFuncWithNestedID

	// Post is used to create an element in the nested collection.
	Post HandlerFuncWithID

	// Put is used to upsert an element in the nested collection, only if
	// a nested UUID is specified in the path.
	Put HandlerFuncWithNestedID

	// Delete is used to delete an element in the collection, only if a
	// nested UUID is specified in the path.
	Delete HandlerFuncWithNestedID

	// Authorizer enforces basic entry-level permissioning on methods.
	Authorizer NestedCollectionAuthorizer
}

// ServeHTTP implements `http.Handler`.
func (h *nestedCollectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Fail closed by default
	authorizer := h.Authorizer
	if authorizer == nil {
		Error(ctx, w, ucerr.Wrap(ErrUnauthorized), http.StatusForbidden)
		return
	}

	// Since this is nested, it must have a parent CollectionHandler with a valid ID.
	parentID := mustGetUUID(ctx)

	rawPath := r.URL.EscapedPath()
	nestedID, prefixToStrip, err := parseRoute(rawPath)
	if err != nil {
		Error(ctx, w, ucerr.Friendlyf(nil, "unable to parse non-nil ID from path '%s'", r.URL.Path), http.StatusNotFound)
		return
	}

	if len(prefixToStrip) < len(rawPath) {
		Error(ctx, w, ucerr.Errorf("unexpected remainder in URL after nested ID: %s", r.URL.Path), http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Handle GETs with an ID vs. no ID.
		if nestedID != uuid.Nil {
			if h.GetOne != nil {
				if err := authorizer.GetOne(r, parentID, nestedID); err != nil {
					Error(ctx, w, err, http.StatusForbidden)
					return
				}
				h.GetOne(w, r, parentID, nestedID)
				return
			}
		} else {
			if h.GetAll != nil {
				if err := authorizer.GetAll(r, parentID); err != nil {
					Error(ctx, w, err, http.StatusForbidden)
					return
				}
				h.GetAll(w, r, parentID)
				return
			}
		}
	case http.MethodPost:
		if h.Post != nil {
			if err := authorizer.Post(r, parentID); err != nil {
				Error(ctx, w, err, http.StatusForbidden)
				return
			}
			h.Post(w, r, parentID)
			return
		}
	case http.MethodPut:
		if nestedID != uuid.Nil && h.Put != nil {
			if err := authorizer.Put(r, parentID, nestedID); err != nil {
				Error(ctx, w, err, http.StatusForbidden)
				return
			}
			h.Put(w, r, parentID, nestedID)
			return
		}
	case http.MethodDelete:
		if nestedID != uuid.Nil {
			if h.Delete != nil {
				if err := authorizer.Delete(r, parentID, nestedID); err != nil {
					Error(ctx, w, err, http.StatusForbidden)
					return
				}
				h.Delete(w, r, parentID, nestedID)
				return
			}
		} else {
			if h.DeleteAll != nil {
				if err := authorizer.DeleteAll(r, parentID); err != nil {
					Error(ctx, w, err, http.StatusForbidden)
					return
				}
				h.DeleteAll(w, r, parentID)
				return
			}
		}
	}

	// TODO: return 404 if method is supported but ID doesn't parse? (for PUT and DELETE)
	Error(ctx, w, ucerr.Errorf("unsupported http method '%s'", r.Method), http.StatusMethodNotAllowed)
}

type uuidKey int

const ctxUUID uuidKey = 1

// parseRoute parses the path and returns a valid parsed UUID, a prefix
// which should be stripped from the path before passing on to nested
// handlers, and an error.
// It expects either:
//  1. "" - empty string. Returns a nil UUID and empty prefix to strip, no error.
//  2. /<uuid>[/remainder] - a valid non-nil UUID with an optional remainder string,
//     in which case it returns the parsed UUID, the prefix to strip (which is
//     the leading slash and UUID), and no error.
//
// Anything else is treated as an error.
func parseRoute(path string) (uuid.UUID, string, error) {
	// Easy case: exact match to root of collection, no ID or subpath.
	if path == "" {
		return uuid.Nil, "", nil
	}

	// Bad case: trailing slash.
	if strings.HasSuffix(path, "/") {
		return uuid.Nil, "", ucerr.Errorf("trailing slashes not allowed in path: '%s'", path)
	}

	// We expect a rooted relative path at this point, e.g. /<uuid>[/remainder]
	if !strings.HasPrefix(path, "/") {
		return uuid.Nil, "", ucerr.Errorf("expected rooted subpath: '%s'", path)
	}

	// Split path into the first path segment and the remainder.
	parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
	if len(parts[0]) > 0 {
		// TODO: support non-UUID string identifiers, and do a 'name lookup'
		// to convert to UUID.
		id, err := uuid.FromString(parts[0])
		if err == nil && id.IsNil() {
			return uuid.Nil, "", ucerr.Errorf("nil UUID found in path '%s'", path)
		}
		return id, fmt.Sprintf("/%s", parts[0]), ucerr.Wrap(err)
	}

	// This means the path started with "//" so there was no ID component.
	return uuid.Nil, "", ucerr.Errorf("no UUID found in path '%s'", path)
}

// mustGetUUID retrieves a UUID parsed out of a request path by CollectionHandler.
// Any handler referenced by CollectionHandler can safely use this.
func mustGetUUID(ctx context.Context) uuid.UUID {
	val := ctx.Value(ctxUUID)
	id, ok := val.(uuid.UUID)
	if !ok {
		panic(ucerr.New("uuid not set in context, did you forget to use CollectionHandler?"))
	}
	return id
}

// ServeMux wraps http.ServeMux to handle trailing slashes in a better way
type ServeMux struct {
	*http.ServeMux
}

// NewServeMux returns a new uchttp.ServeMux wrapping a new http.ServeMux
func NewServeMux() *ServeMux {
	return &ServeMux{http.NewServeMux()}
}

// Handle passes through to the underlying http.ServeMux with better handling
func (s ServeMux) Handle(path string, h http.Handler) {
	basePath := strings.TrimSuffix(path, "/")
	basePathTrailingSlash := fmt.Sprintf("%s/", basePath)

	// http.ServeMux's default behavior is:
	// 1. If pattern has trailing slash: match all URLs with prefix (incl
	//    trailing slash), but if URL matches without the trailing slash,
	//    then 301 redirect to the URL with the slash.
	//    e.g. if pattern is "/a/", then "/a/b" and "/a/" and "/a/b/c/" are
	//    all correctly matched. However, a request to "/a" will 301 redirect
	//    to "/a/", which may cause user agents to change method from
	//    POST/PUT/etc to GET, thus breaking the intent.
	// 2. If pattern has no trailing slash: exact match only (e.g. if pattern
	//    is "/a", only match "/a" and not "/a/" or "/a/1234/...").
	//
	// Other common APIs (e.g. Stripe, Auth0, etc) disallow trailing slashes
	// on API calls, so we will do the same.
	//
	// However, for collections with object IDs and nested handlers to work,
	// we need to route some methods (e.g. GET, POST) to the Path without a trailing
	// slash and some methods to the Path followed by an ID or other content.
	// This requires us to handle both "/a" and "/a/.*".
	// Either way, we want to chomp the prefix off the string and let the handler
	// enforce the desired behavior for trailing slashes.
	if basePath != "" {
		s.ServeMux.Handle(basePath, http.StripPrefix(basePath, h))
	}

	// Intentionally don't strip the slash from the base path so we can
	// differentiate between an exact match to basePath vs. basePathTrailingSlash.
	// The latter should be a 404, as we don't allow trailing slashes.
	s.ServeMux.Handle(basePathTrailingSlash, http.StripPrefix(basePath, h))
}

// HandleFunc wraps sets the handler name in the context prior to calling it. The handler name is
// derived from the name of the function that is passed in
func (s ServeMux) HandleFunc(url string, handler http.HandlerFunc) {
	wrappedHandler := GetWrappedHandlerFunc(handler)
	s.ServeMux.HandleFunc(url, wrappedHandler)
}

// This function takes the full name of handler from reflection and returns either the name of the function for the handler or
// if the handler is a lambda function - the first non-lambda name followed by lambda sequence Blah.funcX.funcX ...
func computeNameFromReflection(fullHandlerName string) string {
	splitNames := strings.Split(fullHandlerName, ".")
	var i int
	for i = len(splitNames) - 1; i >= 0; i-- {
		if !strings.HasPrefix(splitNames[i], "func") {
			break
		}
	}
	// In case we never found a name that doesn't start with "func"
	if i < 0 {
		i = 0
	}
	// Instead of package name we use preset service name. Thee full names have form like userclouds.com/plex/internal/oidc.(*handler).socialLogin-fm
	// or userclouds.com/infra/service.HandleGetDeployed or main.initRoutes.func1, but in all cases the right name
	// to use is "Plex". Otherwise all calls from shared packages will count against same counter. TODO - is that enough
	return uclog.GetServiceName() + "." + strings.Join(splitNames[i:], ".")
}
