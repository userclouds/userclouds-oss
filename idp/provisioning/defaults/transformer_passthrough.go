package defaults

import (
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucdb"
)

// TransformerPassthrough is a simple transformer to return unchanged data,
// immediately useful for Accessors that don't need a transformer
var TransformerPassthrough = storage.Transformer{
	SystemAttributeBaseModel: ucdb.MarkAsSystem(ucdb.NewSystemAttributeBaseWithID(policy.TransformerPassthrough.ID)),
	Name:                     "PassthroughUnchangedData",
	Description:              `This policy returns the data unchanged. This is useful for Accessors that don't need a transformer.`,
	InputDataTypeID:          datatype.String.ID,
	OutputDataTypeID:         datatype.String.ID,
	Function:                 `function transform(data, params) { return data; }`,
	TransformType:            storage.InternalTransformTypeFromClient(policy.TransformerPassthrough.TransformType),
}
