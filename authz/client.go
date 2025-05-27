package authz

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/go-http-utils/headers"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/request"
	"userclouds.com/infra/sdkclient"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

const (
	// DefaultObjTypeTTL specifies how long ObjectTypes remain in the cache by default. If you frequently delete ObjectTypes - you should lower this number
	DefaultObjTypeTTL time.Duration = 10 * time.Minute
	// DefaultEdgeTypeTTL specifies how long EdgeTypes remain in the cache by default. If you frequently delete ObjectTypes - you should lower this number
	DefaultEdgeTypeTTL time.Duration = 10 * time.Minute
	// DefaultObjTTL specifies how long Objects remain in the cache by default. If you frequently delete Objects (such as users) - you should lower this number
	DefaultObjTTL time.Duration = 5 * time.Minute
	// DefaultEdgeTTL specifies how long Edges remain in the cache by default. It is assumed that edges churn frequently so this number is set lower
	DefaultEdgeTTL time.Duration = 30 * time.Second
)

type options struct {
	ifNotExists           bool
	bypassCache           bool
	tenantID              uuid.UUID
	organizationID        uuid.UUID
	cacheProvider         cache.Provider
	paginationOptions     []pagination.Option
	jsonclientOptions     []jsonclient.Option
	bypassAuthHeaderCheck bool // if we're using per-request header forwarding via PassthroughAuthorization, don't check for auth header
	source                *string
}

// Option makes authz.Client extensible
type Option interface {
	apply(*options)
}

type optFunc func(*options)

func (o optFunc) apply(opts *options) {
	o(opts)
}

// IfNotExists returns an Option that will cause the client not to return an error if an identical object to the one being created already exists
func IfNotExists() Option {
	return optFunc(func(opts *options) {
		opts.ifNotExists = true
	})
}

// BypassCache returns an Option that will cause the client to bypass the cache for the request (supported for read operations only)
func BypassCache() Option {
	return optFunc(func(opts *options) {
		opts.bypassCache = true
	})
}

// OrganizationID returns an Option that will cause the client to use the specified organization ID for the request
func OrganizationID(organizationID uuid.UUID) Option {
	return optFunc(func(opts *options) {
		opts.organizationID = organizationID
	})
}

// TenantID returns an Option that can be used to specify the tenant ID for creation of the client
func TenantID(tenantID uuid.UUID) Option {
	return optFunc(func(opts *options) {
		opts.tenantID = tenantID
	})
}

// Source returns an Option that will cause the client to include the specified source in the request
func Source(source string) Option {
	return optFunc(func(opts *options) {
		opts.source = &source
	})
}

// Pagination is a wrapper around pagination.Option
func Pagination(opt ...pagination.Option) Option {
	return optFunc(func(opts *options) {
		opts.paginationOptions = append(opts.paginationOptions, opt...)
	})
}

// JSONClient is a wrapper around jsonclient.Option
func JSONClient(opt ...jsonclient.Option) Option {
	return optFunc(func(opts *options) {
		opts.jsonclientOptions = append(opts.jsonclientOptions, opt...)
	})
}

// CacheProvider returns an Option that will cause the client to use given cache provider (can only be used on call to NewClient)
func CacheProvider(cp cache.Provider) Option {
	return optFunc(func(opts *options) {
		opts.cacheProvider = cp
	})
}

// PassthroughAuthorization returns an Option that will cause the client to use the auth header from the request context
func PassthroughAuthorization() Option {
	return optFunc(func(opts *options) {
		opts.jsonclientOptions = append(opts.jsonclientOptions, jsonclient.PerRequestHeader(func(ctx context.Context) (string, string) {
			return headers.Authorization, request.GetAuthHeader(ctx)
		}))
		opts.bypassAuthHeaderCheck = true
	})
}

// Client is a client for the authz service
type Client struct {
	client  *sdkclient.Client
	options options

	// Object type root cache contains:
	//    ObjTypeID (primary key) -> ObjType and objTypePrefix + TypeName (secondary key) -> ObjType
	//    ObjTypeCollection(global collection key) -> []ObjType (all object types)
	// Edge type root cache contains:
	//    EdgeTypeID (primary key) -> EdgeType and edgeTypePrefix + TypeName (secondary key) -> EdgeType
	//    EdgeTypeCollection(global collection key) -> []EdgeType (all edge types)
	// Object root cache contains:
	//    ObjectID (primary key) -> Object and typeID + Object.Alias (secondary key) -> Object
	//    ObjectCollection(global collection key) -> lock only
	//    ObjectID + Edges (per item collection key) -> []Edges (all outgoing/incoming)
	//    ObjectID1 + Edges + ObjectID2 (per item sub collection key) -> []Edges (all between ObjectID1/ObjectID2)
	//    ObjectID1 + Path + ObjectID2 + Attribute (per item sub collection key) -> []AttributeNode (path between ObjectID1 and ObjectID2 for Attribute)
	//    ObjectID + Dependency (dependency key) -> []CacheKeys (all cache keys that depend on this object)
	// Edge root cache contains:
	//    EdgeID (primary key) -> Edge
	//    SourceObjID + TargetObjID + EdgeTypeID (secondary key) -> Edge
	//    EdgeID + Dependency (dependency key) -> []CacheKeys (all cache keys that depend on this edge)

	cp   cache.Provider
	np   cache.KeyNameProvider
	ttlP cache.TTLProvider
	cm   cache.Manager
}

// NewClient creates a new authz client
// Web API base URL, e.g. "http://localhost:1234".
func NewClient(url string, opts ...Option) (*Client, error) {
	opts = append(opts, BypassCache())
	return NewCustomClient(DefaultObjTypeTTL, DefaultEdgeTypeTTL, DefaultObjTTL, DefaultEdgeTTL, url, opts...)
}

// NewCustomClient creates a new authz client with different cache defaults
// Web API base URL, e.g. "http://localhost:1234".
func NewCustomClient(objTypeTTL time.Duration, edgeTypeTTL time.Duration, objTTL time.Duration, edgeTTL time.Duration,
	url string, opts ...Option) (*Client, error) {
	if url == "" {
		return nil, ucerr.Errorf("url is required")
	}

	var options options
	for _, opt := range opts {
		opt.apply(&options)
	}

	cp := options.cacheProvider

	// If cache provider is not specified use default
	if cp == nil {
		cp = cache.NewInMemoryClientCacheProvider(uuid.Must(uuid.NewV4()).String())
	}

	// TODO need to redo this in a way that makes more sense
	if options.bypassCache {
		objTypeTTL = cache.SkipCacheTTL
		edgeTypeTTL = cache.SkipCacheTTL
		objTTL = cache.SkipCacheTTL
		edgeTTL = cache.SkipCacheTTL
	}

	ttlP := NewCacheTTLProvider(objTypeTTL, edgeTypeTTL, objTTL, edgeTTL, 0)

	var np cache.KeyNameProvider
	if !options.tenantID.IsNil() {
		np = NewCacheNameProviderForTenant(options.tenantID)
	} else {
		np = NewCacheNameProvider(fmt.Sprintf("%s_%s", CachePrefix, url))
	}

	c := &Client{
		client:  sdkclient.New(url, "authz", options.jsonclientOptions...),
		options: options,
		cp:      cp,
		np:      np,
		cm:      cache.NewManager(cp, np, ttlP),
		ttlP:    ttlP,
	}

	if !options.bypassAuthHeaderCheck {
		if err := c.client.ValidateBearerTokenHeader(); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}

	return c, nil
}

// FlushCache clears all contents of the cache
func (c *Client) FlushCache() error {
	return ucerr.Wrap(c.cp.Flush(context.Background(), c.np.GetPrefix(), true))
}

// FlushCacheEdges clears the edge cache only.
func (c *Client) FlushCacheEdges() error {
	return ucerr.Wrap(c.cp.Flush(context.Background(), c.np.GetPrefix(), true))
}

// FlushCacheObjectsAndEdges clears the objects/edges cache only.
func (c *Client) FlushCacheObjectsAndEdges() error {
	return ucerr.Wrap(c.cp.Flush(context.Background(), c.np.GetPrefix(), true))
}

// CreateObjectTypeRequest is the request body for creating an object type
type CreateObjectTypeRequest struct {
	ObjectType ObjectType `json:"object_type" yaml:"object_type"`
}

// CreateObjectType creates a new type of object for the authz system.
func (c *Client) CreateObjectType(ctx context.Context, id uuid.UUID, typeName string, opts ...Option) (*ObjectType, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	input := ObjectType{
		BaseModel: ucdb.NewBase(),
		TypeName:  typeName,
	}

	if !id.IsNil() {
		input.ID = id
	}

	return cache.CreateItemClient[ObjectType](ctx, &c.cm, id, &input, ObjectTypeKeyID, c.cm.N.GetKeyNameWithString(ObjectTypeNameKeyID, typeName), options.ifNotExists, options.bypassCache, nil,
		func(i *ObjectType) (*ObjectType, error) {
			req := CreateObjectTypeRequest{*i}
			var resp ObjectType
			if options.ifNotExists {
				exists, existingID, err := c.client.CreateIfNotExists(ctx, "/authz/objecttypes", req, &resp)
				if err != nil {
					return nil, ucerr.Wrap(err)
				}
				if exists {
					if id.IsNil() || existingID == id {
						resp = req.ObjectType
						resp.ID = existingID
					} else {
						return nil, ucerr.Errorf("object type already exists with different ID: %s", existingID)
					}
				}
			} else {
				if err := c.client.Post(ctx, "/authz/objecttypes", req, &resp); err != nil {
					return nil, ucerr.Wrap(err)
				}
			}
			return &resp, nil
		}, func(in *ObjectType, curr *ObjectType) bool {
			return curr.EqualsIgnoringID(in) && (id.IsNil() || curr.ID == id)
		})
}

// FindObjectTypeID resolves an object type name to an ID.
func (c *Client) FindObjectTypeID(ctx context.Context, typeName string, opts ...Option) (uuid.UUID, error) {
	ctx = request.NewRequestID(ctx)

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	if !options.bypassCache {
		v, _, _, err := cache.GetItemFromCache[ObjectType](ctx, c.cm, c.cm.N.GetKeyNameWithString(ObjectTypeNameKeyID, typeName), false)
		if err != nil {
			return uuid.Nil, ucerr.Wrap(err)
		}
		if v != nil {
			return v.ID, nil
		}
	}

	objTypes, err := c.ListObjectTypes(ctx, opts...)
	if err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}

	// Double check in case the cache was invalidated between the get and the lookup
	for _, objType := range objTypes {
		if objType.TypeName == typeName {
			return objType.ID, nil
		}
	}

	return uuid.Nil, ucerr.Errorf("authz object type '%s' not found", typeName)
}

// GetObjectType returns an object type by ID.
func (c *Client) GetObjectType(ctx context.Context, id uuid.UUID, opts ...Option) (*ObjectType, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	return cache.GetItemClient[ObjectType](ctx, c.cm, id, ObjectTypeKeyID, options.bypassCache, func(id uuid.UUID, conflict cache.Sentinel, resp *ObjectType) error {
		if err := c.client.Get(ctx, fmt.Sprintf("/authz/objecttypes/%v", id), resp); err != nil {
			if jsonclient.IsHTTPNotFound(err) {
				return ucerr.Wrap(ucerr.Combine(err, ErrObjectTypeNotFound))
			}
			return ucerr.Wrap(err)
		}
		return nil
	})
}

// ListObjectTypesResponse is the paginated response from listing object types.
type ListObjectTypesResponse struct {
	Data []ObjectType `json:"data" yaml:"data"`
	pagination.ResponseFields
}

// ListObjectTypes lists all object types in the system
func (c *Client) ListObjectTypes(ctx context.Context, opts ...Option) ([]ObjectType, error) {
	ctx = request.NewRequestID(ctx)

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	s := cache.NoLockSentinel
	var useCache = (!options.bypassCache && len(options.paginationOptions) == 0)
	if useCache {
		var v *[]ObjectType
		var err error
		v, _, s, _, err = cache.GetItemsArrayFromCache[ObjectType](ctx, c.cm, c.cm.N.GetKeyNameStatic(ObjectTypeCollectionKeyID), true)
		if err != nil {
			uclog.Errorf(ctx, "ListObjectTypes failed to get item from cache: %v", err)
		} else if v != nil {
			return *v, nil
		}
	}
	// TODO: we should eventually support pagination arguments to this method, but for now we assume
	// there aren't that many object types and just fetch them all. We should do this by combining
	// the caching behavior here with the pagination behavior of ListObjectTypesPaginated
	pager, err := pagination.ApplyOptions()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	objTypes := make([]ObjectType, 0)

	for {
		query := pager.Query()

		var resp ListObjectTypesResponse

		if err := c.client.Get(ctx, fmt.Sprintf("/authz/objecttypes?%s", query.Encode()), &resp); err != nil {
			return nil, ucerr.Wrap(err)
		}

		objTypes = append(objTypes, resp.Data...)

		if useCache {
			cache.SaveItemsFromCollectionToCache(ctx, c.cm, resp.Data, s)
		}
		if !pager.AdvanceCursor(resp.ResponseFields) {
			break
		}
	}
	if useCache {
		ckey := c.cm.N.GetKeyNameStatic(ObjectTypeCollectionKeyID)
		cache.SaveItemsToCollection(ctx, c.cm, ObjectType{}, objTypes, ckey, ckey, s, true)
	}
	return objTypes, nil
}

// ListObjectTypesPaginated lists objects for console in paginated form
func (c *Client) ListObjectTypesPaginated(ctx context.Context, opts ...Option) (*ListObjectTypesResponse, error) {
	ctx = request.NewRequestID(ctx)

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	var resp ListObjectTypesResponse

	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	query := pager.Query()

	if err := c.client.Get(ctx, fmt.Sprintf("/authz/objecttypes?%s", query.Encode()), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// DeleteObjectType deletes an object type by ID.
func (c *Client) DeleteObjectType(ctx context.Context, objectTypeID uuid.UUID) error {
	ctx = request.NewRequestID(ctx)

	// We don't take a delete lock since we will flush the cache after the delete anyway
	if err := c.client.Delete(ctx, fmt.Sprintf("/authz/objecttypes/%s", objectTypeID), nil); err != nil {
		if jsonclient.IsHTTPNotFound(err) {
			return ucerr.Wrap(ucerr.Combine(err, ErrObjectTypeNotFound))
		}
		return ucerr.Wrap(err)
	}

	// There are so many potential inconsistencies when object type is deleted so flush the whole cache
	return ucerr.Wrap(c.FlushCache())
}

// CreateEdgeTypeRequest is the request body for creating an edge type
type CreateEdgeTypeRequest struct {
	EdgeType EdgeType `json:"edge_type" yaml:"edge_type"`
}

// CreateEdgeType creates a new type of edge for the authz system.
func (c *Client) CreateEdgeType(ctx context.Context, id uuid.UUID, sourceObjectTypeID, targetObjectTypeID uuid.UUID, typeName string, attributes Attributes, opts ...Option) (*EdgeType, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	input := EdgeType{
		BaseModel:          ucdb.NewBase(),
		TypeName:           typeName,
		SourceObjectTypeID: sourceObjectTypeID,
		TargetObjectTypeID: targetObjectTypeID,
		Attributes:         attributes,
		OrganizationID:     options.organizationID,
	}
	if !id.IsNil() {
		input.ID = id
	}

	return cache.CreateItemClient[EdgeType](ctx, &c.cm, id, &input, EdgeTypeKeyID, c.cm.N.GetKeyNameWithString(EdgeTypeNameKeyID, typeName), options.ifNotExists, options.bypassCache, nil,
		func(i *EdgeType) (*EdgeType, error) {
			req := CreateEdgeTypeRequest{*i}
			var resp EdgeType
			if options.ifNotExists {
				exists, existingID, err := c.client.CreateIfNotExists(ctx, "/authz/edgetypes", req, &resp)
				if err != nil {
					return nil, ucerr.Wrap(err)
				}
				if exists {
					if id.IsNil() || existingID == id {
						resp = req.EdgeType
						resp.ID = existingID
					} else {
						return nil, ucerr.Errorf("edge type already exists with different ID: %s", existingID)
					}
				}
			} else {
				if err := c.client.Post(ctx, "/authz/edgetypes", req, &resp); err != nil {
					return nil, ucerr.Wrap(err)
				}
			}
			return &resp, nil
		}, func(in *EdgeType, curr *EdgeType) bool {
			return curr.EqualsIgnoringID(in) && (id.IsNil() || curr.ID == id)
		})
}

// UpdateEdgeTypeRequest is the request struct for updating an edge type
type UpdateEdgeTypeRequest struct {
	TypeName   string     `json:"type_name" yaml:"type_name" validate:"notempty"`
	Attributes Attributes `json:"attributes" yaml:"attributes"`
}

// UpdateEdgeType updates an existing edge type in the authz system.
func (c *Client) UpdateEdgeType(ctx context.Context, id uuid.UUID, sourceObjectTypeID, targetObjectTypeID uuid.UUID, typeName string, attributes Attributes, opts ...Option) (*EdgeType, error) {
	ctx = request.NewRequestID(ctx)

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	req := UpdateEdgeTypeRequest{
		TypeName:   typeName,
		Attributes: attributes,
	}

	eT := EdgeType{
		BaseModel:          ucdb.NewBaseWithID(id),
		TypeName:           typeName,
		SourceObjectTypeID: sourceObjectTypeID,
		TargetObjectTypeID: targetObjectTypeID,
		Attributes:         attributes,
		OrganizationID:     options.organizationID,
	}

	// Check if the edge type already exists in the cache and is the same as the update value being passed in
	if !options.bypassCache {
		var v *EdgeType
		var err error

		v, _, _, err = cache.GetItemFromCache[EdgeType](ctx, c.cm, c.cm.N.GetKeyNameWithID(EdgeTypeKeyID, id), false)
		if err != nil {
			uclog.Errorf(ctx, "UpdateEdgeType failed to get item from cache: %v", err)
		} else if v != nil && v.ID == eT.ID && v.EqualsIgnoringID(&eT) {
			return v, nil
		}
	}

	s, err := cache.TakeItemLock(ctx, cache.Update, c.cm, eT)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	defer cache.ReleaseItemLock(ctx, c.cm, cache.Update, eT, s)

	var resp EdgeType
	if err := c.client.Put(ctx, fmt.Sprintf("/authz/edgetypes/%s", id), req, &resp); err != nil {
		if jsonclient.IsHTTPNotFound(err) {
			return nil, ucerr.Wrap(ucerr.Combine(err, ErrEdgeTypeNotFound))
		}
		return nil, ucerr.Wrap(err)
	}

	cache.SaveItemToCache(ctx, c.cm, resp, s, true, nil)

	// For now flush the cache because we don't track all the paths that need to be invalidated
	if err := c.FlushCache(); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// GetEdgeType gets an edge type (relationship) by its type ID.
func (c *Client) GetEdgeType(ctx context.Context, edgeTypeID uuid.UUID, opts ...Option) (*EdgeType, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	return cache.GetItemClient[EdgeType](ctx, c.cm, edgeTypeID, EdgeTypeKeyID, options.bypassCache, func(id uuid.UUID, conflict cache.Sentinel, resp *EdgeType) error {
		if err := c.client.Get(ctx, fmt.Sprintf("/authz/edgetypes/%s", id), resp); err != nil {
			if jsonclient.IsHTTPNotFound(err) {
				return ucerr.Wrap(ucerr.Combine(err, ErrEdgeTypeNotFound))
			}
			return ucerr.Wrap(err)
		}
		return nil
	})
}

// FindEdgeTypeID resolves an edge type name to an ID.
func (c *Client) FindEdgeTypeID(ctx context.Context, typeName string, opts ...Option) (uuid.UUID, error) {
	ctx = request.NewRequestID(ctx)

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	if !options.bypassCache {
		v, _, _, err := cache.GetItemFromCache[EdgeType](ctx, c.cm, c.cm.N.GetKeyNameWithString(EdgeTypeNameKeyID, typeName), false)
		if err != nil {
			uclog.Errorf(ctx, "FindEdgeTypeID failed to get item from cache: %v", err)
		} else if v != nil {
			return v.ID, nil
		}
	}

	edgeTypes, err := c.ListEdgeTypes(ctx, opts...)
	if err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}

	// Double check if the cache was invalidated on the miss
	for _, edgeType := range edgeTypes {
		if edgeType.TypeName == typeName {
			return edgeType.ID, nil
		}
	}
	return uuid.Nil, ucerr.Errorf("authz edge type '%s' not found", typeName)
}

// ListEdgeTypesResponse is the paginated response from listing edge types.
type ListEdgeTypesResponse struct {
	Data []EdgeType `json:"data" yaml:"data"`
	pagination.ResponseFields
}

// Description implements the Described interface for OpenAPI
func (r ListEdgeTypesResponse) Description() string {
	return "This object contains an array of edge types and pagination information"
}

// ListEdgeTypes lists all available edge types
func (c *Client) ListEdgeTypes(ctx context.Context, opts ...Option) ([]EdgeType, error) {
	ctx = request.NewRequestID(ctx)

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}
	var useCache = (!options.bypassCache && len(options.paginationOptions) == 0)

	s := cache.NoLockSentinel
	if useCache {
		var v *[]EdgeType
		var err error
		v, _, s, _, err = cache.GetItemsArrayFromCache[EdgeType](ctx, c.cm, c.cm.N.GetKeyNameStatic(EdgeTypeCollectionKeyID), true)
		if err != nil {
			uclog.Errorf(ctx, "ListEdgeTypes failed to get item from cache: %v", err)
		} else if v != nil {
			return *v, nil
		}
	}

	// TODO: we should eventually support pagination arguments to this method, but for now we assume
	// there aren't that many edge types and just fetch them all. We should do this by combining
	// the caching behavior here with the pagination behavior of ListEdgeTypesPaginated
	pager, err := pagination.ApplyOptions()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	edgeTypes := make([]EdgeType, 0)

	for {
		query := pager.Query()
		if !options.organizationID.IsNil() {
			query.Add("organization_id", options.organizationID.String())
		}

		var resp ListEdgeTypesResponse
		if err := c.client.Get(ctx, fmt.Sprintf("/authz/edgetypes?%s", query.Encode()), &resp); err != nil {
			return nil, ucerr.Wrap(err)
		}

		edgeTypes = append(edgeTypes, resp.Data...)
		if useCache {
			cache.SaveItemsFromCollectionToCache(ctx, c.cm, resp.Data, s)
		}

		if !pager.AdvanceCursor(resp.ResponseFields) {
			break
		}
	}

	if useCache {
		ckey := c.cm.N.GetKeyNameStatic(EdgeTypeCollectionKeyID)
		cache.SaveItemsToCollection(ctx, c.cm, EdgeType{}, edgeTypes, ckey, ckey, s, true)
	}
	return edgeTypes, nil
}

// ListEdgeTypesPaginated lists edges for console in paginated form
func (c *Client) ListEdgeTypesPaginated(ctx context.Context, opts ...Option) (*ListEdgeTypesResponse, error) {
	ctx = request.NewRequestID(ctx)

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	query := pager.Query()

	if !options.organizationID.IsNil() {
		query.Add("organization_id", options.organizationID.String())
	}

	var resp ListEdgeTypesResponse

	if err := c.client.Get(ctx, fmt.Sprintf("/authz/edgetypes?%s", query.Encode()), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// DeleteEdgeType deletes an edge type by ID.
func (c *Client) DeleteEdgeType(ctx context.Context, edgeTypeID uuid.UUID) error {
	ctx = request.NewRequestID(ctx)

	// We don't take a delete lock since we will flush the cache after the delete anyway
	if err := c.client.Delete(ctx, fmt.Sprintf("/authz/edgetypes/%s", edgeTypeID), nil); err != nil {
		if jsonclient.IsHTTPNotFound(err) {
			return ucerr.Wrap(ucerr.Combine(err, ErrEdgeTypeNotFound))
		}
		return ucerr.Wrap(err)
	}
	// There are so many potential inconsistencies when edge type is deleted so flush the whole cache
	return ucerr.Wrap(c.FlushCache())
}

// CreateObjectRequest is the request body for creating an object
type CreateObjectRequest struct {
	Object Object `json:"object" yaml:"object"`
}

// CreateObject creates a new object with a given ID, name, and type.
func (c *Client) CreateObject(ctx context.Context, id, typeID uuid.UUID, alias string, opts ...Option) (*Object, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	input := Object{
		BaseModel:      ucdb.NewBase(),
		Alias:          &alias,
		TypeID:         typeID,
		OrganizationID: options.organizationID,
	}
	if !id.IsNil() {
		input.ID = id
	}

	if alias == "" { // this allows storing multiple objects with "" alias
		input.Alias = nil
	}

	return cache.CreateItemClient[Object](ctx, &c.cm, id, &input, ObjectKeyID, c.cm.N.GetKeyName(ObjAliasNameKeyID, []string{typeID.String(), alias, options.organizationID.String()}), options.ifNotExists, options.bypassCache, nil,
		func(i *Object) (*Object, error) {
			req := CreateObjectRequest{*i}
			var resp Object
			if options.ifNotExists {
				exists, existingID, err := c.client.CreateIfNotExists(ctx, "/authz/objects", req, &resp)
				if err != nil {
					return nil, ucerr.Wrap(err)
				}
				if exists {
					if id.IsNil() || existingID == id {
						resp = req.Object
						resp.ID = existingID
					} else {
						return nil, ucerr.Errorf("object already exists with different ID: %s", existingID)
					}
				}
			} else {
				if err := c.client.Post(ctx, "/authz/objects", req, &resp); err != nil {
					return nil, ucerr.Wrap(err)
				}
			}
			return &resp, nil
		}, func(in *Object, curr *Object) bool {
			return curr.EqualsIgnoringID(in) && (id.IsNil() || curr.ID == id)
		})
}

// GetObject returns an object by ID.
func (c *Client) GetObject(ctx context.Context, id uuid.UUID, opts ...Option) (*Object, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	return cache.GetItemClient[Object](ctx, c.cm, id, ObjectKeyID, options.bypassCache, func(id uuid.UUID, conflict cache.Sentinel, resp *Object) error {
		if err := c.client.Get(ctx, fmt.Sprintf("/authz/objects/%s", id), resp); err != nil {
			if jsonclient.IsHTTPNotFound(err) {
				return ucerr.Wrap(ucerr.Combine(err, ErrObjectNotFound))
			}
			return ucerr.Wrap(err)
		}
		return nil
	})
}

// GetObjectForName returns an object with a given name.
func (c *Client) GetObjectForName(ctx context.Context, typeID uuid.UUID, name string, opts ...Option) (*Object, error) {
	ctx = request.NewRequestID(ctx)

	if typeID == UserObjectTypeID {
		return nil, ucerr.New("_user objects do not currently support lookup by alias")
	}

	if name == "" {
		return nil, ucerr.New("name cannot be empty")
	}

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	if !options.bypassCache {
		var v *Object
		var err error

		v, _, _, err = cache.GetItemFromCache[Object](ctx, c.cm, c.cm.N.GetKeyName(ObjAliasNameKeyID, []string{typeID.String(), name, options.organizationID.String()}), false)
		if err != nil {
			uclog.Errorf(ctx, "GetObjectForName failed to get item from cache: %v", err)
		} else if v != nil {
			return v, nil
		}
	}

	// TODO: support a name-based path, e.g. `/authz/objects/<objectname>`
	pager, err := pagination.ApplyOptions()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	query := pager.Query()
	query.Add("type_id", typeID.String())
	query.Add("name", name)
	resp, err := c.ListObjectsFromQuery(ctx, query, opts...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if len(resp.Data) > 0 {
		return &resp.Data[0], nil
	}
	return nil, ErrObjectNotFound
}

// UpdateObjectRequest is the request struct for updating an object
type UpdateObjectRequest struct {
	ID     uuid.UUID `json:"id" yaml:"id" validate:"notnil"`
	Alias  *string   `json:"alias" yaml:"alias"`
	Source *string   `json:"source" yaml:"source"` // internal use only
}

// UpdateObject updates the alias of an existing user object in the authz system
func (c *Client) UpdateObject(ctx context.Context, id uuid.UUID, alias *string, opts ...Option) (*Object, error) {
	ctx = request.NewRequestID(ctx)

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	if !options.bypassCache {
		var o *Object
		var err error

		o, _, _, err = cache.GetItemFromCache[Object](ctx, c.cm, c.cm.N.GetKeyNameWithID(ObjectKeyID, id), false)
		if err != nil {
			uclog.Errorf(ctx, "UpdateObject failed to get item from cache: %v", err)
		} else if o != nil && ((o.Alias == nil && alias == nil) || (o.Alias != nil && alias != nil && *o.Alias == *alias)) {
			return o, nil
		}
	}

	obj := Object{BaseModel: ucdb.NewBaseWithID(id), TypeID: UserObjectTypeID} // TODO: we don't know the object's original alias, so we can't invalidate it
	s, err := cache.TakeItemLock(ctx, cache.Update, c.cm, obj)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	defer cache.ReleaseItemLock(ctx, c.cm, cache.Update, obj, s)

	req := UpdateObjectRequest{
		ID:     id,
		Alias:  alias,
		Source: options.source,
	}
	var resp Object
	if err := c.client.Put(ctx, fmt.Sprintf("/authz/objects/%s", id), req, &resp); err != nil {
		if jsonclient.IsHTTPNotFound(err) {
			return nil, ErrObjectNotFound
		}
		return nil, ucerr.Wrap(err)
	}

	cache.SaveItemToCache(ctx, c.cm, resp, s, true, nil)
	return &resp, nil
}

// DeleteObject deletes an object by ID.
func (c *Client) DeleteObject(ctx context.Context, id uuid.UUID) error {
	ctx = request.NewRequestID(ctx)

	obj := &Object{BaseModel: ucdb.NewBaseWithID(id)}
	// Stop in flight reads/writes of this object, edges leading to/from this object, paths including this object and object collection from committing to the cache
	obj, _, _, err := cache.GetItemFromCache[Object](ctx, c.cm, obj.GetPrimaryKey(c.cm.N), false)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if obj == nil {
		obj = &Object{BaseModel: ucdb.NewBaseWithID(id)}
	}
	s, err := cache.TakeItemLock(ctx, cache.Delete, c.cm, *obj)
	if err != nil {
		return ucerr.Wrap(err)
	}
	defer cache.ReleaseItemLock(ctx, c.cm, cache.Delete, *obj, s)

	if err := c.client.Delete(ctx, fmt.Sprintf("/authz/objects/%s", id), nil); err != nil {
		if jsonclient.IsHTTPNotFound(err) {
			return ucerr.Wrap(ucerr.Combine(err, ErrObjectNotFound))
		}
		return ucerr.Wrap(err)
	}
	return nil
}

// DeleteEdgesByObject deletes all edges going in or  out of an object by ID.
func (c *Client) DeleteEdgesByObject(ctx context.Context, id uuid.UUID) error {
	ctx = request.NewRequestID(ctx)

	// Stop in flight reads of edges that include this object as source or target as well as paths starting from this object from committing to the cache
	// We don't block reads of collections/paths that end at this object since we may not have full set of edges without reading the server
	obj := Object{BaseModel: ucdb.NewBaseWithID(id)}

	// Taking a lock will delete all edges and paths that include this object as source or target. We intentionally tombstone the dependency key for the object to
	// prevent inflight reads of edge collection from object connected to this one from committing potentially stale results to the cache.
	s, err := cache.TakePerItemCollectionLock[Object](ctx, cache.Delete, c.cm, nil, obj)

	if err != nil {
		return ucerr.Wrap(err)
	}
	defer cache.ReleasePerItemCollectionLock[Object](ctx, c.cm, nil, obj, s)

	if err := c.client.Delete(ctx, fmt.Sprintf("/authz/objects/%s/edges", id), nil); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// ListObjectsResponse represents a paginated response from listing objects.
type ListObjectsResponse struct {
	Data []Object `json:"data" yaml:"data"`
	pagination.ResponseFields
}

// ListObjects lists `limit` objects in sorted order with pagination, starting after a given ID (or uuid.Nil to start from the beginning).
func (c *Client) ListObjects(ctx context.Context, opts ...Option) (*ListObjectsResponse, error) {
	ctx = request.NewRequestID(ctx)

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	query := pager.Query()
	if !options.organizationID.IsNil() {
		query.Add("organization_id", options.organizationID.String())
	}
	return c.ListObjectsFromQuery(ctx, query, opts...)
}

// ListObjectsFromQuery takes in a query that can handle filters passed from console as well as the default method.
func (c *Client) ListObjectsFromQuery(ctx context.Context, query url.Values, opts ...Option) (*ListObjectsResponse, error) {
	ctx = request.NewRequestID(ctx)

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}
	if !options.organizationID.IsNil() {
		query.Add("organization_id", options.organizationID.String())
	}

	s := cache.NoLockSentinel
	if !options.bypassCache {
		var err error
		ckey := c.cm.N.GetKeyNameStatic(ObjectCollectionKeyID)
		_, _, s, _, err = cache.GetItemsArrayFromCache[Object](ctx, c.cm, ckey, true)
		if err != nil {
			uclog.Errorf(ctx, "ListObjectsFromQuery failed to get item from cache: %v", err)
		}
		// Release the lock after the request is done since we are not writing to global collection
		defer cache.ReleasePerItemCollectionLock[Object](ctx, c.cm, []cache.Key{ckey}, Object{}, s)
	}

	// TODO needs to always be paginated get
	var resp ListObjectsResponse
	if err := c.client.Get(ctx, fmt.Sprintf("/authz/objects?%s", query.Encode()), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	cache.SaveItemsFromCollectionToCache(ctx, c.cm, resp.Data, s)

	return &resp, nil
}

// ListEdgesResponse is the paginated response from listing edges.
type ListEdgesResponse struct {
	Data []Edge `json:"data" yaml:"data"`
	pagination.ResponseFields
}

// ListEdges lists `limit` edges.
func (c *Client) ListEdges(ctx context.Context, opts ...Option) (*ListEdgesResponse, error) {
	ctx = request.NewRequestID(ctx)

	// TODO: this function doesn't support organizations yet, because I haven't figured out a performant way to
	// do it.  The problem is that we need to filter by organization ID, but we don't have that information in
	// the edges table, only on the objects they connect.

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	query := pager.Query()

	var resp ListEdgesResponse
	if err := c.client.Get(ctx, fmt.Sprintf("/authz/edges?%s", query.Encode()), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	// We don't save the individual edges in the cache because it is not certain that edges will be accessed by ID or name in the immediate future

	return &resp, nil
}

// ListEdgesOnObject lists `limit` edges (relationships) where the given object is a source or target.
func (c *Client) ListEdgesOnObject(ctx context.Context, objectID uuid.UUID, opts ...Option) (*ListEdgesResponse, error) {
	ctx = request.NewRequestID(ctx)

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	query := pager.Query()

	obj := Object{BaseModel: ucdb.NewBaseWithID(objectID)}
	s := cache.NoLockSentinel
	var edges *[]Edge
	if !options.bypassCache {
		edges, _, s, _, err = cache.GetItemsArrayFromCache[Edge](ctx, c.cm, c.cm.N.GetKeyNameWithID(ObjEdgesKeyID, objectID), true)
		if err != nil {
			uclog.Errorf(ctx, "ListEdgesOnObject failed to get item from cache: %v", err)
		}

		if edges != nil && len(*edges) <= pager.GetLimit() {
			resp := ListEdgesResponse{Data: *edges, ResponseFields: pagination.ResponseFields{HasNext: false}}
			return &resp, nil
		}

		// Only release the sentinel if we didn't get the edges from the cache
		if edges == nil {
			defer cache.ReleasePerItemCollectionLock(ctx, c.cm, nil, obj, s)
		}
	}

	var resp ListEdgesResponse
	if err := c.client.Get(ctx, fmt.Sprintf("/authz/objects/%s/edges?%s", objectID, query.Encode()), &resp); err != nil {
		if jsonclient.IsHTTPNotFound(err) {
			return nil, ErrObjectNotFound
		}
		return nil, ucerr.Wrap(err)
	}

	// Only cache the response if it fits on one page
	if !resp.HasNext && !resp.HasPrev {
		ckey := c.cm.N.GetKeyNameWithID(ObjEdgesKeyID, objectID)
		cache.SaveItemsToCollection(ctx, c.cm, obj, resp.Data, ckey, ckey, s, false)
	}
	return &resp, nil
}

// ListEdgesBetweenObjects lists all edges (relationships) with a given source & target object.
func (c *Client) ListEdgesBetweenObjects(ctx context.Context, sourceObjectID, targetObjectID uuid.UUID, opts ...Option) ([]Edge, error) {
	ctx = request.NewRequestID(ctx)

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	obj := Object{BaseModel: ucdb.NewBaseWithID(sourceObjectID)}
	ckey := c.cm.N.GetKeyName(EdgesObjToObjID, []string{sourceObjectID.String(), targetObjectID.String()})

	s := cache.NoLockSentinel
	if !options.bypassCache {
		var cEdges *[]Edge
		var err error

		// First try to read the all in/out edges from source object
		cEdges, _, _, _, err = cache.GetItemsArrayFromCache[Edge](ctx, c.cm, c.cm.N.GetKeyNameWithID(ObjEdgesKeyID, sourceObjectID), false)
		if err != nil {
			uclog.Errorf(ctx, "ListEdgesBetweenObjects failed to get item from cache: %v", err)
		} else if cEdges != nil {
			filteredEdges := make([]Edge, 0)
			for _, edge := range *cEdges {
				if edge.TargetObjectID == targetObjectID {
					filteredEdges = append(filteredEdges, edge)
				}
			}
			return filteredEdges, nil
		}

		// Next try to read the edges between target object and source object. We could also try to read the edges from target object but in authz graph
		// it is rare to traverse in both directions so those collections would be less likely to be cached.
		cEdges, _, s, _, err = cache.GetItemsArrayFromCache[Edge](ctx, c.cm, ckey, true)
		if err != nil {
			uclog.Errorf(ctx, "ListEdgesBetweenObjects failed to get item from cache: %v", err)
		} else if cEdges != nil {
			return *cEdges, nil
		}

		// Clear the lock in case of an error
		defer cache.ReleasePerItemCollectionLock(ctx, c.cm, []cache.Key{ckey}, obj, s)
	}
	query := url.Values{}
	query.Add("target_object_id", targetObjectID.String())
	var resp ListEdgesResponse
	var edges []Edge
	for {
		if err := c.client.Get(ctx, fmt.Sprintf("/authz/objects/%s/edges?%s", sourceObjectID, query.Encode()), &resp); err != nil {
			if jsonclient.IsHTTPNotFound(err) {
				return nil, ucerr.Wrap(ucerr.Combine(err, ErrObjectNotFound))
			}
			return nil, ucerr.Wrap(err)
		}
		edges = append(edges, resp.Data...)
		if !resp.HasNext {
			break
		}
	}

	cache.SaveItemsToCollection(ctx, c.cm, obj, resp.Data, ckey, ckey, s, false)

	return edges, nil
}

// GetEdge returns an edge by ID.
func (c *Client) GetEdge(ctx context.Context, id uuid.UUID, opts ...Option) (*Edge, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	return cache.GetItemClient[Edge](ctx, c.cm, id, EdgeKeyID, options.bypassCache, func(id uuid.UUID, conflict cache.Sentinel, resp *Edge) error {
		if err := c.client.Get(ctx, fmt.Sprintf("/authz/edges/%s", id), resp); err != nil {
			if jsonclient.IsHTTPNotFound(err) {
				return ucerr.Wrap(ucerr.Combine(err, ErrEdgeNotFound))
			}
			return ucerr.Wrap(err)
		}
		return nil
	})
}

// FindEdge finds an existing edge (relationship) between two objects.
func (c *Client) FindEdge(ctx context.Context, sourceObjectID, targetObjectID, edgeTypeID uuid.UUID, opts ...Option) (*Edge, error) {
	ctx = request.NewRequestID(ctx)

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	s := cache.NoLockSentinel
	if !options.bypassCache {
		var edge *Edge
		var edges *[]Edge
		var err error

		// Try to fetch the individual edge first using secondary key  Source_Target_TypeID
		edge, _, _, err = cache.GetItemFromCache[Edge](ctx, c.cm, c.cm.N.GetKeyName(EdgeFullKeyID, []string{sourceObjectID.String(), targetObjectID.String(), edgeTypeID.String()}), false)
		// Since we are not taking a lock we can ignore cache errors
		if err != nil {
			uclog.Errorf(ctx, "FindEdge failed to get item from cache: %v", err)
		} else if edge != nil {
			return edge, nil
		}
		// If the edges are in the cache by source->target - iterate over that set first
		edges, _, _, _, err = cache.GetItemsArrayFromCache[Edge](ctx, c.cm, c.cm.N.GetKeyName(EdgesObjToObjID, []string{sourceObjectID.String(), targetObjectID.String()}), false)
		// Since we are not taking a lock we can ignore cache errors
		if err == nil && edges != nil {
			for _, edge := range *edges {
				if edge.EdgeTypeID == edgeTypeID {
					return &edge, nil
				}
			}
			// In theory we could return NotFound here but this is a rare enough case that it makes sense to try the server
		}
		// If there is a cache miss, try to get the edges from all in/out edges on the source object
		edges, _, _, _, err = cache.GetItemsArrayFromCache[Edge](ctx, c.cm, c.cm.N.GetKeyNameWithID(ObjEdgesKeyID, sourceObjectID), false)
		// Since we are not taking a lock we can ignore cache errors
		if err == nil && edges != nil {
			for _, edge := range *edges {
				if edge.TargetObjectID == targetObjectID && edge.EdgeTypeID == edgeTypeID {
					return &edge, nil
				}
			}
			// In theory we could return NotFound here but this is a rare enough case that it makes sense to try the server
		}
		// We could also try all in/out edges from targetObjectID collection

		// If we still don't have the edge, try the server but we can't take a lock single edge lock since we don't know the primary key
		_, _, s, _, err = cache.GetItemsArrayFromCache[Object](ctx, c.cm, c.cm.N.GetKeyNameStatic(EdgeCollectionKeyID), true)
		if err != nil {
			uclog.Errorf(ctx, "FindEdge failed to set lock in the cache: %v", err)
		}
	}
	var resp ListEdgesResponse

	query := url.Values{}
	query.Add("source_object_id", sourceObjectID.String())
	query.Add("target_object_id", targetObjectID.String())
	query.Add("edge_type_id", edgeTypeID.String())
	if err := c.client.Get(ctx, fmt.Sprintf("/authz/edges?%s", query.Encode()), &resp); err != nil {
		if jsonclient.IsHTTPNotFound(err) {
			return nil, ucerr.Wrap(ucerr.Combine(err, ErrEdgeNotFound))
		}
		return nil, ucerr.Wrap(err)
	}
	if len(resp.Data) != 1 {
		return nil, ucerr.Errorf("expected 1 edge from FindEdge, got %d", len(resp.Data))
	}

	cache.SaveItemsFromCollectionToCache(ctx, c.cm, resp.Data, s)

	return &resp.Data[0], nil
}

// CreateEdgeRequest is the request body for creating an edge
type CreateEdgeRequest struct {
	Edge Edge `json:"edge" yaml:"edge"`
}

// CreateEdge creates an edge (relationship) between two objects.
func (c *Client) CreateEdge(ctx context.Context, id, sourceObjectID, targetObjectID, edgeTypeID uuid.UUID, opts ...Option) (*Edge, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	input := Edge{
		BaseModel:      ucdb.NewBase(),
		EdgeTypeID:     edgeTypeID,
		SourceObjectID: sourceObjectID,
		TargetObjectID: targetObjectID,
	}
	if !id.IsNil() {
		input.ID = id
	}

	additionalKeys := []cache.Key{c.cm.N.GetKeyName(EdgesObjToObjID, []string{input.SourceObjectID.String(), input.TargetObjectID.String()}), // Source -> Target edges collection
		c.cm.N.GetKeyNameWithID(ObjEdgesKeyID, input.SourceObjectID),                                               // Source all in/out edges collection
		c.cm.N.GetKeyName(EdgesObjToObjID, []string{input.TargetObjectID.String(), input.SourceObjectID.String()}), // Target -> Source edges collection
		c.cm.N.GetKeyNameWithID(ObjEdgesKeyID, input.TargetObjectID),                                               // Target all in/out edges collection
	}

	return cache.CreateItemClient[Edge](ctx, &c.cm, id, &input, EdgeKeyID, c.cm.N.GetKeyName(EdgeFullKeyID, []string{sourceObjectID.String(), targetObjectID.String(), edgeTypeID.String()}), options.ifNotExists, options.bypassCache, additionalKeys,
		func(i *Edge) (*Edge, error) {
			req := CreateEdgeRequest{*i}
			var resp Edge
			if options.ifNotExists {
				exists, existingID, err := c.client.CreateIfNotExists(ctx, "/authz/edges", req, &resp)
				if err != nil {
					return nil, ucerr.Wrap(err)
				}
				if exists {
					if id.IsNil() || existingID == id {
						resp = req.Edge
						resp.ID = existingID
					} else {
						return nil, ucerr.Errorf("edge already exists with different ID: %s", existingID)
					}
				}
			} else {
				if err := c.client.Post(ctx, "/authz/edges", req, &resp); err != nil {
					return nil, ucerr.Wrap(err)
				}
			}
			return &resp, nil
		}, func(in *Edge, v *Edge) bool {
			return v.EqualsIgnoringID(in) && (id.IsNil() || v.ID == id)
		})
}

// DeleteEdge deletes an edge by ID.
func (c *Client) DeleteEdge(ctx context.Context, edgeID uuid.UUID) error {
	ctx = request.NewRequestID(ctx)

	edge, _, _, err := cache.GetItemFromCache[Edge](ctx, c.cm, c.cm.N.GetKeyNameWithID(EdgeKeyID, edgeID), false)
	if err != nil {
		return ucerr.Wrap(err)
	}

	edgeBase := Edge{BaseModel: ucdb.NewBaseWithID(edgeID)}
	if edge == nil {
		edge = &edgeBase
	}
	s, err := cache.TakeItemLock(ctx, cache.Delete, c.cm, *edge)
	if err != nil {
		return ucerr.Wrap(err)
	}
	defer cache.ReleaseItemLock(ctx, c.cm, cache.Delete, *edge, s)

	if err = c.client.Delete(ctx, fmt.Sprintf("/authz/edges/%s", edgeID), nil); err != nil {
		if jsonclient.IsHTTPNotFound(err) {
			return ucerr.Wrap(ucerr.Combine(err, ErrEdgeNotFound))
		}
		return ucerr.Wrap(err)
	}

	return nil
}

// AttributePathNode is a node in a path list from source to target, if CheckAttribute succeeds.
type AttributePathNode struct {
	ObjectID uuid.UUID `json:"object_id" yaml:"object_id" validate:"notnil"`
	EdgeID   uuid.UUID `json:"edge_id" yaml:"edge_id"`
}

// GetID returns nil ID since we never create/update attribute path directly
func (a AttributePathNode) GetID() uuid.UUID {
	return uuid.Nil
}

//go:generate genvalidate AttributePathNode

// CheckAttributeResponse is returned by the checkattribute endpoint.
type CheckAttributeResponse struct {
	HasAttribute bool                `json:"has_attribute" yaml:"has_attribute"`
	Path         []AttributePathNode `json:"path" yaml:"path"`
}

// CheckAttribute returns true if the source object has the given attribute on the target object.
func (c *Client) CheckAttribute(ctx context.Context, sourceObjectID, targetObjectID uuid.UUID, attributeName string, opts ...Option) (*CheckAttributeResponse, error) {
	ctx = request.NewRequestID(ctx)

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	ckey := c.cm.N.GetKeyName(AttributePathObjToObjID, []string{sourceObjectID.String(), targetObjectID.String(), attributeName})

	s := cache.NoLockSentinel
	if !options.bypassCache {
		var path *[]AttributePathNode
		var err error

		path, _, s, _, err = cache.GetItemsArrayFromCache[AttributePathNode](ctx, c.cm, ckey, true)
		if err != nil {
			uclog.Errorf(ctx, "CheckAttribute failed to get item from cache: %v", err)
		} else if path != nil {
			return &CheckAttributeResponse{HasAttribute: true, Path: *path}, nil
		}
	}

	obj := Object{BaseModel: ucdb.NewBaseWithID(sourceObjectID)}

	// Release the lock in case of error
	defer cache.ReleasePerItemCollectionLock(ctx, c.cm, []cache.Key{ckey}, obj, s)

	var resp CheckAttributeResponse
	query := url.Values{}
	query.Add("source_object_id", sourceObjectID.String())
	query.Add("target_object_id", targetObjectID.String())
	query.Add("attribute", attributeName)
	if err := c.client.Get(ctx, fmt.Sprintf("/authz/checkattribute?%s", query.Encode()), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	if resp.HasAttribute {
		// We can only cache positive responses, since we don't know when the path will be added to invalidate the negative result.
		cache.SaveItemsToCollection(ctx, c.cm, obj, resp.Path, ckey, ckey, s, false)
	}
	return &resp, nil
}

// ListAttributes returns a list of attributes that the source object has on the target object.
func (c *Client) ListAttributes(ctx context.Context, sourceObjectID, targetObjectID uuid.UUID) ([]string, error) {
	ctx = request.NewRequestID(ctx)

	var resp []string
	query := url.Values{}
	query.Add("source_object_id", sourceObjectID.String())
	query.Add("target_object_id", targetObjectID.String())
	if err := c.client.Get(ctx, fmt.Sprintf("/authz/listattributes?%s", query.Encode()), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}
	// This is currently unreachable until we return a path for each attribute from the server that we can use for invalidation.
	return resp, nil
}

// ListObjectsReachableWithAttributeResponse is the response from the ListObjectsReachableWithAttribute endpoint.
type ListObjectsReachableWithAttributeResponse struct {
	Data []uuid.UUID `json:"data" yaml:"data"`
}

// ListObjectsReachableWithAttribute returns a list of object IDs of a certain type that are reachable from the source object with the given attribute
func (c *Client) ListObjectsReachableWithAttribute(ctx context.Context, sourceObjectID uuid.UUID, targetObjectTypeID uuid.UUID, attributeName string) ([]uuid.UUID, error) {
	ctx = request.NewRequestID(ctx)

	var resp ListObjectsReachableWithAttributeResponse
	query := url.Values{}
	query.Add("source_object_id", sourceObjectID.String())
	query.Add("target_object_type_id", targetObjectTypeID.String())
	query.Add("attribute", attributeName)
	if err := c.client.Get(ctx, fmt.Sprintf("/authz/listobjectsreachablewithattribute?%s", query.Encode()), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	// This is currently unreachable until we return a path for each reachable object from the server that we can use for invalidation.
	return resp.Data, nil
}

// ListOrganizationsResponse is the response from the ListOrganizations endpoint.
type ListOrganizationsResponse struct {
	Data []Organization `json:"data" yaml:"data"`
	pagination.ResponseFields
}

// ListOrganizationsFromQuery takes in a query that can handle filters passed from console as well as the default method.
func (c *Client) ListOrganizationsFromQuery(ctx context.Context, query url.Values, opts ...Option) (*ListOrganizationsResponse, error) {
	ctx = request.NewRequestID(ctx)

	var options options
	for _, opt := range opts {
		opt.apply(&options)
	}

	var resp ListOrganizationsResponse
	if err := c.client.Get(ctx, fmt.Sprintf("/authz/organizations?%s", query.Encode()), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// ListOrganizationsPaginated lists `limit` organizations in sorted order with pagination, starting after a given ID (or uuid.Nil to start from the beginning).
func (c *Client) ListOrganizationsPaginated(ctx context.Context, opts ...Option) (*ListOrganizationsResponse, error) {
	ctx = request.NewRequestID(ctx)

	var options options
	for _, opt := range opts {
		opt.apply(&options)
	}

	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	query := pager.Query()
	return c.ListOrganizationsFromQuery(ctx, query)
}

// ListOrganizations lists all organizations for a tenant
func (c *Client) ListOrganizations(ctx context.Context, opts ...Option) ([]Organization, error) {
	ctx = request.NewRequestID(ctx)

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	s := cache.NoLockSentinel
	if !options.bypassCache {
		var v *[]Organization
		var err error
		v, _, s, _, err = cache.GetItemsArrayFromCache[Organization](ctx, c.cm, c.cm.N.GetKeyNameStatic(OrganizationCollectionKeyID), true)
		if err != nil {
			uclog.Errorf(ctx, "ListOrganizations failed to get item from cache: %v", err)
		} else if v != nil {
			return *v, nil
		}
	}

	// TODO: we should eventually support pagination arguments to this method, but for now we assume
	// there aren't that many organizations and just fetch them all.

	orgs := make([]Organization, 0)

	pager, err := pagination.ApplyOptions()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	for {
		query := pager.Query()

		var resp ListOrganizationsResponse
		if err := c.client.Get(ctx, fmt.Sprintf("/authz/organizations?%s", query.Encode()), &resp); err != nil {
			return nil, ucerr.Wrap(err)
		}

		orgs = append(orgs, resp.Data...)

		cache.SaveItemsFromCollectionToCache(ctx, c.cm, resp.Data, s)

		if !pager.AdvanceCursor(resp.ResponseFields) {
			break
		}
	}
	ckey := c.cm.N.GetKeyNameStatic(OrganizationCollectionKeyID)
	cache.SaveItemsToCollection(ctx, c.cm, ObjectType{}, orgs, ckey, ckey, s, true)
	return orgs, nil
}

// GetOrganization retrieves a single organization by its UUID
func (c *Client) GetOrganization(ctx context.Context, id uuid.UUID, opts ...Option) (*Organization, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	return cache.GetItemClient[Organization](ctx, c.cm, id, OrganizationKeyID, options.bypassCache, func(id uuid.UUID, conflict cache.Sentinel, resp *Organization) error {

		if err := c.client.Get(ctx, fmt.Sprintf("/authz/organizations/%s", id), &resp); err != nil {
			return ucerr.Wrap(err)
		}
		return nil
	})
}

// GetOrganizationForName retrieves a single organization by its name
func (c *Client) GetOrganizationForName(ctx context.Context, name string, opts ...Option) (*Organization, error) {
	ctx = request.SetRequestData(ctx, nil, uuid.Must(uuid.NewV4()))

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}
	s := cache.NoLockSentinel
	if !options.bypassCache {
		var v *Organization
		var err error

		v, _, s, err = cache.GetItemFromCache[Organization](ctx, c.cm, c.cm.N.GetKeyNameWithString(OrganizationNameKeyID, name), false)
		if err != nil {
			uclog.Errorf(ctx, "GetOrganization failed to get item from cache: %v", err)
		} else if v != nil {
			return v, nil
		}
	}

	pager, err := pagination.ApplyOptions(
		pagination.Filter(fmt.Sprintf("('name',EQ,'%s')", name)),
	)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	query := pager.Query()
	orgs, err := c.ListOrganizationsFromQuery(ctx, query)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if len(orgs.Data) != 1 {
		return nil, ucerr.Errorf("expected 1 organization from GetOrganizationForName, got %d", len(orgs.Data))
	}

	cache.SaveItemToCache(ctx, c.cm, orgs.Data[0], s, false, nil)

	return &orgs.Data[0], nil
}

// CreateOrganizationRequest is the request struct to the CreateOrganization endpoint
type CreateOrganizationRequest struct {
	Organization Organization `json:"organization" yaml:"organization"`
}

// CreateOrganization creates an organization
// Note that if the `IfNotExists` option is used, the organizations must match exactly (eg. name and region),
// otherwise a 409 Conflict error will still be returned.
func (c *Client) CreateOrganization(ctx context.Context, id uuid.UUID, name string, region region.DataRegion, opts ...Option) (*Organization, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	input := Organization{
		BaseModel: ucdb.NewBase(),
		Name:      name,
		Region:    region,
	}

	if !id.IsNil() {
		input.ID = id
	}

	return cache.CreateItemClient[Organization](ctx, &c.cm, id, &input, OrganizationKeyID, c.cm.N.GetKeyNameWithString(OrganizationNameKeyID, name), options.ifNotExists, options.bypassCache, nil,
		func(i *Organization) (*Organization, error) {
			req := CreateOrganizationRequest{*i}
			var resp Organization
			if options.ifNotExists {
				exists, existingID, err := c.client.CreateIfNotExists(ctx, "/authz/organizations", req, &resp)
				if err != nil {
					return nil, ucerr.Wrap(err)
				}
				if exists {
					if id.IsNil() || existingID == id {
						resp = req.Organization
						resp.ID = existingID
					} else {
						return nil, ucerr.Errorf("organization exists with different ID: %s", existingID)
					}
				}
			} else {
				if err := c.client.Post(ctx, "/authz/organizations", req, &resp); err != nil {
					return nil, ucerr.Wrap(err)
				}
			}
			return &resp, nil
		}, func(in *Organization, v *Organization) bool {
			return v.Name == in.Name && v.Region == in.Region && (id.IsNil() || v.ID == id)
		})
}

// UpdateOrganizationRequest is the request struct to the UpdateOrganization endpoint
type UpdateOrganizationRequest struct {
	Name   string            `json:"name" yaml:"name" validate:"notempty"`
	Region region.DataRegion `json:"region" yaml:"region"` // this is a UC Region (not an AWS region)
}

// UpdateOrganization updates an organization
func (c *Client) UpdateOrganization(ctx context.Context, id uuid.UUID, name string, region region.DataRegion, opts ...Option) (*Organization, error) {
	ctx = request.NewRequestID(ctx)
	var err error

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	req := UpdateOrganizationRequest{
		Name:   name,
		Region: region,
	}

	org := Organization{BaseModel: ucdb.NewBaseWithID(id)}
	s := cache.NoLockSentinel
	if !options.bypassCache {
		s, err = cache.TakeItemLock(ctx, cache.Update, c.cm, org)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
	}
	var resp Organization
	if err := c.client.Put(ctx, fmt.Sprintf("/authz/organizations/%s", id), req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	cache.SaveItemToCache(ctx, c.cm, resp, s, true, nil)

	return &resp, nil
}
