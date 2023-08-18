package messaging

import (
	"context"
	"sync"
)

// Exchange encapsulates a request/response pair.
type Exchange[Request, Response any] struct {
	Context  context.Context
	Request  Request
	Response chan<- Failable[Response]
}

// Ok sends a successful response.
func (e Exchange[Request, Response]) Ok(res Response) {
	e.Response <- Failable[Response]{value: res}
}

// Err sends an error response.
func (e Exchange[Request, Response]) Err(err error) {
	e.Response <- Failable[Response]{err: err}
}

// ExchangeQueue is a queue of request/response exchanges.
type ExchangeQueue[Request, Response any] struct {
	init  sync.Once
	queue chan Exchange[Request, Response]
}

// Recv returns a channel that, when read, dequeues the next exchange.
func (q *ExchangeQueue[Request, Response]) Recv() <-chan Exchange[Request, Response] {
	return q.getQueue()
}

// Send returns a channel that, when written, enqueues an exchange.
func (q *ExchangeQueue[Request, Response]) Send() chan<- Exchange[Request, Response] {
	return q.getQueue()
}

// Exchange performs a synchronous request/response exchange.
func (q *ExchangeQueue[Request, Response]) Exchange(ctx context.Context, req Request) (res Response, err error) {
	response := make(chan Failable[Response], 1)

	select {
	case <-ctx.Done():
		return res, ctx.Err()
	case q.Send() <- Exchange[Request, Response]{ctx, req, response}:
	}

	select {
	case <-ctx.Done():
		return res, ctx.Err()
	case f := <-response:
		return f.Get()
	}
}

func (q *ExchangeQueue[Request, Response]) getQueue() chan Exchange[Request, Response] {
	q.init.Do(func() {
		q.queue = make(chan Exchange[Request, Response])
	})
	return q.queue
}
