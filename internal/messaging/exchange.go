package messaging

import (
	"context"
	"sync"
)

// Exchange encapsulates a request/response pair.
type Exchange[Req, Res any] struct {
	Context  context.Context
	Request  Req
	Response chan<- Failable[Res]
}

// Ok sends a successful response.
func (e Exchange[Req, Res]) Ok(res Res) {
	e.Response <- Failable[Res]{value: res}
}

// Err sends an error response.
func (e Exchange[Req, Res]) Err(err error) {
	e.Response <- Failable[Res]{err: err}
}

// ExchangeQueue is a queue of request/response exchanges.
type ExchangeQueue[Req, Res any] struct {
	init  sync.Once
	queue chan Exchange[Req, Res]
}

// Recv returns a channel that, when read, dequeues the next exchange.
func (q *ExchangeQueue[Req, Res]) Recv() <-chan Exchange[Req, Res] {
	return q.getQueue()
}

// Send returns a channel that, when written, enqueues an exchange.
func (q *ExchangeQueue[Req, Res]) Send() chan<- Exchange[Req, Res] {
	return q.getQueue()
}

// Do performs a synchronous request/response exchange.
func (q *ExchangeQueue[Req, Res]) Do(ctx context.Context, req Req) (Res, error) {
	response := make(chan Failable[Res], 1)

	select {
	case <-ctx.Done():
		var zero Res
		return zero, ctx.Err()
	case q.Send() <- Exchange[Req, Res]{ctx, req, response}:
	}

	select {
	case <-ctx.Done():
		var zero Res
		return zero, ctx.Err()
	case f := <-response:
		return f.Get()
	}
}

func (q *ExchangeQueue[Req, Res]) getQueue() chan Exchange[Req, Res] {
	q.init.Do(func() {
		q.queue = make(chan Exchange[Req, Res])
	})
	return q.queue
}
