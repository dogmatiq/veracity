package engineconfig

import (
	"github.com/dogmatiq/ferrite"
	"github.com/google/uuid"
)

var nodeID = ferrite.
	String("VERACITY_NODE_ID", "a unique identifier for this cluster node").
	WithConstraint(
		"must be a UUID",
		func(v string) bool {
			id, err := uuid.Parse(v)
			return err != nil && id != uuid.Nil
		},
	).
	Optional(ferrite.WithRegistry(FerriteRegistry))

func (c *Config) finalizeNodeID() {
	if c.NodeID != uuid.Nil {
		return
	}

	if c.UseEnv {
		if id, ok := nodeID.Value(); ok {
			c.NodeID = uuid.MustParse(id)
			return
		}
	}

	c.NodeID = uuid.New()
}
