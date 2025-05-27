package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/authz/ucauthz"
	"userclouds.com/console/internal/auth"
	idpAuthz "userclouds.com/idp/authz"
	"userclouds.com/infra/crypto"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/messaging/email/emailaddress"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/security"
	"userclouds.com/plex"
	"userclouds.com/plex/manager"
	"userclouds.com/userevent"
)

func (h *handler) ensureAccessToTenantRoles(r *http.Request, tenantID uuid.UUID) error {
	// Ensure user is an admin the tenant or a UC admin
	tenant, err := h.storage.GetTenant(r.Context(), tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	isAdmin, companyErr := h.ensureEmployeeAccessToTenant(r, tenant)
	if companyErr == nil && isAdmin {
		return nil
	}

	if ucErr := h.ensureUCAdmin(r); ucErr == nil {
		return nil
	}

	return ucerr.Wrap(companyErr)
}

func (h *handler) newTenantRolesAuthorizer() uchttp.NestedCollectionAuthorizer {
	return &uchttp.NestedMethodAuthorizer{
		GetOneF: func(r *http.Request, tenantID uuid.UUID, userID uuid.UUID) error {
			// Allow any user to list roles for a user
			return nil
		},
		GetAllF: func(r *http.Request, tenantID uuid.UUID) error {
			// Allow any user to list roles
			return nil
		},
		PostF: func(r *http.Request, tenantID uuid.UUID) error {
			return ucerr.Wrap(h.ensureAccessToTenantRoles(r, tenantID))
		},
		PutF: func(r *http.Request, tenantID uuid.UUID, userID uuid.UUID) error {
			return ucerr.Wrap(h.ensureAccessToTenantRoles(r, tenantID))
		},
		DeleteF: func(r *http.Request, tenantID uuid.UUID, userID uuid.UUID) error {
			return ucerr.Wrap(h.ensureAccessToTenantRoles(r, tenantID))
		},
	}
}

func (h *handler) ensureAccessToCompanyRoles(r *http.Request, companyID uuid.UUID) error {
	// Ensure user is an admin of the company
	companyErr := h.ensureCompanyAdmin(r, companyID)
	if companyErr == nil {
		return nil
	}

	if ucErr := h.ensureUCAdmin(r); ucErr == nil {
		return nil
	}

	return ucerr.Wrap(companyErr)
}

func (h *handler) newCompanyRolesAuthorizer() uchttp.NestedCollectionAuthorizer {
	return &uchttp.NestedMethodAuthorizer{
		GetAllF: func(r *http.Request, companyID uuid.UUID) error {
			// Allow any user to list roles
			return nil
		},
		PostF: func(r *http.Request, companyID uuid.UUID) error {
			return ucerr.Wrap(h.ensureAccessToCompanyRoles(r, companyID))
		},
		PutF: func(r *http.Request, companyID uuid.UUID, userID uuid.UUID) error {
			return ucerr.Wrap(h.ensureAccessToCompanyRoles(r, companyID))
		},
		DeleteF: func(r *http.Request, companyID uuid.UUID, userID uuid.UUID) error {
			return ucerr.Wrap(h.ensureAccessToCompanyRoles(r, companyID))
		},
	}
}

// TODO: Need to figure out long term plan for super-admin (i.e. UserClouds admin/support)
// pages; do we keep them as hidden pages of the console UI or make a dedicated/separate site?
func (h *handler) listAllCompanies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := h.ensureUCAdmin(r); err != nil {
		jsonapi.MarshalErrorL(ctx, w, ucerr.Wrap(uchttp.ErrUnauthorized), "Unauthorized", jsonapi.Code(http.StatusForbidden))
		return
	}

	allcompanies := make([]companyconfig.Company, 0)

	pager, err := companyconfig.NewCompanyPaginatorFromOptions(
		pagination.Limit(pagination.MaxLimit))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	for {
		companies, respFields, err := h.storage.ListCompaniesPaginated(ctx, *pager)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}

		allcompanies = append(allcompanies, companies...)

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	jsonapi.Marshal(w, allcompanies)
}

type tenantUserRole struct {
	UserID           uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	OrganizationRole string    `json:"organization_role"`
	PolicyRole       string    `json:"policy_role"`
}

func (tur *tenantUserRole) applyRole(m authz.Membership) {
	switch m.Role {
	case ucauthz.AdminRole:
		tur.OrganizationRole = m.Role
	case ucauthz.MemberRole:
		if tur.OrganizationRole == "" {
			tur.OrganizationRole = m.Role
		}
	case idpAuthz.EdgeTypeNameUserGroupPolicyFullAccess:
		tur.PolicyRole = m.Role
	case idpAuthz.EdgeTypeNameUserGroupPolicyReadAccess:
		if tur.PolicyRole == "" {
			tur.PolicyRole = m.Role
		}
	}
}

func (h *handler) getTenantRolesForEmployee(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, employeeID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	// first make sure that employee is a member of the tenant company on console

	employee, err := h.consoleRBACClient.GetUser(ctx, employeeID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	memberships, err := employee.GetMemberships(ctx)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	foundCompany := false
	for _, m := range memberships {
		if m.Group.ID == tenant.CompanyID {
			foundCompany = true
			break
		}
	}

	if !foundCompany {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("employee '%v' not found in console for company '%v'", employeeID, tenant.CompanyID))
		return
	}

	// get employee name from console idp

	profiles, err := h.consoleIDPClient.GetUserBaseProfiles(ctx, []uuid.UUID{employeeID})
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	if len(profiles) != 1 {
		jsonapi.MarshalError(ctx, w, ucerr.Errorf("employee '%v' not found in console idp", employeeID))
		return
	}
	tur := tenantUserRole{
		UserID: employeeID,
		Name:   profiles[0].Name,
	}

	// now check employee memberships on tenant

	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	rbacClient := authz.NewRBACClient(authZClient)
	employee, err = rbacClient.GetUser(ctx, employeeID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	memberships, err = employee.GetMemberships(ctx)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	for _, m := range memberships {
		if m.Group.ID == tenant.CompanyID {
			tur.applyRole(m)
		}
	}

	jsonapi.Marshal(w, tur)
}

func (h *handler) listTenantRolesForEmployees(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		// Use MarshalSQLError to handle 'tenant not found' properly.
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	// Initialize a map of roles for all employees (from console tenant)
	consoleGroup, err := h.consoleRBACClient.GetGroup(ctx, tenant.CompanyID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	consoleMemberships, err := consoleGroup.GetMemberships(ctx)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// Get the names of all employees in the console tenant
	memberIDSet := set.NewUUIDSet()
	for _, membership := range consoleMemberships {
		memberIDSet.Insert(membership.User.ID)
	}
	memberIDArray := memberIDSet.Items()
	profiles, err := h.consoleIDPClient.GetUserBaseProfiles(ctx, memberIDArray)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	nameMap := map[string]string{}
	for _, profile := range profiles {
		nameMap[profile.ID] = profile.Name
	}

	// Populate the map with the roles for each employee based on console tenant memberships
	userRoleMap := make(map[uuid.UUID]tenantUserRole)
	for _, membership := range consoleMemberships {
		if _, ok := userRoleMap[membership.User.ID]; !ok {
			ur := tenantUserRole{
				UserID: membership.User.ID,
				Name:   nameMap[membership.User.ID.String()],
			}
			userRoleMap[membership.User.ID] = ur
		}
	}

	// Get the employee memberships from the tenant
	authZClient, err := h.newAuthZClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	rbacClient := authz.NewRBACClient(authZClient)
	group, err := rbacClient.GetGroup(ctx, tenant.CompanyID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	memberships, err := group.GetMemberships(ctx)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// Update the roles for each employee in the tenant
	for _, membership := range memberships {
		ur, ok := userRoleMap[membership.User.ID]
		if ok {
			ur.applyRole(membership)
			userRoleMap[membership.User.ID] = ur
		} else {
			// Any employees in the tenant should also be employees of the company in the console tenant, but don't fail if not
			uclog.Errorf(ctx, "employee %s not found in console tenant", membership.User.ID)
		}
	}

	// Convert the map to an array and return
	userRoles := []tenantUserRole{}
	for _, ur := range userRoleMap {
		userRoles = append(userRoles, ur)
	}
	jsonapi.Marshal(w, userRoles)
}

func (h *handler) addTenantEmployeeRoleHelper(ctx context.Context, userID uuid.UUID, role string, tenant companyconfig.Tenant) error {
	// Check that the employee exists in the console IDP and is a member of the company
	user, err := h.consoleRBACClient.GetUser(ctx, userID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	memberships, err := user.GetMemberships(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}
	found := false
	for _, membership := range memberships {
		if membership.Group.ID == tenant.CompanyID {
			found = true
			break
		}
	}
	if !found {
		return ucerr.Errorf("user %s is not a member of company %s", user.ID, tenant.CompanyID)
	}

	tenantDB, err := h.tenantCache.GetTenantDB(ctx, tenant.ID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	mgr := manager.NewFromDB(tenantDB, h.cacheConfig)
	if err := auth.AddEmployeeRoleToTenant(ctx, h.consoleTenant.TenantID, mgr, h.storage, tenant, userID, role); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

type updateOrganizationRolesForEmployeeRequest struct {
	OrganizationRole string `json:"organization_role"`
	PolicyRole       string `json:"policy_role"`
}

func (h *handler) updateTenantRolesForEmployee(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, userID uuid.UUID) {
	ctx := r.Context()

	var req updateOrganizationRolesForEmployeeRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

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
	group, err := rbacClient.GetGroup(ctx, tenant.CompanyID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	newUser := false
	user, err := rbacClient.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, authz.ErrObjectNotFound) {
			newUser = true
			// If the user is not found on the tenant, add them
			if err := h.addTenantEmployeeRoleHelper(ctx, userID, req.OrganizationRole, *tenant); err != nil {
				jsonapi.MarshalError(ctx, w, err)
				return
			}
			if req.PolicyRole != "" {
				user, err = rbacClient.GetUser(ctx, userID)
				if err != nil {
					jsonapi.MarshalError(ctx, w, err)
					return
				}
				if _, err := group.AddUserRole(ctx, *user, req.PolicyRole); err != nil {
					jsonapi.MarshalError(ctx, w, err)
					return
				}
			}
		} else {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
	}

	if !newUser {
		if err := group.RemoveUser(ctx, *user); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}

		for _, role := range []string{req.OrganizationRole, req.PolicyRole} {
			if role != "" {
				if _, err := group.AddUserRole(ctx, *user, role); err != nil {
					jsonapi.MarshalError(ctx, w, err)
					return
				}
			}
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

type companyUserRole struct {
	UserID           uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	OrganizationRole string    `json:"organization_role"`
}

func (h *handler) listCompanyRoleForEmployees(w http.ResponseWriter, r *http.Request, companyID uuid.UUID) {
	ctx := r.Context()

	group, err := h.consoleRBACClient.GetGroup(ctx, companyID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	memberships, err := group.GetMemberships(ctx)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// Get the names of all employees in the console tenant
	memberIDSet := set.NewUUIDSet()
	for _, membership := range memberships {
		memberIDSet.Insert(membership.User.ID)
	}
	memberIDArray := memberIDSet.Items()
	profiles, err := h.consoleIDPClient.GetUserBaseProfiles(ctx, memberIDArray)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	nameMap := map[string]string{}
	for _, profile := range profiles {
		nameMap[profile.ID] = profile.Name
	}

	userRoleMap := make(map[uuid.UUID]companyUserRole)
	for _, membership := range memberships {
		ur, ok := userRoleMap[membership.User.ID]
		if ok {
			if membership.Role == ucauthz.AdminRole || (membership.Role == ucauthz.MemberRole && ur.OrganizationRole == "") {
				ur.OrganizationRole = membership.Role
			}
		} else {
			ur = companyUserRole{
				UserID:           membership.User.ID,
				Name:             nameMap[membership.User.ID.String()],
				OrganizationRole: membership.Role}
		}
		userRoleMap[membership.User.ID] = ur
	}

	// Convert the map to an array and return
	userRoles := []companyUserRole{}
	for _, ur := range userRoleMap {
		userRoles = append(userRoles, ur)
	}

	jsonapi.Marshal(w, userRoles)
}

func (h *handler) deleteEmployeeFromCompany(w http.ResponseWriter, r *http.Request, companyID uuid.UUID, userID uuid.UUID) {
	ctx := r.Context()

	tenants, err := h.storage.ListTenantsForCompany(ctx, companyID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	if err := auth.DeleteEmployeeFromCompany(ctx, h.storage, h.tenantCache, h.consoleRBACClient, userID, companyID, tenants, h.cacheConfig); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type updateCompanyRoleForEmployeeRequest struct {
	OrganizationRole string `json:"organization_role"`
}

func (h *handler) updateCompanyRoleForEmployee(w http.ResponseWriter, r *http.Request, companyID uuid.UUID, userID uuid.UUID) {
	ctx := r.Context()

	var req updateCompanyRoleForEmployeeRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	group, err := h.consoleRBACClient.GetGroup(ctx, companyID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	user, err := h.consoleRBACClient.GetUser(ctx, userID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if err := group.RemoveUser(ctx, *user); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if _, err := group.AddUserRole(ctx, *user, req.OrganizationRole); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DefaultInviteExpiry is the default expiration time for a Console invite
const DefaultInviteExpiry = time.Hour * 24 * 30

func (h *handler) createInviteKey(ctx context.Context, email string, companyID uuid.UUID, companyRole string, tenantRoles companyconfig.TenantRoles) (*companyconfig.InviteKey, error) {
	inviteKey := companyconfig.InviteKey{
		BaseModel:    ucdb.NewBase(),
		Type:         companyconfig.InviteKeyTypeExistingCompany,
		Key:          crypto.GenerateOpaqueAccessToken(),
		Expires:      time.Now().UTC().Add(DefaultInviteExpiry),
		Used:         false,
		CompanyID:    companyID,
		Role:         companyRole,
		TenantRoles:  tenantRoles,
		InviteeEmail: email,
	}

	if err := h.storage.SaveInviteKey(ctx, &inviteKey); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &inviteKey, nil
}

// InviteUserRequest is the request struct to invite a user to create a new company or join
// an existing company. It is public only for testing purposes.
type InviteUserRequest struct {
	InviteeEmails string                    `json:"invitee_emails"`
	CompanyRole   string                    `json:"company_role"`
	TenantRoles   companyconfig.TenantRoles `json:"tenant_roles"`
}

//go:generate genvalidate InviteUserRequest

func (iur InviteUserRequest) extraValidate() error {
	emails := strings.Split(iur.InviteeEmails, ",")

	if len(emails) > 250 {
		return ucerr.Friendlyf(nil, "250 is the maximum number of emails that can be invited at once")
	}

	for _, email := range emails {
		a := emailaddress.Address(email)
		if err := a.Validate(); err != nil {
			return ucerr.Wrap(err) // this error is friendly at the base level already
		}
	}

	return nil
}

// inviteUserToCompany handles inviting new or existing users to an existing company.
// This is expected to be used by developers adopting UC who are inviting teammates to help manage their company/tenants.
func (h *handler) inviteUserToCompany(w http.ResponseWriter, r *http.Request, companyID uuid.UUID) {
	ctx := r.Context()

	if err := h.ensureAccessToCompanyRoles(r, companyID); err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.Wrap(err), jsonapi.Code(http.StatusForbidden))
		return
	}

	var req InviteUserRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	uclog.Infof(ctx, "inviting email(s) %s to company %s", req.InviteeEmails, companyID)
	company, err := h.storage.GetCompany(ctx, companyID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	inviteText := fmt.Sprintf("You are being invited to join as an employee of the %s Company on UserClouds.", company.Name)

	// if we're specifying a region, stay true to that region
	callbackURL := auth.GetConsoleURL(r.Host, h.getConsoleURLCallback())
	callbackURL.Path = auth.InviteCallbackPath
	uclog.Debugf(ctx, "callback URL for invite: %s", callbackURL.String())

	plexAppManager := manager.NewFromDB(h.consoleTenantDB, h.cacheConfig)
	loginApps, err := plexAppManager.GetLoginApps(ctx, h.consoleTenant.TenantID, companyID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	if len(loginApps) != 1 {
		jsonapi.MarshalError(ctx, w, ucerr.Friendlyf(nil, "expected 1 login app for this company, found %d", len(loginApps)))
		return
	}

	tokenSource, err := m2m.GetM2MTokenSource(ctx, h.consoleTenant.TenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	plexClient := plex.NewClient(h.consoleTenant.TenantURL, tokenSource, security.PassXForwardedFor())

	for email := range strings.SplitSeq(req.InviteeEmails, ",") {
		inviteeEmail := strings.TrimSpace(email)

		inviteKey, err := h.createInviteKey(ctx, inviteeEmail, companyID, req.CompanyRole, req.TenantRoles)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		state := fmt.Sprintf("%s#%s", auth.InviteStatePrefix, inviteKey.Key)
		userInfo := auth.MustGetUserInfo(r)

		inviteReq := plex.SendInviteRequest{
			InviteeEmail:  inviteeEmail,
			InviterUserID: userInfo.Claims.Subject,
			InviterName:   userInfo.Claims.Name,
			InviterEmail:  userInfo.Claims.Email,
			ClientID:      loginApps[0].ClientID,
			State:         state,
			RedirectURL:   callbackURL.String(),
			InviteText:    inviteText,
			Expires:       inviteKey.Expires,
		}

		if err := plexClient.SendInvite(ctx, inviteReq); err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}

		if err := h.consoleUserEventClient.ReportEvents(ctx, []userevent.UserEvent{
			{
				BaseModel: ucdb.NewBase(),
				Type:      "invite",
				UserAlias: userInfo.Claims.Subject,
				Payload: userevent.Payload{
					"InviteType":   companyconfig.InviteKeyTypeExistingCompany,
					"InviteeEmail": inviteeEmail,
					"Company":      companyID.String(),
				},
			},
		}); err != nil {
			uclog.Errorf(ctx, "error reporting invite event: %v", err)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

type inviteData struct {
	ID            uuid.UUID                 `json:"id"`
	Created       time.Time                 `json:"created"`
	Expires       time.Time                 `json:"expires"`
	Used          bool                      `json:"used"`
	Role          string                    `json:"role"`
	TenantRoles   companyconfig.TenantRoles `json:"tenant_roles"`
	InviteType    string                    `json:"invite_type"`
	InviteeEmail  string                    `json:"invitee_email"`
	InviteeUserID uuid.UUID                 `json:"invitee_user_id"`
}

type listInvitesResponse struct {
	Data []inviteData `json:"data"`
	pagination.ResponseFields
}

// listInvites
func (h *handler) listInvites(w http.ResponseWriter, r *http.Request, companyID uuid.UUID) {
	ctx := r.Context()

	// ensure user has proper access
	if err := h.ensureAccessToCompanyRoles(r, companyID); err != nil {
		jsonapi.MarshalError(ctx, w, ucerr.Wrap(err), jsonapi.Code(http.StatusForbidden))
		return
	}

	pager, err := companyconfig.NewInviteKeyPaginatorFromRequest(
		r,
		pagination.Filter(fmt.Sprintf("((('company_id',EQ,'%v'),AND,('used',EQ,'%s')),AND,('expires',GT,'%v'))", companyID, "false", time.Now().UTC().UnixMicro())),
		pagination.SortKey(pagination.Key("expires,id")),
	)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	inviteKeys, respFields, err := h.storage.ListInviteKeysPaginated(ctx, *pager)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	iData := []inviteData{}
	for _, inviteKey := range inviteKeys {
		iData = append(iData, inviteData{
			ID:           inviteKey.ID,
			Created:      inviteKey.Created,
			Expires:      inviteKey.Expires,
			Role:         inviteKey.Role,
			TenantRoles:  inviteKey.TenantRoles,
			InviteeEmail: inviteKey.InviteeEmail,
		})
	}

	jsonapi.Marshal(w, listInvitesResponse{
		Data:           iData,
		ResponseFields: *respFields,
	})
}
