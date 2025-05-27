package defaults

import (
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucdb"
)

// TransformerUUID transformer for data -> uuid mapping
var TransformerUUID = storage.Transformer{
	SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(policy.TransformerUUID.ID),
	Name:                     "UUID",
	Description:              "This transformer generates a UUID token for the data.",
	InputDataTypeID:          datatype.String.ID,
	OutputDataTypeID:         datatype.String.ID,
	TransformType:            storage.InternalTransformTypeFromClient(policy.TransformerUUID.TransformType),
	Function: `
function uuidv4() {
        return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
                var r = Math.random() * 16 | 0, v = c === 'x' ? r : ((r & 0x3) | 0x8);
                return v.toString(16);
        });
};

function transform(data, params) {
        return JSON.stringify(uuidv4());
};`,
	Parameters: `{}`,
}
