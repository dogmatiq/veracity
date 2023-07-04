package fsm

import (
	"context"
	"sync"
)

// Exchange encapsulates a request/response pair.
type Exchange[Q, S any] struct {
	Context  context.Context
	Request  Q                 // re[Q]uest
	Response FailableFuture[S] // re[S]ponse
}

// RequestQueue models a request queue.
type RequestQueue[Q, S any] struct {
	init      sync.Once
	exchanges chan *Exchange[Q, S]
}

// Requested returns a channel that is readable when a request is received.
func (q *RequestQueue[Q, S]) Requested() <-chan *Exchange[Q, S] {
	return q.channel()
}

// Exchange sends a request and returns its response.
func (q *RequestQueue[Q, S]) Exchange(ctx context.Context, req Q) (S, error) {
	ex := &Exchange[Q, S]{
		Context: ctx,
		Request: req,
	}

	select {
	case <-ctx.Done():
		var zero S
		return zero, ctx.Err()
	case q.channel() <- ex:
	}

	select {
	case <-ctx.Done():
		var zero S
		return zero, ctx.Err()
	case <-ex.Response.Ready():
		return ex.Response.Get()
	}
}

func (q *RequestQueue[Q, S]) channel() chan *Exchange[Q, S] {
	q.init.Do(func() {
		q.exchanges = make(chan *Exchange[Q, S])
	})
	return q.exchanges
}
