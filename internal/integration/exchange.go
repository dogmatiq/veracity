package integration

import "github.com/dogmatiq/enginekit/protobuf/envelopepb"

type EnqueueCommandExchange struct {
	Command *envelopepb.Envelope

	Done chan<- error
}
