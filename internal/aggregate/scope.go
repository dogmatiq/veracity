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

	destroy *DestroyAction
}

func (s *scope) InstanceID() string {
	return s.InstID
}

func (s *scope) Destroy() {
	if s.destroy != nil {
		return
	}

	s.destroy = &DestroyAction{}
	s.Revision.Actions = append(
		s.Revision.Actions,
		&RevisionAction{
			OneOf: &RevisionAction_Destroy{
				Destroy: s.destroy,
			},
		},
	)

	s.Logger.Info("scheduled instance for destruction")
}

func (s *scope) RecordEvent(m dogma.Message) {
	if s.destroy != nil {
		s.destroy.IsCancelled = true
		s.destroy = nil
		s.Logger.Info("canceled instance destruction")
	}

	env := s.Packer.Pack(
		m,
		envelope.WithCause(s.CommandEnvelope),
		envelope.WithHandler(s.HandlerIdentity),
		envelope.WithInstanceID(s.InstID),
	)

	s.Root.ApplyEvent(m)
	s.Revision.Actions = append(
		s.Revision.Actions,
		&RevisionAction{
			OneOf: &RevisionAction_RecordEvent{
				RecordEvent: &RecordEventAction{
					Envelope: env,
				},
			},
		},
	)

	s.Logger.Info(
		"recorded a domain event",
		zapx.Envelope("event", env),
	)
}

func (s *scope) Log(f string, v ...interface{}) {
	message := fmt.Sprintf(f, v...)

	s.Revision.Actions = append(
		s.Revision.Actions,
		&RevisionAction{
			OneOf: &RevisionAction_Log{
				Log: &LogAction{
					Message: message,
				},
			},
		},
	)

	s.Logger.Info(
		"logged an application message",
		zap.String("message", message),
	)
}
