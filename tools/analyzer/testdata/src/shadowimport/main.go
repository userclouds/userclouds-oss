package shadowimport

import (
	"fmt"
	"strings"
)

func goodFunction() {
	// These should not be flagged - not shadowing imports
	foo := "bar"
	baz := 123
	_ = foo
	_ = baz
}

func badFunction() {
	// This should be flagged - shadows the fmt import
	fmt.Println("hello")
	fmt := "hello" // want "variable assignment \"fmt\" shadows imported package name from line 4"
	_ = fmt

	// This should be flagged - shadows the strings import
	_ = strings.Contains("foo", "bar")
	strings := []string{"a", "b"} // want "variable assignment \"strings\" shadows imported package name from line 5"
	_ = strings
}

func moreTests() {
	// This should be flagged - var declaration shadows import
	var fmt int // want "variable definition \"fmt\" shadows imported package name from line 4"
	_ = fmt

	// Multiple variables in one declaration
	var (
		a       int
		strings bool // want "variable definition \"strings\" shadows imported package name from line 5"
		c       float64
	)
	_ = a
	_ = strings
	_ = c
}
