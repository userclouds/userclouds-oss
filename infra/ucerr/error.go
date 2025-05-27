package ucerr

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// UCError lets us figure out if this is a wrapped error
type UCError interface {
	Error() string // include this so UCError implements Error for erroras linter
	Friendly() string
	FriendlyStructure() any
}

type ucError struct {
	text      string // this is intended for internal use
	friendly  string // (optional) this will get propagated to the user (or developer-user)
	structure any    // if non-nil, then FriendlyStructure() will a marshalable struct as its value

	underlying []error

	function string
	filename string
	line     int
}

// Option defines a way to modify ucerr behavior
type Option interface {
	apply(*options)
}

type options struct {
	skipFrames int
}

type optFunc func(*options)

func (o optFunc) apply(os *options) {
	o(os)
}

var errorWrappingSuffix = ": %w"
var wrappedText = "(wrapped)"

// Return a path relative to the repo root
// If the path is not within the repo, return the path unmodified.
func repoRelativePath(s string) string {
	// need the trailing slash here so the paths are obviously relative
	return strings.TrimPrefix(s, fmt.Sprintf("%s/", repoRelativeBasePath()))
}

// Given a fully qualified go function name "pkgname.[type].func",
// return "func" (or return string unchanged if no period found).
func funcName(s string) string {
	if idx := strings.LastIndex(s, "."); idx >= 0 {
		return s[idx+1:]
	}
	return s
}

// Error implements error
func (e ucError) Error() string {
	var messages []string
	for _, u := range e.underlying {
		if u == nil {
			continue
		}
		messages = append(messages, fmt.Sprintf("%s\n", u.Error()))
	}

	msg := strings.Join(messages, "\n")

	// fall back to friendly message if no internal message is defined
	t := e.text
	if e.text == "" {
		t = fmt.Sprintf("[friendly] %s", e.friendly)
	}

	return fmt.Sprintf("%s%s (File %s:%d, in %s)", msg, t, e.filename, e.line, e.function)
}

// Unwrap implements errors.Unwrap for errors.Is
// Note that errors.Is supports Unwrap returning error or []error, but
// errors.Unwrap (which we don't really use) only supports interface { Unwrap() error }
func (e *ucError) Unwrap() []error {
	if e == nil || len(e.underlying) == 0 {
		return nil
	}
	return e.underlying
}

// New creates a new ucerr
func New(text string) error {
	wraps := errors.New(text) // this makes the base error just the text, no stack trace
	return new(text, "", wraps, 1, nil)
}

// Errorf is our local version of fmt.Errorf including callsite info
func Errorf(temp string, args ...any) error {
	var wrapped error
	// if using %w to wrap another error, use our wrapping mechanism
	if strings.HasSuffix(temp, errorWrappingSuffix) {
		temp = strings.TrimSuffix(temp, errorWrappingSuffix)
		// use the safe cast in case this fails
		var ok bool
		wrapped, ok = args[len(args)-1].(error)
		if !ok {
			wrapped = New("seems as if ucerr.Errorf() was called with a non-error %w")
		}
		args = args[0 : len(args)-1]
	}
	return new(fmt.Sprintf(temp, args...), "", wrapped, 1, nil)
}

// Friendlyf wraps an error with a user-friendly message
func Friendlyf(err error, format string, args ...any) error {
	s := fmt.Sprintf(format, args...)
	return new("", s, err, 1, nil)
}

// WrapWithFriendlyStructure wraps an error with a structured error
func WrapWithFriendlyStructure(err error, structure any) error {
	return new("", "", err, 1, structure)
}

// Wrap wraps an existing error with an additional level of the callstack
func Wrap(err error, opts ...Option) error {
	if err == nil {
		return nil
	}
	var options options
	for _, opt := range opts {
		opt.apply(&options)
	}
	return new(wrappedText, "", err, options.skipFrames+1, nil)
}

// ExtraSkip tells Wrap to skip an extra frame in the stack when wrapping an error
// This allows calls like uchttp.Error() and jsonapi.MarshalError() to call Wrap()
// and capture the stack frame that actually logged the error (since we rarely call
// eg, jsonapi.MarshalError(ucerr.Wrap(err)), we lose useful debugging data)
func ExtraSkip() Option {
	return optFunc(func(o *options) { o.skipFrames++ })
}

// Combine lets you merge multiple errors and return them all (eg. 2 of 5 in a batch failed)
// TODO: write some tests, fix up output nicely
func Combine(l, r error) error {
	if l == nil && r == nil {
		return nil
	}

	// choose the non-nil base
	base, other := l, r
	if base == nil {
		other, base = base, other
	}

	newBase, ok := base.(*ucError) // lint: ecast-safe
	if !ok {
		newBase = Wrap(base, ExtraSkip()).(*ucError) // lint: ecast-safe
	}

	if _, ok := other.(*ucError); !ok { // lint: ecast-safe
		other = Wrap(other, ExtraSkip())
	}
	newBase.underlying = append([]error{other}, newBase.underlying...)
	return newBase
}

// skips is the number of stack frames (besides new itself) to skip
func new(text, friendly string, wraps error, skips int, structure any) error {
	function, filename, line := whereAmI(skips + 1)
	err := &ucError{
		text:       text,
		friendly:   friendly,
		underlying: []error{wraps},
		function:   function,
		filename:   filename,
		line:       line,
		structure:  structure,
	}
	return err
}

// s == stack frames to skip not including myself
func whereAmI(s int) (string, string, int) {
	pc, filename, line, ok := runtime.Caller(s + 1)
	if !ok {
		return "", "", 0
	}
	f := runtime.FuncForPC(pc)
	return funcName(f.Name()), repoRelativePath(filename), line
}

// Friendly returns the friendly message, if any, or default string
// Currently takes the first one in the stack, although we could
// eventually extend this to allow composing etc
func (e ucError) Friendly() string {
	if e.friendly != "" {
		return e.friendly
	}

	var uce UCError
	for _, u := range e.underlying {
		if errors.As(u, &uce) {
			return uce.Friendly()
		}
	}

	return "an unspecified error occurred"
}

// FriendlyStructure returns something that can be marshaled to JSON for the client to
// access programmatically
func (e ucError) FriendlyStructure() any {
	if e.structure != nil {
		return e.structure
	}

	var uce UCError
	for _, u := range e.underlying {
		if errors.As(u, &uce) {
			return uce.FriendlyStructure()
		}
	}

	return nil
}

// UserFriendlyMessage is just a simple wrapper to handle casting error -> ucError
func UserFriendlyMessage(err error) string {
	var uce UCError
	if errors.As(err, &uce) {
		return uce.Friendly()
	}

	// note subtle difference in language from Friendly() identifies an
	// (unlikely) place where we didn't wrap an error with a ucError ever
	return "an unknown error occurred"
}

// UserFriendlyStructure exposes the structured error data if error is a ucError
func UserFriendlyStructure(err error) any {
	var uce UCError
	if errors.As(err, &uce) {
		return uce.FriendlyStructure()
	}

	return nil
}
