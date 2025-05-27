package authn

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/idp"
	"userclouds.com/idp/config"
	"userclouds.com/idp/internal/constants"
	"userclouds.com/idp/internal/shared"
	"userclouds.com/idp/internal/storage"
	userstoreInternal "userclouds.com/idp/internal/userstore"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
)

type handler struct {
	companyConfigStorage *companyconfig.Storage
	searchUpdateConfig   *config.SearchUpdateConfig
}

// NewHandler returns an authn handler
func NewHandler(cfg *config.Config, searchUpdateConfig *config.SearchUpdateConfig, companyConfigStorage *companyconfig.Storage) http.Handler {
	h := &handler{
		companyConfigStorage: companyConfigStorage,
		searchUpdateConfig:   searchUpdateConfig,
	}
	hb := builder.NewHandlerBuilder()

	handlerBuilder(hb, h)

	hb.HandleFunc("/mfacode", h.HandleMFACodeRequest).
		HandleFunc("/mfaclearprimarychannel", h.HandleMFAClearPrimaryChannelRequest).
		HandleFunc("/mfacreatechannel", h.HandleMFACreateChannelRequest).
		HandleFunc("/mfadeletechannel", h.HandleMFADeleteChannelRequest).
		HandleFunc("/mfagetchannels", h.HandleMFAGetChannelsRequest).
		HandleFunc("/mfamakeprimarychannel", h.HandleMFAMakePrimaryChannelRequest).
		HandleFunc("/mfareissuerecoverycode", h.HandleMFAReissueRecoveryCodeRequest).
		HandleFunc("/mfaresponse", h.HandleMFAResponse)

	hb.HandleFunc("/uplogin", h.UsernamePasswordLoginHandler)

	// TODO: should this just be part of updateUser? Or a nested handler on /users/<id>?
	hb.HandleFunc("/upupdate", h.UpdateUsernamePasswordHandler)

	hb.HandleFunc("/addauthntouser", h.AddAuthnToUserHandler)

	// baseprofileswithauthn collection returns base profiles of specified users with authn info (these are used by plex)
	hb.CollectionHandler("/baseprofileswithauthn").
		GetOne(h.getUserBaseProfileWithAuthN).
		GetAll(h.listUserBaseProfilesWithAuthN).
		WithAuthorizer(h.newRoleBasedAuthorizer())

	return hb.Build()
}

//go:generate genhandler /authn collection,User,h.newRoleBasedAuthorizer(),/users collection,UserBaseProfile,h.newRoleBasedAuthorizer(),/baseprofiles

func (h *handler) newRoleBasedAuthorizer() uchttp.CollectionAuthorizer {
	return &uchttp.MethodAuthorizer{
		GetAllF: func(r *http.Request) error {
			return ucerr.Wrap(h.ensureTenantMember(r, false))
		},
		GetOneF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(r, false))
		},
		PostF: func(r *http.Request) error {
			return ucerr.Wrap(h.ensureTenantMember(r, true))
		},
		PutF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(r, true))
		},
		DeleteF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(r, true))
		},
		DeleteAllF: func(r *http.Request) error {
			return ucerr.Wrap(h.ensureTenantMember(r, true))
		},
		NestedF: func(r *http.Request, _ uuid.UUID) error {
			return ucerr.Wrap(h.ensureTenantMember(r, false))
		},
	}
}

func (h *handler) ensureTenantMember(r *http.Request, adminOnly bool) error {
	return nil // TODO: figure out how to do this w/o calling authz service
}

// UsernamePasswordLoginHandler allows a user to log in with username & password.
// NOTE: this is actually implementing an OIDC method, and returns OIDC-friendly
// error responses.
func (h *handler) UsernamePasswordLoginHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantAuthn := MustGetTenantAuthn(ctx)

	var req idp.UsernamePasswordLoginRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.NewRequestError(err))
		return
	}

	uclog.Debugf(ctx, "username '%s' logging in with password credentials", req.Username)

	// TODO: this isn't the right long term solution, but if you somehow try to login against IDP that
	// has incompletely-synced users, we shouldn't allow the placeholder password to be used.
	if req.Password == idp.PlaceholderPassword {
		uclog.Errorf(ctx, "this should never happen - user '%s' has placeholder password", req.Username)
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("login against follower / failed sync"), jsonapi.Code(http.StatusBadRequest))
		return
	}

	user, err := tenantAuthn.Manager.CheckUsernamePassword(ctx, req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrUsernamePasswordIncorrect) {
			// OAuth spec says to return 399 and this error payload (instead of 401).
			// https://datatracker.ietf.org/doc/html/rfc6749#section-5.2
			// "The provided authorization grant (e.g., authorization
			// code, resource owner credentials) ... is invalid."
			// See also: https://stackoverflow.com/questions/22586825/oauth-2-0-why-does-the-authorization-server-return-400-instead-of-401-when-the
			jsonapi.MarshalError(ctx, w, ucerr.ErrIncorrectUsernamePassword)
		} else if errors.Is(err, ErrUsernameNotFound) {
			// NOTE: for security, we should not expose details of invalid user vs. invalid password to end users,
			// but this method (eventually) validates a client secret which means it's safer to expose details.
			uclog.Debugf(ctx, "username & password NOT in our DB")
			jsonapi.MarshalError(ctx, w, ucerr.ErrIncorrectUsernamePassword)
		} else {
			uclog.Debugf(ctx, "error checking username/password: %v", err)
			jsonapi.MarshalError(ctx, w, ucerr.NewServerError(err))
		}
		return
	}

	uclog.Debugf(ctx, "password matches in our DB, checking whether MFA is required")

	mfaRequired, channels, evaluateChannels, err := h.getMFASettings(ctx, tenantAuthn, req.ClientID, user.ID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.NewServerError(err))
		return
	}

	resp := idp.LoginResponse{
		UserID:                       user.ID,
		EvaluateSupportedMFAChannels: evaluateChannels,
	}

	if mfaRequired || channels.HasPrimaryChannel() {
		uclog.Debugf(ctx, "MFA is required for user '%v'", user.ID)

		mfaReq := storage.NewMFARequest(user.ID, channels.ChannelTypes)
		if err := tenantAuthn.ConfigStorage.SaveMFARequest(ctx, mfaReq); err != nil {
			jsonapi.MarshalError(ctx, w, ucerr.NewServerError(err))
			return
		}

		resp.Status = idp.LoginStatusMFARequired
		resp.MFAToken = mfaReq.ID.String()
		resp.SupportedMFAChannels = channels
	} else {
		uclog.Debugf(ctx, "MFA is not required for user '%v'", user.ID)

		resp.Status = idp.LoginStatusSuccess
	}

	jsonapi.Marshal(w, resp)
}

// UpdateUsernamePasswordHandler updates the u/p on the secondary IDP
func (h *handler) UpdateUsernamePasswordHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantAuthn := MustGetTenantAuthn(ctx)

	var request idp.UpdateUsernamePasswordRequest
	if err := jsonapi.Unmarshal(r, &request); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := tenantAuthn.Manager.UpdateUsernamePassword(ctx, request.Username, request.Password); err != nil {
		// This will return 404 if user is not found
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, nil, jsonapi.Code(http.StatusNoContent))
}

// AddAuthnToUserHandler handles requests to add an authentication method to a user
func (h *handler) AddAuthnToUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantAuthn := MustGetTenantAuthn(ctx)

	var req idp.AddAuthnToUserRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if req.AuthnType == idp.AuthnTypePassword {
		if err := tenantAuthn.Manager.AddPasswordAuthnToUser(ctx, req.UserID, req.Username, req.Password); err != nil {
			// Return 404 if user is not found
			jsonapi.MarshalSQLError(ctx, w, err)
			return
		}
	} else if req.AuthnType == idp.AuthnTypeOIDC {
		if err := tenantAuthn.Manager.AddOIDCAuthnToUser(ctx, req.UserID, req.OIDCProvider, req.OIDCIssuerURL, req.OIDCSubject); err != nil {
			// Return 404 if user is not found
			jsonapi.MarshalSQLError(ctx, w, err)
			return
		}
	} else {
		jsonapi.MarshalError(ctx, w, ucerr.New("unsupported authn type"))
		return
	}

	jsonapi.Marshal(w, nil, jsonapi.Code(http.StatusNoContent))
}

func newGetUserResponse(
	ctx context.Context,
	user *storage.BaseUser,
	reg region.DataRegion,
	accessPrimaryDBOnly bool,
) (*idp.UserResponse, []auditlog.Entry, error) {

	userResponse := &idp.UserResponse{
		ID:             user.ID,
		UpdatedAt:      user.Updated.Unix(),
		OrganizationID: user.OrganizationID,
	}

	userData, entries, err := userstoreInternal.GetUsers(ctx, true, reg, accessPrimaryDBOnly, user.ID.String())
	if err != nil {
		return nil, entries, ucerr.Wrap(err)
	}

	var profile userstore.Record
	if err = json.Unmarshal([]byte(userData[0]), &profile); err != nil {
		return nil, entries, ucerr.Wrap(err)
	}

	tenantAuthn := MustGetTenantAuthn(ctx)
	cm, err := storage.NewUserstoreColumnManager(ctx, tenantAuthn.ConfigStorage)
	if err != nil {
		return nil, entries, ucerr.Wrap(err)
	}
	columns := cm.GetColumns()

	// We iterate over all columns because we assume that the GetUserAccessor uses all columns
	// If that ever changes, then we should use getAccessorColumns() to get the columns instead
	for _, c := range columns {
		if _, ok := profile[c.Name]; !ok {
			profile[c.Name] = nil
		}
	}

	userResponse.Profile = profile

	return userResponse, entries, nil
}

func newGetUserBaseProfileResponse(ctx context.Context,
	user *storage.BaseUser,
	reg region.DataRegion,
	accessPrimaryDBOnly bool,
) (*idp.UserBaseProfileResponse, []auditlog.Entry, error) {
	userResponse, entries, err := newGetUserResponse(ctx, user, reg, accessPrimaryDBOnly)
	if err != nil {
		return nil, entries, ucerr.Wrap(err)
	}

	resp := &idp.UserBaseProfileResponse{
		ID:             userResponse.ID.String(),
		UpdatedAt:      userResponse.UpdatedAt,
		OrganizationID: userResponse.OrganizationID.String(),
		UserBaseProfile: idp.UserBaseProfile{
			Email:         userResponse.Profile.StringValue("email"),
			EmailVerified: userResponse.Profile.BoolValue("email_verified"),
			Name:          userResponse.Profile.StringValue("name"),
			Nickname:      userResponse.Profile.StringValue("nickname"),
			Picture:       userResponse.Profile.StringValue("picture"),
		},
	}

	return resp, entries, nil
}

// OpenAPI Summary: Create User
// OpenAPI Tags: Users
// OpenAPI Description: This endpoint creates a user.
func (h *handler) createUser(ctx context.Context, req idp.CreateUserAndAuthnRequest) (*idp.UserResponse, int, []auditlog.Entry, error) {
	tenantAuthn := MustGetTenantAuthn(ctx)

	// Check that the organization ID for the user being created is valid
	organizationID, err := shared.ValidateUserOrganizationForRequest(ctx, req.OrganizationID)
	if err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	if !tenantAuthn.UseOrganizations {
		req.OrganizationID = tenantAuthn.CompanyID
	} else {
		if organizationID.IsNil() {
			return nil, http.StatusBadRequest, nil, ucerr.Errorf("organization ID is required")
		}
		req.OrganizationID = organizationID
	}

	var user *storage.BaseUser
	var code int
	switch req.AuthnType {
	case "": // TODO: I don't love this setup, but it keeps the logic consistent until we have time to refactor
		user, code, err = tenantAuthn.Manager.CreateUser(ctx, h.searchUpdateConfig, req)
	case idp.AuthnTypePassword:
		user, code, err = tenantAuthn.Manager.CreateUserWithPassword(ctx, h.searchUpdateConfig, req)
	case idp.AuthnTypeOIDC:
		user, code, err = tenantAuthn.Manager.CreateUserWithOIDCLogin(ctx, h.searchUpdateConfig, req)
	default:
		return nil, http.StatusBadRequest, nil, ucerr.Errorf("invalid authn type in request: %v", req.AuthnType)
	}

	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		case http.StatusForbidden:
			return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	userResponse, entries, err := newGetUserResponse(ctx, user, req.Region, true)
	if err != nil {
		return nil, http.StatusInternalServerError, entries, ucerr.Wrap(err)
	}

	return userResponse, http.StatusCreated, entries, nil
}

// OpenAPI Summary: Update User
// OpenAPI Tags: Users
// OpenAPI Description: This endpoint updates a specified user.
func (h *handler) updateUser(ctx context.Context, id uuid.UUID, req idp.UpdateUserRequest) (*idp.UserResponse, int, []auditlog.Entry, error) {
	// TODO: we are treating this like a PATCH right now and supporting partial updates
	tenantAuthn := MustGetTenantAuthn(ctx)

	var user *storage.BaseUser
	var reg region.DataRegion
	var err error
	if req.Region != "" {
		// If a region is specified, get the user from that region
		user, err = tenantAuthn.UserMultiRegionStorage.GetBaseUserFromRegion(ctx, req.Region, id, false)
		reg = req.Region
	} else {
		// If no region is specified, try to get the user from all regions
		user, reg, err = tenantAuthn.UserMultiRegionStorage.GetBaseUser(ctx, id, false)
	}
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	if _, err = shared.ValidateUserOrganizationForRequest(ctx, user.OrganizationID); err != nil {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	if req.Profile != nil {
		cm, err := storage.NewUserstoreColumnManager(ctx, tenantAuthn.ConfigStorage)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
		if err := coerceRecordToSchema(cm, req.Profile); err != nil {
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}

		mutator, err := tenantAuthn.ConfigStorage.GetLatestMutator(ctx, constants.UpdateUserMutatorID)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}

		values := map[string]idp.ValueAndPurposes{}

		for _, columnID := range mutator.ColumnIDs {
			c := cm.GetColumnByID(columnID)
			if c == nil {
				return nil, http.StatusInternalServerError, nil, ucerr.Errorf("column %v not found", columnID)
			}

			if value, found := req.Profile[c.Name]; found {
				// If the specified value is nil, remove the operational purpose
				// for any existing value. Otherwise, add the operational purpose
				// for the specified value.

				if value == nil {
					if c.Attributes.Constraints.PartialUpdates {
						values[c.Name] = idp.ValueAndPurposes{
							ValueDeletions: idp.MutatorColumnCurrentValue,
							PurposeDeletions: []userstore.ResourceID{
								{ID: constants.OperationalPurposeID},
							},
						}
					} else {
						values[c.Name] = idp.ValueAndPurposes{
							Value: idp.MutatorColumnCurrentValue,
							PurposeDeletions: []userstore.ResourceID{
								{ID: constants.OperationalPurposeID},
							},
						}
					}
				} else if c.Attributes.Constraints.PartialUpdates {
					values[c.Name] = idp.ValueAndPurposes{
						ValueAdditions: value,
						PurposeAdditions: []userstore.ResourceID{
							{ID: constants.OperationalPurposeID},
						},
					}
				} else {
					values[c.Name] = idp.ValueAndPurposes{
						Value: value,
						PurposeAdditions: []userstore.ResourceID{
							{ID: constants.OperationalPurposeID},
						},
					}
				}
			} else {
				// The column is unspecified, so do not change any values or
				// associated purposes.

				if c.Attributes.Constraints.PartialUpdates {
					values[c.Name] = idp.ValueAndPurposes{
						ValueAdditions: idp.MutatorColumnCurrentValue,
					}
				} else {
					values[c.Name] = idp.ValueAndPurposes{
						Value: idp.MutatorColumnCurrentValue,
					}
				}
			}
		}

		authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}

		userIDs, code, err := userstoreInternal.ExecuteMutator(
			ctx,
			idp.ExecuteMutatorRequest{
				MutatorID:      constants.UpdateUserMutatorID,
				Context:        policy.ClientContext{},
				SelectorValues: []any{id},
				RowData:        values,
				Region:         reg,
			},
			tenantAuthn.ID,
			authzClient,
			h.searchUpdateConfig,
		)
		if err != nil {
			switch code {
			case http.StatusBadRequest:
				return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
			case http.StatusConflict:
				return nil, http.StatusConflict, nil, ucerr.Wrap(err)
			default:
				return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
			}
		}
		if len(userIDs) != 1 {
			return nil, http.StatusInternalServerError, nil, ucerr.Errorf("expected exactly one value from mutator")
		}
	}

	user, err = tenantAuthn.UserMultiRegionStorage.GetBaseUserFromRegion(ctx, reg, id, true)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	userResponse, entries, err := newGetUserResponse(ctx, user, reg, true)
	if err != nil {
		return nil, http.StatusInternalServerError, entries, ucerr.Wrap(err)
	}

	return userResponse, http.StatusOK, entries, nil
}

// OpenAPI Summary: Delete User
// OpenAPI Tags: Users
// OpenAPI Description: This endpoint deletes a user by ID.
func (h *handler) deleteUser(ctx context.Context, id uuid.UUID, _ url.Values) (int, []auditlog.Entry, error) {
	code, err := userstoreInternal.DeleteUser(ctx, h.searchUpdateConfig, id)
	if err != nil {
		switch code {
		case http.StatusForbidden:
			return http.StatusForbidden, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return http.StatusNoContent, nil, nil
}

func (h *handler) getUserByOIDCAuthn(ctx context.Context, req UsersPaginatedFilters) (*idp.UserBaseProfileAndAuthnResponse, int, []auditlog.Entry, error) {
	tenantAuthn := MustGetTenantAuthn(ctx)

	// we use ProviderTypeUnsupported as a sentinel value since you might actually
	// want to filter by none some day, and we let an empty string mean do-not-filter
	// TODO: there really probably should be a better enum value for do-not-filter?
	provider := oidc.ProviderTypeUnsupported
	if prov := req.ProviderFilter; prov != nil {
		if err := provider.UnmarshalText([]byte(*prov)); err != nil {
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}
	}
	if provider == oidc.ProviderTypeUnsupported {
		return nil, http.StatusBadRequest, nil, ucerr.Friendlyf(nil, "must provide valid 'provider' to filter on")
	}

	issuerURLFilter := ""
	if req.IssuerURLFilter != nil {
		issuerURLFilter = *req.IssuerURLFilter
	}

	subjectFilter := ""
	if req.SubjectFilter != nil {
		subjectFilter = *req.SubjectFilter
	}

	authn, err := tenantAuthn.ConfigStorage.GetOIDCAuthnForSubject(ctx, provider, issuerURLFilter, subjectFilter)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	user, reg, err := tenantAuthn.UserMultiRegionStorage.GetBaseUser(ctx, authn.UserID, false)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	if _, err := shared.ValidateUserOrganizationForRequest(ctx, user.OrganizationID); err != nil && !checkCrossOrgLoginApp(ctx, tenantAuthn.CompanyID, user) {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	resp, entries, err := newGetUserBaseProfileResponse(ctx, user, reg, false)
	if err != nil {
		return nil, http.StatusInternalServerError, entries, ucerr.Wrap(err)
	}

	respWithAuthN, err := addAuthNToUserBaseProfileResponse(ctx, tenantAuthn.ConfigStorage, resp, authn.UserID)
	if err != nil {
		return nil, http.StatusInternalServerError, entries, ucerr.Wrap(err)
	}

	return respWithAuthN, http.StatusOK, entries, nil
}

func (h *handler) getUsersByEmail(ctx context.Context, req UsersPaginatedFilters) ([]idp.UserBaseProfileAndAuthnResponse, int, []auditlog.Entry, error) {

	// TODO: figure out organizations and also if this endpoint should continue to exist

	tenantAuthn := MustGetTenantAuthn(ctx)

	emailFilter := ""
	if req.EmailFilter != nil {
		emailFilter = *req.EmailFilter
	}
	users, err := tenantAuthn.UserMultiRegionStorage.ListUsersForEmail(ctx, tenantAuthn.ConfigStorage, emailFilter)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	authnType := idp.AuthnTypeAll
	if req.AuthnTypeFilter != nil {
		authnType = idp.AuthnType(*req.AuthnTypeFilter)
	}
	if err := authnType.Validate(); err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	entries := []auditlog.Entry{}
	resp := []idp.UserBaseProfileAndAuthnResponse{}
	for _, user := range users {
		if authnType != idp.AuthnTypeAll {
			hasAuthn, err := tenantAuthn.ConfigStorage.UserHasAuthnType(ctx, user.ID, authnType)
			if err != nil {
				return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
			}
			if !hasAuthn {
				continue
			}
		}
		userResponse, e, err := newGetUserBaseProfileResponse(ctx, &user, "", false) // TODO: this could be optimized by keeping track of each user's data region
		entries = append(entries, e...)
		if err != nil {
			return nil, http.StatusInternalServerError, entries, ucerr.Wrap(err)
		}
		respWithAuthN, err := addAuthNToUserBaseProfileResponse(ctx, tenantAuthn.ConfigStorage, userResponse, user.ID)
		if err != nil {
			return nil, http.StatusInternalServerError, entries, ucerr.Wrap(err)
		}
		resp = append(resp, *respWithAuthN)
	}

	return resp, http.StatusOK, entries, nil
}

// UsersPaginatedFilters are the filters for the ListUsers API
type UsersPaginatedFilters struct {
	AuthnTypeFilter *string `description:"Used alongside EmailFilter to filter users based on email and authentication type. AuthNType can be password (for username/password authentication), social (for authentication via an OIDC provider, like Facebook) or all (for both)." query:"authn_type"`
	EmailFilter     *string `description:"Filters users based on email address" query:"email"`
	IssuerURLFilter *string `description:"Used alongside ProviderFilter to filter users based on OIDC provider" query:"issuer_url"`
	ProviderFilter  *string `description:"Used alongside IssuerURLFilter to filter users based on OIDC provider" query:"provider"`
	SubjectFilter   *string `description:"Used alongside ProviderFilter and IssuerURLFilter to filter users based on OIDC provider & subject ID" query:"subject"`
}

func (f UsersPaginatedFilters) hasEmailFilter() bool {
	return f.EmailFilter != nil
}

func (f UsersPaginatedFilters) hasOIDCFilter() bool {
	return f.ProviderFilter != nil && f.IssuerURLFilter != nil && f.SubjectFilter != nil
}

// Validate validates the filters
func (f UsersPaginatedFilters) Validate() error {
	// There are 2 filter modes allowed:
	// 1. OIDC Provider + OIDC IssuerURL + OIDC Subject - narrows down to a single user associated with an OIDC IDP (e.g. Google)
	//    with a unique OIDC subject.
	// 2. Email and (optional) authn filter - narrows down to a single or few users with an email address, optionally
	//    further constrained to a single authn method (username+password or oidc).
	// We don't allow mixing or matching.
	hasEmailFilter := f.hasEmailFilter()
	if f.AuthnTypeFilter != nil && !hasEmailFilter {
		return ucerr.Friendlyf(nil, "'authn_type' may only be specified with 'email'")
	}
	hasOIDCFilter := f.hasOIDCFilter()
	if !hasOIDCFilter && (f.ProviderFilter != nil || f.IssuerURLFilter != nil || f.SubjectFilter != nil) {
		return ucerr.Friendlyf(nil, "'provider', 'issuer_url', and 'subject' must all be specified, or none specified")
	}
	if hasOIDCFilter && hasEmailFilter {
		return ucerr.Friendlyf(nil, "'provider', 'issuer_url', and 'subject' filtering is incompatible with 'email' and 'authn_type'")
	}

	return nil
}

// ListUsersParams are the parameters for the ListUsers API
type ListUsersParams struct {
	pagination.QueryParams
	OrganizationID *string `description:"Filter the users based on an organization ID" query:"organization_id"`
}

// OpenAPI Summary: List Users
// OpenAPI Tags: Users
// OpenAPI Description: This endpoint returns a paginated list of users in a tenant. The list can be filtered to only include users inside a specified organization.
func (h *handler) listUsers(ctx context.Context, req ListUsersParams) (*idp.ListUsersResponse, int, []auditlog.Entry, error) {
	tenantAuthn := MustGetTenantAuthn(ctx)

	var pagerOptions []pagination.Option

	if req.OrganizationID != nil {
		orgID, err := uuid.FromString(*req.OrganizationID)
		if err != nil {
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}
		if _, err := shared.ValidateUserOrganizationForRequest(ctx, orgID); err != nil {
			return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
		}

		pagerOptions =
			append(pagerOptions,
				pagination.Filter(fmt.Sprintf("('organization_id',EQ,'%v')", orgID)))
	} else if err := h.checkEmployeeRequest(ctx); err != nil {
		orgID, err := shared.ValidateUserOrganizationForRequest(ctx, uuid.Nil)
		if err != nil {
			return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
		}

		pagerOptions =
			append(pagerOptions,
				pagination.Filter(fmt.Sprintf("(('organization_id',EQ,'%v'),OR,('organization_id',EQ,'00000000-0000-0000-0000-000000000000'))", orgID)))
	}

	pager, err := storage.NewBaseUserPaginatorFromQuery(req, pagerOptions...)
	if err != nil {
		return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
	}

	users, respFields, err := tenantAuthn.UserMultiRegionStorage.ListBaseUsersPaginated(ctx, *pager, false)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	userResps := []idp.UserResponse{}

	userIDMap := map[uuid.UUID]storage.BaseUser{}
	var userIDs []string
	for _, user := range users {
		userIDs = append(userIDs, user.ID.String())
		userIDMap[user.ID] = user
	}

	userData, entries, err := userstoreInternal.GetUsers(ctx, true, "", false, userIDs...)
	if err != nil {
		return nil, http.StatusInternalServerError, entries, ucerr.Wrap(err)
	}

	for _, value := range userData {
		var profile userstore.Record
		if err = json.Unmarshal([]byte(value), &profile); err != nil {
			return nil, http.StatusInternalServerError, entries, ucerr.Wrap(err)
		}

		uid, ok := profile["id"].(string)
		if !ok {
			// historically this has happened when the default GetUserAccessor had system columns accidentally removed :)
			return nil, http.StatusInternalServerError, entries, ucerr.New("id not in profile, potential bug in GetUserAccessor")
		}

		userID := uuid.Must(uuid.FromString(uid))
		user := userIDMap[userID]
		userResps = append(userResps, idp.UserResponse{
			ID:             user.ID,
			UpdatedAt:      user.Updated.Unix(),
			OrganizationID: user.OrganizationID,
			Profile:        profile,
		})
	}

	return &idp.ListUsersResponse{
		Data:           userResps,
		ResponseFields: *respFields,
	}, http.StatusOK, entries, nil
}

// OpenAPI Summary: Get User
// OpenAPI Tags: Users
// OpenAPI Description: This endpoint gets a user by ID.
func (h *handler) getUser(ctx context.Context, id uuid.UUID, _ url.Values) (*idp.UserResponse, int, []auditlog.Entry, error) {
	tenantAuthn := MustGetTenantAuthn(ctx)

	user, reg, err := tenantAuthn.UserMultiRegionStorage.GetBaseUser(ctx, id, false)
	if err != nil {
		return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
	}

	if _, err := shared.ValidateUserOrganizationForRequest(ctx, user.OrganizationID); err != nil && !checkCrossOrgLoginApp(ctx, tenantAuthn.CompanyID, user) {
		return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
	}

	userResponse, entries, err := newGetUserResponse(ctx, user, reg, false)
	if err != nil {
		return nil, http.StatusInternalServerError, entries, ucerr.Wrap(err)
	}

	return userResponse, http.StatusOK, entries, nil
}

func addAuthNToUserBaseProfileResponse(ctx context.Context, s *storage.Storage, userBaseProfile *idp.UserBaseProfileResponse, userID uuid.UUID) (*idp.UserBaseProfileAndAuthnResponse, error) {
	userResponse := &idp.UserBaseProfileAndAuthnResponse{
		ID:              userBaseProfile.ID,
		UpdatedAt:       userBaseProfile.UpdatedAt,
		OrganizationID:  userBaseProfile.OrganizationID,
		UserBaseProfile: userBaseProfile.UserBaseProfile,
	}

	// Query all AuthNs for the user
	passwordAuthns, err := s.ListPasswordAuthnsForUserID(ctx, userID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	oidcAuthns, err := s.ListOIDCAuthnsForUserID(ctx, userID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	userMFASettings, err := s.GetUserMFAConfiguration(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, ucerr.Wrap(err)
	}

	userResponse.Authns = []idp.UserAuthn{}
	for _, v := range passwordAuthns {
		userResponse.Authns = append(userResponse.Authns, idp.UserAuthn{
			AuthnType: idp.AuthnTypePassword,
			Username:  v.Username,
			// Obviously don't include password here; is there a safer way to ensure we don't?
		})
	}
	for _, v := range oidcAuthns {
		userResponse.Authns = append(userResponse.Authns,
			idp.UserAuthn{
				AuthnType:     idp.AuthnTypeOIDC,
				OIDCProvider:  v.Type,
				OIDCIssuerURL: v.OIDCIssuerURL,
				OIDCSubject:   v.OIDCSubject,
			})
	}
	if userMFASettings != nil {
		for _, c := range userMFASettings.MFAChannels.Channels {
			userResponse.MFAChannels =
				append(userResponse.MFAChannels,
					idp.UserMFAChannel{
						ChannelType:        c.ChannelType,
						ChannelDescription: c.GetUserDetailDescription(),
						Primary:            c.Primary,
						Verified:           c.Verified,
						LastVerified:       c.LastVerified,
					})
		}
	}

	return userResponse, nil
}

// ListUserBaseProfilesParams are the parameters for the ListUserBaseProfiles API
type ListUserBaseProfilesParams struct {
	UserIDs        []string `description:"Filter the users based on a list of user IDs" query:"user_ids"`
	OrganizationID *string  `description:"Filter the users based on an organization ID" query:"organization_id"`
	pagination.QueryParams
}

// OpenAPI Summary: List User Base Profiles
// OpenAPI Tags: Users
// OpenAPI Description: This endpoint returns a paginated list of user base profiles in a tenant. The list can be filtered to only include users inside a specified organization.
func (h *handler) listUserBaseProfiles(ctx context.Context, req ListUserBaseProfilesParams) (*idp.ListUserBaseProfilesResponse, int, []auditlog.Entry, error) {
	tenantAuthn := MustGetTenantAuthn(ctx)

	var userIDs []string
	respFields := pagination.ResponseFields{}

	checkMatchingLen := false
	if len(req.UserIDs) == 0 {
		checkMatchingLen = true
		var pagerOptions []pagination.Option

		if req.OrganizationID != nil {
			orgID, err := uuid.FromString(*req.OrganizationID)
			if err != nil {
				return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
			}
			if _, err := shared.ValidateUserOrganizationForRequest(ctx, orgID); err != nil {
				return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
			}

			pagerOptions =
				append(pagerOptions,
					pagination.Filter(fmt.Sprintf("('organization_id',EQ,'%v')", orgID)))
		} else if err := h.checkEmployeeRequest(ctx); err != nil {
			orgID, err := shared.ValidateUserOrganizationForRequest(ctx, uuid.Nil)
			if err != nil {
				return nil, http.StatusForbidden, nil, ucerr.Wrap(err)
			}

			pagerOptions =
				append(pagerOptions,
					pagination.Filter(fmt.Sprintf("(('organization_id',EQ,'%v'),OR,('organization_id',EQ,'00000000-0000-0000-0000-000000000000'))", orgID)))
		}

		pager, err := storage.NewBaseUserPaginatorFromQuery(req, pagerOptions...)
		if err != nil {
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		}

		users, rf, err := tenantAuthn.UserMultiRegionStorage.ListBaseUsersPaginated(ctx, *pager, false)
		if err != nil {
			return nil, uchttp.SQLReadErrorMapper(err), nil, ucerr.Wrap(err)
		}
		respFields = *rf

		for _, user := range users {
			userIDs = append(userIDs, user.ID.String())
		}
	} else {
		for _, id := range req.UserIDs {
			if _, err := uuid.FromString(id); err != nil {
				return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
			}
		}
		userIDs = req.UserIDs
	}

	userResps := []idp.UserBaseProfileResponse{}

	// we don't write ExecuteAccessor log entries for this endpoint, so we don't need to capture them
	userData, _, err := userstoreInternal.GetUsers(ctx, checkMatchingLen, "", false, userIDs...)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	for _, value := range userData {
		var profile userstore.Record
		if err = json.Unmarshal([]byte(value), &profile); err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}

		uid, ok := profile["id"].(string)
		if !ok {
			// historically this has happened when the default GetUserAccessor had system columns accidentally removed :)
			return nil, http.StatusInternalServerError, nil, ucerr.New("id not in profile, potential bug in GetUserAccessor")
		}
		updated, ok := profile["updated"].(string)
		if !ok {
			return nil, http.StatusInternalServerError, nil, ucerr.New("updated not in profile, potential bug in GetUserAccessor")
		}
		updatedAt, err := time.Parse(time.RFC3339, updated)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
		organizationID, ok := profile["organization_id"].(string)
		if !ok {
			return nil, http.StatusInternalServerError, nil, ucerr.New("organization_id not in profile, potential bug in GetUserAccessor")
		}

		userResps = append(userResps, idp.UserBaseProfileResponse{
			ID:             uid,
			UpdatedAt:      updatedAt.Unix(),
			OrganizationID: organizationID,
			UserBaseProfile: idp.UserBaseProfile{
				Email:         profile.StringValue("email"),
				EmailVerified: profile.BoolValue("email_verified"),
				Name:          profile.StringValue("name"),
				Nickname:      profile.StringValue("nickname"),
				Picture:       profile.StringValue("picture"),
			},
		})
	}

	return &idp.ListUserBaseProfilesResponse{
		Data:           userResps,
		ResponseFields: respFields,
	}, http.StatusOK, nil, nil
}

func (h *handler) getUserBaseProfileWithAuthN(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	tenantAuthn := MustGetTenantAuthn(ctx)

	user, reg, err := tenantAuthn.UserMultiRegionStorage.GetBaseUser(ctx, id, false)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	if _, err := shared.ValidateUserOrganizationForRequest(ctx, user.OrganizationID); err != nil && !checkCrossOrgLoginApp(ctx, tenantAuthn.CompanyID, user) {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusForbidden))
		return
	}

	userResponse, _, err := newGetUserBaseProfileResponse(ctx, user, reg, false)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	respWithAuthN, err := addAuthNToUserBaseProfileResponse(ctx, tenantAuthn.ConfigStorage, userResponse, user.ID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, respWithAuthN)
}

func (h *handler) listUserBaseProfilesWithAuthN(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	req := UsersPaginatedFilters{}
	if urlValues.Has("authn_type") && urlValues.Get("authn_type") != "null" {
		v := urlValues.Get("authn_type")
		req.AuthnTypeFilter = &v
	}
	if urlValues.Has("email") && urlValues.Get("email") != "null" {
		v := urlValues.Get("email")
		req.EmailFilter = &v
	}
	if urlValues.Has("issuer_url") && urlValues.Get("issuer_url") != "null" {
		v := urlValues.Get("issuer_url")
		req.IssuerURLFilter = &v
	}
	if urlValues.Has("provider") && urlValues.Get("provider") != "null" {
		v := urlValues.Get("provider")
		req.ProviderFilter = &v
	}
	if urlValues.Has("subject") && urlValues.Get("subject") != "null" {
		v := urlValues.Get("subject")
		req.SubjectFilter = &v
	}

	// TODO: filtering needs a lot more thought, most of these filters would require
	// SQL joins to do right (which we can allow) if we wanted to keep pagination working,
	// *or* we should have a separate non-paginated API for the filtered methods (assuming low cardinality)
	// *or* we should push some filtering to the client if it's reasonable.
	if err := req.Validate(); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// TOOD: the filter APIs don't return paginated responses, which is super weird.
	// The right thing to do here (probably) is to move them to a separate API endpoint
	// instead of multiplexing the List endpoint with both
	if req.hasOIDCFilter() {
		user, code, _, err := h.getUserByOIDCAuthn(ctx, req)
		switch code {
		case http.StatusOK:
			jsonapi.Marshal(w, idp.ListUserBaseProfilesAndAuthNResponse{Data: []idp.UserBaseProfileAndAuthnResponse{*user}})
			return
		case http.StatusBadRequest:
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return
		default:
			jsonapi.MarshalSQLError(ctx, w, err)
			return
		}
	} else if req.hasEmailFilter() {
		users, code, _, err := h.getUsersByEmail(ctx, req)
		switch code {
		case http.StatusOK:
			jsonapi.Marshal(w, idp.ListUserBaseProfilesAndAuthNResponse{Data: users})
			return
		case http.StatusBadRequest:
			jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
			return
		default:
			jsonapi.MarshalSQLError(ctx, w, err)
			return
		}
	}

	// If we get here, we're in the paginated list case
	jsonapi.MarshalError(ctx, w, ucerr.New("listing baseprofiles with authn without OIDC or email filter is not supported"))
}

type noopValidator struct{}

// Validate implements ucdb.Validator
func (n noopValidator) Validate(_ context.Context, _ *ucdb.DB) error {
	return nil
}

// CreateMigrationHandler returns an http handler function that lists the latest migration per tenant DB
// TODO: this has the potential to be pretty expensive so definitely needs to be authenticated
// TODO: can we unify this with the per-tenant DB handles we already create?
func CreateMigrationHandler(storage *companyconfig.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		pager, err := companyconfig.NewTenantPaginatorFromOptions(pagination.Limit(5))
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		// We don't always close channels since if we bail out early, we prefer to have a memory leak (which will be handled by GC) than a panic which will happen
		// when trying to write to a closed channel
		errs := make(chan error, 100)
		versionsBatches := make(chan []int, 100)
		done := make(chan bool, 1)
		wg := sync.WaitGroup{}
		for {
			tenants, respFields, err := storage.ListTenantsPaginated(ctx, *pager)
			if err != nil {
				jsonapi.MarshalError(ctx, w, err)
				return
			}
			wg.Add(1)
			go func() {
				versions, err := checkTenantsMigrations(ctx, storage, tenants)
				if err != nil {
					errs <- err
				} else {
					versionsBatches <- versions
				}
				wg.Done()
			}()
			if !pager.AdvanceCursor(*respFields) {
				break
			}
		}
		var vs []int
		// Running wg.Wait in a go routine so we can collect results from the versionsBatches channel w/o waiting for all go routines to complete

		go func() {
			wg.Wait()
			done <- true
		}()

		for {
			select {
			case err := <-errs:
				jsonapi.MarshalError(ctx, w, err)
				return
			case versions := <-versionsBatches:
				vs = append(vs, versions...)
			case <-done:
				uclog.Infof(ctx, "Checked migrations for %v tenants", len(vs))
				close(done)
				close(errs)
				close(versionsBatches)
				jsonapi.Marshal(w, vs)
				return
			}
		}
	}
}

func checkTenantsMigrations(ctx context.Context, storage *companyconfig.Storage, tenants []companyconfig.Tenant) ([]int, error) {
	uclog.Infof(ctx, "Check migrations for %v tenants", len(tenants))
	vers := make([]int, len(tenants))
	for i, t := range tenants {
		if t.State.IsFailed() {
			uclog.Warningf(ctx, "Skipping tenant %v because it is in a failed state", t.ID)
		}
		cfg, err := storage.GetTenantInternal(ctx, t.ID)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		// NB: we have to use a noopValidator here otherwise we can never return migrations != current codebase
		db, err := ucdb.New(ctx, &cfg.TenantDBConfig, noopValidator{})
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		ver, err := migrate.GetMaxVersion(ctx, db)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		vers[i] = ver

		if err := db.Close(ctx); err != nil {
			uclog.Warningf(ctx, "Failed to close log DB connection with %v", err)
		}
	}
	return vers, nil
}

func (h *handler) checkEmployeeRequest(ctx context.Context) error {
	tenantAuthn := MustGetTenantAuthn(ctx)
	if !tenantAuthn.UseOrganizations {
		return nil
	}

	if tokenOrgID := auth.GetOrganizationUUID(ctx); tokenOrgID == tenantAuthn.CompanyID {
		return nil
	}

	subjID := auth.GetSubjectUUID(ctx)
	if subjID.IsNil() {
		uclog.Errorf(ctx, "no subject ID passed in JWT, this should only happen in tests")
		return nil
	}

	var err error
	subjOrgID, err := tenantAuthn.ConfigStorage.GetObjectOrganizationID(ctx, subjID)
	if err != nil {
		return ucerr.Friendlyf(err, "could not get organization of subject %s", subjID)
	}

	if subjOrgID == tenantAuthn.CompanyID {
		return nil
	}

	return ucerr.Friendlyf(nil, "insufficient permissions to perform this action")
}

// In the event that a tenant has organizations enabled and there are suborganizations, we need to allow login apps for these suborganizations to get
// user info about users based in the main company organization, but only for read operations (no updates or deletes).
// This check enables the caller to suppress the error when the token is a login app and the user is in the tenant's company organization
func checkCrossOrgLoginApp(ctx context.Context, companyID uuid.UUID, user *storage.BaseUser) bool {
	subjectType := auth.GetSubjectType(ctx)
	if subjectType == authz.ObjectTypeLoginApp || subjectType == m2m.SubjectTypeM2M {
		return true
	}

	if user.OrganizationID == companyID {
		return true
	}

	return true
}
