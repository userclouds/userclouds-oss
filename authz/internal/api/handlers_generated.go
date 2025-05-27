// NOTE: automatically generated file -- DO NOT EDIT

package api

import (
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/internal/auditlog"
)

func handlerBuilder(builder *builder.HandlerBuilder, h *handler) {

	builder.CollectionHandler("/edges").
		GetOne(h.getEdgeGenerated).
		Post(h.createEdgeGenerated).
		Delete(h.deleteEdgeGenerated).
		GetAll(h.listEdgesGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	builder.CollectionHandler("/edgetypes").
		GetOne(h.getEdgeTypeGenerated).
		Post(h.createEdgeTypeGenerated).
		Put(h.updateEdgeTypeGenerated).
		Delete(h.deleteEdgeTypeGenerated).
		GetAll(h.listEdgeTypesGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	handlerObject := builder.CollectionHandler("/objects").
		GetOne(h.getObjectGenerated).
		Post(h.createObjectGenerated).
		Put(h.updateObjectGenerated).
		Delete(h.deleteObjectGenerated).
		GetAll(h.listObjectsGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	builder.CollectionHandler("/objecttypes").
		GetOne(h.getObjectTypeGenerated).
		Post(h.createObjectTypeGenerated).
		Delete(h.deleteObjectTypeGenerated).
		GetAll(h.listObjectTypesGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	builder.CollectionHandler("/organizations").
		GetOne(h.getOrganizationGenerated).
		Post(h.createOrganizationGenerated).
		Put(h.updateOrganizationGenerated).
		Delete(h.deleteOrganizationGenerated).
		GetAll(h.listOrganizationsGenerated).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	handlerObject.NestedCollectionHandler("/edges").
		GetAll(h.listEdgesOnObjectGenerated).
		DeleteAll(h.deleteAllEdgesOnObjectGenerated).
		WithAuthorizer(h.newNestedRoleBasedAuthorizer())

	builder.MethodHandler("/checkattribute").Get(h.checkAttributeGenerated)

	builder.MethodHandler("/listattributes").Get(h.listAttributesGenerated)

	builder.MethodHandler("/listobjectsreachablewithattribute").Get(h.listObjectsReachableWithAttributeGenerated)

}

func (h *handler) checkAttributeGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := CheckAttributeParams{}
	if urlValues.Has("attribute") && urlValues.Get("attribute") != "null" {
		v := urlValues.Get("attribute")
		req.Attribute = &v
	}
	if urlValues.Has("source_object_id") && urlValues.Get("source_object_id") != "null" {
		v := urlValues.Get("source_object_id")
		req.SourceObjectID = &v
	}
	if urlValues.Has("target_object_id") && urlValues.Get("target_object_id") != "null" {
		v := urlValues.Get("target_object_id")
		req.TargetObjectID = &v
	}

	var res *authz.CheckAttributeResponse
	res, code, entries, err := h.checkAttribute(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listAttributesGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listAttributesParams{}
	if urlValues.Has("source_object_id") && urlValues.Get("source_object_id") != "null" {
		v := urlValues.Get("source_object_id")
		req.SourceObjectID = &v
	}
	if urlValues.Has("target_object_id") && urlValues.Get("target_object_id") != "null" {
		v := urlValues.Get("target_object_id")
		req.TargetObjectID = &v
	}

	var res []string
	res, code, entries, err := h.listAttributes(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listObjectsReachableWithAttributeGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listObjectsReachableWithAttributeParams{}
	if urlValues.Has("attribute") && urlValues.Get("attribute") != "null" {
		v := urlValues.Get("attribute")
		req.Attribute = &v
	}
	if urlValues.Has("source_object_id") && urlValues.Get("source_object_id") != "null" {
		v := urlValues.Get("source_object_id")
		req.SourceObjectID = &v
	}
	if urlValues.Has("target_object_type_id") && urlValues.Get("target_object_type_id") != "null" {
		v := urlValues.Get("target_object_type_id")
		req.TargetObjectTypeID = &v
	}

	var res *authz.ListObjectsReachableWithAttributeResponse
	res, code, entries, err := h.listObjectsReachableWithAttribute(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createEdgeGenerated(w http.ResponseWriter, r *http.Request) {
	entries := h.createEdgeGeneratedOverride(w, r)
	auditlog.PostMultipleAsync(r.Context(), entries)
}

func (h *handler) deleteEdgeGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteEdge(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	if code == http.StatusNoContent {
		w.WriteHeader(code)
		return
	}

	jsonapi.Marshal(w, nil, jsonapi.Code(code))
}

func (h *handler) getEdgeGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *authz.Edge
	res, code, entries, err := h.getEdge(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listEdgesGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listEdgesParams{}
	if urlValues.Has("edge_type_id") && urlValues.Get("edge_type_id") != "null" {
		v := urlValues.Get("edge_type_id")
		req.EdgeTypeID = &v
	}
	if urlValues.Has("ending_before") && urlValues.Get("ending_before") != "null" {
		v := urlValues.Get("ending_before")
		req.EndingBefore = &v
	}
	if urlValues.Has("filter") && urlValues.Get("filter") != "null" {
		v := urlValues.Get("filter")
		req.Filter = &v
	}
	if urlValues.Has("limit") && urlValues.Get("limit") != "null" {
		v := urlValues.Get("limit")
		req.Limit = &v
	}
	if urlValues.Has("sort_key") && urlValues.Get("sort_key") != "null" {
		v := urlValues.Get("sort_key")
		req.SortKey = &v
	}
	if urlValues.Has("sort_order") && urlValues.Get("sort_order") != "null" {
		v := urlValues.Get("sort_order")
		req.SortOrder = &v
	}
	if urlValues.Has("source_object_id") && urlValues.Get("source_object_id") != "null" {
		v := urlValues.Get("source_object_id")
		req.SourceObjectID = &v
	}
	if urlValues.Has("starting_after") && urlValues.Get("starting_after") != "null" {
		v := urlValues.Get("starting_after")
		req.StartingAfter = &v
	}
	if urlValues.Has("target_object_id") && urlValues.Get("target_object_id") != "null" {
		v := urlValues.Get("target_object_id")
		req.TargetObjectID = &v
	}
	if urlValues.Has("version") && urlValues.Get("version") != "null" {
		v := urlValues.Get("version")
		req.Version = &v
	}

	var res *authz.ListEdgesResponse
	res, code, entries, err := h.listEdges(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteAllEdgesOnObjectGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteAllEdgesOnObject(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	if code == http.StatusNoContent {
		w.WriteHeader(code)
		return
	}

	jsonapi.Marshal(w, nil, jsonapi.Code(code))
}

func (h *handler) listEdgesOnObjectGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := ListEdgesOnObjectParams{}
	if urlValues.Has("ending_before") && urlValues.Get("ending_before") != "null" {
		v := urlValues.Get("ending_before")
		req.EndingBefore = &v
	}
	if urlValues.Has("filter") && urlValues.Get("filter") != "null" {
		v := urlValues.Get("filter")
		req.Filter = &v
	}
	if urlValues.Has("limit") && urlValues.Get("limit") != "null" {
		v := urlValues.Get("limit")
		req.Limit = &v
	}
	if urlValues.Has("sort_key") && urlValues.Get("sort_key") != "null" {
		v := urlValues.Get("sort_key")
		req.SortKey = &v
	}
	if urlValues.Has("sort_order") && urlValues.Get("sort_order") != "null" {
		v := urlValues.Get("sort_order")
		req.SortOrder = &v
	}
	if urlValues.Has("starting_after") && urlValues.Get("starting_after") != "null" {
		v := urlValues.Get("starting_after")
		req.StartingAfter = &v
	}
	if urlValues.Has("target_object_id") && urlValues.Get("target_object_id") != "null" {
		v := urlValues.Get("target_object_id")
		req.TargetObjectID = &v
	}
	if urlValues.Has("version") && urlValues.Get("version") != "null" {
		v := urlValues.Get("version")
		req.Version = &v
	}

	var res *authz.ListEdgesResponse
	res, code, entries, err := h.listEdgesOnObject(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createEdgeTypeGenerated(w http.ResponseWriter, r *http.Request) {
	entries := h.createEdgeTypeGeneratedOverride(w, r)
	auditlog.PostMultipleAsync(r.Context(), entries)
}

func (h *handler) deleteEdgeTypeGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteEdgeType(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	if code == http.StatusNoContent {
		w.WriteHeader(code)
		return
	}

	jsonapi.Marshal(w, nil, jsonapi.Code(code))
}

func (h *handler) getEdgeTypeGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *authz.EdgeType
	res, code, entries, err := h.getEdgeType(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listEdgeTypesGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listEdgeTypesParams{}
	if urlValues.Has("ending_before") && urlValues.Get("ending_before") != "null" {
		v := urlValues.Get("ending_before")
		req.EndingBefore = &v
	}
	if urlValues.Has("filter") && urlValues.Get("filter") != "null" {
		v := urlValues.Get("filter")
		req.Filter = &v
	}
	if urlValues.Has("limit") && urlValues.Get("limit") != "null" {
		v := urlValues.Get("limit")
		req.Limit = &v
	}
	if urlValues.Has("organization_id") && urlValues.Get("organization_id") != "null" {
		v := urlValues.Get("organization_id")
		req.OrganizationID = &v
	}
	if urlValues.Has("sort_key") && urlValues.Get("sort_key") != "null" {
		v := urlValues.Get("sort_key")
		req.SortKey = &v
	}
	if urlValues.Has("sort_order") && urlValues.Get("sort_order") != "null" {
		v := urlValues.Get("sort_order")
		req.SortOrder = &v
	}
	if urlValues.Has("source_object_type_id") && urlValues.Get("source_object_type_id") != "null" {
		v := urlValues.Get("source_object_type_id")
		req.SourceObjectTypeID = &v
	}
	if urlValues.Has("starting_after") && urlValues.Get("starting_after") != "null" {
		v := urlValues.Get("starting_after")
		req.StartingAfter = &v
	}
	if urlValues.Has("target_object_type_id") && urlValues.Get("target_object_type_id") != "null" {
		v := urlValues.Get("target_object_type_id")
		req.TargetObjectTypeID = &v
	}
	if urlValues.Has("version") && urlValues.Get("version") != "null" {
		v := urlValues.Get("version")
		req.Version = &v
	}

	var res *authz.ListEdgeTypesResponse
	res, code, entries, err := h.listEdgeTypes(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateEdgeTypeGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req authz.UpdateEdgeTypeRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *authz.EdgeType
	res, code, entries, err := h.updateEdgeType(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createObjectGenerated(w http.ResponseWriter, r *http.Request) {
	entries := h.createObjectGeneratedOverride(w, r)
	auditlog.PostMultipleAsync(r.Context(), entries)
}

func (h *handler) deleteObjectGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteObject(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	if code == http.StatusNoContent {
		w.WriteHeader(code)
		return
	}

	jsonapi.Marshal(w, nil, jsonapi.Code(code))
}

func (h *handler) getObjectGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *authz.Object
	res, code, entries, err := h.getObject(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listObjectsGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listObjectsParams{}
	if urlValues.Has("ending_before") && urlValues.Get("ending_before") != "null" {
		v := urlValues.Get("ending_before")
		req.EndingBefore = &v
	}
	if urlValues.Has("filter") && urlValues.Get("filter") != "null" {
		v := urlValues.Get("filter")
		req.Filter = &v
	}
	if urlValues.Has("limit") && urlValues.Get("limit") != "null" {
		v := urlValues.Get("limit")
		req.Limit = &v
	}
	if urlValues.Has("name") && urlValues.Get("name") != "null" {
		v := urlValues.Get("name")
		req.Name = &v
	}
	if urlValues.Has("organization_id") && urlValues.Get("organization_id") != "null" {
		v := urlValues.Get("organization_id")
		req.OrganizationID = &v
	}
	if urlValues.Has("sort_key") && urlValues.Get("sort_key") != "null" {
		v := urlValues.Get("sort_key")
		req.SortKey = &v
	}
	if urlValues.Has("sort_order") && urlValues.Get("sort_order") != "null" {
		v := urlValues.Get("sort_order")
		req.SortOrder = &v
	}
	if urlValues.Has("starting_after") && urlValues.Get("starting_after") != "null" {
		v := urlValues.Get("starting_after")
		req.StartingAfter = &v
	}
	if urlValues.Has("type_id") && urlValues.Get("type_id") != "null" {
		v := urlValues.Get("type_id")
		req.TypeID = &v
	}
	if urlValues.Has("version") && urlValues.Get("version") != "null" {
		v := urlValues.Get("version")
		req.Version = &v
	}

	var res *authz.ListObjectsResponse
	res, code, entries, err := h.listObjects(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateObjectGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req authz.UpdateObjectRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *authz.Object
	res, code, entries, err := h.updateObject(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createObjectTypeGenerated(w http.ResponseWriter, r *http.Request) {
	entries := h.createObjectTypeGeneratedOverride(w, r)
	auditlog.PostMultipleAsync(r.Context(), entries)
}

func (h *handler) deleteObjectTypeGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteObjectType(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	if code == http.StatusNoContent {
		w.WriteHeader(code)
		return
	}

	jsonapi.Marshal(w, nil, jsonapi.Code(code))
}

func (h *handler) getObjectTypeGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *authz.ObjectType
	res, code, entries, err := h.getObjectType(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listObjectTypesGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listObjectTypesParams{}
	if urlValues.Has("ending_before") && urlValues.Get("ending_before") != "null" {
		v := urlValues.Get("ending_before")
		req.EndingBefore = &v
	}
	if urlValues.Has("filter") && urlValues.Get("filter") != "null" {
		v := urlValues.Get("filter")
		req.Filter = &v
	}
	if urlValues.Has("limit") && urlValues.Get("limit") != "null" {
		v := urlValues.Get("limit")
		req.Limit = &v
	}
	if urlValues.Has("sort_key") && urlValues.Get("sort_key") != "null" {
		v := urlValues.Get("sort_key")
		req.SortKey = &v
	}
	if urlValues.Has("sort_order") && urlValues.Get("sort_order") != "null" {
		v := urlValues.Get("sort_order")
		req.SortOrder = &v
	}
	if urlValues.Has("starting_after") && urlValues.Get("starting_after") != "null" {
		v := urlValues.Get("starting_after")
		req.StartingAfter = &v
	}
	if urlValues.Has("version") && urlValues.Get("version") != "null" {
		v := urlValues.Get("version")
		req.Version = &v
	}

	var res *authz.ListObjectTypesResponse
	res, code, entries, err := h.listObjectTypes(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createOrganizationGenerated(w http.ResponseWriter, r *http.Request) {
	entries := h.createOrganizationGeneratedOverride(w, r)
	auditlog.PostMultipleAsync(r.Context(), entries)
}

func (h *handler) deleteOrganizationGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteOrganization(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	if code == http.StatusNoContent {
		w.WriteHeader(code)
		return
	}

	jsonapi.Marshal(w, nil, jsonapi.Code(code))
}

func (h *handler) getOrganizationGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	var res *authz.Organization
	res, code, entries, err := h.getOrganization(ctx, id, urlValues)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listOrganizationsGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listOrganizationsParams{}
	if urlValues.Has("ending_before") && urlValues.Get("ending_before") != "null" {
		v := urlValues.Get("ending_before")
		req.EndingBefore = &v
	}
	if urlValues.Has("filter") && urlValues.Get("filter") != "null" {
		v := urlValues.Get("filter")
		req.Filter = &v
	}
	if urlValues.Has("limit") && urlValues.Get("limit") != "null" {
		v := urlValues.Get("limit")
		req.Limit = &v
	}
	if urlValues.Has("sort_key") && urlValues.Get("sort_key") != "null" {
		v := urlValues.Get("sort_key")
		req.SortKey = &v
	}
	if urlValues.Has("sort_order") && urlValues.Get("sort_order") != "null" {
		v := urlValues.Get("sort_order")
		req.SortOrder = &v
	}
	if urlValues.Has("starting_after") && urlValues.Get("starting_after") != "null" {
		v := urlValues.Get("starting_after")
		req.StartingAfter = &v
	}
	if urlValues.Has("version") && urlValues.Get("version") != "null" {
		v := urlValues.Get("version")
		req.Version = &v
	}

	var res *authz.ListOrganizationsResponse
	res, code, entries, err := h.listOrganizations(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateOrganizationGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req authz.UpdateOrganizationRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *authz.Organization
	res, code, entries, err := h.updateOrganization(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}
