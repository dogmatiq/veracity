package cluster_test

import (
	"strconv"
	"time"

	"github.com/dogmatiq/veracity/internal/cluster"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Partitioner", func() {
	var partitioner *cluster.Partitioner

	BeforeEach(func() {
		partitioner = &cluster.Partitioner{}
	})

	When("there are no nodes", func() {
		It("panics", func() {
			Expect(func() {
				partitioner.Route("<workload>")
			}).To(PanicWith("partitioner has no nodes"))
		})
	})

	When("there are nodes", func() {
		var nodes map[uuid.UUID]struct{}

		BeforeEach(func() {
			nodes = map[uuid.UUID]struct{}{}

			for i := 0; i < 10; i++ {
				id := uuid.New()
				partitioner.AddNode(id)
				nodes[id] = struct{}{}
			}
		})

		It("distributes workloads across all nodes", func() {
			start := time.Now()
			timeout := 5 * time.Second

			i := 0

			// Loop as many times is possible with a different workload until
			// each node is returned at least once.
			for {
				if time.Since(start) > timeout {
					Expect(nodes).To(
						BeEmpty(),
						"timed-out waiting for workloads to be distributed",
					)
				}

				id := partitioner.Route(strconv.Itoa(i))
				i++

				delete(nodes, id)

				if len(nodes) == 0 {
					break
				}
			}
		})

		It("consistently routes to a specific workload to the same node", func() {
			id := partitioner.Route("<workload>")
			for i := 0; i < 10; i++ {
				Expect(partitioner.Route("<workload>")).To(Equal(id))
			}
		})

		It("does not route to nodes that have been removed", func() {
			removed := partitioner.Route("<workload>")
			partitioner.RemoveNode(removed)

			id := partitioner.Route("<workload>")
			Expect(id).NotTo(Equal(removed))
		})
	})
})
