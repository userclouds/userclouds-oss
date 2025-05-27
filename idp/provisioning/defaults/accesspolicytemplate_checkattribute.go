package defaults

import (
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/infra/ucdb"
)

// accessPolicyTemplateCheckAttribute is the implementation of the CheckAttribute template
var accessPolicyTemplateCheckAttribute = storage.AccessPolicyTemplate{
	SystemAttributeBaseModel: ucdb.MarkAsSystem(ucdb.NewSystemAttributeBaseWithID(policy.AccessPolicyTemplateCheckAttribute.ID)),
	Name:                     "CheckAttribute",
	Description:              "This template returns the value of checkAttribute on the given parameters.",
	Function: `function policy(context, params) {
                const id1 = params.userIDUsage === "id1" ? context.user.id : params.id1;
                const id2 = params.userIDUsage === "id2" ? context.user.id : params.id2;
                const attribute = params.attribute;
                if (!id1 || !id2 || !attribute) {
                        return false;
                }
                try {
                        return checkAttribute(id1, id2, attribute);
                } catch (e) {
                        return false;
                }
        }`,
}
