package cluster

import "github.com/dogmatiq/enginekit/protobuf/uuidpb"

// A Node is a member of the cluster.
type Node struct {
	// ID is a unique identifier for the node.
	//
	// It may be generated when the node starts up, or it may be static on a
	// given machine or container.
	ID *uuidpb.UUID

	// Addresses is a list of network addresses for the node's gRPC server, in
	// order of preference.
	Addresses []string
}
