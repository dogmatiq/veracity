package cluster

import (
	"sync"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
)

// Partitioner determines which node is responsible for handling a specific
// workload.
type Partitioner struct {
	m     sync.RWMutex
	nodes map[uuid.UUID]struct{}
}

// AddNode adds a node to the partitioner.
func (p *Partitioner) AddNode(id uuid.UUID) {
	p.m.Lock()
	defer p.m.Unlock()

	if p.nodes == nil {
		p.nodes = map[uuid.UUID]struct{}{}
	}

	p.nodes[id] = struct{}{}
}

// RemoveNode removes a node from the partitioner.
func (p *Partitioner) RemoveNode(id uuid.UUID) {
	p.m.Lock()
	defer p.m.Unlock()

	delete(p.nodes, id)
}

// Route returns the ID of the node that should handle the workload identified
// by the given string.
func (p *Partitioner) Route(workload string) uuid.UUID {
	p.m.RLock()
	defer p.m.RUnlock()

	if len(p.nodes) == 0 {
		panic("partitioner has no nodes")
	}

	var (
		bestID    uuid.UUID
		bestScore uint64
		hash      xxhash.Digest
	)

	for id := range p.nodes {
		hash.Write(id[:])
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
