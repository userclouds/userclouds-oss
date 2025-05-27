package set

import (
	"sort"
)

// NewStringSet returns a set of strings.
func NewStringSet(items ...string) Set[string] {
	return New(func(s []string) { sort.Strings(s) }, items...)
}
