package codegen_test

import (
	"testing"

	"userclouds.com/infra/assert"
	. "userclouds.com/infra/codegen"
)

func TestImports(t *testing.T) {
	p := NewImports()

	p.Add("context")
	p.Add("github.com/gofrs/uuid")
	p.Add("userclouds.com/infra/codegen")
	p.Add("userclouds.com/abc")

	assert.Equal(t, p.String(), `	"context"

	"github.com/gofrs/uuid"

	"userclouds.com/abc"
	"userclouds.com/infra/codegen"`)
}
