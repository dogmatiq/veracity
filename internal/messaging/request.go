package messaging

import (
	"context"
	"sync"
)

// Request encapsulates a request.
type Request[Req any] struct {
	Context context.Context
	Request Req
	Error   chan<- error
}

// Ok sends a successful response.
func (e Request[Req]) Ok() {
	e.Error <- nil
}

// Err sends an error response.
func (e Request[Req]) Err(err error) {
	e.Error <- err
}

// RequestQueue is a queue of requests.
type RequestQueue[Req any] struct {
	init  sync.Once
	queue chan Request[Req]
}

// Recv returns a channel that, when read, dequeues the next request.
func (q *RequestQueue[Req]) Recv() <-chan Request[Req] {
	return q.getQueue()
}

// Send returns a channel that, when written, enqueues an request.
func (q *RequestQueue[Req]) Send() chan<- Request[Req] {
	return q.getQueue()
}

// Do performs a synchronous request.
func (q *RequestQueue[Req]) Do(ctx context.Context, req Req) error {
	response := make(chan error, 1)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.Send() <- Request[Req]{ctx, req, response}:
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-response:
		return err
	}
}

func (q *RequestQueue[Req]) getQueue() chan Request[Req] {
	q.init.Do(func() {
		q.queue = make(chan Request[Req])
	})
	return q.queue
}
