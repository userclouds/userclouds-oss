package set_test

import (
	"sort"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/assert"
	. "userclouds.com/infra/uctypes/set"
)

// Test data.
var (
	u1 = uuid.Must(uuid.NewV4())
	u2 = uuid.Must(uuid.NewV4())
	u3 = uuid.Must(uuid.NewV4())
)

func TestUUIDEqual(t *testing.T) {
	base := NewUUIDSet(u1, u2)
	assert.NotEqual(t, NewUUIDSet(u1), base, assert.Errorf("different length"), assert.Diff())
	assert.NotEqual(t, NewUUIDSet(u1, u3), base, assert.Errorf("mismatched element"), assert.Diff())
	assert.Equal(t, NewUUIDSet(u1, u2), base, assert.Diff())
}

func TestUUIDContains(t *testing.T) {
	s := NewUUIDSet(u1, u2)
	if !s.Contains(u2) {
		t.Errorf("set %v should contain %v", s, u2)
	}
	if s.Contains(u3) {
		t.Errorf("set %v shouldn't contain %v", s, u3)
	}
}

func TestUUIDInsert(t *testing.T) {
	s := NewUUIDSet()
	s.Insert(u1)
	assert.Equal(t, s, NewUUIDSet(u1), assert.Errorf("insert failed"))
}

func TestUUIDEvict(t *testing.T) {
	s := NewUUIDSet(u1)
	if s.Evict(u2) {
		t.Errorf("unsuccessful eviction returned true, expected false")
	}
	if !s.Evict(u1) {
		t.Errorf("successful eviction returned false, expected true")
	}
	assert.Equal(t, s, NewUUIDSet(), assert.Errorf("evict didn't remove item"))
}

func TestUUIDDifference(t *testing.T) {
	big := NewUUIDSet(u1, u2)
	small := NewUUIDSet(u1)
	assert.Equal(t, big.Difference(small), NewUUIDSet(u2), assert.Diff())
}

func TestUUIDSymmetricDifference(t *testing.T) {
	left := NewUUIDSet(u1, u2)
	right := NewUUIDSet(u2, u3)
	expect := NewUUIDSet(u1, u3)
	assert.Equal(t, left.SymmetricDifference(right), expect)
}

func TestUUIDIntersection(t *testing.T) {
	big := NewUUIDSet(u1, u2)
	small := NewUUIDSet(u2)
	assert.Equal(t, big.Intersection(small), NewUUIDSet(u2))
}

func TestUUIDUnion(t *testing.T) {
	left := NewUUIDSet(u1)
	right := NewUUIDSet(u2)
	union := left.Union(right).Union(right) // union should be idempotent
	assert.Equal(t, union, NewUUIDSet(u1, u2))
}

func TestUUIDItems(t *testing.T) {
	// run repeatedly, since map iteration order is explicitly random
	sorted := []uuid.UUID{u1, u2}
	uuidSorter(sorted)
	reversed := make([]uuid.UUID, 2)
	reversed[0] = sorted[1]
	reversed[1] = sorted[0]
	s := NewUUIDSet(reversed...)
	for range 100 {
		assert.Equal(t, s.Items(), sorted, assert.Errorf("items unsorted"), assert.Must())
	}
}

func TestUUIDSize(t *testing.T) {
	zero := NewUUIDSet()
	one := NewUUIDSet(u1)
	two := NewUUIDSet(u1, u2)
	dups := NewUUIDSet(u1, u1)

	assert.Equal(t, zero.Size(), 0)
	assert.Equal(t, one.Size(), 1)
	assert.Equal(t, two.Size(), 2)
	assert.Equal(t, dups.Size(), 1)
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
