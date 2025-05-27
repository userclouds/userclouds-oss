package main

import (
	"database/sql"
	"errors"
)

func main() {
	test()
}

func test() string {
	var errorNotFound = errors.New("not found")

	var err error

	// NB: the backslashes are required since analysistest interprets the want comments as regexes
	if err == errorNotFound { // want `use errors.Is\(\) instead of comparing to a specific error`
		return "not found"
	} else if err != errorNotFound { // want `use errors.Is\(\) instead of comparing to a specific error`
		return "not good"
	} else if errors.Is(err, errorNotFound) {
		return "this is good"
	}

	if err == nil {
		if err == sql.ErrNoRows { // want `use errors.Is\(\) instead of comparing to a specific error`
			return "not good"
		}
	}

	type myError struct {
		error
	}

	var myErr myError

	if _, ok := err.(myError); ok { // want `use errors.As\(\) instead of casting to myError`
		return "this is bad"
	} else if errors.As(err, &myErr) {
		return "this is good"
	}

	return ""
}
