package ucjson

import (
	"encoding/json"

	"github.com/userclouds/easyjson"

	"userclouds.com/infra/ucerr"
)

// Marshal marshals the given value to JSON, using the highest-performance
// available method.
func Marshal(v any) ([]byte, error) {
	if e, ok := v.(easyjson.Marshaler); ok {
		return easyjson.Marshal(e)
	}
	return json.Marshal(v)
}

// MarshalIndent marshals the given value to JSON
// this really only exists to let us enforce ucjson.Marshal easily in the ucwrapper linter :)
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

// Unmarshal unmarshals the given JSON data into the given value, using the
// highest-performance available method. Even though technically json.Unmarshal would
// call into (type).UnmarshalJSON(), which easyjson generates, it's still much slower
// because of all the reflection & validation that happens up front in json.Unmarshal.
func Unmarshal(data []byte, v any) error {
	if e, ok := v.(easyjson.Unmarshaler); ok {
		return ucerr.Wrap(easyjson.Unmarshal(data, e))
	}
	return ucerr.Wrap(json.Unmarshal(data, v))
}
