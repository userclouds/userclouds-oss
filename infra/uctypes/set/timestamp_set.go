package set

import (
	"sort"
	"time"
)

// NewTimestampSet returns a set of time.Times
func NewTimestampSet(items ...time.Time) Set[time.Time] {
	return New(timestampSorter, items...)
}

func timestampSorter(items []time.Time) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Before(items[j])
	})
}
