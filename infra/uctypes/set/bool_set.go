package set

import (
	"sort"
)

// NewBoolSet returns a set of bools
func NewBoolSet(items ...bool) Set[bool] {
	return New(boolSorter, items...)
}

func boolSorter(items []bool) {
	sort.Slice(items, func(i, j int) bool {
		return !items[i]
	})
}
