package cluster_test

import (
	"fmt"
	"testing"
	"time"

	. "github.com/dogmatiq/veracity/internal/cluster"
	"github.com/google/uuid"
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
			p.Route("<workload>")
		})
	})

	t.Run("when there are nodes", func(t *testing.T) {
		t.Parallel()

		t.Run("it distributes workloads across all nodes", func(t *testing.T) {
			t.Parallel()

			p := &Partitioner{}
			remaining := map[uuid.UUID]struct{}{}

			for i := 0; i < 10; i++ {
				id := uuid.New()
				p.AddNode(id)
				remaining[id] = struct{}{}
			}

			start := time.Now()
			timeout := 5 * time.Second

			for len(remaining) != 0 {
				if time.Since(start) > timeout {
					t.Fatal("timed-out waiting for workloads to be distributed")
				}

				workload := uuid.NewString()
				id := p.Route(workload)

				delete(remaining, id)
			}
		})

		t.Run("it consistently routes to a specific workload to the same node", func(t *testing.T) {
			t.Parallel()

			p := &Partitioner{}
			for i := 0; i < 10; i++ {
				p.AddNode(uuid.New())
			}

			expect := p.Route("<workload>")

			for i := 0; i < 10; i++ {
				actual := p.Route("<workload>")
				if actual != expect {
					t.Fatalf("got %q, want %q", actual, expect)
				}
			}
		})

		t.Run("it does not route to nodes that have been removed", func(t *testing.T) {
			t.Parallel()

			p := &Partitioner{}
			for i := 0; i < 10; i++ {
				p.AddNode(uuid.New())
			}

			removed := p.Route("<workload>")
			p.RemoveNode(removed)

			id := p.Route("<workload>")
			if id == removed {
				t.Fatalf("got %q, expected a different node", id)
			}
		})
	})
}
