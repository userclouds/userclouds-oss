package tokenizer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/tokenizer"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/auditlog"
)

type listAccessPolicyTemplatesParams struct {
	pagination.QueryParams
	Name      *string `description:"Optional - allows filtering by template name" query:"template_name"`
	Version   *string `description:"Optional - allows filtering by template version" query:"template_version"`
	Versioned *string `description:"Optional - if versioned is true, endpoint will return all versions of each access policy template in the response. Otherwise, only the latest version of each template will be returned." query:"versioned"`
}

// OpenAPI Summary: List Access Policy Templates
// OpenAPI Tags: Access Policy Templates
// OpenAPI Description: This endpoint returns a paginated list of access policy templates in a tenant. The list can be filtered to only include templates with a specified name or version.
func (h handler) listAccessPolicyTemplates(ctx context.Context, req listAccessPolicyTemplatesParams) (*idp.ListAccessPolicyTemplatesResponse, int, []auditlog.Entry, error) {

	s := storage.MustCreateStorage(ctx)
	accessPolicyTemplates := []policy.AccessPolicyTemplate{}
	var responseFields pagination.ResponseFields
	if req.Name != nil {
		apt, err := s.GetAccessPolicyTemplateByName(ctx, *req.Name)
		if err != nil {
			return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}

		if req.Version != nil {
			versionInt, err := strconv.Atoi(*req.Version)
			if err != nil {
				return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
			}
			apt, err = s.GetAccessPolicyTemplateByVersion(ctx, apt.ID, versionInt)
			if err != nil {
				return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
			}
		}
		accessPolicyTemplates = append(accessPolicyTemplates, apt.ToClient())

		return &idp.ListAccessPolicyTemplatesResponse{
			Data: accessPolicyTemplates,
		}, http.StatusOK, nil, nil
	}

	pager, err := storage.NewAccessPolicyTemplatePaginatorFromQuery(req)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	if req.Versioned != nil && *req.Versioned == "true" {

		templates, respFields, err := s.ListAccessPolicyTemplatesPaginated(ctx, *pager)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}

		for _, apt := range templates {
			accessPolicyTemplates = append(accessPolicyTemplates, apt.ToClient())
		}
		responseFields = *respFields

	} else {
		templates, respFields, err := s.GetLatestAccessPolicyTemplatesPaginated(ctx, *pager)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}

		for _, apt := range templates {
			accessPolicyTemplates = append(accessPolicyTemplates, apt.ToClient())
		}
		responseFields = *respFields
	}

	return &idp.ListAccessPolicyTemplatesResponse{Data: accessPolicyTemplates, ResponseFields: responseFields}, http.StatusOK, nil, nil
}

// GetAccessPolicyTemplateParams is the request body for the Get Access Policy Template API
type GetAccessPolicyTemplateParams struct {
	Version *string `description:"Optional - if not specified, the latest template will be returned" query:"template_version"`
}

// OpenAPI Summary: Get Access Policy Template
// OpenAPI Tags: Access Policy Templates
// OpenAPI Description: This endpoint gets an access policy template by ID.
func (h handler) getAccessPolicyTemplate(ctx context.Context, id uuid.UUID, req GetAccessPolicyTemplateParams) (*policy.AccessPolicyTemplate, int, []auditlog.Entry, error) {
	s := storage.MustCreateStorage(ctx)
	var apt *storage.AccessPolicyTemplate
	var err error

	if req.Version != nil {
		version, err := strconv.Atoi(*req.Version)
		if err != nil {
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}

		apt, err = s.GetAccessPolicyTemplateByVersion(ctx, id, version)
		if err != nil {
			return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}
	} else {
		apt, err = s.GetLatestAccessPolicyTemplate(ctx, id)
		if err != nil {
			return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}
	}

	clientAPT := apt.ToClient()
	return &clientAPT, http.StatusOK, nil, nil
}

// OpenAPI Summary: Create Access Policy Template
// OpenAPI Tags: Access Policy Templates
// OpenAPI Description: This endpoint creates an access policy template.
func (h handler) createAccessPolicyTemplate(ctx context.Context, req tokenizer.CreateAccessPolicyTemplateRequest) (*policy.AccessPolicyTemplate, int, []auditlog.Entry, error) {
	if req.AccessPolicyTemplate.IsSystem {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "System templates cannot be created")
	}

	s := storage.MustCreateStorage(ctx)
	// if a matching template can be loaded, we shouldn't use using create
	if req.AccessPolicyTemplate.ID != uuid.Nil {
		if storageAPT, err := s.GetLatestAccessPolicyTemplate(ctx, req.AccessPolicyTemplate.ID); !errors.Is(err, sql.ErrNoRows) {

			if err != nil {
				return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
			}

			if storageAPT.ToClient().EqualsIgnoringNilID(req.AccessPolicyTemplate) {
				return nil, http.StatusConflict, nil,
					ucerr.WrapWithFriendlyStructure(
						jsonclient.Error{StatusCode: http.StatusConflict},
						jsonclient.SDKStructuredError{
							Error:     "This template already exists",
							ID:        storageAPT.ID,
							Identical: true,
						},
					)
			}

			return nil, http.StatusConflict, nil,
				ucerr.WrapWithFriendlyStructure(
					jsonclient.Error{StatusCode: http.StatusConflict},
					jsonclient.SDKStructuredError{
						Error: "A template already exists with the same ID",
						ID:    storageAPT.ID,
					},
				)
		}
	}

	if storageAPT, err := s.GetAccessPolicyTemplateByName(ctx, req.AccessPolicyTemplate.Name); !errors.Is(err, sql.ErrNoRows) {

		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}

		if storageAPT.ToClient().EqualsIgnoringNilID(req.AccessPolicyTemplate) {
			return nil, http.StatusConflict, nil,
				ucerr.WrapWithFriendlyStructure(
					jsonclient.Error{StatusCode: http.StatusConflict},
					jsonclient.SDKStructuredError{
						Error:     "This template already exists",
						ID:        storageAPT.ID,
						Identical: true,
					},
				)
		}

		return nil, http.StatusConflict, nil,
			ucerr.WrapWithFriendlyStructure(
				jsonclient.Error{StatusCode: http.StatusConflict},
				jsonclient.SDKStructuredError{
					Error: fmt.Sprintf(`A template with the name '%s' already exists`, req.AccessPolicyTemplate.Name),
					ID:    storageAPT.ID,
				},
			)
	}

	apt := req.AccessPolicyTemplate
	if apt.ID.IsNil() {
		apt.ID = uuid.Must(uuid.NewV4())
	}

	storageAPT := storage.NewAccessPolicyTemplateFromClient(apt)
	if err := storageAPT.Validate(); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	if err := s.SaveAccessPolicyTemplate(ctx, &storageAPT); err != nil {
		return nil, uchttp.SQLWriteErrorMapper(err), nil, ucerr.Wrap(err)
	}

	h.createEventTypesForAccessPolicyTemplate(ctx, storageAPT.ID, storageAPT.Version)

	apt = storageAPT.ToClient()
	return &apt, http.StatusCreated, nil, nil
}

// OpenAPI Summary: Update Access Policy Template
// OpenAPI Tags: Access Policy Templates
// OpenAPI Description: This endpoint updates a specified access policy template.
func (h handler) updateAccessPolicyTemplate(ctx context.Context, id uuid.UUID, req tokenizer.UpdateAccessPolicyTemplateRequest) (*policy.AccessPolicyTemplate, int, []auditlog.Entry, error) {

	if req.AccessPolicyTemplate.ID != uuid.Nil && req.AccessPolicyTemplate.ID != id {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "URL ID doesn't match request body ID")
	}

	s := storage.MustCreateStorage(ctx)
	storageAPT, err := s.GetLatestAccessPolicyTemplate(ctx, id)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	if storageAPT.IsSystem {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "system templates cannot be updated")
	}
	if req.AccessPolicyTemplate.IsSystem != storageAPT.IsSystem {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "the IsSystem attribute cannot be updated by the client")
	}

	updateNames := false
	if storageAPT.Name != req.AccessPolicyTemplate.Name {
		// check that the new name isn't already in use
		storageAPT, err := s.GetAccessPolicyTemplateByName(ctx, req.AccessPolicyTemplate.Name)
		if storageAPT != nil {
			if id != storageAPT.ID {
				return nil, http.StatusConflict, nil,
					ucerr.WrapWithFriendlyStructure(
						jsonclient.Error{StatusCode: http.StatusConflict},
						jsonclient.SDKStructuredError{
							Error: fmt.Sprintf(`A template with the name '%s' already exists`, req.AccessPolicyTemplate.Name),
							ID:    storageAPT.ID,
						},
					)
			}

			// if the ids are the same, this is just a case change
			// to the name, which we should allow
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
		updateNames = true
	}

	if storageAPT.Version != req.AccessPolicyTemplate.Version {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "Access policy template version in request does not match current version")
	}

	storageAPT.Name = req.AccessPolicyTemplate.Name
	storageAPT.Description = req.AccessPolicyTemplate.Description
	storageAPT.Function = req.AccessPolicyTemplate.Function
	storageAPT.Version++

	if err := s.SaveAccessPolicyTemplate(ctx, storageAPT); err != nil {
		return nil, uchttp.SQLWriteErrorMapper(err), nil, ucerr.Wrap(err)
	}

	if updateNames {
		// change the policy template name for all versions
		storageAPTs, err := s.GetAllAccessPolicyTemplateVersions(ctx, id)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
		for _, storageAPT := range storageAPTs {
			if storageAPT.Name != req.AccessPolicyTemplate.Name {
				storageAPT.Name = req.AccessPolicyTemplate.Name
				if err := s.PriorVersionSaveAccessPolicyTemplate(ctx, &storageAPT); err != nil {
					return nil, uchttp.SQLWriteErrorMapper(err), nil, ucerr.Wrap(err)
				}
			}
		}
	}

	clientAPT := storageAPT.ToClient()
	return &clientAPT, http.StatusOK, nil, nil
}

// DeleteAccessPolicyTemplateParams is the request body for the Delete Access Policy Template API
type DeleteAccessPolicyTemplateParams struct {
	Version *string `description:"Required - specifies the version of the template to delete, use 'all' to delete all versions" query:"template_version"`
}

// OpenAPI Summary: Delete Access Policy Template
// OpenAPI Tags: Access Policy Templates
// OpenAPI Description: This endpoint deletes an access policy template by ID.
func (h handler) deleteAccessPolicyTemplate(ctx context.Context, id uuid.UUID, req DeleteAccessPolicyTemplateParams) (int, []auditlog.Entry, error) {

	if req.Version == nil || *req.Version == "" {
		return http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "version is required")
	}

	s := storage.MustCreateStorage(ctx)

	if strings.ToLower(*req.Version) == "all" {
		apt, err := s.GetLatestAccessPolicyTemplate(ctx, id)
		if err != nil {
			return uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}
		if apt.IsSystem {
			return http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "system templates cannot be deleted")
		}

		if err := s.DeleteAllAccessPolicyTemplateVersions(ctx, id); err != nil {
			if errors.Is(err, storage.ErrStillInUse) {
				return http.StatusConflict, nil, ucerr.Wrap(err)
			}
			return http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}

	} else {
		versionInt, err := strconv.Atoi(*req.Version)
		if err != nil {
			return http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "version must be an integer or 'all'")
		}

		apt, err := s.GetAccessPolicyTemplateByVersion(ctx, id, versionInt)
		if err != nil {
			return uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}
		if apt.IsSystem {
			return http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "system templates cannot be deleted")
		}

		if err := s.DeleteAccessPolicyTemplateByVersion(ctx, id, versionInt); err != nil {
			if errors.Is(err, storage.ErrStillInUse) {
				return http.StatusConflict, nil, ucerr.Wrap(err)
			}
			return http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return http.StatusNoContent, nil, nil
}

// OpenAPI Summary: Test Access Policy Template
// OpenAPI Tags: Access Policies
// OpenAPI Description: This endpoint tests an access policy template. It receives test context and returns a boolean indicating whether access is allowed or denied.
func (h handler) testAccessPolicyTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := storage.MustCreateStorage(ctx)

	var req tokenizer.TestAccessPolicyTemplateRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	bs, err := json.Marshal(req.Context)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	uclog.Debugf(ctx, "testing access policy template: %+v", req)

	storageAPT := storage.NewAccessPolicyTemplateFromClient(req.AccessPolicyTemplate)
	if storageAPT.ID.IsNil() {
		storageAPT.ID = uuid.Must(uuid.NewV4())
	}

	if err := storageAPT.Validate(); err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	apte, err := newAccessPolicyTemplateExecutor(authzClient, s)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	defer apte.cleanup()

	allowed, err := apte.execute(ctx, &storageAPT, req.Context, string(bs), req.Params)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	uclog.Debugf(ctx, "got result: %v", allowed)

	jsonapi.Marshal(w, tokenizer.TestAccessPolicyResponse{Allowed: allowed, Debug: map[string]any{"console": apte.getConsoleOutput()}})
}
