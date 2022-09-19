package aggregate

import (
	"fmt"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/zapx"
	"go.uber.org/zap"
)

// scope is an implementation of dogma.AggregateCommandScope.
type scope struct {
	HandlerIdentity *envelopespec.Identity
	InstID          string
	Root            dogma.AggregateRoot
	Packer          *envelope.Packer
	Logger          *zap.Logger
	CommandEnvelope *envelopespec.Envelope
	Revision        *RevisionRecord
}

func (s *scope) InstanceID() string {
	return s.InstID
}

func (s *scope) Destroy() {
	if !s.Revision.InstanceDestroyed {
		s.Revision.InstanceDestroyed = true
		s.Logger.Info("scheduled instance for destruction")
	}
}

func (s *scope) RecordEvent(m dogma.Message) {
	if s.Revision.InstanceDestroyed {
		s.Revision.InstanceDestroyed = false
		s.Logger.Info("canceled instance destruction")
	}

	env := s.Packer.Pack(
		m,
		envelope.WithCause(s.CommandEnvelope),
		envelope.WithHandler(s.HandlerIdentity),
		envelope.WithInstanceID(s.InstID),
	)

	s.Root.ApplyEvent(m)
	s.Revision.EventEnvelopes = append(s.Revision.EventEnvelopes, env)

	s.Logger.Info(
		"recorded a domain event",
		zapx.Envelope("event", env),
	)
}

func (s *scope) Log(f string, v ...interface{}) {
	message := fmt.Sprintf(f, v...)

	s.Logger.Info(
		"logged an application message",
		zap.String("message", message),
	)
}
