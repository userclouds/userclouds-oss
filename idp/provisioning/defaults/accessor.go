package defaults

import (
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/constants"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/ucdb"
)

var defaultAccessorsByID = map[uuid.UUID]storage.Accessor{}
var defaultAccessors = []storage.Accessor{
	{
		SystemAttributeBaseModel: ucdb.MarkAsSystem(ucdb.NewSystemAttributeBaseWithID(constants.GetUserAccessorID)),
		Name:                     "GetUsers",
		Description:              "Accessor used to retrieve users' profile data via the GetUser API",
		DataLifeCycleState:       column.DataLifeCycleStateLive,
		AccessPolicyID:           policy.AccessPolicyAllowAll.ID,
		SelectorConfig:           userstore.UserSelectorConfig{WhereClause: "{id} = ANY (?)"},
		PurposeIDs:               []uuid.UUID{constants.OperationalPurposeID},
	},
}

// GetDefaultAccessors returns the default accessors
func GetDefaultAccessors() []*storage.Accessor {
	var accessors []*storage.Accessor
	for _, da := range defaultAccessors {
		accessors = append(accessors, &da)
	}
	return accessors
}

// IsDefaultAccessor returns true if id refers to a default accessor
func IsDefaultAccessor(id uuid.UUID) bool {
	if _, found := defaultAccessorsByID[id]; found {
		return true
	}

	return false
}

func init() {
	for _, da := range defaultAccessors {
		if _, found := defaultAccessorsByID[da.ID]; found {
			panic(fmt.Sprintf("accessor %s has conflicting id %v", da.Name, da.ID))
		}
		defaultAccessorsByID[da.ID] = da
	}
}
