// Package set provides a set type with some useful helper functions.
package set

// originally implemented by @akshayjshah
// ported to UserClouds by @stgarrity 5/2023
// used with permission
//
// I was going to rewrite this entirely using generics, but uuidset seems
// pretty important for us, and uuid.UUID does not implement `constraints.Ordered`,
// so there is no (obvious?) way to sort it. This would leave us with a Set that
// potentially returns something subtly different after seemingly idempotenent
// operations, which would make testing really suck. So, we're going to go most
// of the way with a generic implementation that gets wrapped by type-specific
// ctors (eg. stringset & uuidset) that just pass in a sort function.

import (
	"fmt"
	"maps"
)

// Set is a unique, unordered collection of elements.
type Set[T comparable] struct {
	set    map[T]struct{}
	sorter func([]T)
}

// New creates a new set.
func New[T comparable](sorter func([]T), items ...T) Set[T] {
	s := make(map[T]struct{}, len(items))
	for _, item := range items {
		s[item] = struct{}{}
	}
	return Set[T]{set: s, sorter: sorter}
}

// Equal checks whether two sets contain the same elements.
func (s Set[T]) Equal(other Set[T]) bool {
	if s.Size() != other.Size() {
		return false
	}
	for item := range s.set {
		if _, ok := other.set[item]; !ok {
			return false
		}
	}
	return true
}

// Contains checks if the set contains the element.
func (s Set[T]) Contains(item T) bool {
	_, ok := s.set[item]
	return ok
}

// IsSupersetOf checks if the set contains all the elements of the other set.
func (s Set[T]) IsSupersetOf(other Set[T]) bool {
	if s.Size() < other.Size() {
		return false
	}

	return other.Difference(s).Size() == 0
}

// Insert inserts the items into the set.
func (s Set[T]) Insert(items ...T) {
	for _, item := range items {
		s.set[item] = struct{}{}
	}
}

// Evict removes an item from the set and returns whether the item was present.
func (s Set[T]) Evict(item T) bool {
	_, ok := s.set[item]
	delete(s.set, item)
	return ok
}

// Difference returns elements of the set not in the other.
func (s Set[T]) Difference(other Set[T]) Set[T] {
	output := New(s.sorter)
	for item := range s.set {
		if !other.Contains(item) {
			// Avoid the varargs allocation from Insert.
			output.set[item] = struct{}{}
		}
	}
	return output
}

// SymmetricDifference returns elements of both sets that are not in the other.
//
// This is a shortcut for calling x.Difference(y) + y.Difference(x).
func (s Set[T]) SymmetricDifference(other Set[T]) Set[T] {
	a := s.Difference(other)
	b := other.Difference(s)
	maps.Copy(a.set, b.set)
	return a
}

// Intersection returns elements common to both sets.
func (s Set[T]) Intersection(other Set[T]) Set[T] {
	small, big := s, other
	if small.Size() > big.Size() {
		big, small = small, big
	}
	output := New(s.sorter)
	for item := range small.set {
		if big.Contains(item) {
			output.set[item] = struct{}{}
		}
	}
	return output
}

// Union returns a set with elements in either of the sets.
func (s Set[T]) Union(other Set[T]) Set[T] {
	output := New(s.sorter)
	for item := range s.set {
		output.set[item] = struct{}{}
	}
	for item := range other.set {
		if !output.Contains(item) {
			output.set[item] = struct{}{}
		}
	}
	return output
}

// Items returns a sorted slice of the set elements.
func (s Set[T]) Items() []T {
	output := make([]T, 0, s.Size())
	for item := range s.set {
		output = append(output, item)
	}
	s.sorter(output)
	return output
}

// Size returns the number of items in the set
func (s Set[T]) Size() int {
	return len(s.set)
}

// String implements fmt.Stringer.
func (s Set[T]) String() string {
	return fmt.Sprint(s.Items())
}
