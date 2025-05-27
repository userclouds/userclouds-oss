package assert

import "testing"

func TestEquals(t *testing.T) {
	Equal(t, 1, 1)
}

func TestIsNil(t *testing.T) {
	var err *error
	IsNil(t, err)
}
