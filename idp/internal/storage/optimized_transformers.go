package storage

import (
	"github.com/gofrs/uuid"

	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucdb"
)

// TransformerFunc is a transformer function prototype
type TransformerFunc func(data string, params string) string

type transformerRecord struct {
	fun TransformerFunc
	t   Transformer
}

var nativeTransformers = map[uuid.UUID]transformerRecord{
	policy.TransformerUUID.ID: {
		fun: transformerUUID,
		t: Transformer{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(policy.TransformerUUID.ID),
			Name:                     policy.TransformerUUID.Name,
			TransformType:            InternalTransformTypeFromClient(policy.TransformerUUID.TransformType),
			InputDataTypeID:          datatype.String.ID,
			OutputDataTypeID:         datatype.UUID.ID,
		},
	},
	policy.TransformerPassthrough.ID: {
		fun: transformerPassthrough,
		t: Transformer{
			SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(policy.TransformerPassthrough.ID),
			Name:                     policy.TransformerPassthrough.Name,
			TransformType:            InternalTransformTypeFromClient(policy.TransformerPassthrough.TransformType),
			InputDataTypeID:          datatype.String.ID,
			OutputDataTypeID:         datatype.String.ID,
		},
	},
}

// GetNativeTransformer returns a pointer to a native transformer func if one exists for the transformer id
func GetNativeTransformer(transformerID uuid.UUID) *TransformerFunc {
	if record, found := nativeTransformers[transformerID]; found {
		return &record.fun
	}

	return nil
}

// This function corresponds to provisioning.TransformerUUID
func transformerUUID(data string, params string) string {
	return uuid.Must(uuid.NewV4()).String()
}

// This function corresponds to provisioning.TransformerPassthrough
func transformerPassthrough(data string, params string) string {
	return data
}
