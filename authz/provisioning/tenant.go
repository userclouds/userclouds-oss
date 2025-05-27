package provisioning

import (
	"fmt"

	"userclouds.com/authz"
	"userclouds.com/authz/ucauthz"
	idpAuthz "userclouds.com/idp/authz"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/provisioning/types"
)

// ConsoleAuthZEdgeTypes is an array containing default AuthZ edge types
var ConsoleAuthZEdgeTypes = []authz.EdgeType{
	{BaseModel: ucdb.NewBaseWithID(ucauthz.AdminRoleTypeID), TypeName: "_admin_deprecated", SourceObjectTypeID: authz.GroupObjectTypeID,
		TargetObjectTypeID: authz.UserObjectTypeID},
	{BaseModel: ucdb.NewBaseWithID(ucauthz.MemberRoleTypeID), TypeName: "_member_deprecated", SourceObjectTypeID: authz.GroupObjectTypeID,
		TargetObjectTypeID: authz.UserObjectTypeID},
	{BaseModel: ucdb.NewBaseWithID(ucauthz.AdminEdgeTypeID),
		TypeName:           ucauthz.EdgeTypeAdmin,
		SourceObjectTypeID: authz.UserObjectTypeID,
		TargetObjectTypeID: authz.GroupObjectTypeID,
		Attributes: []authz.Attribute{
			{Name: ucauthz.AdminRole, Direct: true},
			{Name: idpAuthz.AttributePolicyCreate, Inherit: true},
			{Name: idpAuthz.AttributePolicyRead, Inherit: true},
			{Name: idpAuthz.AttributePolicyUpdate, Inherit: true},
			{Name: idpAuthz.AttributePolicyDelete, Inherit: true},
		},
	},
	{BaseModel: ucdb.NewBaseWithID(ucauthz.MemberEdgeTypeID),
		TypeName:           ucauthz.EdgeTypeMember,
		SourceObjectTypeID: authz.UserObjectTypeID,
		TargetObjectTypeID: authz.GroupObjectTypeID,
		Attributes: []authz.Attribute{
			{Name: ucauthz.MemberRole, Direct: true},
		},
	},
}

// NewTenantAuthZ return an initialized Provisionable object for initializing default AuthZ objects
// TODO - this function appear to be dead code only called from tests, we can remove it assuming that there is another coverage for equivalent code
func NewTenantAuthZ(name string, pi types.ProvisionInfo, company *companyconfig.Company) (types.Provisionable, error) {
	provs := make([]types.Provisionable, 0)
	name = fmt.Sprintf("%s:AuthZFoundationTypes", name)

	// Validate inputs
	if pi.TenantID.IsNil() {
		return nil, ucerr.Errorf("Can't provision nil tenant")
	}
	if pi.TenantDB == nil {
		return nil, ucerr.Errorf("Can't provision tenant with nil tenant DB")
	}
	if company == nil || company.ID.IsNil() || company.Name == "" {
		return nil, ucerr.Errorf("Can't provision tenant with invalid company %v", company)
	}

	// Add a logging message in front of each operation to help with debugging (serial)
	logMesg := types.NewLogMessageProvisioner(fmt.Sprintf("AuthZ Foundations for Tenant ID %v for company %s id %v", pi.TenantID, company.Name, company.ID))
	provs = append(provs, logMesg)

	// Create the Default AuthZ types (User, Group, App) and the Console AuthZ edges
	typesProv := NewEntityAuthZ(name, pi, authz.DefaultAuthZObjectTypes, append(authz.DefaultAuthZEdgeTypes, ConsoleAuthZEdgeTypes...), nil, nil, types.Validate)
	provs = append(provs, typesProv)

	// Provision the default organization representing the company in the tenant (parallel). In our current design this object represents every system resource
	// outside of a scope of a different organization
	orgProv, err := NewOrganizationProvisioner(name, pi, company.ID /* default company org has same ID as the company */, company.Name, "")
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	provs = append(provs, orgProv)

	p := types.NewParallelProvisioner(provs, name)
	return p, nil
}
