package aggregate

import (
	"github.com/dogmatiq/dogma"
)

// scope is an implementation of dogma.AggregateCommandScope.
type scope struct {
	instanceID string
	root       dogma.AggregateRoot
	tx         Transaction
	logger     Logger
}

func (s *scope) InstanceID() string {
	return s.instanceID
}

func (s *scope) RecordEvent(m dogma.Message) {
	s.tx.Events = append(s.tx.Events, m)

	logFormat := "recorded event: %s"
	if s.tx.Destroyed {
		logFormat += "restored state, " + logFormat
		s.tx.Destroyed = false
	}

	s.logger.Log(
		logFormat,
		dogma.DescribeMessage(m),
	)

	s.root.ApplyEvent(m)
}

func (s *scope) Destroy() {
	if !s.tx.Destroyed {
		s.tx.Destroyed = true
		s.logger.Log("state destroyed")
	}
}

func (s *scope) Log(f string, v ...interface{}) {
	s.logger.Log(f, v...)
}
