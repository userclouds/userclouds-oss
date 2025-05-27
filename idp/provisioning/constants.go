package provisioning

import (
	"userclouds.com/authz"
	idpAuthz "userclouds.com/idp/authz"
	"userclouds.com/infra/ucdb"
)

// DefaultAuthZObjectTypes is an array containing default Tokenizer AuthZ object types
var DefaultAuthZObjectTypes = []authz.ObjectType{
	{BaseModel: ucdb.NewBaseWithID(idpAuthz.PoliciesTypeID), TypeName: idpAuthz.ObjectTypeNamePolicies},
	{BaseModel: ucdb.NewBaseWithID(idpAuthz.PolicyAccessTemplateTypeID), TypeName: idpAuthz.ObjectTypeNameAccessPolicyTemplate},
	{BaseModel: ucdb.NewBaseWithID(idpAuthz.PolicyAccessTypeID), TypeName: idpAuthz.ObjectTypeNameAccessPolicy},
	{BaseModel: ucdb.NewBaseWithID(idpAuthz.PolicyTransformerTypeID), TypeName: idpAuthz.ObjectTypeNameTransformer},
}

// DefaultAuthZObjects is an array containing default Tokenizer AuthZ objects
var DefaultAuthZObjects = []authz.Object{
	{BaseModel: ucdb.NewBaseWithID(idpAuthz.PoliciesObjectID), TypeID: idpAuthz.PoliciesTypeID /* this should go into Name "_policies" + idpAuthz.PoliciesObjectID.String() */},
}

// DefaultAuthZEdgeTypes is an array containing default Tokenizer AuthZ edge types
var DefaultAuthZEdgeTypes = []authz.EdgeType{
	{BaseModel: ucdb.NewBaseWithID(idpAuthz.PolicyAccessExistsEdgeTypeID),
		TypeName:           idpAuthz.EdgeTypeNameAccessPolicyExists,
		SourceObjectTypeID: idpAuthz.PoliciesTypeID,
		TargetObjectTypeID: idpAuthz.PolicyAccessTypeID,
		Attributes: []authz.Attribute{
			{Name: idpAuthz.AttributePolicyCreate, Propagate: true},
			{Name: idpAuthz.AttributePolicyRead, Propagate: true},
			{Name: idpAuthz.AttributePolicyUpdate, Propagate: true},
			{Name: idpAuthz.AttributePolicyDelete, Propagate: true},
		},
	},
	{BaseModel: ucdb.NewBaseWithID(idpAuthz.PolicyTransformerExistsEdgeTypeID),
		TypeName:           idpAuthz.EdgeTypeNameTransformerExists,
		SourceObjectTypeID: idpAuthz.PoliciesTypeID,
		TargetObjectTypeID: idpAuthz.PolicyTransformerTypeID,
		Attributes: []authz.Attribute{
			{Name: idpAuthz.AttributePolicyCreate, Propagate: true},
			{Name: idpAuthz.AttributePolicyRead, Propagate: true},
			{Name: idpAuthz.AttributePolicyUpdate, Propagate: true},
			{Name: idpAuthz.AttributePolicyDelete, Propagate: true},
		},
	},
	{BaseModel: ucdb.NewBaseWithID(idpAuthz.UserPolicyFullAccessEdgeTypeID),
		TypeName:           idpAuthz.EdgeTypeNameUserPolicyFullAccess,
		SourceObjectTypeID: authz.UserObjectTypeID,
		TargetObjectTypeID: idpAuthz.PoliciesTypeID,
		Attributes: []authz.Attribute{
			{Name: idpAuthz.AttributePolicyCreate, Direct: true},
			{Name: idpAuthz.AttributePolicyRead, Direct: true},
			{Name: idpAuthz.AttributePolicyUpdate, Direct: true},
			{Name: idpAuthz.AttributePolicyDelete, Direct: true},
		},
	},
	{BaseModel: ucdb.NewBaseWithID(idpAuthz.UserPolicyReadAccessEdgeTypeID),
		TypeName:           idpAuthz.EdgeTypeNameUserPolicyReadAccess,
		SourceObjectTypeID: authz.UserObjectTypeID,
		TargetObjectTypeID: idpAuthz.PoliciesTypeID,
		Attributes: []authz.Attribute{
			{Name: idpAuthz.AttributePolicyRead, Direct: true},
		},
	},
	{BaseModel: ucdb.NewBaseWithID(idpAuthz.GroupPolicyFullAccessEdgeTypeID),
		TypeName:           idpAuthz.EdgeTypeNameGroupPolicyFullAccess,
		SourceObjectTypeID: authz.GroupObjectTypeID,
		TargetObjectTypeID: idpAuthz.PoliciesTypeID,
		Attributes: []authz.Attribute{
			{Name: idpAuthz.AttributePolicyCreate, Direct: true},
			{Name: idpAuthz.AttributePolicyRead, Direct: true},
			{Name: idpAuthz.AttributePolicyUpdate, Direct: true},
			{Name: idpAuthz.AttributePolicyDelete, Direct: true},
		},
	},
	{BaseModel: ucdb.NewBaseWithID(idpAuthz.GroupPolicyReadAccessEdgeTypeID),
		TypeName:           idpAuthz.EdgeTypeNameGroupPolicyReadAccess,
		SourceObjectTypeID: authz.GroupObjectTypeID,
		TargetObjectTypeID: idpAuthz.PoliciesTypeID,
		Attributes: []authz.Attribute{
			{Name: idpAuthz.AttributePolicyRead, Direct: true},
		},
	},
	{BaseModel: ucdb.NewBaseWithID(idpAuthz.UserGroupPolicyFullAccessEdgeTypeID),
		TypeName:           idpAuthz.EdgeTypeNameUserGroupPolicyFullAccess,
		SourceObjectTypeID: authz.UserObjectTypeID,
		TargetObjectTypeID: authz.GroupObjectTypeID,
		Attributes: []authz.Attribute{
			{Name: idpAuthz.AttributePolicyCreate, Inherit: true},
			{Name: idpAuthz.AttributePolicyRead, Inherit: true},
			{Name: idpAuthz.AttributePolicyUpdate, Inherit: true},
			{Name: idpAuthz.AttributePolicyDelete, Inherit: true},
		},
	},
	{BaseModel: ucdb.NewBaseWithID(idpAuthz.UserGroupPolicyReadAccessEdgeTypeID),
		TypeName:           idpAuthz.EdgeTypeNameUserGroupPolicyReadAccess,
		SourceObjectTypeID: authz.UserObjectTypeID,
		TargetObjectTypeID: authz.GroupObjectTypeID,
		Attributes: []authz.Attribute{
			{Name: idpAuthz.AttributePolicyRead, Inherit: true},
		},
	},
}

// DefaultAuthZEdges is an array containing default Tokenizer AuthZ edges
var DefaultAuthZEdges = []authz.Edge{}
