package defaults

import (
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/constants"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/infra/ucdb"
)

var defaultPurposesByID = map[uuid.UUID]storage.Purpose{}
var defaultPurposes = []storage.Purpose{
	{
		SystemAttributeBaseModel: ucdb.MarkAsSystem(ucdb.NewSystemAttributeBaseWithID(constants.OperationalPurposeID)),
		Name:                     "operational",
		Description:              "Purpose is for basic operation of the site",
	},
	{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(constants.AnalyticsPurposeID),
		Name:                     "analytics",
		Description:              "Purpose is for product improvement analytics",
	},
	{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(constants.MarketingPurposeID),
		Name:                     "marketing",
		Description:              "Purpose is for marketing to users",
	},
	{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(constants.SupportPurposeID),
		Name:                     "support",
		Description:              "Purpose is for support",
	},
	{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(constants.SecurityPurposeID),
		Name:                     "security",
		Description:              "Purpose is for security, fraud, and site integrity usage",
	},
}

// GetDefaultPurposes returns the default purposes
func GetDefaultPurposes() []storage.Purpose {
	var purposes []storage.Purpose
	purposes = append(purposes, defaultPurposes...)
	return purposes
}

// IsDefaultPurpose returns true if id refers to a default purpose
func IsDefaultPurpose(id uuid.UUID) bool {
	if _, found := defaultPurposesByID[id]; found {
		return true
	}

	return false
}

func init() {
	for _, dp := range defaultPurposes {
		if _, found := defaultPurposesByID[dp.ID]; found {
			panic(fmt.Sprintf("purpose %s has conflicting id %v", dp.Name, dp.ID))
		}
		defaultPurposesByID[dp.ID] = dp
	}
}
