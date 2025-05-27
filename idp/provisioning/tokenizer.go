package provisioning

import (
	"userclouds.com/authz"
	provisioningAuthZ "userclouds.com/authz/provisioning"
	idpAuthz "userclouds.com/idp/authz"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/provisioning/types"
)

// ProvisionTokenizerAuthzEntities returns a ProvisionableMaker that can provision tokenizer authz entities
func ProvisionTokenizerAuthzEntities(name string, pi types.ProvisionInfo, company *companyconfig.Company) types.ProvisionableMaker {
	return func() ([]types.Provisionable, error) {
		if pi.TenantDB == nil {
			return nil, ucerr.New("cannot provisioner tokenizer authz entities with nil tenantDB")
		}

		if company == nil {
			return nil, ucerr.New("cannot provision tokenizer authz entities with nil company")
		}

		if err := company.Validate(); err != nil {
			return nil, ucerr.Wrap(err)
		}

		// Provision global tokenizer AuthZ objects
		// This edge connects one global (per tenant) AuthZ Policy object to the AuthZ object for the default organization
		// This implies that the policies are can't be associated with any other organization which is our current design but may need to change

		defaultAuthZEdges := append(
			DefaultAuthZEdges,
			authz.Edge{
				EdgeTypeID:     idpAuthz.GroupPolicyFullAccessEdgeTypeID,
				SourceObjectID: company.ID,
				TargetObjectID: idpAuthz.PoliciesObjectID,
			},
		)

		return []types.Provisionable{
				provisioningAuthZ.NewEntityAuthZ(
					name+":Tokenizer",
					pi,
					DefaultAuthZObjectTypes,
					DefaultAuthZEdgeTypes,
					DefaultAuthZObjects,
					defaultAuthZEdges,
				),
			},
			nil
	}
}
