package shared

import (
	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/apiclient"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/multitenant"
)

// ValidateUserOrganizationForRequest validates that the organization ID passed in the request is valid for the caller
func ValidateUserOrganizationForRequest(ctx context.Context, organizationID uuid.UUID) (creationOrgID uuid.UUID, err error) {
	ts := multitenant.MustGetTenantState(ctx)
	s := storage.NewFromTenantState(ctx, ts)

	if !ts.UseOrganizations {
		return uuid.Nil, nil
	}

	tokenOrgID := auth.GetOrganizationUUID(ctx)

	//
	// Try to validate organization from the token org
	//
	if !organizationID.IsNil() && (tokenOrgID == organizationID || tokenOrgID == ts.CompanyID) {
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

	subjOrgID, err := s.GetObjectOrganizationID(ctx, subjID)
	if err != nil || subjOrgID.IsNil() {
		return uuid.Nil, ucerr.Friendlyf(err, "could not validate organization %s for subject %s", organizationID, subjID)
	}

	if subjOrgID != tokenOrgID {
		// We got a different org ID by reading from the db than we did from the token, log a warning
		uclog.Warningf(ctx, "organization ID from token (%s) does not match organization ID from subject (%s), this should only happen for UC employees", tokenOrgID, subjOrgID)
	}

	// If the passed in organization ID is for the nil org, use the subject's org ID for creation
	if organizationID.IsNil() {
		return subjOrgID, nil
	}

	// Re-do the check from above using subjOrgID
	if subjOrgID == organizationID || subjOrgID == ts.CompanyID {
		return organizationID, nil
	}

	friendlyErr := ucerr.Friendlyf(nil, `requested organizationID %s does not match JWT subject's organizationID %s`, organizationID, subjOrgID)

	// Try to get the names of the organizations to include in the error message
	if authzClient, err := apiclient.NewAuthzClientFromTenantStateWithPassthroughAuth(ctx); err == nil {
		if org, err := authzClient.GetOrganization(ctx, organizationID); err == nil {
			if subjOrg, err := authzClient.GetOrganization(ctx, subjOrgID); err == nil {
				friendlyErr = ucerr.Friendlyf(nil, `requested organization "%s" (%s) does not match JWT subject's organization "%s" (%s)`, org.Name, org.ID, subjOrg.Name, subjOrg.ID)
			}
		}
	}

	return uuid.Nil, ucerr.Wrap(friendlyErr)
}
