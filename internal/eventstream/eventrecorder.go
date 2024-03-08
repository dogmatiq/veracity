package eventstream

import (
	"context"

	"github.com/dogmatiq/veracity/internal/messaging"
)

type eventRecorder struct {
	AppendQueue *messaging.ExchangeQueue[AppendRequest, AppendResponse]
}

func (e *eventRecorder) AppendEvents(ctx context.Context, req AppendRequest) (AppendResponse, error) {
	return e.AppendQueue.Exchange(ctx, req)
}
