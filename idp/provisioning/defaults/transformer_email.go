package defaults

import (
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucdb"
)

// TransformerEmail is a transformer for email in form of x@y.z -> to custom mapping.
// The parameters for the custom mapping are in an array of length 3 corresponding to
// three segments of th email:
//
//	{
//	        PreserveValue: false | true, - causes original value to be preserved
//	        PreserveChars: N, - causes up to N chars from the start of the value to be preserved
//	        FinalLength: N - sets the length of the segment, only applies if the value is not preserved
//	 }
//
// By default the transformer generates id(length 12)@preserve domain.preserve extension
// The domain is only preserved for a list of common domains
// TODO allow preservation of extension with multiple periods like domain.foo.fr
var TransformerEmail = storage.Transformer{
	SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(policy.TransformerEmail.ID),
	Name:                     "EmailToID",
	Description:              "This transformer generates an email token for the given email.",
	InputDataTypeID:          datatype.String.ID,
	OutputDataTypeID:         datatype.String.ID,
	TransformType:            storage.InternalTransformTypeFromClient(policy.TransformerEmail.TransformType),
	Function: `
function id(len) {
        var s = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

        return Array(len).join().split(',').map(function() {
                return s.charAt(Math.floor(Math.random() * s.length));
        }).join('');
}

var commonValues = ["gmail", "hotmail", "yahoo", "msn", "aol", "orange", "wanadoo", "comcast", "live", "apple", "proton", "yandex", "ymail"]

function constructSegment(seg, config) {
        if (config.PreserveValue) {
                return seg
        }

        if (config.PreserveCommonValue && (commonValues.includes(seg))) {
                return seg
        }

        preserveCount = Math.min(config.PreserveChars, seg.length);
        newSeg = seg.slice(0, preserveCount)
        return newSeg + id(config.FinalLength - preserveCount)
}function transform(data, params) {
        emailParts = data.split('@')

        // Make sure we have a username and a domain
        if (emailParts.length !== 2) {
                throw new Error('Invalid Data');
        }

        username = emailParts[0]
        domainParts = emailParts[1].split('.')

        // Check if the domain is valid
        if (domainParts.length < 2) {
                throw new Error('Invalid Data');
        }
        domainName = domainParts[0]
        domainExt = domainParts[1]

        if (params.length != 3) {
                throw new Error('Invalid Params');
        }
        return constructSegment(username, params[0]) + '@' +
                constructSegment(domainName, params[1]) + '.' +
                constructSegment(domainExt, params[2]);
};`,
	Parameters: `
[{
        "PreserveValue": false,
        "PreserveChars": 0,
        "FinalLength": 12
}, {
        "PreserveValue": false,
        "PreserveCommonValue": true,
        "PreserveChars": 0,
        "FinalLength": 6
}, {
        "PreserveValue": true
}]`,
}
