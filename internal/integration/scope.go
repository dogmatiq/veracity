package integration

import (
	"context"
	"fmt"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/identitypb"
	"github.com/dogmatiq/veracity/internal/telemetry"
)

type scope struct {
	ctx       context.Context
	packer    *envelopepb.Packer
	handler   *identitypb.Identity
	command   *envelopepb.Envelope
	events    []*envelopepb.Envelope
	telemetry *telemetry.Recorder
}

func (s *scope) RecordEvent(e dogma.Event) {
	env := s.packer.Pack(
		e,
		envelopepb.WithHandler(s.handler),
		envelopepb.WithCause(s.command),
	)

	s.events = append(s.events, env)

	s.telemetry.Info(
		s.ctx,
		"integration.event_recorded",
		"event recorded",
		telemetry.UUID("event.message_id", env.GetMessageId()),
		telemetry.UUID("event.causation_id", env.GetMessageId()),
		telemetry.UUID("event.correlation_id", env.GetCorrelationId()),
		telemetry.String("event.media_type", env.GetMediaType()),
		telemetry.String("event.description", env.GetDescription()),
	)
}

func (s *scope) Log(format string, args ...any) {
	s.telemetry.Info(
		s.ctx,
		"integration.message_logged",
		fmt.Sprintf(format, args...),
	)
}
