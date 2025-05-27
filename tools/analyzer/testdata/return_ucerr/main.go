package main

import (
	"errors"
)

func namedGroupedParams() (a, b int, err error) {
	return 1, 2, nil
}

func namedGroupedParamsNeedsWrapper() (a, b int, err error) {
	err = errors.New("test")
	return 0, 0, err // want `error return value at pos 2 should be wrapped with ucerr.Wrap()`
}

func testNestedReturns() (*string, error) {
	if err := errors.New("foo"); err != nil {
		if true {
			return nil, nil
		}
		return nil, err // want `ucerr.Wrap`
	}
	return nil, nil
}

func main() {
	namedGroupedParams()
	namedGroupedParamsNeedsWrapper()
	testNestedReturns()
}
