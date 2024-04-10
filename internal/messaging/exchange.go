package messaging

import (
	"context"
	"sync"
)

// Exchange encapsulates a request/response pair.
type Exchange[Req, Res any] struct {
	Request  Req
	Response chan<- Failable[Res]
}

// None is a placeholder type for when a response is not expected.
type None struct{}

// Ok sends a successful response.
func (e Exchange[Req, Res]) Ok(res Res) {
	e.Response <- Failable[Res]{value: res}
}

// Zero sends a successful zero-valued response.
func (e Exchange[Req, Res]) Zero() {
	e.Response <- Failable[Res]{}
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
	ex, response := q.New(req)

	select {
	case <-ctx.Done():
		var zero Res
		return zero, ctx.Err()
	case q.Send() <- ex:
	}

	select {
	case <-ctx.Done():
		var zero Res
		return zero, ctx.Err()
	case f := <-response:
		return f.Get()
	}
}

// New creates a new exchange and response channel.
func (q *ExchangeQueue[Req, Res]) New(req Req) (Exchange[Req, Res], <-chan Failable[Res]) {
	response := make(chan Failable[Res], 1)
	return Exchange[Req, Res]{req, response}, response
}

func (q *ExchangeQueue[Req, Res]) getQueue() chan Exchange[Req, Res] {
	q.init.Do(func() {
		q.queue = make(chan Exchange[Req, Res])
	})
	return q.queue
}
