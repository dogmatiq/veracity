package eventstream

import (
	"context"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/messaging"
)

type EventRecorder struct {
	AppendQueue *messaging.ExchangeQueue[AppendRequest, AppendResponse]
}

func (e *EventRecorder) AppendEvents(ctx context.Context, req AppendRequest) (AppendResponse, error) {
	return e.AppendQueue.Exchange(ctx, req)
}

func (e *EventRecorder) SelectEventStream(context.Context) (streamID *uuidpb.UUID, offset Offset, err error) {
	//TODO: handle partions??
	return uuidpb.Generate(), 0, nil
}
