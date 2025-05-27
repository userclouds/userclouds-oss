package ucerr_test

import (
	"errors"
	"testing"

	"userclouds.com/infra/assert"
	. "userclouds.com/infra/ucerr"
)

func TestWarning(t *testing.T) {
	err := NewWarning("test")
	var w Warning
	assert.True(t, errors.As(err, &w))
	assert.True(t, errors.As(Wrap(err), &w))
}
