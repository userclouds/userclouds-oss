package defaults

import (
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/infra/ucdb"
)

// accessPolicyTemplateAllowAll is the access policy template that allows all access
var accessPolicyTemplateAllowAll = storage.AccessPolicyTemplate{
	SystemAttributeBaseModel: ucdb.MarkAsSystem(ucdb.NewSystemAttributeBaseWithID(policy.AccessPolicyTemplateAllowAll.ID)),
	Name:                     "AllowAll",
	Description:              "This template allows all access.",
	Function: `function policy(context, params) {
                return true;
        }`,
}
