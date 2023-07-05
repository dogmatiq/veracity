package cluster_test

import (
	"fmt"
	"testing"
	"time"

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
			p.Route("<workload>")
		})
	})

	t.Run("when there are nodes", func(t *testing.T) {
		t.Parallel()

		t.Run("it distributes workloads across all nodes", func(t *testing.T) {
			t.Parallel()

			p := &Partitioner{}
			remaining := uuidpb.Set{}

			for i := 0; i < 10; i++ {
				id := uuidpb.Generate()
				p.AddNode(id)
				remaining.Add(id)
			}

			start := time.Now()
			timeout := 5 * time.Second

			for len(remaining) != 0 {
				if time.Since(start) > timeout {
					t.Fatal("timed-out waiting for workloads to be distributed")
				}

				workload := uuidpb.Generate().AsString()
				id := p.Route(workload)

				remaining.Delete(id)
			}
		})

		t.Run("it consistently routes to a specific workload to the same node", func(t *testing.T) {
			t.Parallel()

			p := &Partitioner{}
			for i := 0; i < 10; i++ {
				p.AddNode(uuidpb.Generate())
			}

			expect := p.Route("<workload>")

			for i := 0; i < 10; i++ {
				actual := p.Route("<workload>")
				if !actual.Equal(expect) {
					t.Fatalf("got %q, want %q", actual, expect)
				}
			}
		})

		t.Run("it does not route to nodes that have been removed", func(t *testing.T) {
			t.Parallel()

			p := &Partitioner{}
			for i := 0; i < 10; i++ {
				p.AddNode(uuidpb.Generate())
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
