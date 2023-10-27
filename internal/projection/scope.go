package projection

import (
	"time"
)

type scope struct {
	recordedAt time.Time
}

func (s *scope) RecordedAt() time.Time {
	return s.recordedAt
}

func (s *scope) IsPrimaryDelivery() bool {
	// TODO
	return true
}

func (s *scope) Log(string, ...any) {
	// TODO
}
