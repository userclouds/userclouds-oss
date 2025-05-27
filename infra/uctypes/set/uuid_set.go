package set

import (
	"sort"

	"github.com/gofrs/uuid"
)

// NewUUIDSet returns a set of uuid.UUIDs
func NewUUIDSet(items ...uuid.UUID) Set[uuid.UUID] {
	return New(uuidSorter, items...)
}

func uuidSorter(items []uuid.UUID) {
	sort.Slice(items, func(i, j int) bool {
		left, right := items[i], items[j]
		for char := range left {
			if left[char] != right[char] {
				return left[char] < right[char]
			}
		}
		return false
	})
}
