package assert

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"userclouds.com/infra/jsonclient"
)

// Equal asserts two objects are equal
func Equal(t testing.TB, got, want any, opts ...Option) bool {
	t.Helper() // mark ourselves as irrelevant in the call stack for errors, etc
	os := buildOpts(opts)
	if cmp.Equal(got, want, os.cmpOpts...) {
		return true
	}
	logFailure(t, got, want, opts)
	return false
}

// NotEqual asserts two objects are not equal
func NotEqual(t testing.TB, got, want any, opts ...Option) bool {
	t.Helper() // mark ourselves as irrelevant in the call stack for errors, etc
	os := buildOpts(opts)
	if !cmp.Equal(got, want, os.cmpOpts...) {
		return true
	}
	logFailure(t, got, want, opts)
	return false
}

// True is a shortcut for Equals(true)
func True(t testing.TB, got bool, opts ...Option) bool {
	t.Helper()
	return Equal(t, got, true, opts...)
}

// False is a shortcut for Equals(false)
func False(t testing.TB, got bool, opts ...Option) bool {
	t.Helper()
	return Equal(t, got, false, opts...)
}

// IsNil is a shortcut for Equals(nil)
func IsNil(t testing.TB, got any, opts ...Option) bool {
	t.Helper()
	if isNil(got) {
		return true
	}
	logFailure(t, got, nil, opts)
	return false
}

// NotNil is a shortcut for !Equals(nil)
func NotNil(t testing.TB, got any, opts ...Option) bool {
	t.Helper()
	if !isNil(got) {
		return true
	}
	logFailure(t, got, "!<nil>", opts)
	return false
}

// NoErr is a shortcut for Nil(..., assert.Must())
func NoErr(t testing.TB, err error, opts ...Option) {
	t.Helper()
	opts = append(opts, Must())
	IsNil(t, err, opts...)
}

// Contains asserts that one string contains another
func Contains(t testing.TB, body, substr string, opts ...Option) bool {
	t.Helper()
	if strings.Contains(body, substr) {
		return true
	}
	logFailure(t, body, fmt.Sprintf("a string containing '%s'", substr), opts)
	return false
}

// DoesNotContain asserts that one string does not contain another
func DoesNotContain(t testing.TB, body, substr string, opts ...Option) bool {
	t.Helper()
	if !strings.Contains(body, substr) {
		return true
	}
	logFailure(t, body, fmt.Sprintf("a string not containing '%s'", substr), opts)
	return false
}

// FailContinue marks the test as failed, but continues execution
func FailContinue(t testing.TB, fmt string, args ...any) {
	t.Helper()
	logFailure(t, nil, nil, []Option{Errorf(fmt, args...)})
}

// Fail marks the test as failed and stops execution
func Fail(t testing.TB, fmt string, args ...any) {
	t.Helper()
	logFailure(t, nil, nil, []Option{Must(), Errorf(fmt, args...)})
}

// HTTPError asserts that an error is an HTTP error with the given status code
func HTTPError(t *testing.T, err error, code int) {
	t.Helper()
	NotNil(t, err, Errorf("Expected HTTP error with HTTP status code %d, got nil", code))
	httpCode := jsonclient.GetHTTPStatusCode(err)
	NotEqual(t, httpCode, -1, Errorf("Error is not a HTTP error (%v)", err))
	Equal(t, httpCode, code)
}

func isNil(got any) bool {
	if got == nil {
		return true
	}
	// if it's not immediately nil, we need to understand if it's just the type of the interface
	// that's not nil, or the actual value of the pointer inside the interface.
	val := reflect.ValueOf(got)
	switch val.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return val.IsNil()
	default:
		return false
	}
}

func copyAppendOpts(opt Option, opts ...Option) []Option {
	newOpts := make([]Option, 0, len(opts)+1)
	newOpts = append(newOpts, opts...)
	newOpts = append(newOpts, opt)
	return newOpts
}

// ErrorIs asserts that error is not nil and it matches the expected error
func ErrorIs(t testing.TB, got, expected error, opts ...Option) {
	t.Helper()
	True(t, errors.Is(got, expected), copyAppendOpts(Errorf("Expected error '%v' got: '%v'", expected, got), opts...)...)
}

func buildOpts(opts []Option) options {
	var os options
	for _, o := range opts {
		o.apply(&os)
	}
	return os
}

func logFailure(t testing.TB, got, want any, opts []Option) {
	os := buildOpts(opts)
	var extraMsg string
	if os.msg != "" {
		extraMsg = fmt.Sprintf(": %s", os.msg)
	}

	t.Helper()
	msg := &bytes.Buffer{}
	fmt.Fprintf(msg, "assertion failed%s\n", extraMsg)
	fmt.Fprint(msg, alignLines(" got: ", got))
	fmt.Fprint(msg, alignLines("want: ", want))

	if os.diff && got != nil && want != nil {
		fmt.Fprintf(msg, "\ndiff (-want, +got):\n%v", cmp.Diff(want, got))
	}

	if os.stop {
		t.Fatal(msg.String())
	}

	t.Error(msg.String())
}

// make sure that multiline error messages / stack traces are aligned
func alignLines(prefix string, body any) string {
	msg := &bytes.Buffer{}
	fmt.Fprint(msg, prefix)
	spaces := strings.Repeat(" ", len(prefix))

	b := fmt.Sprintf("%+v", body)
	lines := strings.Split(b, "\n")
	if len(lines) < 1 {
		return msg.String()
	}

	fmt.Fprintf(msg, "%s\n", lines[0])
	for _, l := range lines[1:] {
		fmt.Fprintf(msg, "%s%s\n", spaces, l)
	}

	return msg.String()
}
