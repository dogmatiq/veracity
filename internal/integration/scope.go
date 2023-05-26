package integration

import (
	"github.com/dogmatiq/dogma"
)

type scope struct {
}

func (s *scope) RecordEvent(e dogma.Event) {
}

func (s *scope) Log(format string, args ...any) {
}
