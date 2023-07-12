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
				"add a non-member": func(t *rapid.T) {
					m := rapid.
						Int8().
						Draw(t, "non-member")

					if _, ok := members[m]; ok {
						t.Skip("already a member")
					}

					if set.Has(m) {
						t.Errorf("did not expect set to contain %d", m)
					}

					set.Add(m)

					if !set.Has(m) {
						t.Errorf("expected set to contain %d", m)
					}

					members[m] = struct{}{}
				},
				"re-add an existing member": func(t *rapid.T) {
					if len(members) == 0 {
						t.Skip("set is empty")
					}

					m := rapid.
						SampledFrom(maps.Keys(members)).
						Draw(t, "member")

					before := set.Members()
					set.Add(m)
					after := set.Members()

					test.Expect(
						t,
						after,
						before,
					)
				},
				"delete an existing member": func(t *rapid.T) {
					if len(members) == 0 {
						t.Skip("set is empty")
					}

					m := rapid.
						SampledFrom(maps.Keys(members)).
						Draw(t, "member")

					if !set.Has(m) {
						t.Errorf("expected set to contain %d", m)
					}

					set.Delete(m)

					if set.Has(m) {
						t.Errorf("did not expect set to contain %d", m)
					}

					delete(members, m)
				},
				"delete a non-member": func(t *rapid.T) {
					m := rapid.
						Int8().
						Draw(t, "non-member")

					if _, ok := members[m]; ok {
						t.Skip("already a member")
					}

					before := set.Members()
					set.Delete(m)
					after := set.Members()

					test.Expect(
						t,
						after,
						before,
					)
				},
				"cardinality": func(t *rapid.T) {
					got := set.Len()
					want := len(members)

					if got != want {
						t.Errorf(
							"unexpected cardinality: got %d, want %d",
							got,
							want,
						)
					}
				},
				"members are ordered": func(t *rapid.T) {
					got := set.Members()
					want := maps.Keys(members)
					slices.Sort(want)

					test.Expect(
						t,
						got,
						want,
					)
				},
			},
		)
	})
}
