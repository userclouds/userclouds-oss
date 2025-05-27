package defaults

import (
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucdb"
)

// TransformerSSN transformer for SSN -> id. By default the ID is alphanumeric - ppp-rr-rrrr where
// p is preserved digit and r is a random number
// TODO maybe allow parameter to be a mask with char p,n,l where p is preserved, n is random number, l is random number
// or letter
var TransformerSSN = storage.Transformer{
	SystemAttributeBaseModel: ucdb.NewSystemAttributeBaseWithID(policy.TransformerSSN.ID),
	Name:                     "SSNToID",
	Description:              "This transformer generates a masked SSN.",
	InputDataTypeID:          datatype.String.ID,
	OutputDataTypeID:         datatype.String.ID,
	TransformType:            storage.InternalTransformTypeFromClient(policy.TransformerSSN.TransformType),
	Function: `
function id(len, decimalonly) {
        var s = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
        var d = "0123456789";
        if (decimalonly) {
                return Array(len).join().split(',').map(function() {
                        return d.charAt(Math.floor(Math.random() * d.length));
                }).join('');
        }
        return Array(len).join().split(',').map(function() {
                return s.charAt(Math.floor(Math.random() * s.length));
        }).join('');
}

function constructSegment(seg, decimalonly, preserveS, preserveT) {
        preserveCountS = Math.min(Math.max(preserveS, 0), seg.length);
        preserveCountT = Math.min(Math.max(preserveT, 0), seg.length);

        preserveCount = preserveCountS + preserveCountT
        if (preserveCount >= seg.length) {
                return seg
        }

        newSegS = seg.slice(0, preserveCountS)
        newSegT = seg.slice(seg.length - preserveCountT, seg.length)
        return newSegS + id(seg.length - preserveCount, decimalonly) + newSegT;
}

function validate(str) {
        regexp = /^(?!000|666)[0-8][0-9]{2}(?!00)[0-9]{2}(?!0000)[0-9]{4}$/;

        return regexp.test(str);
}

function transform(data, params) {
        // Strip non numeric characters if present
        orig_data = data;
        data = data.replace(/\D/g, '');
        if (!validate(data)) {
                throw new Error('Invalid SSN Provided');
        }

        if ((params.PreserveCharsTrailing + params.PreserveCharsStart) > 9 ||
                params.PreserveCharsTrailing < 0 || params.PreserveCharsStart < 0) {
                throw new Error('Invalid Params Provided');
        }

        if (params.PreserveValue) {
                return orig_data;
        }

        seg1 = data.slice(0, 3);
        seg2 = data.slice(3, 5);
        seg3 = data.slice(5, 9);
        return constructSegment(
                        seg1,
                        params.DecimalOnly,
                        params.PreserveCharsStart,
                        params.PreserveCharsTrailing - 6
                ) +
                '-' +
                constructSegment(
                        seg2,
                        params.DecimalOnly,
                        params.PreserveCharsStart - 3,
                        params.PreserveCharsTrailing - 4
                ) +
                '-' +
                constructSegment(
                        seg3,
                        params.DecimalOnly,
                        params.PreserveCharsStart - 5,
                        params.PreserveCharsTrailing
                );
};`,
	Parameters: `{
        "PreserveValue": false,
        "DecimalOnly": true,
        "PreserveCharsTrailing": 0,
        "PreserveCharsStart": 3
}`,
}
