package ucerr_test

import (
	"errors"
	"testing"

	"userclouds.com/infra/assert"
	. "userclouds.com/infra/ucerr"
)

func TestNew(t *testing.T) {
	err := New("test error")
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "test error\ntest error (File infra/ucerr/errors_test.go:12, in TestNew)")
}

func TestWrap(t *testing.T) {
	err := errors.New("test base") // lint: ucwrapper-safe
	ue := Wrap(err)
	assert.NotNil(t, ue)
	assert.Equal(t, ue.Error(), "test base\n(wrapped) (File infra/ucerr/errors_test.go:19, in TestWrap)")

	second := Wrap(ue)
	assert.NotNil(t, second)
	assert.Equal(t, second.Error(), `test base
(wrapped) (File infra/ucerr/errors_test.go:19, in TestWrap)
(wrapped) (File infra/ucerr/errors_test.go:23, in TestWrap)`)
}

func TestErrorsIs(t *testing.T) {
	base := errors.New("important error") // lint: ucwrapper-safe
	wrapped := Wrap(base)
	assert.True(t, errors.Is(wrapped, base))
}

type customError struct {
	s string
}

func (e *customError) Error() string {
	return e.s
}
func TestErrorsAs(t *testing.T) {
	base := &customError{s: "custom error type"}
	wrapped := Wrap(base)

	var asBase *customError
	// errors.As() should extract the base type.
	assert.True(t, errors.As(wrapped, &asBase))
	assert.NotEqual(t, base.Error(), wrapped.Error())
	assert.Equal(t, base.Error(), asBase.Error())

	// Type casting does not work the same way as errors.As().
	_, ok := wrapped.(*customError) // lint ecast-safe
	assert.False(t, ok)
}

func TestErrorf(t *testing.T) {
	base := errors.New("test") // lint: ucwrapper-safe

	err1 := Errorf("test with int %d", 100)
	var uce UCError
	assert.True(t, errors.As(err1, &uce))
	assert.Equal(t, uce.Error(), "test with int 100 (File infra/ucerr/errors_test.go:61, in TestErrorf)")

	// There is special handling for a trailing ": %w" suffix.
	err2 := Errorf("test with int %d and string %s and error: %w", 101, "hello", base)
	assert.True(t, errors.As(err2, &uce))
	assert.Equal(t, uce.Error(), "test\ntest with int 101 and string hello and error (File infra/ucerr/errors_test.go:67, in TestErrorf)")

	err3 := Errorf("test with not an error: %w", 101)
	assert.Contains(t, err3.Error(), `seems as if ucerr.Errorf() was called with a non-error %w`)
}

func TestFriendly(t *testing.T) {
	base := errors.New("test") // lint: ucwrapper-safe

	assert.Equal(t, UserFriendlyMessage(base), "an unknown error occurred")

	l1 := Wrap(base)
	l2 := Errorf("me too: %w", l1)

	assert.Equal(t, UserFriendlyMessage(l2), "an unspecified error occurred")

	l3 := Friendlyf(l2, "this is friendly")
	assert.Equal(t, UserFriendlyMessage(l3), "this is friendly")

	l4 := Wrap(l3)
	assert.Equal(t, UserFriendlyMessage(l4), "this is friendly")

	l11 := Friendlyf(base, "friendly only")
	assert.Contains(t, l11.Error(), "test\n[friendly] friendly only")
}

func TestCombineUnwrap(t *testing.T) {
	err1 := New("test1")
	err2 := New("test2")
	err3 := New("test3")
	err4 := New("test4")

	combined := Combine(err1, err2)
	combined = Combine(combined, err3)
	combined = Combine(combined, Wrap(err4))

	assert.True(t, errors.Is(combined, err1))
	assert.True(t, errors.Is(combined, err2))
	assert.True(t, errors.Is(combined, err3))
}
