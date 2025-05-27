package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/authz/config"
	"userclouds.com/authz/internal"
	"userclouds.com/authz/internal/tenantstate"
	idpAuthz "userclouds.com/idp/authz"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/security"
	"userclouds.com/plex/manager"
)

type handler struct {
	companyConfigStorage         *companyconfig.Storage
	checkAttributeServiceNameMap map[uuid.UUID]*string // map from tenantID to check attribute service name
}

// NewHandler returns a new AuthZ API handler
func NewHandler(ctx context.Context, storage *companyconfig.Storage, cfg config.Config) http.Handler {
	h := &handler{companyConfigStorage: storage, checkAttributeServiceNameMap: make(map[uuid.UUID]*string)}
	if cfg.CheckAttributeServiceMap != nil {
		currentRegion := region.Current()
		for tenantID, regionalServers := range cfg.CheckAttributeServiceMap {
			for _, config := range regionalServers {
				if config.Region == currentRegion {
					uclog.Infof(ctx, "Check attribute service name for tenant %s is %s", tenantID, config.ServiceName)
					h.checkAttributeServiceNameMap[tenantID] = &config.ServiceName
				}
			}
		}
	}
	uclog.Infof(ctx, "Check attribute will handle %d tenants", len(h.checkAttributeServiceNameMap))
	hb := builder.NewHandlerBuilder()
	handlerBuilder(hb, h)

	hb.MethodHandler("/check_attribute").Get(h.checkAttributeGenerated)

	hb.CollectionHandler("/migrate/objects").
		Put(h.migrateObject).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	hb.CollectionHandler("/migrate/edgetypes").
		Put(h.migrateEdgeType).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	return hb.Build()
}

//go:generate genhandler /authz collection,ObjectType,h.newRoleBasedAuthorizer(),/objecttypes collection,Object,h.newRoleBasedAuthorizer(),/objects collection,EdgeType,h.newRoleBasedAuthorizer(),/edgetypes collection,Edge,h.newRoleBasedAuthorizer(),/edges collection,Organization,h.newRoleBasedAuthorizer(),/organizations GET,listAttributes,/listattributes nestedcollection,Edge,h.newNestedRoleBasedAuthorizer(),/edges,Object GET,checkAttribute,/checkattribute GET,listObjectsReachableWithAttribute,/listobjectsreachablewithattribute

func (h *handler) newRoleBasedAuthorizer() uchttp.CollectionAuthorizer {
	return &uchttp.MethodAuthorizer{
		GetAllF: func(r *http.Request) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), false))
		},
		GetOneF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), false))
		},
		PostF: func(r *http.Request) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), true))
		},
		PutF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), true))
		},
		DeleteF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), true))
		},
		DeleteAllF: func(r *http.Request) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), true))
		},
		NestedF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), false))
		},
	}
}

func (h *handler) newNestedRoleBasedAuthorizer() uchttp.NestedCollectionAuthorizer {
	return &uchttp.NestedMethodAuthorizer{
		GetAllF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), false))
		},
		GetOneF: func(r *http.Request, _, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), false))
		},
		PostF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), true))
		},
		PutF: func(r *http.Request, _, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), true))
		},
		DeleteF: func(r *http.Request, _, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), true))
		},
		DeleteAllF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(r.Context(), true))
		},
	}
}

func (h *handler) ensureTenantMember(ctx context.Context, adminOnly bool) error {
	return nil // TODO: figure out how to do this performantly
}

type listObjectTypesParams struct {
	pagination.QueryParams
}

// OpenAPI Summary: List Object Types
// OpenAPI Tags: Object Types
// OpenAPI Description: This endpoint returns a paginated list of all object types in a tenant.
func (h *handler) listObjectTypes(ctx context.Context, req listObjectTypesParams) (*authz.ListObjectTypesResponse, int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	pager, err := authz.NewObjectTypePaginatorFromQuery(req)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	types, respFields, err := tenantState.Storage.ListObjectTypesPaginated(ctx, *pager)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &authz.ListObjectTypesResponse{
		Data:           types,
		ResponseFields: *respFields,
	}, http.StatusOK, nil, nil
}

// OpenAPI Summary: Get Object Type
// OpenAPI Tags: Object Types
// OpenAPI Description: This endpoint gets an object type by ID.
func (h *handler) getObjectType(ctx context.Context, id uuid.UUID, _ url.Values) (*authz.ObjectType, int, []auditlog.Entry, error) {
	ts := tenantstate.MustGet(ctx)

	typ, err := ts.Storage.GetObjectType(ctx, id)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	return typ, http.StatusOK, nil, nil
}

// OpenAPI Summary: Create Object Type
// OpenAPI Tags: Object Types
// OpenAPI Description: This endpoint creates a new object type.
func (h *handler) createObjectType(ctx context.Context, req authz.CreateObjectTypeRequest) (*authz.ObjectType, int, []auditlog.Entry, error) {
	if err := h.checkEmployeeRequest(ctx); err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	tenantState := tenantstate.MustGet(ctx)

	if err := tenantState.Storage.InsertObjectType(ctx, &req.ObjectType); err != nil {
		if ucdb.IsUniqueViolation(err) {
			if existing, err := tenantState.Storage.GetObjectTypeForName(ctx, req.ObjectType.TypeName); err == nil {
				if existing.ID == req.ObjectType.ID && existing.EqualsIgnoringID(&req.ObjectType) {
					// This is for back-compat, if IfNotExists() is used it wouldn't be necessary
					return &req.ObjectType, http.StatusCreated, nil, nil
				}
				return nil, http.StatusConflict, nil, ucerr.WrapWithFriendlyStructure(nil, jsonclient.SDKStructuredError{
					Error:     "This object type already exists",
					ID:        existing.ID,
					Identical: true,
				})
			}
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		}
		var pse internal.PreSaveError
		if errors.Is(err, sql.ErrNoRows) || errors.As(err, &pse) {
			// This error occurs if certain fields are changed in a Save* operation
			// that should not be changed.
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &req.ObjectType, http.StatusCreated, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), auditlog.CreateObjectType, auditlog.Payload{
		"ID":   req.ObjectType.ID,
		"Name": req.ObjectType.TypeName,
	}), nil
}

// ProtectedTypes are authz object types that cannot be deleted.
var ProtectedTypes = map[string]bool{
	authz.ObjectTypeUser:                true,
	authz.ObjectTypeGroup:               true,
	idpAuthz.ObjectTypeNameAccessPolicy: true,
	idpAuthz.ObjectTypeNameTransformer:  true,
	idpAuthz.ObjectTypeNamePolicies:     true,
}

// OpenAPI Summary: Delete Object Type
// OpenAPI Tags: Object Types
// OpenAPI Description: This endpoint deletes an object type by ID. It also deletes all objects, edge types and edges which use the object type.
func (h *handler) deleteObjectType(ctx context.Context, id uuid.UUID, _ url.Values) (int, []auditlog.Entry, error) {

	if err := h.checkEmployeeRequest(ctx); err != nil {
		return http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	tenantState := tenantstate.MustGet(ctx)

	ot, err := tenantState.Storage.GetObjectType(ctx, id)
	if err != nil {
		return http.StatusBadRequest, nil, ucerr.Wrap(err)
	}
	if _, exists := ProtectedTypes[ot.TypeName]; exists {
		return http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "Cannot delete protected object type %s", ot.TypeName)
	}

	// This also deletes all objects, edge types, & edges which use this object type
	if err := tenantState.Storage.DeleteObjectType(ctx, ot.ID); err != nil {
		return uchttp.SQLDeleteErrorMapper(err), nil, ucerr.Wrap(err)
	}

	return http.StatusNoContent, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), auditlog.DeleteObjectType, auditlog.Payload{
		"ID":   ot.ID,
		"Name": ot.TypeName,
	}), nil
}

func (h *handler) addOrganizationFilter(ctx context.Context, organizationID *string, filter string) (string, uuid.UUID, int, error) {
	var orgID uuid.UUID
	var err error
	var orgFilter string

	if organizationID != nil {
		orgID, err = uuid.FromString(*organizationID)
		if err != nil {
			return "", uuid.Nil, http.StatusBadRequest, ucerr.Wrap(err)
		}
		if _, err = h.validateOrganizationForRequest(ctx, orgID); err != nil {
			return "", uuid.Nil, http.StatusForbidden, ucerr.Wrap(err)
		}
		orgFilter = fmt.Sprintf("('organization_id',EQ,'%v')", orgID)
	} else if err = h.checkEmployeeRequest(ctx); err != nil {
		orgID, err = h.validateOrganizationForRequest(ctx, uuid.Nil)
		if err != nil {
			return "", uuid.Nil, http.StatusForbidden, ucerr.Wrap(err)
		}
		orgFilter =
			fmt.Sprintf("(('organization_id',EQ,'%v'),OR,('organization_id',EQ,'00000000-0000-0000-0000-000000000000'))",
				orgID)
	}

	if orgFilter != "" {
		if filter != "" {
			filter = fmt.Sprintf("(%s,AND,%s)", filter, orgFilter)
		} else {
			filter = orgFilter
		}
	}

	return filter, orgID, http.StatusOK, nil
}

type listEdgeTypesParams struct {
	pagination.QueryParams
	SourceObjectTypeID *string `description:"The ID of the source object type" query:"source_object_type_id"`
	TargetObjectTypeID *string `description:"The ID of the target object type" query:"target_object_type_id"`
	OrganizationID     *string `description:"The ID of the organization" query:"organization_id"`
}

// OpenAPI Summary: List Edge Types
// OpenAPI Tags: Edge Types
// OpenAPI Description: This endpoint returns a paginated list of edge types in a tenant. The list can be filtered to only include edge types with a specified organization, source object type or target object type.
func (h *handler) listEdgeTypes(ctx context.Context, req listEdgeTypesParams) (*authz.ListEdgeTypesResponse, int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	filter := ""
	if req.SourceObjectTypeID != nil {
		filter = fmt.Sprintf("('source_object_type_id',EQ,'%s')", *req.SourceObjectTypeID)
	}

	if req.TargetObjectTypeID != nil {
		targetFilter := fmt.Sprintf("('target_object_type_id',EQ,'%s')", *req.TargetObjectTypeID)
		if filter != "" {
			filter = fmt.Sprintf("(%s,AND,%s)", filter, targetFilter)
		} else {
			filter = targetFilter
		}
	}

	filter, _, httpStatus, err := h.addOrganizationFilter(ctx, req.OrganizationID, filter)
	if err != nil {
		switch httpStatus {
		case http.StatusForbidden:
			return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	var pagerOptions []pagination.Option
	if filter != "" {
		pagerOptions = append(pagerOptions, pagination.Filter(filter))
	}

	pager, err := authz.NewEdgeTypePaginatorFromQuery(req, pagerOptions...)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	types, respFields, err := tenantState.Storage.ListEdgeTypesPaginated(ctx, *pager)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &authz.ListEdgeTypesResponse{
		Data:           types,
		ResponseFields: *respFields,
	}, http.StatusOK, nil, nil
}

// OpenAPI Summary: Get Edge Type
// OpenAPI Tags: Edge Types
// OpenAPI Description: This endpoint gets an edge type by ID.
func (h *handler) getEdgeType(ctx context.Context, id uuid.UUID, _ url.Values) (*authz.EdgeType, int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	edgeType, err := tenantState.Storage.GetEdgeType(ctx, id)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	if _, err := h.validateOrganizationForRequest(ctx, edgeType.OrganizationID); err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	return edgeType, http.StatusOK, nil, nil
}

// OpenAPI Summary: Create Edge Type
// OpenAPI Tags: Edge Types
// OpenAPI Description: This endpoint creates a new edge type, complete with name, source object type and target object type. Edges of a given type can only link source objects and target objects of the specified types.
func (h *handler) createEdgeType(ctx context.Context, req authz.CreateEdgeTypeRequest) (*authz.EdgeType, int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	orgID, err := h.validateOrganizationForRequest(ctx, req.EdgeType.OrganizationID)
	if err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	req.EdgeType.OrganizationID = orgID

	if err := tenantState.Storage.InsertEdgeType(ctx, &req.EdgeType); err != nil {
		if ucdb.IsUniqueViolation(err) {
			if existing, err := tenantState.Storage.GetEdgeTypeForName(ctx, req.EdgeType.TypeName); err == nil {
				if existing.ID == req.EdgeType.ID && existing.EqualsIgnoringID(&req.EdgeType) {
					// This is for back-compat, if IfNotExists() is used it wouldn't be necessary
					return &req.EdgeType, http.StatusCreated, nil, nil
				}
				return nil, http.StatusConflict, nil, ucerr.WrapWithFriendlyStructure(nil, jsonclient.SDKStructuredError{
					Error:     "This edge type already exists",
					ID:        existing.ID,
					Identical: req.EdgeType.EqualsIgnoringID(existing),
				})
			}
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		}
		var pse internal.PreSaveError
		if errors.Is(err, sql.ErrNoRows) || errors.As(err, &pse) {
			// This error occurs if certain fields are changed in a Save* operation
			// that should not be changed.
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &req.EdgeType, http.StatusCreated, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), auditlog.CreateEdgeType, auditlog.Payload{
		"ID":                 req.EdgeType.ID,
		"Name":               req.EdgeType.TypeName,
		"SourceObjectTypeID": req.EdgeType.SourceObjectTypeID,
		"TargetObjectTypeID": req.EdgeType.TargetObjectTypeID,
		"Attributes":         req.EdgeType.Attributes,
		"OrganizationID":     req.EdgeType.OrganizationID,
	}), nil
}

// OpenAPI Summary: Update Edge Type
// OpenAPI Tags: Edge Types
// OpenAPI Description: This endpoint updates an edge type. It is used to adjust the attributes associated with the edge type.
func (h *handler) updateEdgeType(ctx context.Context, id uuid.UUID, req authz.UpdateEdgeTypeRequest) (*authz.EdgeType, int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	edgeType, err := tenantState.Storage.GetEdgeType(ctx, id)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	_, err = h.validateOrganizationForRequest(ctx, edgeType.OrganizationID)
	if err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	uclog.Debugf(ctx, "updating edge type %v: typename (%s) to (%s), attributes from (%v) to (%v)",
		edgeType.ID, edgeType.TypeName, req.TypeName, edgeType.Attributes, req.Attributes)

	edgeType.TypeName = req.TypeName
	edgeType.Attributes = req.Attributes

	// Fetch existing object and see if this is a no-op
	existingEdgeType, err := tenantState.Storage.GetEdgeType(ctx, id)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}
	if existingEdgeType.EqualsIgnoringID(edgeType) {
		uclog.Debugf(ctx, "edge type %v: no-op update", edgeType.ID)
		return edgeType, http.StatusOK, nil, nil
	}

	if err := tenantState.Storage.SaveEdgeType(ctx, edgeType); err != nil {
		var pse internal.PreSaveError
		if errors.Is(err, sql.ErrNoRows) || errors.As(err, &pse) {
			// This error occurs if certain fields are changed in a Save* operation
			// that should not be changed.
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	if err := tenantState.Storage.FlushCacheForEdgeType(ctx, id); err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return edgeType, http.StatusOK, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), auditlog.UpdateEdgeType, auditlog.Payload{
		"ID":                 edgeType.ID,
		"Name":               edgeType.TypeName,
		"SourceObjectTypeID": edgeType.SourceObjectTypeID,
		"TargetObjectTypeID": edgeType.TargetObjectTypeID,
		"Attributes":         edgeType.Attributes,
		"OrganizationID":     edgeType.OrganizationID,
	}), nil

}

// OpenAPI Summary: Delete Edge Type
// OpenAPI Tags: Edge Types
// OpenAPI Description: This endpoint deletes an edge type by ID. It also deletes all edges which use this edge type.
func (h *handler) deleteEdgeType(ctx context.Context, id uuid.UUID, _ url.Values) (int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	et, err := tenantState.Storage.GetEdgeType(ctx, id)
	if err != nil {
		return uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	if _, err := h.validateOrganizationForRequest(ctx, et.OrganizationID); err != nil {
		return http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	// This also deletes all edges which use this edge type
	if err := tenantState.Storage.DeleteEdgeType(ctx, et.ID); err != nil {
		// TODO: differentiate error types, e.g. object not found?
		return uchttp.SQLDeleteErrorMapper(err), nil, ucerr.Wrap(err)
	}

	return http.StatusNoContent, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), auditlog.DeleteEdgeType, auditlog.Payload{
		"ID":                 et.ID,
		"Name":               et.TypeName,
		"SourceObjectTypeID": et.SourceObjectTypeID,
		"TargetObjectTypeID": et.TargetObjectTypeID,
		"Attributes":         et.Attributes,
		"OrganizationID":     et.OrganizationID,
	}), nil
}

func (h *handler) migrateEdgeType(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	if err := h.checkEmployeeRequest(ctx); err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusForbidden))
		return
	}

	tenantState := tenantstate.MustGet(ctx)

	var req authz.MigrationRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	et, err := tenantState.Storage.GetEdgeType(ctx, id)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	et.OrganizationID = req.OrganizationID
	if err := tenantState.Storage.SaveEdgeType(ctx, et); err != nil {
		var pse internal.PreSaveError
		if errors.Is(err, sql.ErrNoRows) || errors.As(err, &pse) {
			// This error occurs if certain fields are changed in a Save* operation
			// that should not be changed.
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return
		}
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, et)
}

type listObjectsParams struct {
	pagination.QueryParams
	TypeID         *string `description:"Optional - allows filtering by object type ID" query:"type_id"`
	Name           *string `description:"Optional - allows filtering by object name" query:"name"`
	OrganizationID *string `description:"Optional - allows filtering by organization ID." query:"organization_id"`
}

func (h *handler) getObjectByTypeAndName(ctx context.Context, req listObjectsParams) (*authz.ListObjectsResponse, int, error) {
	// TODO: this should be a separate endpoint rather than trying to fold into our list objects API
	tenantState := tenantstate.MustGet(ctx)

	if req.TypeID == nil || req.Name == nil {
		return nil, http.StatusInternalServerError, ucerr.New("getObjectByTypeAndName expects specified type and name")
	}

	pager, err := authz.NewObjectPaginatorFromQuery(req)
	if err != nil {
		return nil, http.StatusBadRequest, ucerr.Wrap(err)
	}

	typeID, err := uuid.FromString(*req.TypeID)
	if err != nil {
		return nil, http.StatusBadRequest, ucerr.Friendlyf(err, "invalid type_id: '%v'", *req.TypeID)
	}

	if *req.Name == "" {
		return nil, http.StatusBadRequest, ucerr.Friendlyf(err, "name cannot be empty")
	}

	_, orgID, httpStatus, err := h.addOrganizationFilter(ctx, req.OrganizationID, "")
	if err != nil {
		return nil, httpStatus, ucerr.Wrap(err)
	}

	var objs []authz.Object
	obj, err := tenantState.Storage.GetObjectForAlias(ctx, typeID, *req.Name, orgID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, http.StatusInternalServerError, ucerr.Wrap(err)
		}
	} else {
		objs = append(objs, *obj)
	}

	data, respFields := pagination.ProcessResults(
		objs,
		pager.GetCursor(),
		pager.GetLimit(),
		pager.IsForward(),
		pager.GetSortKey(),
	)
	return &authz.ListObjectsResponse{
			Data:           data,
			ResponseFields: respFields,
		},
		http.StatusOK,
		nil
}

// OpenAPI Summary: List Objects
// OpenAPI Tags: Objects
// OpenAPI Description: This endpoint returns a paginated list of objects in a tenant. The list can be filtered to only include objects with a specified type, name or organization.
func (h *handler) listObjects(ctx context.Context, req listObjectsParams) (*authz.ListObjectsResponse, int, []auditlog.Entry, error) {
	if req.TypeID != nil && req.Name != nil {
		resp, httpStatus, err := h.getObjectByTypeAndName(ctx, req)
		if err != nil {
			switch httpStatus {
			case http.StatusForbidden:
				return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
			case http.StatusBadRequest:
				return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
			default:
				return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
			}
		}
		return resp, http.StatusOK, nil, nil
	}

	tenantState := tenantstate.MustGet(ctx)

	filter := ""

	if req.TypeID != nil {
		filter = fmt.Sprintf("('type_id',EQ,'%s')", *req.TypeID)
	}

	if req.Name != nil {
		nameFilter := fmt.Sprintf("('alias',EQ,'%s')", *req.Name)
		if filter == "" {
			filter = nameFilter
		} else {
			filter = fmt.Sprintf("(%s,AND,%s)", filter, nameFilter)
		}
	}

	filter, _, httpStatus, err := h.addOrganizationFilter(ctx, req.OrganizationID, filter)
	if err != nil {
		switch httpStatus {
		case http.StatusForbidden:
			return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	var pagerOptions []pagination.Option
	if filter != "" {
		pagerOptions = append(pagerOptions, pagination.Filter(filter))
	}

	pager, err := authz.NewObjectPaginatorFromQuery(req, pagerOptions...)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	objs, respFields, err := tenantState.Storage.ListObjectsPaginated(ctx, *pager)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &authz.ListObjectsResponse{
			Data:           objs,
			ResponseFields: *respFields,
		},
		http.StatusOK,
		nil, nil
}

// OpenAPI Summary: Get Object
// OpenAPI Tags: Objects
// OpenAPI Description: This endpoint gets an object by ID. If the ID provided is that of a User in the IDP, it returns an object representing the user.
func (h *handler) getObject(ctx context.Context, id uuid.UUID, _ url.Values) (*authz.Object, int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	object, err := tenantState.Storage.GetObject(ctx, id)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	_, err = h.validateOrganizationForRequest(ctx, object.OrganizationID)
	if err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	return object, http.StatusOK, nil, nil
}

// OpenAPI Summary: Create Object
// OpenAPI Tags: Objects
// OpenAPI Description: This endpoint creates an object with a given ID, Type ID, and Alias.
func (h *handler) createObject(ctx context.Context, req authz.CreateObjectRequest) (*authz.Object, int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	orgID, err := h.validateOrganizationForRequest(ctx, req.Object.OrganizationID)
	if err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	if !tenantState.UseOrganizations {
		if req.Object.TypeID == authz.UserObjectTypeID || req.Object.TypeID == authz.LoginAppObjectTypeID {
			// users and login apps must always have non-nil org, even on tenants w/o orgs enabled
			req.Object.OrganizationID = tenantState.CompanyID
		} else {
			req.Object.OrganizationID = uuid.Nil
		}
	} else {
		if orgID.IsNil() && (req.Object.TypeID == authz.UserObjectTypeID || req.Object.TypeID == authz.LoginAppObjectTypeID) {
			return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "cannot create this object %v with nil org", req.Object)
		}

		req.Object.OrganizationID = orgID
	}

	if err := tenantState.Storage.InsertObject(ctx, &req.Object); err != nil {
		if ucdb.IsUniqueViolation(err) {
			if req.Object.Alias != nil {
				if existing, err := tenantState.Storage.GetObjectForAlias(ctx, req.Object.TypeID, *req.Object.Alias, orgID); err == nil {
					if existing.ID == req.Object.ID && existing.EqualsIgnoringID(&req.Object) {
						// This is for back-compat, if IfNotExists() is used it wouldn't be necessary
						return &req.Object, http.StatusCreated, nil, nil
					}
					return nil, http.StatusConflict, nil, ucerr.WrapWithFriendlyStructure(nil, jsonclient.SDKStructuredError{
						Error:     "This object already exists",
						ID:        existing.ID,
						Identical: req.Object.EqualsIgnoringID(existing),
					})
				}
			} else if existing, err := tenantState.Storage.GetObject(ctx, req.Object.ID); err == nil && existing.EqualsIgnoringID(&req.Object) {
				// This is for back-compat, if IfNotExists() is used it wouldn't be necessary
				return &req.Object, http.StatusCreated, nil, nil
			}
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		}

		var pse internal.PreSaveError
		if errors.Is(err, sql.ErrNoRows) || errors.As(err, &pse) {
			// This error occurs if certain fields are changed in a Save* operation
			// that should not be changed.
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &req.Object, http.StatusCreated, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), auditlog.CreateObject, auditlog.Payload{
		"ID":             req.Object.ID,
		"Name":           "Object",
		"TypeID":         req.Object.TypeID,
		"Alias":          req.Object.Alias,
		"OrganizationID": req.Object.OrganizationID,
	}), nil
}

func (h *handler) updateObject(ctx context.Context, id uuid.UUID, req authz.UpdateObjectRequest) (*authz.Object, int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	object, err := tenantState.Storage.GetObject(ctx, id)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	_, err = h.validateOrganizationForRequest(ctx, object.OrganizationID)
	if err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	if object.TypeID == authz.UserObjectTypeID {
		if req.Source == nil || *req.Source != "idp" {
			return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "user objects can only be updated by idp")
		}
	}

	curAlias := ""
	newAlias := ""
	if object.Alias != nil {
		curAlias = *object.Alias
	}
	if req.Alias != nil {
		newAlias = *req.Alias
	}
	uclog.Debugf(ctx, "updating object %v: alias (%s) to (%s)", object.ID, curAlias, newAlias)
	object.Alias = req.Alias

	if err := tenantState.Storage.SaveObject(ctx, object); err != nil {
		var pse internal.PreSaveError
		if errors.Is(err, sql.ErrNoRows) || errors.As(err, &pse) {
			// This error occurs if certain fields are changed in a Save* operation
			// that should not be changed.
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}
		return nil, uchttp.SQLWriteErrorMapper(err), nil, ucerr.Wrap(err)
	}

	entries := auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), auditlog.UpdateObject, auditlog.Payload{
		"ID":             object.ID,
		"TypeID":         object.TypeID,
		"Alias":          newAlias,
		"OrganizationID": object.OrganizationID,
	})

	return object, http.StatusOK, entries, nil
}

// OpenAPI Summary: Delete Object
// OpenAPI Tags: Objects
// OpenAPI Description: This endpoint deletes an object by ID. This also deletes all edges that use that object.
func (h *handler) deleteObject(ctx context.Context, id uuid.UUID, _ url.Values) (int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	obj, err := tenantState.Storage.GetObject(ctx, id)
	if err != nil {
		return uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	_, err = h.validateOrganizationForRequest(ctx, obj.OrganizationID)
	if err != nil {
		return http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	// This also deletes all edges.
	if err := tenantState.Storage.DeleteObject(ctx, obj.ID); err != nil {
		// TODO: differentiate error types, e.g. object not found?
		return uchttp.SQLDeleteErrorMapper(err), nil, ucerr.Wrap(err)
	}

	return http.StatusNoContent, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), auditlog.DeleteObject, auditlog.Payload{
		"ID":             obj.ID,
		"Name":           "Object",
		"TypeID":         obj.TypeID,
		"Alias":          obj.Alias,
		"OrganizationID": obj.OrganizationID,
	}), nil
}

func (h *handler) migrateObject(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	if err := h.checkEmployeeRequest(ctx); err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusForbidden))
		return
	}

	tenantState := tenantstate.MustGet(ctx)

	var req authz.MigrationRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	obj, err := tenantState.Storage.GetObject(ctx, id)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	obj.OrganizationID = req.OrganizationID
	if err := tenantState.Storage.SaveObject(ctx, obj); err != nil {
		var pse internal.PreSaveError
		if errors.Is(err, sql.ErrNoRows) || errors.As(err, &pse) {
			// This error occurs if certain fields are changed in a Save* operation
			// that should not be changed.
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return
		}

		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, obj)
}

// ListEdgesOnObjectParams are the parameters for listing edges API.
type ListEdgesOnObjectParams struct {
	pagination.QueryParams
	TargetObjectID *string `description:"The object ID to list edges on" query:"target_object_id"`
}

// OpenAPI Summary: List Edges on Object
// OpenAPI Tags: Edges
// OpenAPI Description: This endpoint returns a paginated list of edges associated with an object, which is specified by ID. The endpoint lists all incoming and outgoing edges (i.e. all edges where the provided object is a source or target).
func (h *handler) listEdgesOnObject(ctx context.Context, objectID uuid.UUID, req ListEdgesOnObjectParams) (*authz.ListEdgesResponse, int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	obj, err := tenantState.Storage.GetObject(ctx, objectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		}
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	_, err = h.validateOrganizationForRequest(ctx, obj.OrganizationID)
	if err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	targetObjectID := uuid.Nil
	if req.TargetObjectID != nil {
		targetObjectID = uuid.FromStringOrNil(*req.TargetObjectID)
	}

	filterString := ""
	if targetObjectID != uuid.Nil {
		filterString = fmt.Sprintf("(('source_object_id',EQ,'%v'),AND,('target_object_id',EQ,'%v'))", objectID, targetObjectID)
	} else {
		filterString = fmt.Sprintf("(('source_object_id',EQ,'%v'),OR,('target_object_id',EQ,'%v'))", objectID, objectID)
	}

	pager, err := authz.NewEdgePaginatorFromQuery(req, pagination.Filter(filterString))
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	edges, respFields, err := tenantState.Storage.ListEdgesPaginatedAndUpdateCollectionCache(ctx, *pager, objectID, targetObjectID)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	return &authz.ListEdgesResponse{
		Data:           edges,
		ResponseFields: *respFields,
	}, http.StatusOK, nil, nil
}

type listEdgesParams struct {
	pagination.QueryParams
	SourceObjectID *string `description:"Optional - allows filtering by source object ID" query:"source_object_id"`
	TargetObjectID *string `description:"Optional - allows filtering by target object ID" query:"target_object_id"`
	EdgeTypeID     *string `description:"Optional - allows filtering by edge ID" query:"edge_type_id"`
}

// OpenAPI Summary: List Edges
// OpenAPI Tags: Edges
// OpenAPI Description: This endpoint returns a paginated list of all edges in a tenant. The list can be filtered to only include edges with a specified organization, source object or target object.
func (h *handler) listEdges(ctx context.Context, req listEdgesParams) (*authz.ListEdgesResponse, int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	edgeTypeID := uuid.Nil
	if req.EdgeTypeID != nil {
		edgeTypeID = uuid.FromStringOrNil(*req.EdgeTypeID)
	}

	if edgeTypeID != uuid.Nil {
		sourceObjectID := uuid.Nil
		if req.SourceObjectID != nil {
			sourceObjectID = uuid.FromStringOrNil(*req.SourceObjectID)
		}

		targetObjectID := uuid.Nil
		if req.TargetObjectID != nil {
			targetObjectID = uuid.FromStringOrNil(*req.TargetObjectID)
		}

		edge, err := tenantState.Storage.FindEdge(ctx, edgeTypeID, sourceObjectID, targetObjectID)
		if err != nil {
			return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}
		return &authz.ListEdgesResponse{Data: []authz.Edge{*edge}}, http.StatusOK, nil, nil
	}

	pager, err := authz.NewEdgePaginatorFromQuery(req)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	edges, respFields, err := tenantState.Storage.ListEdgesPaginated(ctx, *pager)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &authz.ListEdgesResponse{
		Data:           edges,
		ResponseFields: *respFields,
	}, http.StatusOK, nil, nil
}

// OpenAPI Summary: Get Edge
// OpenAPI Tags: Edges
// OpenAPI Description: This endpoint gets an edge by ID.
func (h *handler) getEdge(ctx context.Context, id uuid.UUID, _ url.Values) (*authz.Edge, int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	edge, err := tenantState.Storage.GetEdge(ctx, id)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	if err := h.validateTwoObjectOrganizations(ctx, edge.SourceObjectID, edge.TargetObjectID, false); err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	return edge, http.StatusOK, nil, nil
}

// OpenAPI Summary: Create Edge
// OpenAPI Tags: Edges
// OpenAPI Description: This endpoint creates a directed edge of a given type between a source object and target object, both of which are specified by ID.
func (h *handler) createEdge(ctx context.Context, req authz.CreateEdgeRequest) (*authz.Edge, int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	if err := h.validateTwoObjectOrganizations(ctx, req.Edge.SourceObjectID, req.Edge.TargetObjectID, true); err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	if err := tenantState.Storage.InsertEdge(ctx, &req.Edge); err != nil {
		if ucdb.IsUniqueViolation(err) {
			if existing, err := tenantState.Storage.FindEdge(ctx, req.Edge.EdgeTypeID, req.Edge.SourceObjectID, req.Edge.TargetObjectID); err == nil {
				if existing.ID == req.Edge.ID && existing.EqualsIgnoringID(&req.Edge) {
					// This is for back-compat, if IfNotExists() is used it wouldn't be necessary
					return &req.Edge, http.StatusCreated, nil, nil
				}
				return nil, http.StatusConflict, nil, ucerr.WrapWithFriendlyStructure(nil, jsonclient.SDKStructuredError{
					Error:     "This edge already exists",
					ID:        existing.ID,
					Identical: true,
				})
			}
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		}
		var pse internal.PreSaveError
		if errors.Is(err, sql.ErrNoRows) || errors.As(err, &pse) {
			// This error occurs if certain fields are changed in a Save* operation
			// that should not be changed.
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &req.Edge, http.StatusCreated, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), auditlog.CreateEdge, auditlog.Payload{
		"ID":             req.Edge.ID,
		"Name":           "Edge",
		"TypeID":         req.Edge.EdgeTypeID,
		"SourceObjectID": req.Edge.SourceObjectID,
		"TargetObjectID": req.Edge.TargetObjectID,
	}), nil
}

// OpenAPI Summary: Delete Edges on Object
// OpenAPI Tags: Edges
// OpenAPI Description: This endpoint deletes all edges associated with an object (specified by ID).
func (h *handler) deleteAllEdgesOnObject(ctx context.Context, objectID uuid.UUID, _ url.Values) (int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	obj, err := tenantState.Storage.GetObject(ctx, objectID)
	if err != nil {
		return uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	_, err = h.validateOrganizationForRequest(ctx, obj.OrganizationID)
	if err != nil {
		return http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	if err := tenantState.Storage.DeleteEdgesFromObject(ctx, objectID); err != nil {
		// TODO: differentiate error types, e.g. object not found?
		return uchttp.SQLDeleteErrorMapper(err), nil, ucerr.Wrap(err)
	}

	return http.StatusNoContent, nil, nil
}

// OpenAPI Summary: Delete Edge
// OpenAPI Tags: Edges
// OpenAPI Description: This endpoint deletes an edge by ID.
func (h *handler) deleteEdge(ctx context.Context, id uuid.UUID, _ url.Values) (int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	e, err := tenantState.Storage.GetEdge(ctx, id)
	if err != nil {
		return uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}
	if err := h.validateTwoObjectOrganizations(ctx, e.SourceObjectID, e.TargetObjectID, false); err != nil {
		return http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	if err := tenantState.Storage.DeleteEdge(ctx, e.ID); err != nil {
		return uchttp.SQLDeleteErrorMapper(err), nil, ucerr.Wrap(err)
	}

	return http.StatusNoContent, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), auditlog.DeleteEdge, auditlog.Payload{
		"ID":             e.ID,
		"Name":           "Edge",
		"TypeID":         e.EdgeTypeID,
		"SourceObjectID": e.SourceObjectID,
		"TargetObjectID": e.TargetObjectID,
	}), nil
}

// CheckAttributeParams are the parameters for checking attribute API.
type CheckAttributeParams struct {
	SourceObjectID *string `description:"The object for which permissions are to be checked" query:"source_object_id"`
	TargetObjectID *string `description:"The object on which permissions are to be checked" query:"target_object_id"`
	Attribute      *string `description:"The permission to check" query:"attribute"`
}

// OpenAPI Summary: Check Attribute
// OpenAPI Tags: Permissions
// OpenAPI Description: This endpoint receives a source object ID, target object ID and attribute. It returns a boolean indicating whether the source object has the attribute permission on the target object.
func (h *handler) checkAttribute(ctx context.Context, req CheckAttributeParams) (*authz.CheckAttributeResponse, int, []auditlog.Entry, error) {
	if err := h.ensureTenantMember(ctx, false); err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}
	tenantState := tenantstate.MustGet(ctx)

	if req.SourceObjectID == nil || req.TargetObjectID == nil || req.Attribute == nil || *req.Attribute == "" {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "missing required query parameter")
	}

	sourceObjectID, err := uuid.FromString(*req.SourceObjectID)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}
	targetObjectID, err := uuid.FromString(*req.TargetObjectID)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}
	if err := h.validateTwoObjectOrganizations(ctx, sourceObjectID, targetObjectID, false); err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	found, path, err := tenantState.Storage.CheckAttribute(ctx, h.checkAttributeServiceNameMap[tenantState.TenantID], sourceObjectID, targetObjectID, *req.Attribute)
	if err != nil {
		if ucdb.IsTransactionConflict(err) {
			return nil, http.StatusConflict, nil, ucerr.WrapWithFriendlyStructure(nil, jsonclient.SDKStructuredError{
				Error: "Conflict with write/delete operations in another region. Please retry the call",
			})
		}
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &authz.CheckAttributeResponse{
		HasAttribute: found,
		Path:         path,
	}, http.StatusOK, nil, nil
}

type listAttributesParams struct {
	SourceObjectID *string `description:"Optional - allows filtering to a particular source object ID" query:"source_object_id"`
	TargetObjectID *string `description:"Optional - allows filtering to a particular target object ID" query:"target_object_id"`
}

// OpenAPI Summary: List Attributes
// OpenAPI Tags: Permissions
// OpenAPI Description: This endpoint receives a source object ID and target object ID. It returns a list of attributes that the source object has on the target object.
func (h *handler) listAttributes(ctx context.Context, req listAttributesParams) ([]string, int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	if req.SourceObjectID == nil || req.TargetObjectID == nil {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "missing required query parameter")
	}

	sourceObjectID, err := uuid.FromString(*req.SourceObjectID)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}
	targetObjectID, err := uuid.FromString(*req.TargetObjectID)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}
	if err := h.validateTwoObjectOrganizations(ctx, sourceObjectID, targetObjectID, false); err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	candidateAttributes := map[string]bool{}

	edgeTypeCache := make(map[uuid.UUID]*authz.EdgeType)

	// Get attribute names from outgoing edges from the sourceObject, adding them to the candidateAttributes map with value false
	pagerSource, err := authz.NewEdgePaginatorFromOptions(
		pagination.Filter(fmt.Sprintf("('source_object_id',EQ,'%v')", sourceObjectID)),
		pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}
	for {
		edges, respFields, err := tenantState.Storage.ListEdgesPaginated(ctx, *pagerSource)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
		for _, edge := range edges {
			edgeType, ok := edgeTypeCache[edge.EdgeTypeID]
			if !ok {
				edgeType, err = tenantState.Storage.GetEdgeType(ctx, edge.EdgeTypeID)
				if err != nil {
					return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
				}
				edgeTypeCache[edge.EdgeTypeID] = edgeType
			}
			for _, attr := range edgeType.Attributes {
				if attr.Direct || attr.Inherit {
					candidateAttributes[attr.Name] = false
				}
			}
		}
		if !pagerSource.AdvanceCursor(*respFields) {
			break
		}
	}

	// Get attribute names from incoming edges to the targetObject, setting the ones that are already in candidateAttributes to true
	pagerTarget, err := authz.NewEdgePaginatorFromOptions(
		pagination.Filter(fmt.Sprintf("('target_object_id',EQ,'%v')", targetObjectID)),
		pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}
	for {
		edges, respFields, err := tenantState.Storage.ListEdgesPaginated(ctx, *pagerTarget)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
		for _, edge := range edges {
			edgeType, ok := edgeTypeCache[edge.EdgeTypeID]
			if !ok {
				edgeType, err = tenantState.Storage.GetEdgeType(ctx, edge.EdgeTypeID)
				if err != nil {
					return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
				}
				edgeTypeCache[edge.EdgeTypeID] = edgeType
			}
			for _, attr := range edgeType.Attributes {
				if attr.Direct || attr.Propagate {
					if _, ok := candidateAttributes[attr.Name]; ok {
						candidateAttributes[attr.Name] = true
					}
				}
			}
		}
		if !pagerTarget.AdvanceCursor(*respFields) {
			break
		}
	}

	// Look up all attributes that are candidates and store ones that are found in attributeNames
	attributeNames := []string{}
	for attrName, lookup := range candidateAttributes {
		if lookup {
			found, _, err := tenantState.Storage.CheckAttribute(ctx, h.checkAttributeServiceNameMap[tenantState.TenantID], sourceObjectID, targetObjectID, attrName)
			if err != nil {
				return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
			}
			if found {
				attributeNames = append(attributeNames, attrName)
			}
		}
	}

	return attributeNames, http.StatusOK, nil, nil
}

type listObjectsReachableWithAttributeParams struct {
	SourceObjectID     *string `description:"The source object from which you are searching for reachable target objects" query:"source_object_id"`
	TargetObjectTypeID *string `description:"Optional - allows filtering to a particular target object type" query:"target_object_type_id"`
	Attribute          *string `description:"The permission through which target objects are considered reachable from the source object" query:"attribute"`
}

// OpenAPI Summary: List Objects Reachable with Attribute
// OpenAPI Tags: Permissions
// OpenAPI Description: This endpoint receives a source object ID and attribute. It returns a list of objects reachable from the source object with the attribute.
func (h *handler) listObjectsReachableWithAttribute(ctx context.Context, req listObjectsReachableWithAttributeParams) (*authz.ListObjectsReachableWithAttributeResponse, int, []auditlog.Entry, error) {

	if err := h.ensureTenantMember(ctx, false); err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	tenantState := tenantstate.MustGet(ctx)

	if req.SourceObjectID == nil || req.TargetObjectTypeID == nil || req.Attribute == nil || *req.Attribute == "" {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "missing required query parameter")
	}

	sourceObjectID, err := uuid.FromString(*req.SourceObjectID)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	object, err := tenantState.Storage.GetObject(ctx, sourceObjectID)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	if _, err := h.validateOrganizationForRequest(ctx, object.OrganizationID); err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	targetObjectTypeID, err := uuid.FromString(*req.TargetObjectTypeID)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	attrName := *req.Attribute

	objectIDs, err := internal.ListObjectsReachableWithAttributeBFS(ctx, tenantState.Storage, tenantState.TenantID, sourceObjectID, targetObjectTypeID, attrName)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &authz.ListObjectsReachableWithAttributeResponse{Data: objectIDs}, http.StatusOK, nil, nil
}

// OpenAPI Summary: Create Organization
// OpenAPI Tags: Organizations
// OpenAPI Description: This endpoint creates an organization.
func (h *handler) createOrganization(ctx context.Context, req authz.CreateOrganizationRequest) (*authz.Organization, int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	if !tenantState.UseOrganizations {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "organization creation is disabled")
	}

	if err := h.checkEmployeeRequest(ctx); err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	if err := tenantState.Storage.InsertOrganization(ctx, &req.Organization); err != nil {
		if ucdb.IsUniqueViolation(err) {
			if existing, err := tenantState.Storage.GetOrganizationForName(ctx, req.Organization.Name); err == nil &&
				existing.Region == req.Organization.Region {
				return nil, http.StatusConflict, nil, ucerr.WrapWithFriendlyStructure(nil, jsonclient.SDKStructuredError{
					Error:     "organization already exists",
					ID:        existing.ID,
					Identical: true,
				})
			}
			// most likely caused by mismatched regions
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		}
		var pse internal.PreSaveError
		if errors.Is(err, sql.ErrNoRows) || errors.As(err, &pse) {
			// This error occurs if certain fields are changed in a Save* operation
			// that should not be changed.
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	entries := auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), auditlog.CreateOrganization, auditlog.Payload{
		"ID":     req.Organization.ID,
		"Name":   req.Organization.Name,
		"Region": req.Organization.Region,
	})

	// Not using apiclient.NewAuthzClient due to TODO below
	authzClient, err := authz.NewClient(tenantState.TenantURL.String(), authz.PassthroughAuthorization(), authz.JSONClient(security.PassXForwardedFor()))
	if err != nil {
		return nil, http.StatusInternalServerError, entries, ucerr.Wrap(err)
	}

	// create the login app for this organization
	// TODO (sgarrity 6/23): this logic is duplicated in provisioning because we can't assume
	// that the authz service is up & running to make these calls too ... need a better approach to provisioning
	app, err := manager.NewLoginAppForOrganization(ctx, tenantState.TenantID, req.Organization.Name, req.Organization.ID)
	if err != nil {
		return nil, http.StatusInternalServerError, entries, ucerr.Wrap(err)
	}
	ts := multitenant.MustGetTenantState(ctx)
	mgr := manager.NewFromDB(ts.TenantDB, ts.CacheConfig)
	if err := mgr.AddLoginApp(ctx, tenantState.TenantID, authzClient, *app); err != nil {
		return nil, http.StatusInternalServerError, entries, ucerr.Wrap(err)
	}

	return &req.Organization, http.StatusCreated, entries, nil
}

// OpenAPI Summary: Update Organization
// OpenAPI Tags: Organizations
// OpenAPI Description: This endpoint updates an organization, specified by ID.
func (h *handler) updateOrganization(ctx context.Context, id uuid.UUID, req authz.UpdateOrganizationRequest) (*authz.Organization, int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	if !tenantState.UseOrganizations {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "organization updates are disabled")
	}

	if err := h.checkEmployeeRequest(ctx); err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	organization, err := tenantState.Storage.GetOrganization(ctx, id)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	organization.Name = req.Name
	organization.Region = req.Region
	if err := tenantState.Storage.SaveOrganization(ctx, organization); err != nil {
		if ucdb.IsUniqueViolation(err) {
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		}
		var pse internal.PreSaveError
		if errors.Is(err, sql.ErrNoRows) || errors.As(err, &pse) {
			// This error occurs if certain fields are changed in a Save* operation
			// that should not be changed.
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}

		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return organization, http.StatusOK, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), auditlog.UpdateOrganization, auditlog.Payload{
		"ID":     id,
		"Name":   req.Name,
		"Region": req.Region,
	}), nil
}

// OpenAPI Summary: Delete Organization
// OpenAPI Tags: Organizations
// OpenAPI Description: This endpoint deletes an organization by ID.
func (h *handler) deleteOrganization(ctx context.Context, id uuid.UUID, _ url.Values) (int, []auditlog.Entry, error) {
	if err := h.checkEmployeeRequest(ctx); err != nil {
		return http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	return http.StatusNotImplemented, nil, ucerr.Friendlyf(nil, "deleting organizations is not yet supported")
}

type listOrganizationsParams struct {
	pagination.QueryParams
}

// OpenAPI Summary: List Organizations
// OpenAPI Tags: Organizations
// OpenAPI Description: This endpoint returns a paginated list of all organizations in a tenant.
func (h *handler) listOrganizations(ctx context.Context, req listOrganizationsParams) (*authz.ListOrganizationsResponse, int, []auditlog.Entry, error) {

	if err := h.checkEmployeeRequest(ctx); err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	tenantState := tenantstate.MustGet(ctx)

	pager, err := authz.NewOrganizationPaginatorFromQuery(req)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	orgs, respFields, err := tenantState.Storage.ListOrganizationsPaginated(ctx, *pager)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return &authz.ListOrganizationsResponse{
		Data:           orgs,
		ResponseFields: *respFields,
	}, http.StatusOK, nil, nil
}

// OpenAPI Summary: Get Organization
// OpenAPI Tags: Organizations
// OpenAPI Description: This endpoint gets an organization by ID.
func (h *handler) getOrganization(ctx context.Context, id uuid.UUID, _ url.Values) (*authz.Organization, int, []auditlog.Entry, error) {
	tenantState := tenantstate.MustGet(ctx)

	if !tenantState.UseOrganizations {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "organizations are disabled")
	}

	typ, err := tenantState.Storage.GetOrganization(ctx, id)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	return typ, http.StatusOK, nil, nil
}

func (h *handler) validateOrganizationForRequest(ctx context.Context, organizationID uuid.UUID) (creationOrgID uuid.UUID, err error) {
	tenantState := tenantstate.MustGet(ctx)
	if !tenantState.UseOrganizations {
		return uuid.Nil, nil
	}

	tokenOrgID := auth.GetOrganizationUUID(ctx)

	//
	// Try to validate organization from the token
	//
	if tokenOrgID == tenantState.CompanyID || (organizationID != uuid.Nil && tokenOrgID == organizationID) {
		return organizationID, nil
	}

	//
	// We were unable to validate organization from the token, let's get the organization ID from the subject
	//
	subjID := auth.GetSubjectUUID(ctx)
	if subjID.IsNil() {
		uclog.Errorf(ctx, "no subject ID passed in JWT, this should only happen in tests")
		return organizationID, nil
	}

	subjObject, err := tenantState.Storage.GetObject(ctx, subjID)
	if err != nil || subjObject.OrganizationID.IsNil() {
		return uuid.Nil, ucerr.Friendlyf(err, "could not get valid organization %s for subject %s", organizationID, subjID)
	}
	subjOrgID := subjObject.OrganizationID

	if subjOrgID != tokenOrgID {
		// We got a different org ID by reading from the db than we did from the token, log a warning
		uclog.Warningf(ctx, "organization ID from token (%s) does not match organization ID from subject (%s), this should only happen for UC employees", tokenOrgID, subjOrgID)

		// Re-do the check from above using subjOrgID
		if subjOrgID == tenantState.CompanyID || (organizationID != uuid.Nil && subjOrgID == organizationID) {
			return organizationID, nil
		}
	}

	// If the passed in organization ID is for the nil org, use the subject's org ID for creation
	if organizationID.IsNil() {
		return subjOrgID, nil
	}

	friendlyErr := ucerr.Friendlyf(nil, `requested organizationID %s does not match JWT subject's organizationID %s`, organizationID, subjOrgID)

	// Try to get the names of the organizations to include in the error message
	if org, err := tenantState.Storage.GetOrganization(ctx, organizationID); err == nil {
		if subjOrg, err := tenantState.Storage.GetOrganization(ctx, subjOrgID); err == nil {
			friendlyErr = ucerr.Friendlyf(nil, `requested organization "%s" (%s) does not match JWT subject's organization "%s" (%s)`, org.Name, org.ID, subjOrg.Name, subjOrg.ID)
		}
	}
	return uuid.Nil, ucerr.Wrap(friendlyErr)
}

// This function validates the organizations for two objects for the incoming request. If "both" is set to true, it validates the organization on both objects.
// If "both" is set to false, the function returns nil if either object passes validation.
func (h *handler) validateTwoObjectOrganizations(ctx context.Context, objectID1, objectID2 uuid.UUID, both bool) error {
	tenantState := tenantstate.MustGet(ctx)
	if !tenantState.UseOrganizations {
		return nil
	}

	obj1, err := tenantState.Storage.GetObject(ctx, objectID1)
	if err != nil {
		return ucerr.Wrap(err)
	}

	_, err = h.validateOrganizationForRequest(ctx, obj1.OrganizationID)
	if err != nil {
		if both {
			return ucerr.Wrap(err)
		}
	} else if !both || objectID1 == objectID2 {
		return nil
	}

	obj2, err := tenantState.Storage.GetObject(ctx, objectID2)
	if err != nil {
		return ucerr.Wrap(err)
	}

	_, err = h.validateOrganizationForRequest(ctx, obj2.OrganizationID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

func (h *handler) checkEmployeeRequest(ctx context.Context) error {
	tenantState := tenantstate.MustGet(ctx)
	if !tenantState.UseOrganizations {
		return nil
	}

	if tokenOrgID := auth.GetOrganizationUUID(ctx); tokenOrgID == tenantState.CompanyID {
		return nil
	}

	// fallback to looking up organization ID from the objects table using on subject ID
	subjID := auth.GetSubjectUUID(ctx)
	if subjID.IsNil() {
		uclog.Errorf(ctx, "no subject ID passed in JWT, this should only happen in tests")
		return nil
	}

	subjObject, err := tenantState.Storage.GetObject(ctx, subjID)
	if err != nil {
		return ucerr.Friendlyf(err, "could not validate fetch organization for subject %s", subjID)
	}
	if subjObject.OrganizationID == tenantState.CompanyID {
		return nil
	}

	return ucerr.Friendlyf(nil, "insufficient permissions to perform this action")
}
