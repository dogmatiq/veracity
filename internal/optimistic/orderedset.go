package optimistic

import (
	"slices"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"golang.org/x/exp/constraints"
)

// OrderedSet is a read-optimized lock-free ordered set.
type OrderedSet[M any, C Comparator[M]] struct {
	Comparator C
	members    Value[[]M]
}

// Add adds a member to the set.
func (s *OrderedSet[M, C]) Add(m M) {
	s.members.Apply(
		func(members []M) ([]M, bool) {
			if i, ok := slices.BinarySearchFunc(members, m, s.Comparator.Compare); !ok {
				return slices.Insert(members, i, m), true
			}
			return members, false
		},
	)
}

// Delete removes a member from the set.
func (s *OrderedSet[M, C]) Delete(m M) {
	s.members.Apply(
		func(members []M) ([]M, bool) {
			if i, ok := slices.BinarySearchFunc(members, m, s.Comparator.Compare); ok {
				return slices.Delete(members, i, i+1), true
			}
			return members, false
		},
	)
}

// Has returns true if m is a member of the set.
func (s *OrderedSet[M, C]) Has(m M) bool {
	_, ok := slices.BinarySearchFunc(s.members.Load(), m, s.Comparator.Compare)
	return ok
}

// Len returns the number of members in the set.
func (s *OrderedSet[M, C]) Len() int {
	return len(s.members.Load())
}

// Members returns the members of the set, in order.
// The returned slice must not be modified.
func (s *OrderedSet[M, C]) Members() []M {
	return s.members.Load()
}

// Comparator is a type that compares two values.
type Comparator[T any] interface {
	// Compare performs a 3-way comparison of a and b.
	Compare(a, b T) int
}

// LessComparator is a [Comparator] for types that implement a LessComparator()
// method.
type LessComparator[T interface{ Less(T) bool }] struct{}

// Compare performs a 3-way comparison of a and b.
func (LessComparator[M]) Compare(a, b *uuidpb.UUID) int {
	switch {
	case a.Less(b):
		return -1
	case b.Less(a):
		return +1
	default:
		return 0
	}
}

// OrderedComparator is a Comparator that compares values that match the
// constraints.OrderedComparator constraint.
type OrderedComparator[T constraints.Ordered] struct{}

// Compare performs a 3-way comparison of a and b.
func (OrderedComparator[T]) Compare(a, b T) int {
	if a < b {
		return -1
	}
	if a > b {
		return +1
	}
	return 0
}
