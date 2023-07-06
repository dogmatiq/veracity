package cluster

import (
	"sync/atomic"

	"github.com/cespare/xxhash/v2"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"golang.org/x/exp/slices"
)

// Partitioner determines which node is responsible for handling a specific
// workload.
type Partitioner struct {
	// nodes is an atomic pointer to an ordered set of node IDs.
	nodes atomic.Pointer[[]*uuidpb.UUID]
}

// AddNode adds a node to the partitioner.
func (p *Partitioner) AddNode(id *uuidpb.UUID) {
	p.mutate(
		func(nodes []*uuidpb.UUID) []*uuidpb.UUID {
			if i, ok := slices.BinarySearchFunc(nodes, id, compareUUID); !ok {
				nodes = slices.Insert(nodes, i, id)
			}
			return nodes
		},
	)
}

// RemoveNode removes a node from the partitioner.
func (p *Partitioner) RemoveNode(id *uuidpb.UUID) {
	p.mutate(
		func(nodes []*uuidpb.UUID) []*uuidpb.UUID {
			if i, ok := slices.BinarySearchFunc(nodes, id, compareUUID); ok {
				nodes = slices.Delete(nodes, i, i+1)
			}
			return nodes
		},
	)
}

// mutate applies a mutation function to the set of nodes.
//
// fn is passed a copy of the current set of nodes. Its result replaces the
// current set of nodes. fn must be safe to be retried multiple times.
func (p *Partitioner) mutate(
	fn func([]*uuidpb.UUID) []*uuidpb.UUID,
) {
	for {
		ptr := p.nodes.Load()

		var nodes []*uuidpb.UUID
		if ptr != nil && len(*ptr) != 0 {
			nodes = slices.Clone(*ptr)
		}

		nodes = fn(nodes)

		if p.nodes.CompareAndSwap(ptr, &nodes) {
			return
		}
	}
}

// Route returns the ID of the node that should handle the given workload.
func (p *Partitioner) Route(workload *uuidpb.UUID) *uuidpb.UUID {
	nodes := p.nodes.Load()

	if nodes == nil || len(*nodes) == 0 {
		panic("partitioner has no nodes")
	}

	var (
		hash         xxhash.Digest
		winningNode  *uuidpb.UUID
		winningScore uint64
	)

	for _, node := range *nodes {
		hash.Write(node.AsBytes())
		hash.Write(workload.AsBytes())

		score := hash.Sum64()
		hash.Reset()

		if score > winningScore {
			winningNode = node
			winningScore = score
		}
	}

	return winningNode
}

// compareUUID is a comparator for UUID types, suitable for use with
// [slices.BinarySearchFunc].
func compareUUID(a, b *uuidpb.UUID) int {
	if a.Upper == b.Upper {
		return int(b.Lower - a.Lower)
	}
	return int(b.Upper - a.Upper)
}
