package defaults

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/lib/pq"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/infra/ucdb"
)

var defaultAccessPoliciesByID = map[uuid.UUID]storage.AccessPolicy{}
var defaultAccessPolicies = []storage.AccessPolicy{
	// access policy that allows all access
	{
		SystemAttributeBaseModel: ucdb.MarkAsSystem(ucdb.NewSystemAttributeBaseWithID(policy.AccessPolicyAllowAll.ID)),
		Name:                     "AllowAll",
		Description:              "This policy allows all access.",
		ComponentIDs:             []uuid.UUID{policy.AccessPolicyTemplateAllowAll.ID},
		ComponentParameters:      []string{""},
		ComponentTypes:           pq.Int32Array{int32(storage.AccessPolicyComponentTypeTemplate)},
		PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
	},
	// access policy that denies all access
	{
		SystemAttributeBaseModel: ucdb.MarkAsSystem(ucdb.NewSystemAttributeBaseWithID(policy.AccessPolicyDenyAll.ID)),
		Name:                     "DenyAll",
		Description:              "This policy denies all access.",
		ComponentIDs:             []uuid.UUID{policy.AccessPolicyTemplateDenyAll.ID},
		ComponentParameters:      []string{""},
		ComponentTypes:           pq.Int32Array{int32(storage.AccessPolicyComponentTypeTemplate)},
		PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
	},
	// access policy that applies to all accessors
	{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(policy.AccessPolicyGlobalAccessorID),
		Name:                     "GlobalBaselinePolicyForAccessors",
		Description:              "This policy applies to all accessors.",
		ComponentIDs:             []uuid.UUID{policy.AccessPolicyTemplateAllowAll.ID},
		ComponentParameters:      []string{""},
		ComponentTypes:           pq.Int32Array{int32(storage.AccessPolicyComponentTypeTemplate)},
		PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
	},
	// access policy that applies to all mutators
	{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(policy.AccessPolicyGlobalMutatorID),
		Name:                     "GlobalBaselinePolicyForMutators",
		Description:              "This policy applies to all mutators.",
		ComponentIDs:             []uuid.UUID{policy.AccessPolicyTemplateAllowAll.ID},
		ComponentParameters:      []string{""},
		ComponentTypes:           pq.Int32Array{int32(storage.AccessPolicyComponentTypeTemplate)},
		PolicyType:               storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd),
	},
}

// GetDefaultAccessPolicies returns the default access policies
func GetDefaultAccessPolicies() []storage.AccessPolicy {
	var accessPolicies []storage.AccessPolicy
	accessPolicies = append(accessPolicies, defaultAccessPolicies...)
	return accessPolicies
}

// IsDefaultAccessPolicy returns true if id refers to a default access policy
func IsDefaultAccessPolicy(id uuid.UUID) bool {
	if _, found := defaultAccessPoliciesByID[id]; found {
		return true
	}

	return false
}

func init() {
	for _, dap := range defaultAccessPolicies {
		if _, found := defaultAccessPoliciesByID[dap.ID]; found {
			panic(fmt.Sprintf("access policy %s has conflicting id %v", dap.Name, dap.ID))
		}
		defaultAccessPoliciesByID[dap.ID] = dap
	}
}
