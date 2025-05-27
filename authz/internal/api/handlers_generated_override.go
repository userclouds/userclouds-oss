package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/ucdb"
	"userclouds.com/internal/auditlog"
)

func (h *handler) createEdgeGeneratedOverride(w http.ResponseWriter, r *http.Request) []auditlog.Entry {
	ctx := r.Context()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return nil
	}
	br := bytes.NewReader(b)
	dec := json.NewDecoder(br)

	var req authz.CreateEdgeRequest
	if dec.Decode(&req) != nil || req.Validate() != nil {
		br.Reset(b)
		var edge authz.Edge
		if err := dec.Decode(&edge); err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return nil
		} else if err := edge.Validate(); err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return nil
		}
		req.Edge = edge
	}

	var res *authz.Edge
	res, code, entries, err := h.createEdge(ctx, req)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return entries
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))

	return entries
}

func (h *handler) createObjectGeneratedOverride(w http.ResponseWriter, r *http.Request) []auditlog.Entry {
	ctx := r.Context()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return nil
	}
	br := bytes.NewReader(b)
	dec := json.NewDecoder(br)

	var req authz.CreateObjectRequest
	if dec.Decode(&req) != nil || req.Validate() != nil {
		br.Reset(b)
		var object authz.Object
		if err := dec.Decode(&object); err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return nil
		} else if err := object.Validate(); err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return nil
		}
		req.Object = object
	}

	var res *authz.Object
	res, code, entries, err := h.createObject(ctx, req)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return entries
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))

	return entries
}

func (h *handler) createObjectTypeGeneratedOverride(w http.ResponseWriter, r *http.Request) []auditlog.Entry {
	ctx := r.Context()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return nil
	}
	br := bytes.NewReader(b)
	dec := json.NewDecoder(br)

	var req authz.CreateObjectTypeRequest
	if dec.Decode(&req) != nil || req.Validate() != nil {
		br.Reset(b)
		var objectType authz.ObjectType
		if err := dec.Decode(&objectType); err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return nil
		} else if err := objectType.Validate(); err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return nil
		}
		req.ObjectType = objectType
	}

	var res *authz.ObjectType
	res, code, entries, err := h.createObjectType(ctx, req)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return entries
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))

	return entries
}

func (h *handler) createEdgeTypeGeneratedOverride(w http.ResponseWriter, r *http.Request) []auditlog.Entry {
	ctx := r.Context()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return nil
	}
	br := bytes.NewReader(b)
	dec := json.NewDecoder(br)

	var req authz.CreateEdgeTypeRequest
	if dec.Decode(&req) != nil || req.Validate() != nil {
		br.Reset(b)
		var edgeType authz.EdgeType
		if err := dec.Decode(&edgeType); err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return nil
		} else if err := edgeType.Validate(); err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return nil
		}
		req.EdgeType = edgeType
	}

	var res *authz.EdgeType
	res, code, entries, err := h.createEdgeType(ctx, req)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return entries
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))

	return entries
}

type oldCreateOrganizationRequest struct {
	ID     uuid.UUID         `json:"id"`
	Name   string            `json:"name" validate:"notempty"`
	Region region.DataRegion `json:"region"` // this is a UC Region (not an AWS region)
}

func (h *handler) createOrganizationGeneratedOverride(w http.ResponseWriter, r *http.Request) []auditlog.Entry {
	ctx := r.Context()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return nil
	}
	br := bytes.NewReader(b)
	dec := json.NewDecoder(br)

	var req authz.CreateOrganizationRequest
	if dec.Decode(&req) != nil || req.Validate() != nil {
		br.Reset(b)
		var oldReq oldCreateOrganizationRequest
		if err := dec.Decode(&oldReq); err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return nil
		}
		req.Organization = authz.Organization{
			BaseModel: ucdb.NewBaseWithID(oldReq.ID),
			Name:      oldReq.Name,
			Region:    oldReq.Region,
		}
		if err := req.Validate(); err != nil {
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return nil
		}
	}

	var res *authz.Organization
	res, code, entries, err := h.createOrganization(ctx, req)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return entries
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))

	return entries
}
