package projection

import (
	"context"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/signaling"
	"golang.org/x/sync/errgroup"
)

// A Supervisor coordinates projection workers.
type Supervisor struct {
	Handler       dogma.ProjectionMessageHandler
	Packer        *envelope.Packer
	EventConsumer EventConsumer
	StreamIDs     []*uuidpb.UUID

	shutdown signaling.Latch
}

func (s *Supervisor) Run(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	for _, streamID := range s.StreamIDs {
		streamID := streamID // capture loop variable

		eg.Go(
			func() error {
				worker := &worker{
					Handler:       s.Handler,
					EventConsumer: s.EventConsumer,
					Packer:        s.Packer,
					StreamID:      streamID,
					Shutdown:      &s.shutdown,
				}
				return worker.Run(ctx)
			},
		)
	}

	return eg.Wait()
}

// Shutdown stops the supervisor when it next becomes idle.
func (s *Supervisor) Shutdown() {
	s.shutdown.Signal()
}
