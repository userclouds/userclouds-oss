package api

import (
	"errors"
	"net/http"
	"slices"

	"github.com/gofrs/uuid"

	"userclouds.com/authz/ucauthz"
	"userclouds.com/console/internal/auth"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	internalAuth "userclouds.com/internal/auth"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/provisioning"
	"userclouds.com/internal/provisioning/types"
	"userclouds.com/plex/manager"
)

func (h *handler) getCompanyRoles(r *http.Request, companyID uuid.UUID) ([]string, error) {
	ctx := r.Context()
	userInfo := auth.MustGetUserInfo(r)
	userID, err := userInfo.GetUserID()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	group, err := h.consoleRBACClient.GetGroup(ctx, companyID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	user, err := h.consoleRBACClient.GetUser(ctx, userID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	roles, err := group.GetUserRoles(ctx, *user)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return roles, nil
}

// this grabs a list of all companies that this user is an admin of
func (h *handler) getAdminCompaniesForUser(r *http.Request) ([]uuid.UUID, error) {
	ctx := r.Context()
	userInfo := auth.MustGetUserInfo(r)
	userID, err := userInfo.GetUserID()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	user, err := h.consoleRBACClient.GetUser(ctx, userID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	memberships, err := user.GetMemberships(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	companyIDs := []uuid.UUID{}
	for _, membership := range memberships {
		if membership.Role == ucauthz.AdminRole {
			companyIDs = append(companyIDs, membership.Group.ID)
		}
	}
	return companyIDs, nil
}

func (h *handler) ensureCompanyAdmin(r *http.Request, companyID uuid.UUID) error {
	roles, err := h.getCompanyRoles(r, companyID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if slices.Contains(roles, ucauthz.AdminRole) {
		return nil
	}
	return ucerr.Friendlyf(nil, "You must be an admin of the company to perform this action")
}

func (h *handler) ensureCompanyEmployee(r *http.Request, companyID uuid.UUID) error {
	roles, err := h.getCompanyRoles(r, companyID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	for _, role := range roles {
		if role == ucauthz.AdminRole || role == ucauthz.MemberRole {
			return nil
		}
	}
	return ucerr.Friendlyf(nil, "You must be an admin or member of the company to perform this action")
}

func (h *handler) ensureUCAdmin(r *http.Request) error {
	ctx := r.Context()
	if st := internalAuth.GetSubjectType(ctx); st == m2m.SubjectTypeM2M {
		return nil
	}

	roles, err := h.getCompanyRoles(r, h.consoleTenant.CompanyID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if slices.Contains(roles, ucauthz.AdminRole) {
		return nil
	}
	return ucerr.Errorf("user must be an admin of userclouds company %s", h.consoleTenant.CompanyID)
}

func (h *handler) newCompanyAuthorizer() uchttp.CollectionAuthorizer {
	return &uchttp.MethodAuthorizer{
		GetAllF: func(r *http.Request) error {
			// If the user exists on the console tenant (will be checked in the function),
			// then they are an employee of some company
			return nil
		},
		PostF: func(r *http.Request) error {
			// Logic to check create permissions is simpler inside the handler
			return nil
		},
		PutF: func(r *http.Request, id uuid.UUID) error {
			return ucerr.Wrap(h.ensureCompanyAdmin(r, id))
		},
		DeleteF: func(r *http.Request, id uuid.UUID) error {
			return ucerr.Wrap(h.ensureCompanyAdmin(r, id))
		},
		NestedF: func(r *http.Request, id uuid.UUID) error {
			// Ensure user has some kind of relationship to company or is a UC admin
			// before any further access is allowed.
			companyErr := h.ensureCompanyEmployee(r, id)
			ucErr := h.ensureUCAdmin(r)
			// If either check passes, allow access
			if companyErr == nil || ucErr == nil {
				return nil
			}
			return ucerr.Wrap(companyErr)
		},
	}
}

type companyInfo struct {
	companyconfig.Company
	IsAdmin bool `json:"is_admin"`
}

func (h *handler) listCompanies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userInfo := auth.MustGetUserInfo(r)
	userID, err := userInfo.GetUserID()
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	user, err := h.consoleRBACClient.GetUser(ctx, userID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// Get all the groups that this user is an admin or member of
	memberships, err := user.GetMemberships(ctx)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	groupIDs := map[uuid.UUID]bool{} // bool indicates if user is admin
	for _, membership := range memberships {
		if membership.Role == ucauthz.AdminRole {
			groupIDs[membership.Group.ID] = true
		} else if _, ok := groupIDs[membership.Group.ID]; !ok {
			groupIDs[membership.Group.ID] = false
		}
	}

	visibleCompanies := []companyInfo{}

	pager, err := companyconfig.NewCompanyPaginatorFromOptions(
		pagination.Limit(pagination.MaxLimit))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	for {
		// Match the list of companies against the ids of groups
		companies, respFields, err := h.storage.ListCompaniesPaginated(ctx, *pager)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}

		for _, company := range companies {
			if isAdmin, ok := groupIDs[company.ID]; ok {
				visibleCompanies = append(visibleCompanies, companyInfo{Company: company, IsAdmin: isAdmin})
			}
		}

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	jsonapi.Marshal(w, visibleCompanies)
}

// CreateCompanyRequest is public for testing purposes
type CreateCompanyRequest struct {
	Company companyconfig.Company `json:"company"`
}

// CreateCompanyResponse is public for testing purposes
func (h *handler) createCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateCompanyRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := req.Company.Validate(); err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	userInfo := auth.MustGetUserInfo(r)
	userID, err := userInfo.GetUserID()
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusForbidden))
		return
	}

	// Permissions check - user must be a UC admin OR signups must be enabled
	if err := h.ensureUCAdmin(r); err != nil {
		// TODO: this seems like slightly strange factoring to me, but it makes no sense to allow
		// signups on the console tenant but *not* allow company creation
		mgr := manager.NewFromDB(h.consoleTenantDB, h.cacheConfig)

		tc, err := mgr.GetTenantPlex(ctx, h.consoleTenant.TenantID)
		if err != nil {
			jsonapi.MarshalSQLError(ctx, w, err)
			return
		}

		// if signups are disabled, then checks have failed
		if tc.PlexConfig.DisableSignUps && !slices.Contains(tc.PlexConfig.BootstrapAccountEmails, userInfo.Claims.Email) {
			jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(err, "signups are disabled"), jsonapi.Code(http.StatusForbidden))
			return
		}
	}
	pi := types.ProvisionInfo{
		CompanyStorage: h.storage,
		TenantDB:       h.consoleTenantDB,
		LogDB:          nil,
		CacheCfg:       h.cacheConfig,
		TenantID:       h.consoleTenant.TenantID,
	}

	pc, err := provisioning.NewProvisionableCompany(ctx,
		"console",
		pi,
		&req.Company,
		h.consoleTenant.CompanyID,
		provisioning.Owner(userID))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := pc.Provision(ctx); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, companyInfo{Company: req.Company, IsAdmin: true})
}

// UpdateCompanyRequest is public for testing purposes
type UpdateCompanyRequest struct {
	Company struct {
		// Use pointers here because individual fields are optional
		Name *string                    `json:"name,omitempty"`
		Type *companyconfig.CompanyType `json:"type,omitempty"`
	}
}

// Validate implements the Validatable interface
func (ucr *UpdateCompanyRequest) Validate() error {
	if ucr.Company.Name == nil && ucr.Company.Type == nil {
		return ucerr.Friendlyf(nil, "No fields to update")
	}
	if ucr.Company.Name != nil && len(*ucr.Company.Name) == 0 {
		return ucerr.Friendlyf(nil, "Company name cannot be empty")
	}
	return nil
}

// UpdateCompanyResponse is public for testing purposes
type UpdateCompanyResponse struct {
	Company companyInfo `json:"company"`
}

func (h *handler) updateCompany(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req UpdateCompanyRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	company, err := h.storage.GetCompany(ctx, id)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	if req.Company.Name != nil && company.Name != *req.Company.Name {
		uclog.Infof(ctx, "updating company name from '%s' to '%s' for company ID '%s'", company.Name, *req.Company.Name, company.ID)
		company.Name = *req.Company.Name

	}
	if req.Company.Type != nil && company.Type != *req.Company.Type {
		uclog.Infof(ctx, "updating company type from '%v' to '%v' for company ID '%s'", company.Type, *req.Company.Type, company.ID)
		company.Type = *req.Company.Type
	}

	if err := h.storage.SaveCompany(ctx, company); err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, UpdateCompanyResponse{Company: companyInfo{Company: *company, IsAdmin: true}})
}

func (h *handler) deleteCompany(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	// Get the employees of the company
	companyGroup, err := h.consoleRBACClient.GetGroup(ctx, id)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	companyMembers, err := companyGroup.GetMemberships(ctx)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// Delete the company objects
	company, err := h.storage.GetCompany(ctx, id)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}
	pi := types.ProvisionInfo{
		CompanyStorage: h.storage,
		TenantDB:       h.consoleTenantDB,
		LogDB:          nil,
		CacheCfg:       h.cacheConfig,
		TenantID:       h.consoleTenant.TenantID,
	}
	po, err := provisioning.NewProvisionableCompany(ctx, "DeleteCompanyAPI", pi, company, h.consoleTenant.CompanyID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := po.Cleanup(ctx); err != nil {
		if errors.Is(err, companyconfig.ErrCompanyStillHasTenants) {
			err := ucerr.Errorf("cannot deprovision company '%s' (id: %s) because it has tenants which must be deprovisioned first", company.Name, company.ID)
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// Delete all employees of the company from Console that are not employees of other companies
	for _, member := range companyMembers {
		// Get the companies the employee belongs to
		employeeMemberships, err := member.User.GetMemberships(ctx)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}

		// If the employee belongs to more than one company, then they are not deleted
		otherCompany := false
		for _, employeeMembership := range employeeMemberships {
			if employeeMembership.Group.ID != companyGroup.ID {
				otherCompany = true
				break
			}
		}

		if !otherCompany {
			if err := h.consoleIDPClient.DeleteUser(ctx, member.User.ID); err != nil {
				jsonapi.MarshalError(ctx, w, err)
				return
			}
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
