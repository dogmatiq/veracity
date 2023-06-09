package integration

import "github.com/dogmatiq/interopspec/envelopespec"

type EnqueueCommandExchange struct {
	Command *envelopespec.Envelope

	Done chan<- struct{}
}
