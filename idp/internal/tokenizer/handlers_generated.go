// NOTE: automatically generated file -- DO NOT EDIT

package tokenizer

import (
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/tokenizer"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/internal/auditlog"
)

func handlerBuilder(builder *builder.HandlerBuilder, h *handler) {

	builder.CollectionHandler("/policies/access").
		GetOne(h.getAccessPolicyGenerated).
		Post(h.createAccessPolicyGenerated).
		Put(h.updateAccessPolicyGenerated).
		Delete(h.deleteAccessPolicyGenerated).
		GetAll(h.listAccessPoliciesGenerated).
		WithAuthorizer(h.newTokenizerAuthorizer())

	builder.CollectionHandler("/policies/accesstemplate").
		GetOne(h.getAccessPolicyTemplateGenerated).
		Post(h.createAccessPolicyTemplateGenerated).
		Put(h.updateAccessPolicyTemplateGenerated).
		Delete(h.deleteAccessPolicyTemplateGenerated).
		GetAll(h.listAccessPolicyTemplatesGenerated).
		WithAuthorizer(h.newAccessPolicyTemplateAuthorizer())

	builder.CollectionHandler("/policies/secret").
		Post(h.createSecretGenerated).
		Delete(h.deleteSecretGenerated).
		GetAll(h.listSecretsGenerated).
		WithAuthorizer(h.newSecretAuthorizer())

	builder.CollectionHandler("/policies/transformation").
		GetOne(h.getTransformerGenerated).
		Post(h.createTransformerGenerated).
		Put(h.updateTransformerGenerated).
		Delete(h.deleteTransformerGenerated).
		GetAll(h.listTransformersGenerated).
		WithAuthorizer(h.newTokenizerAuthorizer())

	builder.MethodHandler("/tokens").
		Post(h.createTokenGenerated).
		Delete(h.deleteTokenGenerated).
		End()

	builder.MethodHandler("/tokens/actions/inspect").Post(h.inspectTokenGenerated)

	builder.MethodHandler("/tokens/actions/lookup").Post(h.lookupTokensGenerated)

	builder.MethodHandler("/tokens/actions/lookuporcreate").Post(h.lookupOrCreateTokensGenerated)

	builder.MethodHandler("/tokens/actions/resolve").Post(h.resolveTokenGenerated)

}

func (h *handler) inspectTokenGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req tokenizer.InspectTokenRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *tokenizer.InspectTokenResponse
	res, code, entries, err := h.inspectToken(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) lookupOrCreateTokensGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req tokenizer.LookupOrCreateTokensRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *tokenizer.LookupOrCreateTokensResponse
	res, code, entries, err := h.lookupOrCreateTokens(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) lookupTokensGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req tokenizer.LookupTokensRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *tokenizer.LookupTokensResponse
	res, code, entries, err := h.lookupTokens(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) resolveTokenGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req tokenizer.ResolveTokensRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res []tokenizer.ResolveTokenResponse
	res, code, entries, err := h.resolveToken(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createTokenGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req tokenizer.CreateTokenRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *tokenizer.CreateTokenResponse
	res, code, entries, err := h.createToken(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteTokenGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := deleteTokenParams{}
	if urlValues.Has("token") && urlValues.Get("token") != "null" {
		v := urlValues.Get("token")
		req.Token = &v
	}

	code, entries, err := h.deleteToken(ctx, req)
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

func (h *handler) createAccessPolicyGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req tokenizer.CreateAccessPolicyRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *policy.AccessPolicy
	res, code, entries, err := h.createAccessPolicy(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteAccessPolicyGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := DeleteAccessPolicyParams{}
	if urlValues.Has("policy_version") && urlValues.Get("policy_version") != "null" {
		v := urlValues.Get("policy_version")
		req.Version = &v
	}

	code, entries, err := h.deleteAccessPolicy(ctx, id, req)
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

func (h *handler) getAccessPolicyGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := GetAccessPolicyParams{}
	if urlValues.Has("policy_version") && urlValues.Get("policy_version") != "null" {
		v := urlValues.Get("policy_version")
		req.Version = &v
	}

	var res *policy.AccessPolicy
	res, code, entries, err := h.getAccessPolicy(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listAccessPoliciesGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listAccessPoliciesParams{}
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
	if urlValues.Has("policy_name") && urlValues.Get("policy_name") != "null" {
		v := urlValues.Get("policy_name")
		req.Name = &v
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
	if urlValues.Has("policy_version") && urlValues.Get("policy_version") != "null" {
		v := urlValues.Get("policy_version")
		req.Version = &v
	}
	if urlValues.Has("versioned") && urlValues.Get("versioned") != "null" {
		v := urlValues.Get("versioned")
		req.Versioned = &v
	}

	var res *idp.ListAccessPoliciesResponse
	res, code, entries, err := h.listAccessPolicies(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateAccessPolicyGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req tokenizer.UpdateAccessPolicyRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *policy.AccessPolicy
	res, code, entries, err := h.updateAccessPolicy(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createAccessPolicyTemplateGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req tokenizer.CreateAccessPolicyTemplateRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *policy.AccessPolicyTemplate
	res, code, entries, err := h.createAccessPolicyTemplate(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteAccessPolicyTemplateGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := DeleteAccessPolicyTemplateParams{}
	if urlValues.Has("template_version") && urlValues.Get("template_version") != "null" {
		v := urlValues.Get("template_version")
		req.Version = &v
	}

	code, entries, err := h.deleteAccessPolicyTemplate(ctx, id, req)
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

func (h *handler) getAccessPolicyTemplateGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := GetAccessPolicyTemplateParams{}
	if urlValues.Has("template_version") && urlValues.Get("template_version") != "null" {
		v := urlValues.Get("template_version")
		req.Version = &v
	}

	var res *policy.AccessPolicyTemplate
	res, code, entries, err := h.getAccessPolicyTemplate(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listAccessPolicyTemplatesGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listAccessPolicyTemplatesParams{}
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
	if urlValues.Has("template_name") && urlValues.Get("template_name") != "null" {
		v := urlValues.Get("template_name")
		req.Name = &v
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
	if urlValues.Has("template_version") && urlValues.Get("template_version") != "null" {
		v := urlValues.Get("template_version")
		req.Version = &v
	}
	if urlValues.Has("versioned") && urlValues.Get("versioned") != "null" {
		v := urlValues.Get("versioned")
		req.Versioned = &v
	}

	var res *idp.ListAccessPolicyTemplatesResponse
	res, code, entries, err := h.listAccessPolicyTemplates(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateAccessPolicyTemplateGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req tokenizer.UpdateAccessPolicyTemplateRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *policy.AccessPolicyTemplate
	res, code, entries, err := h.updateAccessPolicyTemplate(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createSecretGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req tokenizer.CreateSecretRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *policy.Secret
	res, code, entries, err := h.createSecret(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteSecretGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteSecret(ctx, id, urlValues)
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

func (h *handler) listSecretsGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listSecretsParams{}
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

	var res *idp.ListSecretsResponse
	res, code, entries, err := h.listSecrets(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) createTransformerGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req tokenizer.CreateTransformerRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *policy.Transformer
	res, code, entries, err := h.createTransformer(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) deleteTransformerGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	code, entries, err := h.deleteTransformer(ctx, id, urlValues)
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

func (h *handler) getTransformerGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := GetTransformerParams{}
	if urlValues.Has("transformer_version") && urlValues.Get("transformer_version") != "null" {
		v := urlValues.Get("transformer_version")
		req.Version = &v
	}

	var res *policy.Transformer
	res, code, entries, err := h.getTransformer(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) listTransformersGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := listTransformersParams{}
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
	if urlValues.Has("transformer_name") && urlValues.Get("transformer_name") != "null" {
		v := urlValues.Get("transformer_name")
		req.Name = &v
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
	if urlValues.Has("transformer_version") && urlValues.Get("transformer_version") != "null" {
		v := urlValues.Get("transformer_version")
		req.Version = &v
	}
	if urlValues.Has("versioned") && urlValues.Get("versioned") != "null" {
		v := urlValues.Get("versioned")
		req.Versioned = &v
	}

	var res *idp.ListTransformersResponse
	res, code, entries, err := h.listTransformers(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

func (h *handler) updateTransformerGenerated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req tokenizer.UpdateTransformerRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res *policy.Transformer
	res, code, entries, err := h.updateTransformer(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}
