package assert

import (
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// Option defines a way to modify assert behavior
type Option interface {
	apply(*options)
}

type options struct {
	msg     string
	stop    bool
	cmpOpts []cmp.Option // passed straight through
	diff    bool         // show diff on failure
}

type optFunc func(*options)

func (o optFunc) apply(os *options) {
	o(os)
}

// Errorf adds a more specific message to the failure
func Errorf(msg string, args ...any) Option {
	return optFunc(func(os *options) {
		if os.msg == "" {
			os.msg = fmt.Sprintf(msg, args...)
		} else {
			os.msg = fmt.Sprintf("%s\n%s", os.msg, fmt.Sprintf(msg, args...))
		}
	})
}

// Must stops the test if this assert fails
// Useful if you're just going to run into eg. a nil pointer deref next
func Must() Option {
	return optFunc(func(os *options) {
		os.stop = true
	})
}

// CmpOpt adds a cmp.Option from the underlying lib
// Useful for things like .IgnoreUnexported
func CmpOpt(o cmp.Option) Option {
	return optFunc(func(os *options) {
		os.cmpOpts = append(os.cmpOpts, o)
	})
}

// Diff prints a diff between got & want on failure
func Diff() Option {
	return optFunc(func(os *options) {
		os.diff = true
	})
}

// CmpIgnoreTimes is a cmp.Option that ignores BaseModel.Created, BaseModel.Updated, and BaseModel.Version
// This is useful if you want to display a diff without times (eg. `assert.Errorf("diff: %s", cmp.Diff(got, want, IgnoreTimes()))“)
func CmpIgnoreTimes() cmp.Option {
	return cmp.FilterPath(func(p cmp.Path) bool {
		return strings.Contains(p.GoString(), "BaseModel.Created") ||
			strings.Contains(p.GoString(), "BaseModel.Updated") ||
			strings.Contains(p.GoString(), "BaseModel.Version")
	}, cmp.Ignore())
}

// IgnoreTimes ignores BaseModel.Created, BaseModel.Updated, and BaseModel.Version for cmp operations
// This is a helper to pass directly into eg. `assert.Equal`
func IgnoreTimes() Option {
	return CmpOpt(CmpIgnoreTimes())
}
