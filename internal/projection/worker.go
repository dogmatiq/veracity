package projection

import (
	"context"
	"encoding/binary"
	"errors"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/eventstream"
	"github.com/dogmatiq/veracity/internal/fsm"
	"github.com/dogmatiq/veracity/internal/signaling"
)

// A worker dispatches events from a single eventstream to a projection handler.
type worker struct {
	Handler       dogma.ProjectionMessageHandler
	Packer        *envelope.Packer
	EventConsumer EventConsumer
	StreamID      *uuidpb.UUID
	Shutdown      *signaling.Latch

	resource       []byte
	currentVersion []byte
	events         chan eventstream.Event
	consumerError  chan error
	cancelConsumer context.CancelFunc
}

// resourceFromStream returns the resource ID for a stream.
func resourceFromStream(streamID *uuidpb.UUID) []byte {
	return streamID.AsBytes()
}

// offsetFromVersion returns the offset for a version.
func offsetFromVersion(version []byte) (eventstream.Offset, error) {
	switch len(version) {
	case 0:
		return 0, nil
	case 8:
		return eventstream.Offset(binary.BigEndian.Uint64(version)), nil
	default:
		return 0, errors.New("invalid version")
	}
}

// offsetToVersion returns the version for an offset.
func offsetToVersion(offset eventstream.Offset) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(offset))
	return b
}

func (w *worker) Run(ctx context.Context) error {
	defer func() {
		if w.cancelConsumer != nil {
			w.cancelConsumer()
		}
	}()
	return fsm.Start(ctx, w.initState)
}

func (w *worker) initState(ctx context.Context) fsm.Action {
	if w.cancelConsumer != nil {
		w.cancelConsumer()

		select {
		case <-ctx.Done():
		case <-w.consumerError:
		}
	}

	w.resource = resourceFromStream(w.StreamID)
	w.events = make(chan eventstream.Event)
	w.consumerError = make(chan error, 1)

	var err error
	w.currentVersion, err = w.Handler.ResourceVersion(ctx, w.resource)
	if err != nil {
		return fsm.Fail(err)
	}

	offset, err := offsetFromVersion(w.currentVersion)
	if err != nil {
		return fsm.Fail(err)
	}

	var consumeCtx context.Context
	consumeCtx, w.cancelConsumer = context.WithCancel(ctx)
	go func() {
		w.consumerError <- w.EventConsumer.Consume(consumeCtx, w.StreamID, offset, w.events)
	}()

	return fsm.EnterState(w.idleState)
}

func (w *worker) idleState(ctx context.Context) fsm.Action {
	select {
	case <-ctx.Done():
		return fsm.Stop()

	case <-w.Shutdown.Signaled():
		return fsm.Stop()

	case err := <-w.consumerError:
		if err == nil {
			panic("consumer returned nil")
		}
		return fsm.Fail(err)

	case ese := <-w.events:
		return fsm.With(ese).EnterState(w.handleEventState)
	}
}

// handleEventState handles events.
func (w *worker) handleEventState(
	ctx context.Context,
	ese eventstream.Event,
) fsm.Action {
	e, err := w.Packer.Unpack(ese.Envelope)
	if err != nil {
		return fsm.Fail(err)
	}

	n := offsetToVersion(ese.Offset + 1)

	ok, err := w.Handler.HandleEvent(
		ctx,
		w.resource,
		w.currentVersion,
		n,
		&scope{
			recordedAt: ese.Envelope.GetCreatedAt().AsTime(),
		},
		e,
	)
	if err != nil {
		return fsm.Fail(err)
	}
	if !ok {
		return fsm.EnterState(w.initState)
	}

	w.currentVersion = n

	return fsm.EnterState(w.idleState)
}
