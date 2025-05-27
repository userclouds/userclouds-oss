package tokenizer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"

	"userclouds.com/authz"
	"userclouds.com/idp"
	"userclouds.com/idp/internal"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/tokenizer"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/security"
)

func validateAndPopulateAccessPolicyFields(ctx context.Context, s *storage.Storage, accessPolicy *policy.AccessPolicy) error {

	if err := accessPolicy.Validate(); err != nil {
		return ucerr.Friendlyf(err, "invalid access policy: %v", accessPolicy)
	}

	for i, component := range accessPolicy.Components {
		if component.Policy != nil {
			if err := component.Policy.Validate(); err != nil {
				return ucerr.Friendlyf(err, "invalid component policy: %v", component.Policy)
			}

			if component.Policy.ID != uuid.Nil {
				ap, err := s.GetLatestAccessPolicy(ctx, component.Policy.ID)
				if err != nil {
					return ucerr.Friendlyf(err, "invalid component policy: %v", component.Policy)
				}
				if component.Policy.Name != "" && ap.Name != component.Policy.Name {
					return ucerr.Friendlyf(nil, "component policy name doesn't match ID: %v", component.Policy)
				}
				accessPolicy.Components[i].Policy.Name = ap.Name

			} else {
				ap, err := s.GetAccessPolicyByName(ctx, component.Policy.Name)
				if err != nil {
					return ucerr.Friendlyf(err, "invalid component policy: %v", component.Policy)
				}
				accessPolicy.Components[i].Policy.ID = ap.ID
			}

		} else if component.Template != nil {
			if err := component.Template.Validate(); err != nil {
				return ucerr.Friendlyf(err, "invalid component template: %v", component.Template)
			}

			if component.Template.ID != uuid.Nil {
				t, err := s.GetLatestAccessPolicyTemplate(ctx, component.Template.ID)
				if err != nil {
					return ucerr.Friendlyf(err, "invalid component template: %v", component.Template)
				}
				if component.Template.Name != "" && t.Name != component.Template.Name {
					return ucerr.Friendlyf(nil, "component template name doesn't match ID: %v", component.Template)
				}
				accessPolicy.Components[i].Template.Name = t.Name

			} else {
				t, err := s.GetAccessPolicyTemplateByName(ctx, component.Template.Name)
				if err != nil {
					return ucerr.Friendlyf(err, "invalid component template: %v", component.Template)
				}
				accessPolicy.Components[i].Template.ID = t.ID
			}
		}
	}

	return nil
}

func examineSubPolicies(ctx context.Context, s *storage.Storage, ap *storage.AccessPolicy, visited set.Set[uuid.UUID], recStack set.Set[uuid.UUID]) (map[string]string, error) {

	combinedRequiredContext := map[string]string{}

	visited.Insert(ap.ID)
	recStack.Insert(ap.ID)

	for i, componentID := range ap.ComponentIDs {
		if ap.ComponentTypes[i] != int32(storage.AccessPolicyComponentTypePolicy) {
			continue
		}

		if recStack.Contains(componentID) {
			return nil, ucerr.Friendlyf(nil, "Loop detected in policy %s (ID: %v)", ap.Name, ap.ID)
		}

		if visited.Contains(componentID) {
			continue
		}

		componentAP, err := s.GetLatestAccessPolicy(ctx, componentID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		maps.Copy(combinedRequiredContext, componentAP.Metadata.RequiredContext)

		rc, err := examineSubPolicies(ctx, s, componentAP, visited, recStack)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		maps.Copy(combinedRequiredContext, rc)
	}

	recStack.Evict(ap.ID)

	return combinedRequiredContext, nil
}

func convertClientPolicyComponentsToStoragePolicyComponents(clientComponents []policy.AccessPolicyComponent, ap *storage.AccessPolicy) error {
	ap.ComponentIDs = nil
	ap.ComponentParameters = nil
	ap.ComponentTypes = nil

	for _, component := range clientComponents {
		if component.Template != nil {
			ap.ComponentIDs = append(ap.ComponentIDs, component.Template.ID)
			if component.TemplateParameters == "" {
				ap.ComponentParameters = append(ap.ComponentParameters, "{}")
			} else {
				ap.ComponentParameters = append(ap.ComponentParameters, component.TemplateParameters)
			}
			ap.ComponentTypes = append(ap.ComponentTypes, int32(storage.AccessPolicyComponentTypeTemplate))
		} else if component.Policy != nil {
			ap.ComponentIDs = append(ap.ComponentIDs, component.Policy.ID)
			ap.ComponentParameters = append(ap.ComponentParameters, "")
			ap.ComponentTypes = append(ap.ComponentTypes, int32(storage.AccessPolicyComponentTypePolicy))
		} else {
			return ucerr.New("Invalid component")
		}
	}

	return nil
}

type listAccessPoliciesParams struct {
	pagination.QueryParams
	Name      *string `description:"Optional - allows filtering by access policy name" query:"policy_name"`
	Version   *string `description:"Optional - allows filtering by access policy version" query:"policy_version"`
	Versioned *string `description:"Optional - set versioned to be true to include all prior versions of access policies in response" query:"versioned"`
}

// OpenAPI Summary: List Access Policies
// OpenAPI Tags: Access Policies
// OpenAPI Description: This endpoint returns a list of access policies in a tenant. The list can be filtered to only include policies with a specified name or version.
func (h handler) listAccessPolicies(ctx context.Context, req listAccessPoliciesParams) (*idp.ListAccessPoliciesResponse, int, []auditlog.Entry, error) {

	s := storage.MustCreateStorage(ctx)
	clientAccessPolicies := []policy.AccessPolicy{}
	var responseFields pagination.ResponseFields
	if req.Name != nil {
		ap, err := s.GetAccessPolicyByName(ctx, *req.Name)
		if err != nil {
			return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}

		if req.Version != nil {
			versionInt, err := strconv.Atoi(*req.Version)
			if err != nil {
				return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(err, "Invalid access policy version specified")
			}
			ap, err = s.GetAccessPolicyByVersion(ctx, ap.ID, versionInt)
			if err != nil {
				return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
			}
		}

		clientAP := ap.ToClientModel()
		if err = validateAndPopulateAccessPolicyFields(ctx, s, clientAP); err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
		clientAccessPolicies = append(clientAccessPolicies, *clientAP)

		return &idp.ListAccessPoliciesResponse{
			Data: clientAccessPolicies,
		}, http.StatusOK, nil, nil
	}

	pager, err := storage.NewAccessPolicyPaginatorFromQuery(req)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	if req.Versioned != nil && *req.Versioned == "true" {

		policies, respFields, err := s.ListAccessPoliciesPaginated(ctx, *pager)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}

		for i := range policies {
			clientAP := policies[i].ToClientModel()
			if err = validateAndPopulateAccessPolicyFields(ctx, s, clientAP); err != nil {
				return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
			}
			clientAccessPolicies = append(clientAccessPolicies, *clientAP)
		}
		responseFields = *respFields

	} else {
		policies, respFields, err := s.GetLatestAccessPoliciesPaginated(ctx, *pager)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}

		for i := range policies {
			clientAP := policies[i].ToClientModel()
			if err = validateAndPopulateAccessPolicyFields(ctx, s, clientAP); err != nil {
				return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
			}

			clientAccessPolicies = append(clientAccessPolicies, *clientAP)
		}
		responseFields = *respFields
	}

	return &idp.ListAccessPoliciesResponse{Data: clientAccessPolicies, ResponseFields: responseFields}, http.StatusOK, nil, nil
}

// GetAccessPolicyParams are the parameters for the Get Access Policy API
type GetAccessPolicyParams struct {
	Version *string `description:"Optional - if not specified, the latest policy will be returned" query:"policy_version"`
}

// OpenAPI Summary: Get Access Policy
// OpenAPI Tags: Access Policies
// OpenAPI Description: This endpoint gets an access policy.
func (h handler) getAccessPolicy(ctx context.Context, id uuid.UUID, req GetAccessPolicyParams) (*policy.AccessPolicy, int, []auditlog.Entry, error) {

	s := storage.MustCreateStorage(ctx)
	var ap *storage.AccessPolicy
	var err error

	if req.Version != nil {
		version, err := strconv.Atoi(*req.Version)
		if err != nil {
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}

		ap, err = s.GetAccessPolicyByVersion(ctx, id, version)
		if err != nil {
			return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}
	} else {
		ap, err = s.GetLatestAccessPolicy(ctx, id)
		if err != nil {
			return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}
	}

	clientAP := ap.ToClientModel()
	if err = validateAndPopulateAccessPolicyFields(ctx, s, clientAP); err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return clientAP, http.StatusOK, nil, nil
}

// OpenAPI Summary: Create Access Policy
// OpenAPI Tags: Access Policies
// OpenAPI Description: This endpoint creates an access policy.
func (h handler) createAccessPolicy(ctx context.Context, req tokenizer.CreateAccessPolicyRequest) (*policy.AccessPolicy, int, []auditlog.Entry, error) {
	if req.AccessPolicy.IsSystem {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "the IsSystem attribute cannot be set by the client")
	}

	s := storage.MustCreateStorage(ctx)
	if err := validateAndPopulateAccessPolicyFields(ctx, s, &req.AccessPolicy); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	ap := storage.AccessPolicy{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(req.AccessPolicy.ID),
		Name:                     req.AccessPolicy.Name,
		Description:              req.AccessPolicy.Description,
		PolicyType:               storage.InternalPolicyTypeFromClient(req.AccessPolicy.PolicyType),
		TagIDs:                   req.AccessPolicy.TagIDs,
		Version:                  0,
		IsAutogenerated:          req.AccessPolicy.IsAutogenerated,
		Metadata:                 storage.AccessPolicyMetadata{RequiredContext: req.AccessPolicy.RequiredContext},
		Thresholds:               storage.AccessPolicyThresholdsFromClient(req.AccessPolicy.Thresholds),
	}

	if err := convertClientPolicyComponentsToStoragePolicyComponents(req.AccessPolicy.Components, &ap); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	// if a matching policy can be loaded, we shouldn't use using create
	// TODO: would be nice to unify this with loadAccessPolicy but the logic is slightly different
	if ap.ID != uuid.Nil {
		if p, err := s.GetLatestAccessPolicy(ctx, req.AccessPolicy.ID); !errors.Is(err, sql.ErrNoRows) {

			if err != nil {
				return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
			}

			if ap.EqualsIgnoringNilID(p) {
				return nil,
					http.StatusConflict, nil,
					ucerr.WrapWithFriendlyStructure(
						jsonclient.Error{StatusCode: http.StatusConflict},
						jsonclient.SDKStructuredError{
							Error:     "This policy already exists",
							ID:        p.ID,
							Identical: true,
						},
					)
			}

			return nil,
				http.StatusConflict, nil,
				ucerr.WrapWithFriendlyStructure(
					jsonclient.Error{StatusCode: http.StatusConflict},
					jsonclient.SDKStructuredError{
						Error: "A policy already exists with the same ID",
						ID:    req.AccessPolicy.ID,
					},
				)
		}
	}

	if p, err := s.GetAccessPolicyByName(ctx, req.AccessPolicy.Name); !errors.Is(err, sql.ErrNoRows) {

		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}

		if ap.EqualsIgnoringNilID(p) {
			return nil,
				http.StatusConflict, nil,
				ucerr.WrapWithFriendlyStructure(
					jsonclient.Error{StatusCode: http.StatusConflict},
					jsonclient.SDKStructuredError{
						Error:     "This policy already exists",
						ID:        p.ID,
						Identical: true,
					},
				)
		}

		return nil, http.StatusConflict, nil,
			ucerr.WrapWithFriendlyStructure(
				jsonclient.Error{StatusCode: http.StatusConflict},
				jsonclient.SDKStructuredError{
					Error: fmt.Sprintf(`A policy with the name '%s' already exists`, req.AccessPolicy.Name),
					ID:    p.ID,
				},
			)
	}

	if ap.ID.IsNil() {
		ap.ID = uuid.Must(uuid.NewV4())
	}

	if ap.Metadata.RequiredContext == nil {
		ap.Metadata.RequiredContext = map[string]string{}
	}
	rc, err := examineSubPolicies(ctx, s, &ap, set.NewUUIDSet(), set.NewUUIDSet())
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}
	maps.Copy(ap.Metadata.RequiredContext, rc)

	authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}
	if err := SaveAccessPolicyWithAuthz(ctx, s, authzClient, &ap); err != nil {
		return nil, uchttp.SQLWriteErrorMapper(err), nil, ucerr.Wrap(err)
	}

	h.createEventTypesForAccessPolicy(ctx, ap.ID, ap.Version)
	clientAP := ap.ToClientModel()
	entries := auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), internal.AuditLogEventTypeCreateAccessPolicy, auditlog.Payload{"ID": ap.ID, "Name": ap.Name, "Version": ap.Version})
	if err = validateAndPopulateAccessPolicyFields(ctx, s, clientAP); err != nil {
		return nil, http.StatusInternalServerError, entries, ucerr.Wrap(err)
	}

	return clientAP, http.StatusCreated, entries, nil
}

// OpenAPI Summary: Update Access Policy
// OpenAPI Tags: Access Policies
// OpenAPI Description: This endpoint updates a specified access policy.
func (h handler) updateAccessPolicy(ctx context.Context, id uuid.UUID, req tokenizer.UpdateAccessPolicyRequest) (*policy.AccessPolicy, int, []auditlog.Entry, error) {
	if req.AccessPolicy.ID != uuid.Nil && req.AccessPolicy.ID != id {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "URL ID doesn't match request body ID")
	}

	s := storage.MustCreateStorage(ctx)
	if err := validateAndPopulateAccessPolicyFields(ctx, s, &req.AccessPolicy); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	ap, err := s.GetLatestAccessPolicy(ctx, id)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	if ap.IsSystem {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "system policies cannot be updated")
	}

	if ap.IsSystem != req.AccessPolicy.IsSystem {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "the IsSystem attribute cannot be updated by the client")
	}

	updateNames := false
	if ap.Name != req.AccessPolicy.Name {
		// verify that the new name isn't already in use
		ap, err := s.GetAccessPolicyByName(ctx, req.AccessPolicy.Name)
		if ap != nil {
			if id != ap.ID {
				return nil, http.StatusConflict, nil,
					ucerr.WrapWithFriendlyStructure(
						jsonclient.Error{StatusCode: http.StatusConflict},
						jsonclient.SDKStructuredError{
							Error: fmt.Sprintf(`A policy with the name '%s' already exists`, req.AccessPolicy.Name),
							ID:    ap.ID,
						},
					)
			}

			// if the IDs are the same, this is just a case change
			// to the name, which we should allow
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
		updateNames = true
	}

	if ap.Version != req.AccessPolicy.Version {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "Access policy version in request does not match current version")
	}

	ap.Name = req.AccessPolicy.Name
	ap.Description = req.AccessPolicy.Description
	ap.PolicyType = storage.InternalPolicyTypeFromClient(req.AccessPolicy.PolicyType)
	ap.TagIDs = req.AccessPolicy.TagIDs
	ap.Thresholds = storage.AccessPolicyThresholdsFromClient(req.AccessPolicy.Thresholds)
	ap.IsAutogenerated = req.AccessPolicy.IsAutogenerated
	ap.Metadata.RequiredContext = req.AccessPolicy.RequiredContext
	if ap.Metadata.RequiredContext == nil {
		ap.Metadata.RequiredContext = map[string]string{}
	}
	ap.Version++

	if err := convertClientPolicyComponentsToStoragePolicyComponents(req.AccessPolicy.Components, ap); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	rc, err := examineSubPolicies(ctx, s, ap, set.NewUUIDSet(), set.NewUUIDSet())
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}
	maps.Copy(ap.Metadata.RequiredContext, rc)

	// No need to SaveWithAuthz here because this is an update so the edge already exists
	if err := s.SaveAccessPolicy(ctx, ap); err != nil {
		return nil, uchttp.SQLWriteErrorMapper(err), nil, ucerr.Wrap(err)
	}

	if updateNames {
		// change the policy name for all versions
		aps, err := s.GetAllAccessPolicyVersions(ctx, id)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
		for _, ap := range aps {
			if ap.Name != req.AccessPolicy.Name {
				ap.Name = req.AccessPolicy.Name
				if err := s.PriorVersionSaveAccessPolicy(ctx, &ap); err != nil {
					return nil, uchttp.SQLWriteErrorMapper(err), nil, ucerr.Wrap(err)
				}
			}
		}
	}

	entries := auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), internal.AuditLogEventTypeUpdateAccessPolicy,
		auditlog.Payload{"ID": id, "Name": ap.Name, "Version": ap.Version})

	clientAP := ap.ToClientModel()
	if err = validateAndPopulateAccessPolicyFields(ctx, s, clientAP); err != nil {
		return nil, http.StatusInternalServerError, entries, ucerr.Wrap(err)
	}

	return clientAP, http.StatusOK, entries, nil
}

// DeleteAccessPolicyParams are the parameters for the Delete Access Policy API
type DeleteAccessPolicyParams struct {
	Version *string `description:"Required - specifies the version of the policy to delete, use 'all' to delete all versions" query:"policy_version"`
}

// OpenAPI Summary: Delete Access Policy
// OpenAPI Tags: Access Policies
// OpenAPI Description: This endpoint deletes an access policy by ID.
func (h handler) deleteAccessPolicy(ctx context.Context, id uuid.UUID, req DeleteAccessPolicyParams) (int, []auditlog.Entry, error) {

	if req.Version == nil || *req.Version == "" {
		return http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "version is required")
	}

	s := storage.MustCreateStorage(ctx)
	authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
	if err != nil {
		return http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	var ap *storage.AccessPolicy
	if strings.ToLower(*req.Version) == "all" {
		ap, err = s.GetLatestAccessPolicy(ctx, id)
		if err != nil {
			return uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}
		if ap.IsSystem {
			return http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "system policies cannot be deleted")
		}

		if err := DeleteAllAccessPolicyVersionsWithAuthz(ctx, s, authzClient, ap); err != nil {
			if errors.Is(err, storage.ErrStillInUse) {
				return http.StatusConflict, nil, ucerr.Wrap(err)
			}
			return http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}

		for i := 0; i <= ap.Version; i++ {
			if err := h.deleteEventsForAccessPolicy(ctx, id, i); err != nil {
				uclog.Errorf(ctx, "error deleting events for access policy: %v", err)
			}
		}

	} else {
		versionInt, err := strconv.Atoi(*req.Version)
		if err != nil {
			return http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "version must be an integer or 'all'")
		}

		ap, err = s.GetAccessPolicyByVersion(ctx, id, versionInt)
		if err != nil {
			return uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}
		if ap.IsSystem {
			return http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "system policies cannot be deleted")
		}

		if err := DeleteAccessPolicyWithAuthz(ctx, s, authzClient, ap); err != nil {
			if errors.Is(err, storage.ErrStillInUse) {
				return http.StatusConflict, nil, ucerr.Wrap(err)
			}
			return http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}

		if err := h.deleteEventsForAccessPolicy(ctx, id, ap.Version); err != nil {
			uclog.Errorf(ctx, "error deleting events for access policy: %v", err)
		}
	}

	return http.StatusNoContent, auditlog.NewEntryArray(auth.GetAuditLogActor(ctx), internal.AuditLogEventTypeDeleteAccessPolicy,
		auditlog.Payload{"ID": id, "Name": ap.Name, "Version": *req.Version}), nil
}

// OpenAPI Summary: Test Access Policy
// OpenAPI Tags: Access Policies
// OpenAPI Description: This endpoint tests an access policy. It receives test context and returns a boolean indicating whether access is allowed or denied.
func (h handler) testAccessPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req tokenizer.TestAccessPolicyRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	uclog.Debugf(ctx, "testing access policy: %+v", req)

	authZClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	s := storage.MustCreateStorage(ctx)
	allowed, consoleOutput, err := ExecuteAccessPolicy(ctx, &req.AccessPolicy, req.Context, authZClient, s)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	uclog.Debugf(ctx, "got result: %v", allowed)

	jsonapi.Marshal(w, tokenizer.TestAccessPolicyResponse{Allowed: allowed, Debug: map[string]any{"console": consoleOutput}})
}

// BuildBaseAPContext builds an AccessPolicyContext from the request context, leaving the user unset
func BuildBaseAPContext(
	ctx context.Context,
	clientContext policy.ClientContext,
	action policy.Action,
	purposes ...storage.Purpose,
) policy.AccessPolicyContext {
	ip := "unknown"
	if sec := security.GetSecurityStatus(ctx); sec != nil {
		ip = sec.IPs[0]
	}

	purposeNames := make([]string, 0, len(purposes))
	for _, p := range purposes {
		purposeNames = append(purposeNames, p.Name)
	}

	var claims jwt.MapClaims
	if j := auth.GetRawJWT(ctx); j != "" {
		var err error
		claims, err = ucjwt.ParseJWTClaimsUnverified(j)
		if err != nil {
			uclog.Errorf(ctx, "error parsing claims: %v", err)
		}
	}

	return policy.AccessPolicyContext{
		Server: policy.ServerContext{
			IPAddress:    ip,
			Action:       action,
			PurposeNames: purposeNames,
			Claims:       claims,
		},
		Client: clientContext,
	}
}

// ExecuteAccessPolicy executes an access policy
func ExecuteAccessPolicy(
	ctx context.Context,
	ap *policy.AccessPolicy,
	accessPolicyContext policy.AccessPolicyContext,
	authzClient *authz.Client,
	s *storage.Storage,
) (bool, string, error) {
	apte, err := newAccessPolicyTemplateExecutor(authzClient, s)
	if err != nil {
		return false, "", ucerr.Wrap(err)
	}
	defer apte.cleanup()

	allowed, err := executeAccessPolicy(ctx, ap, accessPolicyContext, apte, s)
	if err != nil {
		return false, apte.getConsoleOutput(), ucerr.Wrap(err)
	}

	return allowed, apte.getConsoleOutput(), nil
}

func executeAccessPolicy(
	ctx context.Context,
	ap *policy.AccessPolicy,
	accessPolicyContext policy.AccessPolicyContext,
	apte *accessPolicyTemplateExecutor,
	s *storage.Storage,
) (bool, error) {
	// Request may contain PII so don't log it in prod
	uclog.DebugfPII(ctx, "executing access policy: %+v", ap)

	start := time.Now().UTC()
	defer logAPDuration(ctx, ap.ID, ap.Version, start)

	if accessPolicyContext.Client == nil {
		accessPolicyContext.Client = policy.ClientContext{}
	}
	bs, err := json.Marshal(accessPolicyContext)
	if err != nil {
		return false, ucerr.Wrap(err)
	}
	logAPCall(ctx, ap.ID, ap.Version)

	allowed := false

	for _, component := range ap.Components {
		if component.Policy != nil {
			p, err := s.GetLatestAccessPolicy(ctx, component.Policy.ID)
			if err != nil {
				return false, ucerr.Wrap(err)
			}

			allowed, err = executeAccessPolicy(ctx, p.ToClientModel(), accessPolicyContext, apte, s)
			if err != nil {
				return false, ucerr.Wrap(err)
			}

		} else if component.Template != nil {

			template, err := s.GetLatestAccessPolicyTemplate(ctx, component.Template.ID)
			if err != nil {
				return false, ucerr.Wrap(err)
			}

			allowed, err = apte.execute(ctx, template, accessPolicyContext, string(bs), component.TemplateParameters)
			if err != nil {
				return false, ucerr.Wrap(err)
			}

		} else {
			logAPError(ctx, ap.ID, ap.Version)
			return false, ucerr.Errorf("unknown component type: %v", component)

		}

		if allowed {
			if ap.PolicyType == policy.PolicyTypeCompositeOr {
				// access policy passes if any component passes
				break
			}
		} else {
			if ap.PolicyType == policy.PolicyTypeCompositeAnd {
				// access policy does not pass if any component does not pass
				break
			}
		}
	}

	logAPResult(ctx, ap.ID, ap.Version, allowed)
	return allowed, nil
}
