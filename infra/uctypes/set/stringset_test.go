package set_test

import (
	"sort"
	"testing"

	"userclouds.com/infra/assert"
	. "userclouds.com/infra/uctypes/set"
)

const (
	s1 = "s1"
	s2 = "s2"
	s3 = "s3"
)

func TestStringEqual(t *testing.T) {
	base := NewStringSet(s1, s2)
	assert.NotEqual(t, NewStringSet(s1), base, assert.Errorf("different length"), assert.Diff())
	assert.NotEqual(t, NewStringSet(s1, s3), base, assert.Errorf("mismatched element"), assert.Diff())
	assert.Equal(t, NewStringSet(s1, s2), base, assert.Diff())
}

func TestStringContains(t *testing.T) {
	s := NewStringSet(s1, s2)
	if !s.Contains(s2) {
		t.Errorf("set %v should contain %v", s, s2)
	}
	if s.Contains(s3) {
		t.Errorf("set %v shouldn't contain %v", s, s3)
	}
}

func TestStringInsert(t *testing.T) {
	s := NewStringSet()
	s.Insert(s1)
	assert.Equal(t, s, NewStringSet(s1), assert.Errorf("insert failed"))
}

func TestStringEvict(t *testing.T) {
	s := NewStringSet(s1)
	if s.Evict(s2) {
		t.Errorf("unsuccessful eviction returned true, expected false")
	}
	if !s.Evict(s1) {
		t.Errorf("successful eviction returned false, expected true")
	}
	assert.Equal(t, s, NewStringSet(), assert.Errorf("evict didn't remove item"))
}

func TestStringDifference(t *testing.T) {
	big := NewStringSet(s1, s2)
	small := NewStringSet(s1)
	assert.Equal(t, big.Difference(small), NewStringSet(s2), assert.Diff())
}

func TestStringSymmetricDifference(t *testing.T) {
	left := NewStringSet(s1, s2)
	right := NewStringSet(s2, s3)
	expect := NewStringSet(s1, s3)
	assert.Equal(t, left.SymmetricDifference(right), expect)
}

func TestStringIntersection(t *testing.T) {
	big := NewStringSet(s1, s2)
	small := NewStringSet(s2)
	assert.Equal(t, big.Intersection(small), NewStringSet(s2))
}

func TestStringUnion(t *testing.T) {
	left := NewStringSet(s1)
	right := NewStringSet(s2)
	union := left.Union(right).Union(right) // union should be idempotent
	assert.Equal(t, union, NewStringSet(s1, s2))
}

func TestStringItems(t *testing.T) {
	// run repeatedly, since map iteration order is explicitly random
	sorted := []string{s1, s2}
	sort.Strings(sorted)
	reversed := make([]string, 2)
	reversed[0] = sorted[1]
	reversed[1] = sorted[0]
	s := NewStringSet(reversed...)
	for range 100 {
		assert.Equal(t, s.Items(), sorted, assert.Errorf("items unsorted"), assert.Must())
	}
}

func TestStringSize(t *testing.T) {
	zero := NewStringSet()
	one := NewStringSet(s1)
	two := NewStringSet(s1, s2)

	assert.Equal(t, zero.Size(), 0)
	assert.Equal(t, one.Size(), 1)
	assert.Equal(t, two.Size(), 2)
}

func TestStringSuperset(t *testing.T) {
	big := NewStringSet(s1, s2)
	small := NewStringSet(s1)
	assert.True(t, big.IsSupersetOf(small))
	assert.False(t, small.IsSupersetOf(big))
}
