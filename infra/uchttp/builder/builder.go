package builder

import (
	"fmt"
	"net/http"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
)

type collectionHandler struct {
	authorizer               uchttp.CollectionAuthorizer
	deleteHandler            uchttp.HandlerFuncWithID
	deleteAllHandler         http.HandlerFunc
	getAllHandler            http.HandlerFunc
	getOneHandler            uchttp.HandlerFuncWithID
	nestedCollectionHandlers map[string]nestedCollectionHandler
	nestedMethodHandlers     map[string]nestedMethodHandler
	postHandler              http.HandlerFunc
	putHandler               uchttp.HandlerFuncWithID
}

// Validate implements the Validatable interface
func (h *collectionHandler) Validate() error {
	if h.authorizer == nil {
		return ucerr.New(`no authorizer specified for collectionHandler`)
	}

	if h.deleteHandler == nil &&
		h.deleteAllHandler == nil &&
		h.getAllHandler == nil &&
		h.getOneHandler == nil &&
		len(h.nestedCollectionHandlers) == 0 &&
		len(h.nestedMethodHandlers) == 0 &&
		h.postHandler == nil &&
		h.putHandler == nil {
		return ucerr.New(`no handler methods specified for collectionHandler`)
	}

	uniquePaths := map[string]bool{}

	for path, nch := range h.nestedCollectionHandlers {
		uniquePaths[path] = true

		if err := nch.Validate(); err != nil {
			return ucerr.Errorf(`nested collection handler for second level path "%s" is invalid: %v`, path, err)
		}
	}

	for path, nmh := range h.nestedMethodHandlers {
		if uniquePaths[path] {
			return ucerr.Errorf(`second level path "%s" is not unique`, path)
		}
		uniquePaths[path] = true

		if err := nmh.Validate(); err != nil {
			return ucerr.Errorf(`nested method handler for second level path "%s" is invalid: %v`, path, err)
		}
	}

	return nil
}

type methodHandler struct {
	deleteHandler http.HandlerFunc
	getHandler    http.HandlerFunc
	postHandler   http.HandlerFunc
	putHandler    http.HandlerFunc
}

// Validate implements the Validatable interface
func (h *methodHandler) Validate() error {
	if h.deleteHandler == nil &&
		h.getHandler == nil &&
		h.postHandler == nil &&
		h.putHandler == nil {
		return ucerr.New(`no handler methods specified for methodHandler`)
	}

	return nil
}

type nestedCollectionHandler struct {
	authorizer       uchttp.NestedCollectionAuthorizer
	deleteHandler    uchttp.HandlerFuncWithNestedID
	deleteAllHandler uchttp.HandlerFuncWithID
	getAllHandler    uchttp.HandlerFuncWithID
	getOneHandler    uchttp.HandlerFuncWithNestedID
	postHandler      uchttp.HandlerFuncWithID
	putHandler       uchttp.HandlerFuncWithNestedID
}

// Validate implements the Validatable interface
func (h *nestedCollectionHandler) Validate() error {
	if h.authorizer == nil {
		return ucerr.New(`no authorizer specified for nestedCollectionHandler`)
	}

	if h.deleteHandler == nil &&
		h.getAllHandler == nil &&
		h.deleteAllHandler == nil &&
		h.getOneHandler == nil &&
		h.postHandler == nil &&
		h.putHandler == nil {
		return ucerr.New(`no handler methods specified for nestedCollectionHandler`)
	}

	return nil
}

type nestedMethodHandler struct {
	deleteHandler uchttp.HandlerFuncWithID
	getHandler    uchttp.HandlerFuncWithID
	postHandler   uchttp.HandlerFuncWithID
	putHandler    uchttp.HandlerFuncWithID
}

// Validate implements the Validatable interface
func (h *nestedMethodHandler) Validate() error {
	if h.deleteHandler == nil &&
		h.getHandler == nil &&
		h.postHandler == nil &&
		h.putHandler == nil {
		return ucerr.New(`no handler methods specified for nestedMethodHandler`)
	}

	return nil
}

// HandlerBuilder is used to configure a top level handler
type HandlerBuilder struct {
	collectionHandlers map[string]collectionHandler
	handlerFuncs       map[string]http.HandlerFunc
	handlers           map[string]http.Handler
	methodHandlers     map[string]methodHandler
	firstLevelPath     string
	secondLevelPath    string
}

// Validate implements the Validatable interface
func (b *HandlerBuilder) Validate() error {
	if len(b.collectionHandlers) == 0 &&
		len(b.handlerFuncs) == 0 &&
		len(b.handlers) == 0 &&
		len(b.methodHandlers) == 0 {
		return ucerr.New(`no handled paths specified`)
	}

	uniquePaths := map[string]bool{}

	for path, ch := range b.collectionHandlers {
		uniquePaths[path] = true

		if err := ch.Validate(); err != nil {
			return ucerr.Errorf(`collection handler for first level path "%s" is invalid: %v`, path, err)
		}
	}

	for path, hf := range b.handlerFuncs {
		if uniquePaths[path] {
			return ucerr.Errorf(`first level path "%s" is not unique`, path)
		}
		uniquePaths[path] = true

		if hf == nil {
			return ucerr.Errorf(`handler func for first level path "%s" is nil`, path)
		}
	}

	for path, h := range b.handlers {
		if uniquePaths[path] {
			return ucerr.Errorf(`first level path "%s" is not unique`, path)
		}
		uniquePaths[path] = true

		if h == nil {
			return ucerr.Errorf(`handler for first level path "%s" is nil`, path)
		}
	}

	for path, mh := range b.methodHandlers {
		if uniquePaths[path] {
			return ucerr.Errorf(`first level path "%s" is not unique`, path)
		}
		uniquePaths[path] = true

		if err := mh.Validate(); err != nil {
			return ucerr.Errorf(`method handler for first level path "%s" is invalid: %v`, path, err)
		}
	}

	return nil
}

// CollectionHandlerBuilder is used to configure a collection handler for a handler
type CollectionHandlerBuilder struct {
	HandlerBuilder
}

// MethodHandlerBuilder is used to configure a method handler for a handler
type MethodHandlerBuilder struct {
	HandlerBuilder
}

// NestedCollectionHandlerBuilder is used to configure a nested collection handler for a collection handler
type NestedCollectionHandlerBuilder struct {
	CollectionHandlerBuilder
}

// NestedMethodHandlerBuilder is used to configure a nested method handler for a collection handler
type NestedMethodHandlerBuilder struct {
	CollectionHandlerBuilder
}

// NewHandlerBuilder create a new handler builder
func NewHandlerBuilder() *HandlerBuilder {
	return &HandlerBuilder{
		collectionHandlers: map[string]collectionHandler{},
		handlerFuncs:       map[string]http.HandlerFunc{},
		handlers:           map[string]http.Handler{},
		methodHandlers:     map[string]methodHandler{},
	}
}

// Build will ensure the handler builder is valid, and if so, will build the serve mux for the configured paths
func (b *HandlerBuilder) Build() *uchttp.ServeMux {
	if err := b.Validate(); err != nil {
		panic(fmt.Sprintf(`attempting to build an invalid handler: "%v"`, err))
	}

	mux := uchttp.NewServeMux()

	for path, h := range b.handlerFuncs {
		mux.HandleFunc(path, h)
	}

	for path, h := range b.handlers {
		mux.Handle(path, h)
	}

	for path, mh := range b.methodHandlers {
		mux.Handle(path,
			uchttp.NewMethodHandler(mh.getHandler,
				mh.postHandler,
				mh.putHandler,
				mh.deleteHandler))
	}

	for firstLevelPath, ch := range b.collectionHandlers {
		var collectionMux *uchttp.ServeMux
		if len(ch.nestedCollectionHandlers) > 0 || len(ch.nestedMethodHandlers) > 0 {
			collectionMux = uchttp.NewServeMux()

			for secondLevelPath, nch := range ch.nestedCollectionHandlers {
				collectionMux.Handle(secondLevelPath,
					uchttp.NewNestedCollectionHandler(nch.getAllHandler,
						nch.getOneHandler,
						nch.postHandler,
						nch.putHandler,
						nch.deleteAllHandler,
						nch.deleteHandler,
						nch.authorizer))
			}

			for secondLevelPath, nmh := range ch.nestedMethodHandlers {
				collectionMux.Handle(secondLevelPath,
					uchttp.NewNestedMethodHandler(nmh.getHandler,
						nmh.postHandler,
						nmh.putHandler,
						nmh.deleteHandler))
			}
		}

		mux.Handle(firstLevelPath,
			uchttp.NewCollectionHandler(ch.getAllHandler,
				ch.getOneHandler,
				ch.postHandler,
				ch.putHandler,
				ch.deleteAllHandler,
				ch.deleteHandler,
				collectionMux,
				ch.authorizer))
	}

	return mux
}

// HandleFunc registers a handler func for the specified path
func (b *HandlerBuilder) HandleFunc(path string, h http.HandlerFunc) *HandlerBuilder {
	if _, found := b.handlerFuncs[path]; found {
		panic(fmt.Sprintf(`attempting to register a handleFunc more than once for path: "%s"`, path))
	}

	b.firstLevelPath = ""
	b.secondLevelPath = ""
	b.handlerFuncs[path] = h
	return b
}

// Handle registers a handler for the specified path
func (b *HandlerBuilder) Handle(path string, h http.Handler) *HandlerBuilder {
	if _, found := b.handlers[path]; found {
		panic(fmt.Sprintf(`attempting to register a handler more than once for path: "%s"`, path))
	}

	b.firstLevelPath = ""
	b.secondLevelPath = ""
	b.handlers[path] = h
	return b
}

// CollectionHandler starts a new collection handler for the specified first level path
func (b *HandlerBuilder) CollectionHandler(path string) *CollectionHandlerBuilder {
	if _, found := b.collectionHandlers[path]; !found {
		b.collectionHandlers[path] = collectionHandler{
			nestedCollectionHandlers: map[string]nestedCollectionHandler{},
			nestedMethodHandlers:     map[string]nestedMethodHandler{},
		}
	}

	b.firstLevelPath = path
	b.secondLevelPath = ""
	return &CollectionHandlerBuilder{*b}
}

// MethodHandler starts a new method handler for the specified first level path
func (b *HandlerBuilder) MethodHandler(path string) *MethodHandlerBuilder {
	if _, found := b.methodHandlers[path]; !found {
		b.methodHandlers[path] = methodHandler{}
	}

	b.firstLevelPath = path
	b.secondLevelPath = ""
	return &MethodHandlerBuilder{*b}
}

// Delete sets the Delete handler for a method handler
func (b *MethodHandlerBuilder) Delete(h http.HandlerFunc) *MethodHandlerBuilder {
	mh := b.methodHandlers[b.firstLevelPath]
	if mh.deleteHandler != nil {
		panic(fmt.Sprintf(`attempting to register a Delete method more than once for path "%s"`, b.firstLevelPath))
	}

	mh.deleteHandler = h
	b.methodHandlers[b.firstLevelPath] = mh
	return b
}

// Get sets the Get handler for a method handler
func (b *MethodHandlerBuilder) Get(h http.HandlerFunc) *MethodHandlerBuilder {
	mh := b.methodHandlers[b.firstLevelPath]
	if mh.getHandler != nil {
		panic(fmt.Sprintf(`attempting to register a Get method more than once for path "%s"`, b.firstLevelPath))
	}

	mh.getHandler = h
	b.methodHandlers[b.firstLevelPath] = mh
	return b
}

// Post sets the Post handler for a method handler
func (b *MethodHandlerBuilder) Post(h http.HandlerFunc) *MethodHandlerBuilder {
	mh := b.methodHandlers[b.firstLevelPath]
	if mh.postHandler != nil {
		panic(fmt.Sprintf(`attempting to register a Post method more than once for path "%s"`, b.firstLevelPath))
	}

	mh.postHandler = h
	b.methodHandlers[b.firstLevelPath] = mh
	return b
}

// Put sets the Put handler for a method handler
func (b *MethodHandlerBuilder) Put(h http.HandlerFunc) *MethodHandlerBuilder {
	mh := b.methodHandlers[b.firstLevelPath]
	if mh.putHandler != nil {
		panic(fmt.Sprintf(`attempting to register a Put method more than once for path "%s"`, b.firstLevelPath))
	}

	mh.putHandler = h
	b.methodHandlers[b.firstLevelPath] = mh
	return b
}

// End ends the configuration of a method handler (useful for codegen)
func (b *MethodHandlerBuilder) End() *MethodHandlerBuilder {
	return b
}

// Delete sets the Delete handler for a collection handler
func (b *CollectionHandlerBuilder) Delete(h uchttp.HandlerFuncWithID) *CollectionHandlerBuilder {
	ch := b.collectionHandlers[b.firstLevelPath]
	if ch.deleteHandler != nil {
		panic(fmt.Sprintf(`attempting to register a collection Delete method more than once for path "%s"`, b.firstLevelPath))
	}

	ch.deleteHandler = h
	b.collectionHandlers[b.firstLevelPath] = ch
	return b
}

// DeleteAll sets the DeleteAll handler for a collection handler
func (b *CollectionHandlerBuilder) DeleteAll(h http.HandlerFunc) *CollectionHandlerBuilder {
	ch := b.collectionHandlers[b.firstLevelPath]
	if ch.deleteAllHandler != nil {
		panic(fmt.Sprintf(`attempting to register a collection DeleteAll method more than once for path "%s"`, b.firstLevelPath))
	}

	ch.deleteAllHandler = h
	b.collectionHandlers[b.firstLevelPath] = ch
	return b
}

// GetAll sets the GetAll handler for a collection handler
func (b *CollectionHandlerBuilder) GetAll(h http.HandlerFunc) *CollectionHandlerBuilder {
	ch := b.collectionHandlers[b.firstLevelPath]
	if ch.getAllHandler != nil {
		panic(fmt.Sprintf(`attempting to register a collection GetAll method more than once for path "%s"`, b.firstLevelPath))
	}

	ch.getAllHandler = h
	b.collectionHandlers[b.firstLevelPath] = ch
	return b
}

// GetOne sets the GetOne handler for a collection handler
func (b *CollectionHandlerBuilder) GetOne(h uchttp.HandlerFuncWithID) *CollectionHandlerBuilder {
	ch := b.collectionHandlers[b.firstLevelPath]
	if ch.getOneHandler != nil {
		panic(fmt.Sprintf(`attempting to register a collection GetOne method more than once for path "%s"`, b.firstLevelPath))
	}

	ch.getOneHandler = h
	b.collectionHandlers[b.firstLevelPath] = ch
	return b
}

// Post sets the Post handler for a collection handler
func (b *CollectionHandlerBuilder) Post(h http.HandlerFunc) *CollectionHandlerBuilder {
	ch := b.collectionHandlers[b.firstLevelPath]
	if ch.postHandler != nil {
		panic(fmt.Sprintf(`attempting to register a collection Post method more than once for path "%s"`, b.firstLevelPath))
	}

	ch.postHandler = h
	b.collectionHandlers[b.firstLevelPath] = ch
	return b
}

// Put sets the Put handler for a collection handler
func (b *CollectionHandlerBuilder) Put(h uchttp.HandlerFuncWithID) *CollectionHandlerBuilder {
	ch := b.collectionHandlers[b.firstLevelPath]
	if ch.putHandler != nil {
		panic(fmt.Sprintf(`attempting to register a collection Put method more than once for path "%s"`, b.firstLevelPath))
	}

	ch.putHandler = h
	b.collectionHandlers[b.firstLevelPath] = ch
	return b
}

// WithAuthorizer sets the CollectionAuthorizer for a collection handler
func (b *CollectionHandlerBuilder) WithAuthorizer(a uchttp.CollectionAuthorizer) *CollectionHandlerBuilder {
	ch := b.collectionHandlers[b.firstLevelPath]
	if ch.authorizer != nil {
		panic(fmt.Sprintf(`attempting to register a collection Authorizer more than once for path "%s"`, b.firstLevelPath))
	}

	ch.authorizer = a
	b.collectionHandlers[b.firstLevelPath] = ch
	return b
}

// NestedCollectionHandler starts a new nested collection handler for the specified second level path
func (b *CollectionHandlerBuilder) NestedCollectionHandler(path string) *NestedCollectionHandlerBuilder {
	if _, found := b.collectionHandlers[b.firstLevelPath].nestedCollectionHandlers[path]; found {
		panic(fmt.Sprintf(`attempting to create a nested collection handler more than once for path "%s %s"`, b.firstLevelPath, path))
	}

	b.secondLevelPath = path
	b.collectionHandlers[b.firstLevelPath].nestedCollectionHandlers[b.secondLevelPath] = nestedCollectionHandler{}
	return &NestedCollectionHandlerBuilder{*b}
}

// NestedMethodHandler starts a new nested method handler for the specified second level path
func (b *CollectionHandlerBuilder) NestedMethodHandler(path string) *NestedMethodHandlerBuilder {
	if _, found := b.collectionHandlers[b.firstLevelPath].nestedMethodHandlers[path]; found {
		panic(fmt.Sprintf(`attempting to create a nested method handler more than once for path "%s %s"`, b.firstLevelPath, path))
	}

	b.secondLevelPath = path
	b.collectionHandlers[b.firstLevelPath].nestedMethodHandlers[b.secondLevelPath] = nestedMethodHandler{}
	return &NestedMethodHandlerBuilder{*b}
}

// Delete sets the delete handler for a nested collection handler
func (b *NestedCollectionHandlerBuilder) Delete(h uchttp.HandlerFuncWithNestedID) *NestedCollectionHandlerBuilder {
	nch := b.collectionHandlers[b.firstLevelPath].nestedCollectionHandlers[b.secondLevelPath]
	if nch.deleteHandler != nil {
		panic(fmt.Sprintf(`attempting to register a nested collection Delete Method more than once for path "%s %s"`, b.firstLevelPath, b.secondLevelPath))
	}

	nch.deleteHandler = h
	b.collectionHandlers[b.firstLevelPath].nestedCollectionHandlers[b.secondLevelPath] = nch
	return b
}

// GetAll sets the GetAll handler for a nested collection handler
func (b *NestedCollectionHandlerBuilder) GetAll(h uchttp.HandlerFuncWithID) *NestedCollectionHandlerBuilder {
	nch := b.collectionHandlers[b.firstLevelPath].nestedCollectionHandlers[b.secondLevelPath]
	if nch.getAllHandler != nil {
		panic(fmt.Sprintf(`attempting to register a nested collection GetAll Method more than once for path "%s %s"`, b.firstLevelPath, b.secondLevelPath))
	}

	nch.getAllHandler = h
	b.collectionHandlers[b.firstLevelPath].nestedCollectionHandlers[b.secondLevelPath] = nch
	return b
}

// DeleteAll sets the DeleteAll handler for a nested collection handler
func (b *NestedCollectionHandlerBuilder) DeleteAll(h uchttp.HandlerFuncWithID) *NestedCollectionHandlerBuilder {
	nch := b.collectionHandlers[b.firstLevelPath].nestedCollectionHandlers[b.secondLevelPath]
	if nch.deleteAllHandler != nil {
		panic(fmt.Sprintf(`attempting to register a nested collection DeleteAll Method more than once for path "%s %s"`, b.firstLevelPath, b.secondLevelPath))
	}

	nch.deleteAllHandler = h
	b.collectionHandlers[b.firstLevelPath].nestedCollectionHandlers[b.secondLevelPath] = nch
	return b
}

// GetOne sets the GetOne handler for a nested collection handler
func (b *NestedCollectionHandlerBuilder) GetOne(h uchttp.HandlerFuncWithNestedID) *NestedCollectionHandlerBuilder {
	nch := b.collectionHandlers[b.firstLevelPath].nestedCollectionHandlers[b.secondLevelPath]
	if nch.getOneHandler != nil {
		panic(fmt.Sprintf(`attempting to register a nested collection GetOne Method more than once for path "%s %s"`, b.firstLevelPath, b.secondLevelPath))
	}

	nch.getOneHandler = h
	b.collectionHandlers[b.firstLevelPath].nestedCollectionHandlers[b.secondLevelPath] = nch
	return b
}

// Post sets the Post handler for a nested collection handler
func (b *NestedCollectionHandlerBuilder) Post(h uchttp.HandlerFuncWithID) *NestedCollectionHandlerBuilder {
	nch := b.collectionHandlers[b.firstLevelPath].nestedCollectionHandlers[b.secondLevelPath]
	if nch.postHandler != nil {
		panic(fmt.Sprintf(`attempting to register a nested collection Post Method more than once for path "%s %s"`, b.firstLevelPath, b.secondLevelPath))
	}

	nch.postHandler = h
	b.collectionHandlers[b.firstLevelPath].nestedCollectionHandlers[b.secondLevelPath] = nch
	return b
}

// Put sets the Put handler for a nested collection handler
func (b *NestedCollectionHandlerBuilder) Put(h uchttp.HandlerFuncWithNestedID) *NestedCollectionHandlerBuilder {
	nch := b.collectionHandlers[b.firstLevelPath].nestedCollectionHandlers[b.secondLevelPath]
	if nch.putHandler != nil {
		panic(fmt.Sprintf(`attempting to register a nested collection Put Method more than once for path "%s %s"`, b.firstLevelPath, b.secondLevelPath))
	}

	nch.putHandler = h
	b.collectionHandlers[b.firstLevelPath].nestedCollectionHandlers[b.secondLevelPath] = nch
	return b
}

// WithAuthorizer sets the NestedCollectionAuthorizer for a nested collection handler
func (b *NestedCollectionHandlerBuilder) WithAuthorizer(a uchttp.NestedCollectionAuthorizer) *NestedCollectionHandlerBuilder {
	nch := b.collectionHandlers[b.firstLevelPath].nestedCollectionHandlers[b.secondLevelPath]
	if nch.authorizer != nil {
		panic(fmt.Sprintf(`attempting to register a nested collection Authorizer more than once for path "%s %s"`, b.firstLevelPath, b.secondLevelPath))
	}

	nch.authorizer = a
	b.collectionHandlers[b.firstLevelPath].nestedCollectionHandlers[b.secondLevelPath] = nch
	return b
}

// Delete sets the Delete handler for a nested method handler
func (b *NestedMethodHandlerBuilder) Delete(h uchttp.HandlerFuncWithID) *NestedMethodHandlerBuilder {
	nmh := b.collectionHandlers[b.firstLevelPath].nestedMethodHandlers[b.secondLevelPath]
	if nmh.deleteHandler != nil {
		panic(fmt.Sprintf(`attempting to register a nested Delete method more than once for path "%s %s"`, b.firstLevelPath, b.secondLevelPath))
	}

	nmh.deleteHandler = h
	b.collectionHandlers[b.firstLevelPath].nestedMethodHandlers[b.secondLevelPath] = nmh
	return b
}

// Get sets the Get handler for a nested method handler
func (b *NestedMethodHandlerBuilder) Get(h uchttp.HandlerFuncWithID) *NestedMethodHandlerBuilder {
	nmh := b.collectionHandlers[b.firstLevelPath].nestedMethodHandlers[b.secondLevelPath]
	if nmh.getHandler != nil {
		panic(fmt.Sprintf(`attempting to register a nested Get method more than once for path "%s %s"`, b.firstLevelPath, b.secondLevelPath))
	}

	nmh.getHandler = h
	b.collectionHandlers[b.firstLevelPath].nestedMethodHandlers[b.secondLevelPath] = nmh
	return b
}

// Post sets the Post handler for a nested method handler
func (b *NestedMethodHandlerBuilder) Post(h uchttp.HandlerFuncWithID) *NestedMethodHandlerBuilder {
	nmh := b.collectionHandlers[b.firstLevelPath].nestedMethodHandlers[b.secondLevelPath]
	if nmh.postHandler != nil {
		panic(fmt.Sprintf(`attempting to register a nested Post method more than once for path "%s %s"`, b.firstLevelPath, b.secondLevelPath))
	}

	nmh.postHandler = h
	b.collectionHandlers[b.firstLevelPath].nestedMethodHandlers[b.secondLevelPath] = nmh
	return b
}

// Put sets the Put handler for a nested method handler
func (b *NestedMethodHandlerBuilder) Put(h uchttp.HandlerFuncWithID) *NestedMethodHandlerBuilder {
	nmh := b.collectionHandlers[b.firstLevelPath].nestedMethodHandlers[b.secondLevelPath]
	if nmh.putHandler != nil {
		panic(fmt.Sprintf(`attempting to register a nested Put method more than once for path "%s %s"`, b.firstLevelPath, b.secondLevelPath))
	}

	nmh.putHandler = h
	b.collectionHandlers[b.firstLevelPath].nestedMethodHandlers[b.secondLevelPath] = nmh
	return b
}
