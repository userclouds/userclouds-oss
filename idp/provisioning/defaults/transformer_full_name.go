package defaults

import (
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucdb"
)

// TransformerFullName transformer for full name, by default preserving the first letters of first and last name
var TransformerFullName = storage.Transformer{
	SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(policy.TransformerFullName.ID),
	Name:                     "FullNameToID",
	Description:              "This transformer generates a masked name.",
	InputDataTypeID:          datatype.String.ID,
	OutputDataTypeID:         datatype.String.ID,
	TransformType:            storage.InternalTransformTypeFromClient(policy.TransformerFullName.TransformType),
	Function: `
function id(len) {
        var s = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
        return Array(len).join().split(',').map(function() {
                return s.charAt(Math.floor(Math.random() * s.length));
        }).join('');
}

function constructSegment(seg, config) {
        if (config.PreserveValue) {
                return seg
        }
        preserveCount = Math.min(config.PreserveChars, seg.length);
        newSeg = seg.slice(0, preserveCount)
        return newSeg + id(config.FinalLength - preserveCount)
}

function transform(data, params) {
        nameParts = data.split(' ')

        // Assume that if we have a single name, treat it as a first name
        firstName = data;
        lastName = "";
        if (nameParts.length > 0) {
                firstName = nameParts[0]
        }

        // Skip middle name if provided
        if (nameParts.length > 1) {
                lastName = nameParts[nameParts.length - 1]
        }

        if (params.length != 2) {
                throw new Error('Invalid Params');
        }

        return constructSegment(firstName, params[0]) + ' ' +
                constructSegment(lastName, params[1]);
};`,
	Parameters: `[{
        "PreserveValue": true
}, {
        "PreserveValue": false,
        "PreserveChars": 1,
        "FinalLength": 12
}]`,
}
