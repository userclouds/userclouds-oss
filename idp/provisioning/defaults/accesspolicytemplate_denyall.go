package defaults

import (
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/infra/ucdb"
)

// accessPolicyTemplateDenyAll is the access policy template that denies all access
var accessPolicyTemplateDenyAll = storage.AccessPolicyTemplate{
	SystemAttributeBaseModel: ucdb.MarkAsSystem(ucdb.NewSystemAttributeBaseWithID(policy.AccessPolicyTemplateDenyAll.ID)),
	Name:                     "DenyAll",
	Description:              "This template denies all access.",
	Function: `function policy(context, params) {
                return false;
        }`,
}
