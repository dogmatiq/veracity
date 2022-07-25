package aggregate

import (
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/parcel"
	"go.uber.org/zap"
)

// scope is an implementation of dogma.AggregateCommandScope.
type scope struct {
	// Command is the parcel containing the command being handled.
	Command parcel.Parcel

	// HandlerIdentity is the identity of the handler.
	HandlerIdentity *envelopespec.Identity

	// ID is the ID of the aggregate instance to which the command was routed.
	ID string

	// Root is the aggregate root of the instance that to which the command was
	// routed.
	Root dogma.AggregateRoot

	// Packer is used to create parcels containing the recorded events.
	Packer *parcel.Packer

	// IsDestroyed is true if Destroy() has been called and there have been no
	// calls to RecordEvent() since.
	IsDestroyed bool

	// EventEnvelopes is a slice of envelopes containing the recorded events.
	EventEnvelopes []*envelopespec.Envelope

	// Logger is the target for log messages from the handler.
	Logger *zap.SugaredLogger
}

// InstanceID returns the ID of the targeted aggregate instance.
func (s *scope) InstanceID() string {
	return s.ID
}

// Destroy destroys the targeted instance.
func (s *scope) Destroy() {
	if !s.IsDestroyed {
		s.IsDestroyed = true
	}
}

// RecordEvent records the occurrence of an event as a result of the command
// message that is being handled.
func (s *scope) RecordEvent(m dogma.Message) {
	s.Root.ApplyEvent(m)

	s.IsDestroyed = false
	s.EventEnvelopes = append(
		s.EventEnvelopes,
		s.Packer.PackChildEvent(
			s.Command,
			m,
			s.HandlerIdentity,
			s.ID,
		).Envelope,
	)
}

// Log records an informational message within the context of the message
// that is being handled.
func (s *scope) Log(f string, v ...interface{}) {
	s.Logger.Infof(f, v...)
}
