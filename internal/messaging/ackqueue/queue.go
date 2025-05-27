package ackqueue

import (
	"context"
	"sync"
)

// Request is a container for an "acknowledgable" value of type T, which is sent
// via a [Queue].
type Request[T any] struct {
	Value           T
	Acknowledgement chan<- error
}

// Ack acknowledges the value, indicating that it has been successfully
// processed by the receiver.
func (v Request[T]) Ack() {
	v.Acknowledgement <- nil
}

// Nack negatively acknowledges the value, indicating that it has not been
// successfully processed by the receiver.
func (v Request[T]) Nack(err error) {
	if err == nil {
		panic("ackqueue: cannot nack with a nil error")
	}
	v.Acknowledgement <- err
}

// Queue is an in-memory queue of T values that can be (negatively) acknowledged
// by the receiver.
type Queue[T any] struct {
	init sync.Once
	ch   chan Request[T]
}

// New creates a new [Request] containing v, and returns the channel on
// which its acknowledgment can be received.
func (q *Queue[T]) New(v T) (Request[T], <-chan error) {
	ack := make(chan error, 1)
	return Request[T]{v, ack}, ack
}

// Recv returns a channel that, when read, pops the next value from the queue.
func (q *Queue[T]) Recv() <-chan Request[T] {
	return q.channel()
}

// Send returns a channel that, when written, pushes a value to the queue.
func (q *Queue[T]) Send() chan<- Request[T] {
	return q.channel()
}

// Push pushes a value to the queue and waits for its acknowledgement.
func (q *Queue[T]) Push(ctx context.Context, v T) error {
	req, ack := q.New(v)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.Send() <- req:
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-ack:
		return err
	}
}

func (q *Queue[T]) channel() chan Request[T] {
	q.init.Do(func() {
		q.ch = make(chan Request[T])
	})
	return q.ch
}
