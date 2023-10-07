package integration

import (
	"github.com/dogmatiq/dogma"
)

type scope struct {
	evs []dogma.Event
}

func (s *scope) RecordEvent(e dogma.Event) {
	s.evs = append(s.evs, e)
}

func (s *scope) Log(string, ...any) {
	// TODO
}
