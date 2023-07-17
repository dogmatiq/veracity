package optimistic_test

import (
	"testing"

	. "github.com/dogmatiq/veracity/internal/optimistic"
	"github.com/dogmatiq/veracity/internal/test"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"pgregory.net/rapid"
)

func TestOrderedSet(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(t *rapid.T) {
		var set OrderedSet[int8, OrderedComparator[int8]]
		members := map[int8]struct{}{}

		t.Repeat(
			map[string]func(*rapid.T){
				"": func(t *rapid.T) {
					test.Expect(
						t,
						"set cardinality is incorrect",
						set.Len(),
						len(members),
					)

					sorted := maps.Keys(members)
					slices.Sort(sorted)

					test.Expect(
						t,
						"set members are disjoint or out of order",
						set.Members(),
						sorted,
					)
				},
				"add a non-member": func(t *rapid.T) {
					m := rapid.
						Int8().
						Draw(t, "non-member")

					if _, ok := members[m]; ok {
						t.Skip("already a member")
					}

					set.Add(m)

					members[m] = struct{}{}
				},
				"re-add an existing member": func(t *rapid.T) {
					if len(members) == 0 {
						t.Skip("set is empty")
					}

					m := rapid.
						SampledFrom(maps.Keys(members)).
						Draw(t, "member")

					set.Add(m)
				},
				"delete an existing member": func(t *rapid.T) {
					if len(members) == 0 {
						t.Skip("set is empty")
					}

					m := rapid.
						SampledFrom(maps.Keys(members)).
						Draw(t, "member")

					set.Delete(m)

					delete(members, m)
				},
				"delete a non-member": func(t *rapid.T) {
					m := rapid.
						Int8().
						Draw(t, "non-member")

					if _, ok := members[m]; ok {
						t.Skip("already a member")
					}

					set.Delete(m)
				},
			},
		)
	})
}
