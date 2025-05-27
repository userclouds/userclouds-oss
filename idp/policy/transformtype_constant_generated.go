// NOTE: automatically generated file -- DO NOT EDIT

package policy

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t TransformType) MarshalText() ([]byte, error) {
	switch t {
	case TransformTypePassThrough:
		return []byte("passthrough"), nil
	case TransformTypeTokenizeByReference:
		return []byte("tokenizebyreference"), nil
	case TransformTypeTokenizeByValue:
		return []byte("tokenizebyvalue"), nil
	case TransformTypeTransform:
		return []byte("transform"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown TransformType value '%s'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *TransformType) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "passthrough":
		*t = TransformTypePassThrough
	case "tokenizebyreference":
		*t = TransformTypeTokenizeByReference
	case "tokenizebyvalue":
		*t = TransformTypeTokenizeByValue
	case "transform":
		*t = TransformTypeTransform
	default:
		return ucerr.Friendlyf(nil, "unknown TransformType value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *TransformType) Validate() error {
	switch *t {
	case TransformTypePassThrough:
		return nil
	case TransformTypeTokenizeByReference:
		return nil
	case TransformTypeTokenizeByValue:
		return nil
	case TransformTypeTransform:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown TransformType value '%s'", *t)
	}
}

// Enum implements Enum
func (t TransformType) Enum() []any {
	return []any{
		"passthrough",
		"tokenizebyreference",
		"tokenizebyvalue",
		"transform",
	}
}

// AllTransformTypes is a slice of all TransformType values
var AllTransformTypes = []TransformType{
	TransformTypePassThrough,
	TransformTypeTokenizeByReference,
	TransformTypeTokenizeByValue,
	TransformTypeTransform,
}
