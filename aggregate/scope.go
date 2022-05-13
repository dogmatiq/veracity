package aggregate

import (
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
)

// scope is an implementation of dogma.AggregateCommandScope.
type scope struct {
	// IsDestroyed is true if Destroy() has been called and there have been no
	// calls to RecordEvent() since.
	IsDestroyed bool

	// EventEnvelopes is a slice of envelopes containing the recorded events.
	EventEnvelopes []*envelopespec.Envelope
}

// InstanceID returns the ID of the targeted aggregate instance.
func (s *scope) InstanceID() string {
	panic("not implemented")
}

// Destroy destroys the targeted instance.
func (s *scope) Destroy() {
	panic("not implemented")
}

// RecordEvent records the occurrence of an event as a result of the command
// message that is being handled.
func (s *scope) RecordEvent(m dogma.Message) {
	panic("not implemented")
}

// Log records an informational message within the context of the message
// that is being handled.
func (s *scope) Log(f string, v ...interface{}) {
	panic("not implemented")
}
