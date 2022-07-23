package aggregate_test

import (
	"github.com/dogmatiq/configkit"
	. "github.com/dogmatiq/veracity/aggregate"
)

// observerStub observes the lifecycle events of a supervisor.
type observerStub struct {
	OnCommandReceivedFunc   func(h configkit.Identity, id string, cmd *Command)
	OnCommandDispatchedFunc func(h configkit.Identity, id string, cmd *Command)
	OnCommandAckFunc        func(h configkit.Identity, id string, cmd *Command)
	OnCommandNackFunc       func(h configkit.Identity, id string, cmd *Command, reported, cause error)
	OnWorkerStartFunc       func(h configkit.Identity, id string)
	OnWorkerLoadedFunc      func(h configkit.Identity, id string)
	OnWorkerIdleFunc        func(h configkit.Identity, id string)
	OnWorkerIdleAckFunc     func(h configkit.Identity, id string)
	OnWorkerShutdownFunc    func(h configkit.Identity, id string, err error)
}

func (s *observerStub) OnCommandReceived(
	h configkit.Identity,
	id string,
	cmd *Command,
) {
	if s.OnCommandReceivedFunc != nil {
		s.OnCommandReceivedFunc(h, id, cmd)
	}
}

func (s *observerStub) OnCommandDispatched(
	h configkit.Identity,
	id string,
	cmd *Command,
) {
	if s.OnCommandDispatchedFunc != nil {
		s.OnCommandDispatchedFunc(h, id, cmd)
	}
}

func (s *observerStub) OnCommandAck(
	h configkit.Identity,
	id string,
	cmd *Command,
) {
	if s.OnCommandAckFunc != nil {
		s.OnCommandAckFunc(h, id, cmd)
	}
}

func (s *observerStub) OnCommandNack(
	h configkit.Identity,
	id string,
	cmd *Command,
	reported, cause error,
) {
	if s.OnCommandNackFunc != nil {
		s.OnCommandNackFunc(h, id, cmd, reported, cause)
	}
}

func (s *observerStub) OnWorkerStart(
	h configkit.Identity,
	id string,
) {
	if s.OnWorkerStartFunc != nil {
		s.OnWorkerStartFunc(h, id)
	}
}

func (s *observerStub) OnWorkerLoaded(
	h configkit.Identity,
	id string,
) {
	if s.OnWorkerLoadedFunc != nil {
		s.OnWorkerLoadedFunc(h, id)
	}
}

func (s *observerStub) OnWorkerIdle(
	h configkit.Identity,
	id string,
) {
	if s.OnWorkerIdleFunc != nil {
		s.OnWorkerIdleFunc(h, id)
	}
}

func (s *observerStub) OnWorkerIdleAck(
	h configkit.Identity,
	id string,
) {
	if s.OnWorkerIdleAckFunc != nil {
		s.OnWorkerIdleAckFunc(h, id)
	}
}

func (s *observerStub) OnWorkerShutdown(
	h configkit.Identity,
	id string,
	err error,
) {
	if s.OnWorkerShutdownFunc != nil {
		s.OnWorkerShutdownFunc(h, id, err)
	}
}
