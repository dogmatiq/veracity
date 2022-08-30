package executor

import (
	"github.com/dogmatiq/dogma"
)

// scope is an implementation of dogma.AggregateCommandScope.
type scope struct {
	// Executor *CommandExecutor
	// Command  parcel.Parcel
	// Events   []parcel.Parcel
	root dogma.AggregateRoot
}

// InstanceID returns the ID of the targeted aggregate instance.
func (s *scope) InstanceID() string {
	// return s.Executor.InstanceID
	return ""
}

// Destroy destroys the targeted instance.
func (s *scope) Destroy() {
	panic("not implemented")
}

// RecordEvent records the occurrence of an event as a result of the command
// message that is being handled.
func (s *scope) RecordEvent(m dogma.Message) {
	s.root.ApplyEvent(m)

	// // s.IsDestroyed = false
	// s.Events = append(
	// 	s.Events,
	// 	s.Executor.Packer.PackChildEvent(
	// 		s.Command,
	// 		m,
	// 		s.Executor.HandlerIdentity,
	// 		s.Executor.InstanceID,
	// 	),
	// )
}

// Log records an informational message within the context of the message
// that is being handled.
func (s *scope) Log(f string, v ...interface{}) {
	// s.Logger.Infof(f, v...)
	// panic("not implemented")
}
