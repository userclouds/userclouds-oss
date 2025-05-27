package api

import (
	"net/http"
	"strconv"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/console/internal/auth"
	"userclouds.com/idp"
	idpAuthz "userclouds.com/idp/authz"
	"userclouds.com/idp/tokenizer"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// Returns the current user's permissions on the global policy object
func (h *handler) listGlobalPolicyPermissions(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	h.listPolicyPermissions(w, r, tenantID, idpAuthz.PoliciesObjectID)
}

type policyPermissionsResponse struct {
	Create bool `json:"create"`
	Read   bool `json:"read"`
	Update bool `json:"update"`
	Delete bool `json:"delete"`
}

// Returns the current user's permission on the specified policy object
func (h *handler) listPolicyPermissions(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, policyID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	rbacClient := authz.NewRBACClient(authZClient)

	userInfo, err := auth.GetUserInfo(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	userID, err := userInfo.GetUserID()
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	user, err := rbacClient.GetUser(ctx, userID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	permissions, err := user.ListResourceAttributes(ctx, authz.Resource{ID: policyID})
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp := policyPermissionsResponse{}
	for _, permission := range permissions {
		if permission == idpAuthz.AttributePolicyCreate {
			resp.Create = true
		} else if permission == idpAuthz.AttributePolicyRead {
			resp.Read = true
		} else if permission == idpAuthz.AttributePolicyUpdate {
			resp.Update = true
		} else if permission == idpAuthz.AttributePolicyDelete {
			resp.Delete = true
		}
	}

	jsonapi.Marshal(w, resp)
}

func (h handler) listAccessPolicyTemplates(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	versioned := r.URL.Query().Get("versioned")

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	options := idp.Pagination(pager.GetOptions()...)

	resp, err := c.ListAccessPolicyTemplates(ctx, versioned != "", options)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h handler) getAccessPolicyTemplate(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if versionString := r.URL.Query().Get("version"); versionString != "" {
		version, err := strconv.Atoi(versionString)
		if err != nil {
			jsonapi.MarshalError(ctx, w, ucerr.New("Invalid access policy version specified"), jsonapi.Code(http.StatusBadRequest))
			return
		}

		resp, err := c.GetAccessPolicyTemplateByVersion(ctx, userstore.ResourceID{ID: id}, version)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		jsonapi.Marshal(w, resp)
		return
	}

	resp, err := c.GetAccessPolicyTemplate(ctx, userstore.ResourceID{ID: id})
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	jsonapi.Marshal(w, resp)
}

func (h handler) updateAccessPolicyTemplate(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()

	var req tokenizer.UpdateAccessPolicyTemplateRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if req.AccessPolicyTemplate.ID != id {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "path (%v) and request (%v) IDs must match", id, req.AccessPolicyTemplate.ID))
		return
	}

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := c.UpdateAccessPolicyTemplate(ctx, req.AccessPolicyTemplate)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h handler) createAccessPolicyTemplate(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	var req tokenizer.CreateAccessPolicyTemplateRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	ap, err := c.CreateAccessPolicyTemplate(ctx, req.AccessPolicyTemplate)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, *ap)
}

type deleteAccessPolicyTemplateRequest struct {
	Version int `json:"version"`
}

func (h handler) deleteAccessPolicyTemplate(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()

	var req deleteAccessPolicyTemplateRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := c.DeleteAccessPolicyTemplate(ctx, id, req.Version); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h handler) listAccessPolicies(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	versioned := r.URL.Query().Get("versioned")

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	options := idp.Pagination(pager.GetOptions()...)

	resp, err := c.ListAccessPolicies(ctx, versioned != "", options)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h handler) getAccessPolicy(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if versionString := r.URL.Query().Get("version"); versionString != "" {
		version, err := strconv.Atoi(versionString)
		if err != nil {
			jsonapi.MarshalError(ctx, w, ucerr.New("Invalid access policy version specified"))
			return
		}

		resp, err := c.GetAccessPolicyByVersion(ctx, userstore.ResourceID{ID: id}, version)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		jsonapi.Marshal(w, resp)
		return
	}

	resp, err := c.GetAccessPolicy(ctx, userstore.ResourceID{ID: id})
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	jsonapi.Marshal(w, resp)
}

func (h handler) updateAccessPolicy(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()

	var req tokenizer.UpdateAccessPolicyRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if req.AccessPolicy.ID != id {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "path (%v) and request (%v) IDs must match", id, req.AccessPolicy.ID), jsonapi.Code(http.StatusBadRequest))
		return
	}

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := c.UpdateAccessPolicy(ctx, req.AccessPolicy)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h handler) createAccessPolicy(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	var req tokenizer.CreateAccessPolicyRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	ap, err := c.CreateAccessPolicy(ctx, req.AccessPolicy)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, *ap)
}

type deleteAccessPolicyRequest struct {
	Version int `json:"version"`
}

func (h handler) deleteAccessPolicy(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()

	var req deleteAccessPolicyRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := c.DeleteAccessPolicy(ctx, id, req.Version); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h handler) testRunAccessPolicy(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	var req tokenizer.TestAccessPolicyRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	uclog.Debugf(ctx, "testing access policy: %v", req)

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := c.TestAccessPolicy(ctx, req.AccessPolicy, req.Context)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h handler) testRunAccessPolicyTemplate(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	var req tokenizer.TestAccessPolicyTemplateRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	uclog.Debugf(ctx, "testing access policy template: %v", req)

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := c.TestAccessPolicyTemplate(ctx, req.AccessPolicyTemplate, req.Context, req.Params)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h handler) listTransformers(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	options := idp.Pagination(pager.GetOptions()...)

	resp, err := c.ListTransformers(ctx, options)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h handler) getTransformer(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if versionString := r.URL.Query().Get("version"); versionString != "" {
		version, err := strconv.Atoi(versionString)
		if err != nil {
			jsonapi.MarshalError(ctx, w, ucerr.New("Invalid transformer version specified"))
			return
		}

		resp, err := c.GetTransformerByVersion(ctx, userstore.ResourceID{ID: id}, version)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		jsonapi.Marshal(w, resp)
		return
	}

	resp, err := c.GetTransformer(ctx, userstore.ResourceID{ID: id})
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	jsonapi.Marshal(w, resp)
}

func (h handler) createTransformer(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	var req tokenizer.CreateTransformerRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	tf, err := c.CreateTransformer(ctx, req.Transformer)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, tf)
}

func (h handler) updateTransformer(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()

	var req tokenizer.UpdateTransformerRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if req.Transformer.ID != id {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "path (%v) and request (%v) IDs must match", id, req.Transformer.ID), jsonapi.Code(http.StatusBadRequest))
		return
	}

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	tf, err := c.UpdateTransformer(ctx, req.Transformer)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, tf)
}

func (h handler) deleteTransformer(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := c.DeleteTransformer(ctx, id); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h handler) testRunTransformer(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	var req tokenizer.TestTransformerRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	uclog.Debugf(ctx, "testing transformer: %v", req)

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	resp, err := c.TestTransformer(ctx, req.Data, req.Transformer)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h handler) listSecrets(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	pager, err := pagination.NewPaginatorFromRequest(r)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	options := idp.Pagination(pager.GetOptions()...)

	resp, err := c.ListSecrets(ctx, options)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, resp)
}

func (h handler) createSecret(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	var req tokenizer.CreateSecretRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	gp, err := c.CreateSecret(ctx, req.Secret)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, gp)
}

func (h handler) deleteSecret(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()

	c, err := h.newTokenizerClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := c.DeleteSecret(ctx, id); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
