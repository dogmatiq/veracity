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

// ExchangeQueue is a queue of request/response exchanges.
type ExchangeQueue[Request, Response any] struct {
	init sync.Once
	ch   chan Exchange[Request, Response]
}

// Push returns a channel that, when written, enqueues an exchange.
func (q *ExchangeQueue[Request, Response]) Push() chan<- Exchange[Request, Response] {
	return q.queue()
}

// Pop returns a channel that, when read, dequeues the next exchange.
func (q *ExchangeQueue[Request, Response]) Pop() <-chan Exchange[Request, Response] {
	return q.queue()
}

// Exchange performs a synchronous request/response exchange.
func (q *ExchangeQueue[Request, Response]) Exchange(ctx context.Context, req Request) (res Response, err error) {
	ch := make(chan Failable[Response], 1)

	select {
	case <-ctx.Done():
		return res, ctx.Err()
	case q.Push() <- Exchange[Request, Response]{ctx, req, ch}:
	}

	select {
	case <-ctx.Done():
		return res, ctx.Err()
	case f := <-ch:
		return f.Get()
	}
}

func (q *ExchangeQueue[Request, Response]) queue() chan Exchange[Request, Response] {
	q.init.Do(func() {
		q.ch = make(chan Exchange[Request, Response])
	})
	return q.ch
}
