package policy

import (
	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucdb"
)

// AccessPolicyGlobalAccessorID is the ID of the global accessor policy
var AccessPolicyGlobalAccessorID = uuid.Must(uuid.FromString("a78f1f88-3684-4e59-a01d-c121e259ec96"))

// AccessPolicyGlobalMutatorID is the ID of the global mutator policy
var AccessPolicyGlobalMutatorID = uuid.Must(uuid.FromString("804e84f1-7fa4-4bb4-b785-4c89e1ceaba0"))

// AccessPolicyAllowAll access policy that allows anything
var AccessPolicyAllowAll = AccessPolicy{
	ID: uuid.Must(uuid.FromString("3f380e42-0b21-4570-a312-91e1b80386fa")),
}

// AccessPolicyTemplateAllowAll access policy that allows anything
var AccessPolicyTemplateAllowAll = AccessPolicyTemplate{
	SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(uuid.Must(uuid.FromString("1e742248-fdde-4c88-9ea7-2c2106ec7aa8"))),
}

// AccessPolicyDenyAll access policy that denies everything
var AccessPolicyDenyAll = AccessPolicy{
	ID: uuid.Must(uuid.FromString("c9c14750-b8f3-4507-bd3f-5c6562f0a6e6")),
}

// AccessPolicyTemplateDenyAll access policy that denies everything
var AccessPolicyTemplateDenyAll = AccessPolicyTemplate{
	SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(uuid.Must(uuid.FromString("c88d97a6-a3ae-4af8-b018-2bcddf1fa606"))),
}

// AccessPolicyTemplateCheckAttribute is a template that calls CheckAttribute
var AccessPolicyTemplateCheckAttribute = AccessPolicyTemplate{
	SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(uuid.Must(uuid.FromString("aad2bf25-311f-467e-9169-a6a89b6d34a6"))),
}

// TransformerUUID transformer for replacing data with a uuid
var TransformerUUID = Transformer{
	ID:            uuid.Must(uuid.FromString("e3743f5b-521e-4305-b232-ee82549e1477")),
	Name:          "UUID",
	TransformType: TransformTypeTokenizeByValue,
}

// TransformerEmail transformer for email by default preserving the domain but not username
var TransformerEmail = Transformer{
	ID:            uuid.Must(uuid.FromString("0cedf7a4-86ab-450a-9426-478ad0a60faa")),
	TransformType: TransformTypeTokenizeByValue,
}

// TransformerFullName transformer for full name, by default preserving the first letters of first and
// last name
var TransformerFullName = Transformer{
	ID:            uuid.Must(uuid.FromString("b9bf352f-b1ee-4fb2-a2eb-d0c346c6404b")),
	TransformType: TransformTypeTransform,
}

// TransformerSSN transformer for SSN
var TransformerSSN = Transformer{
	ID:            uuid.Must(uuid.FromString("3f65ee22-2241-4694-bbe3-72cefbe59ff2")),
	TransformType: TransformTypeTransform,
}

// TransformerCreditCard transformer for credit card numbers
var TransformerCreditCard = Transformer{
	ID:            uuid.Must(uuid.FromString("618a4ae7-9979-4ee8-bac5-db87335fe4d9")),
	TransformType: TransformTypeTransform,
}

// TransformerPassthrough is a transformer that passes through the data without changing it
// (most immediately useful in secured Accessors)
var TransformerPassthrough = Transformer{
	ID:            uuid.Must(uuid.FromString("c0b5b2a1-0b1f-4b9f-8b1a-1b1f4b9f8b1a")),
	Name:          "PassthroughUnchangedData",
	TransformType: TransformTypePassThrough,
}
