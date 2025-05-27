package jsonapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"

	"github.com/go-http-utils/headers"

	"userclouds.com/infra"
	"userclouds.com/infra/request"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// Option defines a method of passing optional args to jsonapi
type Option interface {
	Apply(*opts)
}

type opts struct {
	code int
}

type codeOpt struct {
	code int
}

func (c codeOpt) Apply(o *opts) {
	o.code = c.code
}

// Code can be passed as an option to MarshalError to override HTTP 500
func Code(c int) Option {
	return codeOpt{code: c}
}

// JSONError is just an error from jsonapi.Unmarshal that automatically has a
// developer-friendly error message (what decoding failed, but no stack trace),
// and a default HTTP 400 status code.
type JSONError struct {
	error    // embedded so we inherit Error implementation
	friendly string
}

// Code implements errorWithCode
func (JSONError) Code() int {
	return http.StatusBadRequest
}

// Friendly implements UCError
func (j JSONError) Friendly() string {
	return j.friendly
}

// FriendlyStructure implements UCError
func (JSONError) FriendlyStructure() any {
	return nil
}

// Unmarshal handles decoding JSON POST bodies. If the response object is Validateable, we'll do that too
func Unmarshal(r *http.Request, i any) error {
	if r.ContentLength != 0 {
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(i); err != nil {
			// NB: this is slightly complicated, but if this was an error we generated (eg. in custom
			// Unmarshaler logic), we want to use the friendly error message so that we don't leak a minor stack.
			// But if this was an error directly from `json` (like
			// "json: cannot unmarshal string into Go struct field jsonMarshalTestStruct.id of type int"), we want to
			// use the actual error to help debugging.
			f := ucerr.UserFriendlyMessage(err)
			var uce ucerr.UCError
			if !errors.As(err, &uce) {
				f = err.Error()
			}
			return ucerr.Wrap(JSONError{
				error:    err,
				friendly: f, // don't leak stacks
			})
		}
	}

	v, ok := i.(infra.Validateable)
	if ok {
		if err := v.Validate(); err != nil {
			return ucerr.Wrap(JSONError{
				error:    err,
				friendly: ucerr.UserFriendlyMessage(err),
			})
		}
	}
	return nil
}

func marshal(w http.ResponseWriter, r any, options opts) {
	if options.code == -1 {
		uclog.Errorf(context.Background(), "jsonapi.marshal called without a status code option, defaulting to %d", http.StatusOK)
		options.code = http.StatusOK
	}

	w.Header().Set(headers.ContentType, "application/json")
	w.WriteHeader(options.code)

	// TODO: I don't love using reflect here, but it's the only way I can think of to handle empty slices
	// consistently. The problem is that json.Encoder.Encode() will write "null" for an uninitialized slice,
	// and sometimes our APIs hand `jsonapi.Marshal()` an uninitialized slice. (eg #824 where console is just
	// writing out a potentially-empty list of accessors to the UI). The alternative would be checking len==0
	// at all possible call sites, and ensuring that the slice is initialized to empty there, but that's lame.
	// FWIW, json.Marshal already uses reflect under the covers so it's unlikely we are the major perf bottleneck
	// There's a related problem with embedded slices in structs, but `omitempty` generally works for those.
	if reflect.TypeOf(r) != nil && reflect.TypeOf(r).Kind() == reflect.Slice && reflect.ValueOf(r).Len() == 0 {
		w.Write([]byte("[]"))
		return
	} else if r == nil {
		// Nothing to encode
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(r); err != nil {
		// NB: we are using context.Background() here because we don't have a request context, and it
		// doesn't seem worth passing context to every jsonapi.Marshal call just for this (although I'm
		// open to it if people disagreed))
		// Also note that if/when this fails, we've already written the status code header, so we can't
		// change it here...
		requestID := request.GetRequestIDFromHeader(w.Header())
		uclog.Errorf(context.Background(), "jsonapi.Marshal failed on request %v: %v", requestID, err)
	}
}

// Marshal sends a JSON-encoded default HTTP 200 responses
func Marshal(w http.ResponseWriter, r any, os ...Option) {
	options := applyOptions(os...)
	if options.code == -1 {
		options.code = http.StatusOK
	}

	marshal(w, r, options)
}

func applyOptions(os ...Option) opts {
	options := opts{code: -1}
	for _, o := range os {
		o.Apply(&options)
	}
	return options
}
