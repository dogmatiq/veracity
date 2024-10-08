package cluster_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/dogmatiq/enginekit/collections/sets"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	. "github.com/dogmatiq/veracity/internal/cluster"
)

func TestPartitioner(t *testing.T) {
	t.Parallel()

	t.Run("when there are no nodes", func(t *testing.T) {
		t.Parallel()

		t.Run("it panics", func(t *testing.T) {
			t.Parallel()

			defer func() {
				actual := fmt.Sprint(recover())
				expect := "partitioner has no nodes"

				if actual != expect {
					t.Errorf("got %q, want %q", actual, expect)
				}
			}()

			p := &Partitioner{}
			p.Route(uuidpb.Generate())
		})
	})

	t.Run("when there are nodes", func(t *testing.T) {
		t.Parallel()

		t.Run("it distributes workloads across all nodes", func(t *testing.T) {
			t.Parallel()

			p := &Partitioner{}
			remaining := sets.NewOrderedByMember[*uuidpb.UUID]()

			for range 10 {
				id := uuidpb.Generate()
				p.AddNode(id)
				remaining.Add(id)
			}

			start := time.Now()
			timeout := 5 * time.Second

			for remaining.Len() != 0 {
				if time.Since(start) > timeout {
					t.Fatal("timed-out waiting for workloads to be distributed")
				}

				id := p.Route(uuidpb.Generate())
				remaining.Remove(id)
			}
		})

		t.Run("it consistently routes to a specific workload to the same node", func(t *testing.T) {
			t.Parallel()

			p := &Partitioner{}
			for range 10 {
				p.AddNode(uuidpb.Generate())
			}

			workload := uuidpb.Generate()
			expect := p.Route(workload)

			for i := range 10 {
				actual := p.Route(workload)
				if !actual.Equal(expect) {
					t.Fatalf("attempt #%d: got %q, want %q", i+1, actual, expect)
				}
			}
		})

		t.Run("it does not route to nodes that have been removed", func(t *testing.T) {
			t.Parallel()

			p := &Partitioner{}
			for range 10 {
				p.AddNode(uuidpb.Generate())
			}

			workload := uuidpb.Generate()
			removed := p.Route(workload)
			p.RemoveNode(removed)

			id := p.Route(workload)
			if id == removed {
				t.Fatalf("got %q, expected a different node", id)
			}
		})
	})
}
