package authz

import (
	"github.com/gofrs/uuid"
)

// ObjectTypeNameTransformer
// Tokenizer AuthZ object types & edge types (roles) provisioned for every tenant.
const (
	ObjectTypeNamePolicies                = "_policies"
	ObjectTypeNameAccessPolicy            = "_access_policy"
	ObjectTypeNameAccessPolicyTemplate    = "_access_policy_template"
	ObjectTypeNameTransformer             = "_transformer"
	EdgeTypeNameAccessPolicyExists        = "_policy_exist_access"
	EdgeTypeNameTransformerExists         = "_policy_exist_transformation"
	EdgeTypeNameUserPolicyFullAccess      = "_user_policy_full"
	EdgeTypeNameUserPolicyReadAccess      = "_user_policy_read"
	EdgeTypeNameGroupPolicyFullAccess     = "_group_policy_full"
	EdgeTypeNameGroupPolicyReadAccess     = "_group_policy_read"
	EdgeTypeNameUserGroupPolicyFullAccess = "_user_group_policy_full"
	EdgeTypeNameUserGroupPolicyReadAccess = "_user_group_policy_read"
	AttributePolicyCreate                 = "_policy_create"
	AttributePolicyRead                   = "_policy_read"
	AttributePolicyUpdate                 = "_policy_update"
	AttributePolicyDelete                 = "_policy_delete"
)

// PoliciesTypeID is the ID of global policy object type "_policies"
var PoliciesTypeID = uuid.Must(uuid.FromString("e410fb38-2158-4431-8b66-fb547b55f2da"))

// PoliciesObjectID is the ID of global policy object "_policies"
var PoliciesObjectID = uuid.Must(uuid.FromString("61553ed5-749e-4d8a-98c4-fa80e048e3fb"))

// PolicyAccessTypeID is the ID of the access policy object type
var PolicyAccessTypeID = uuid.Must(uuid.FromString("9bb5d92b-41df-4dc2-8ac1-418365c408ab"))

// PolicyAccessTemplateTypeID is the ID of the access policy template object type
var PolicyAccessTemplateTypeID = uuid.Must(uuid.FromString("23e22596-c1f0-43f6-9173-fa0449b2d1ad"))

// PolicyTransformerTypeID is the ID of the transformer object type
var PolicyTransformerTypeID = uuid.Must(uuid.FromString("260c7221-729c-478b-843b-e00168a0cdc6"))

// PolicyAccessExistsEdgeTypeID is edge type that connects from the global policy object to all access policies
var PolicyAccessExistsEdgeTypeID = uuid.Must(uuid.FromString("2dbccbff-44fe-4ccf-b104-2ac5cb48bea0"))

// PolicyTransformerExistsEdgeTypeID is edge type that connects from the global policy object to all transformers
var PolicyTransformerExistsEdgeTypeID = uuid.Must(uuid.FromString("ca3d9771-1077-4fd8-a3f2-3ceaab1e63f3"))

// UserPolicyFullAccessEdgeTypeID is the ID for an attribute set that allows a user full access to a policy
var UserPolicyFullAccessEdgeTypeID = uuid.Must(uuid.FromString("58840f76-65b6-4006-8286-8927246ee9f9"))

// UserPolicyReadAccessEdgeTypeID is the ID for an attribute set that allows a user to read a policy
var UserPolicyReadAccessEdgeTypeID = uuid.Must(uuid.FromString("bb2baf72-4acc-4b73-9ea5-8d936af91a55"))

// GroupPolicyFullAccessEdgeTypeID is the ID for an attribute set that allows a group full access to a policy
var GroupPolicyFullAccessEdgeTypeID = uuid.Must(uuid.FromString("1569fa26-2458-4041-8cf3-5532f0d0670a"))

// GroupPolicyReadAccessEdgeTypeID is the ID for an attribute set that allows a group to read a policy
var GroupPolicyReadAccessEdgeTypeID = uuid.Must(uuid.FromString("1eb8cb99-9a1f-4e14-9651-b32e1e2df614"))

// UserGroupPolicyFullAccessEdgeTypeID is the ID for an attribute set that allows a user full access to a policy via a group
var UserGroupPolicyFullAccessEdgeTypeID = uuid.Must(uuid.FromString("51734876-a296-469f-9978-c4d3565cd877"))

// UserGroupPolicyReadAccessEdgeTypeID is the ID for an attribute set that allows a user read access to a policy via a group
var UserGroupPolicyReadAccessEdgeTypeID = uuid.Must(uuid.FromString("f11c57e1-91fb-4fdc-9357-a955339eedb9"))
