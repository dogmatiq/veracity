package engineconfig

import (
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/ferrite"
)

var nodeID = ferrite.
	String("VERACITY_NODE_ID", "a unique identifier for this cluster node").
	WithConstraint(
		"must be a UUID",
		func(v string) bool {
			id, err := uuidpb.FromString(v)
			if err != nil {
				return false
			}

			return id.Validate() == nil
		},
	).
	Optional(ferrite.WithRegistry(FerriteRegistry))

func (c *Config) finalizeNodeID() {
	if c.NodeID != nil {
		return
	}

	if c.UseEnv {
		if v, ok := nodeID.Value(); ok {
			id, err := uuidpb.FromString(v)
			if err != nil {
				panic(err)
			}
			c.NodeID = id
			return
		}
	}

	c.NodeID = uuidpb.Generate()
}
