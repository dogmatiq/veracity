package eventstream

import (
	"context"
	"errors"
	"log/slog"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/journal"
	"github.com/dogmatiq/veracity/internal/eventstream/internal/eventstreamjournal"
	"github.com/dogmatiq/veracity/internal/fsm"
	"github.com/dogmatiq/veracity/internal/messaging"
	"github.com/dogmatiq/veracity/internal/signaling"
)

// errShuttingDown is sent in response to requests that are not serviced
// because of an error within the event stream supervisor or a worker.
var errShuttingDown = errors.New("event stream sub-system is shutting down")

// A Supervisor coordinates event stream workers.
type Supervisor struct {
	Journals         journal.BinaryStore
	AppendQueue      messaging.ExchangeQueue[AppendRequest, AppendResponse]
	SubscribeQueue   messaging.ExchangeQueue[*Subscriber, messaging.None]
	UnsubscribeQueue messaging.ExchangeQueue[*Subscriber, messaging.None]
	Logger           *slog.Logger

	shutdown      signaling.Latch
	workers       uuidpb.Map[*worker]
	workerStopped chan workerResult
}

type workerResult struct {
	StreamID *uuidpb.UUID
	Err      error
}

// Run starts the supervisor.
func (s *Supervisor) Run(ctx context.Context) error {
	s.workers = uuidpb.Map[*worker]{}
	s.workerStopped = make(chan workerResult)

	return fsm.Start(
		ctx,
		s.idleState,
		fsm.WithFinalState(s.shutdownState),
	)
}

// Shutdown stops the supervisor when it next becomes idle.
func (s *Supervisor) Shutdown() {
	s.shutdown.Signal()
}

// idleState waits for an append request.
func (s *Supervisor) idleState(ctx context.Context) fsm.Action {
	select {
	case <-ctx.Done():
		return fsm.Stop()

	case res := <-s.workerStopped:
		s.workers.Delete(res.StreamID)
		if res.Err != nil {
			return fsm.Fail(res.Err)
		}
		return fsm.StayInCurrentState()

	case ex := <-s.AppendQueue.Recv():
		return fsm.With(ex).EnterState(s.appendState)

	case req := <-s.SubscribeQueue.Recv():
		return fsm.With(req).EnterState(s.subscribeState)

	case req := <-s.UnsubscribeQueue.Recv():
		return fsm.With(req).EnterState(s.unsubscribeState)
	}
}

// appendState forwards an append request to the appropriate worker.
func (s *Supervisor) appendState(
	ctx context.Context,
	ex messaging.Exchange[AppendRequest, AppendResponse],
) fsm.Action {
	w, err := s.workerByStreamID(ctx, ex.Request.StreamID)
	if err != nil {
		ex.Err(errShuttingDown)
		return fsm.Fail(err)
	}

	return forwardToWorker(ctx, s, w.AppendQueue.Send(), ex)
}

// subscribeState forwards a subscribe request to the appropriate worker.
func (s *Supervisor) subscribeState(
	ctx context.Context,
	ex messaging.Exchange[*Subscriber, messaging.None],
) fsm.Action {
	w, err := s.workerByStreamID(ctx, ex.Request.StreamID)
	if err != nil {
		ex.Err(errShuttingDown)
		return fsm.Fail(err)
	}

	return forwardToWorker(ctx, s, w.SubscribeQueue.Send(), ex)
}

// unsubscribeState forwards an unsubscribe request to the appropriate worker.
func (s *Supervisor) unsubscribeState(
	ctx context.Context,
	ex messaging.Exchange[*Subscriber, messaging.None],
) fsm.Action {
	w, ok := s.workers.TryGet(ex.Request.StreamID)
	if !ok {
		ex.Zero()
		return fsm.EnterState(s.idleState)
	}

	return forwardToWorker(ctx, s, w.UnsubscribeQueue.Send(), ex)
}

func forwardToWorker[
	T interface{ Err(error) },
](
	ctx context.Context,
	s *Supervisor,
	q chan<- T,
	v T,
) fsm.Action {
	select {
	case <-ctx.Done():
		v.Err(errShuttingDown)
		return fsm.Stop()

	case res := <-s.workerStopped:
		s.workers.Delete(res.StreamID)
		if res.Err != nil {
			v.Err(errShuttingDown)
			return fsm.Fail(res.Err)
		}
		return fsm.StayInCurrentState()

	case q <- v:
		return fsm.EnterState(s.idleState)
	}
}

// shutdownState signals all workers to shutdown and waits for them to finish.
func (s *Supervisor) shutdownState(context.Context) fsm.Action {
	for _, w := range s.workers {
		w.Shutdown.Signal()
	}

	for len(s.workers) > 0 {
		res := <-s.workerStopped
		s.workers.Delete(res.StreamID)
	}

	return fsm.Stop()
}

// workerByStreamID returns the worker that manages the state of the stream with
// the given ID. The worker is started if it is not already running.
func (s *Supervisor) workerByStreamID(
	ctx context.Context,
	streamID *uuidpb.UUID,
) (*worker, error) {
	if w, ok := s.workers.TryGet(streamID); ok {
		return w, nil
	}

	w, err := s.startWorkerForStreamID(ctx, streamID)
	if err != nil {
		return nil, err
	}

	s.workers.Set(streamID, w)

	return w, nil
}

// startWorkerForStreamID starts a new worker for the stream with the given ID.
func (s *Supervisor) startWorkerForStreamID(
	ctx context.Context,
	streamID *uuidpb.UUID,
) (*worker, error) {
	j, err := eventstreamjournal.Open(ctx, s.Journals, streamID)
	if err != nil {
		return nil, err
	}

	w := &worker{
		StreamID: streamID,
		Journal:  j,
		Logger: s.Logger.With(
			slog.String("stream_id", streamID.AsString()),
		),
	}

	go func() {
		defer j.Close()

		s.workerStopped <- workerResult{
			StreamID: streamID,
			Err:      w.Run(ctx),
		}
	}()

	return w, nil
}
