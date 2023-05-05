package cluster

import "github.com/google/uuid"

// A Node is a member of the cluster.
type Node struct {
	ID uuid.UUID
}
