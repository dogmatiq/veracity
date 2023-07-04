package cluster

import (
	"sync"

	"github.com/cespare/xxhash/v2"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
)

// Partitioner determines which node is responsible for handling a specific
// workload.
type Partitioner struct {
	m     sync.RWMutex
	nodes uuidpb.Set
}

// AddNode adds a node to the partitioner.
func (p *Partitioner) AddNode(id *uuidpb.UUID) {
	p.m.Lock()
	defer p.m.Unlock()

	if p.nodes == nil {
		p.nodes = uuidpb.Set{}
	}

	p.nodes.Add(id)
}

// RemoveNode removes a node from the partitioner.
func (p *Partitioner) RemoveNode(id *uuidpb.UUID) {
	p.m.Lock()
	defer p.m.Unlock()

	p.nodes.Delete(id)
}

// Route returns the ID of the node that should handle the workload identified
// by the given string.
func (p *Partitioner) Route(workload string) *uuidpb.UUID {
	p.m.RLock()
	defer p.m.RUnlock()

	if len(p.nodes) == 0 {
		panic("partitioner has no nodes")
	}

	var (
		bestID    *uuidpb.UUID
		bestScore uint64
		hash      xxhash.Digest
	)

	for id := range p.nodes {
		id := id.AsUUID()
		hash.Write(id.AsBytes())
		hash.WriteString(workload)
		score := hash.Sum64()
		hash.Reset()

		if score > bestScore {
			bestID = id
			bestScore = score
		}
	}

	return bestID
}
