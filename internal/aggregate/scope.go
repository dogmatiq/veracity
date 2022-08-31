package aggregate

import (
	"github.com/dogmatiq/dogma"
)

// scope is an implementation of dogma.AggregateCommandScope.
type scope struct {
	ID          string
	Root        dogma.AggregateRoot
	Events      []dogma.Message
	IsDestroyed bool
	Logger      func(f string, v ...interface{})
}

// InstanceID returns the ID of the targeted aggregate instance.
func (s *scope) InstanceID() string {
	return s.ID
}

// Destroy destroys the targeted instance.
func (s *scope) Destroy() {
	s.IsDestroyed = true
}

// RecordEvent records the occurrence of an event as a result of the command
// message that is being handled.
func (s *scope) RecordEvent(m dogma.Message) {
	s.Root.ApplyEvent(m)
	s.Events = append(s.Events, m)
	s.IsDestroyed = false
}

// Log records an informational message within the context of the message
// that is being handled.
func (s *scope) Log(f string, v ...interface{}) {
	s.Logger(f, v...)
}
