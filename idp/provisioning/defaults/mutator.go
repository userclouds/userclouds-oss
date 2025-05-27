package defaults

import (
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/constants"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/ucdb"
)

var defaultMutatorsByID = map[uuid.UUID]storage.Mutator{}
var defaultMutators = []storage.Mutator{
	{
		SystemAttributeBaseModel: ucdb.MarkAsSystem(ucdb.NewSystemAttributeBaseWithID(constants.UpdateUserMutatorID)),
		Name:                     "UpdateUser",
		Description:              "Mutator used to modify a user’s data by CreateUser and UpdateUser APIs",
		AccessPolicyID:           policy.AccessPolicyAllowAll.ID,
		SelectorConfig:           userstore.UserSelectorConfig{WhereClause: "{id} = ?"},
	},
}

// GetDefaultMutators returns the default mutators
func GetDefaultMutators() []*storage.Mutator {
	var mutators []*storage.Mutator
	for _, dm := range defaultMutators {
		mutators = append(mutators, &dm)
	}
	return mutators
}

// IsDefaultMutator returns true if id refers to a default mutator
func IsDefaultMutator(id uuid.UUID) bool {
	if _, found := defaultMutatorsByID[id]; found {
		return true
	}

	return false
}

func init() {
	for _, dm := range defaultMutators {
		if _, found := defaultMutatorsByID[dm.ID]; found {
			panic(fmt.Sprintf("mutator %s has conflicting id %v", dm.Name, dm.ID))
		}
		defaultMutatorsByID[dm.ID] = dm
	}
}
