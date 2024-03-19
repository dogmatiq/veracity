package cluster

import (
	"github.com/cespare/xxhash/v2"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/optimistic"
)

// Partitioner determines which node is responsible for handling a specific
// workload.
type Partitioner struct {
	// nodes is an atomic pointer to an ordered set of node IDs.
	nodes optimistic.OrderedSet[*uuidpb.UUID, optimistic.LessComparator[*uuidpb.UUID]]
}

// AddNode adds a node to the partitioner.
func (p *Partitioner) AddNode(id *uuidpb.UUID) {
	p.nodes.Add(id)
}

// RemoveNode removes a node from the partitioner.
func (p *Partitioner) RemoveNode(id *uuidpb.UUID) {
	p.nodes.Delete(id)
}

// Route returns the ID of the node that should handle the given workload.
func (p *Partitioner) Route(workload *uuidpb.UUID) *uuidpb.UUID {
	nodes := p.nodes.Members()

	if len(nodes) == 0 {
		panic("partitioner has no nodes")
	}

	var (
		hash   xxhash.Digest
		winner *uuidpb.UUID
		score  uint64
	)

	for _, node := range nodes {
		hash.Reset()
		hash.Write(node.AsBytes())
		hash.Write(workload.AsBytes())

		if s := hash.Sum64(); s > score {
			winner = node
			score = s
		}
	}

	return winner
}
