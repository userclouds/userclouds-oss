package testhelpers

import (
	"sort"

	"github.com/gofrs/uuid"
)

type idList []uuid.UUID

// Len implements sort.Interface
func (ids idList) Len() int {
	return len(ids)
}

// Swap implements sort.Interface
func (ids idList) Swap(left, right int) {
	tmp := ids[left]
	ids[left] = ids[right]
	ids[right] = tmp
}

// Less implements sort.Interface
func (ids idList) Less(left, right int) bool {
	return ids[left].String() < ids[right].String()
}

// MakeSortedUUIDs returns N generated UUIDs in ascending order
func MakeSortedUUIDs(n int) []uuid.UUID {
	ids := make(idList, n)
	for i := range n {
		ids[i] = uuid.Must(uuid.NewV4())
	}
	sort.Sort(ids)
	return ids
}
