package integration

import (
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/identitypb"
)

type scope struct {
	packer  *envelopepb.Packer
	handler *identitypb.Identity
	command *envelopepb.Envelope
	events  []*envelopepb.Envelope
}

func (s *scope) RecordEvent(e dogma.Event) {
	s.events = append(
		s.events,
		s.packer.Pack(
			e,
			envelopepb.WithHandler(s.handler),
			envelopepb.WithCause(s.command),
		),
	)
}

func (s *scope) Log(string, ...any) {
	// TODO
}
