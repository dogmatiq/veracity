package queue

import (
	"github.com/dogmatiq/interopspec/envelopespec"
)

// message represents a message on a queue.
type message struct {
	Envelope *envelopespec.Envelope
	Priority uint64

	index int
}

// pqueue is an implementation of heap.Interface that implements an in-memory
// priority queue of messages.
type pqueue struct {
	messages []*message
}

func (q *pqueue) Len() int {
	return len(q.messages)
}

func (q *pqueue) Less(i, j int) bool {
	return q.messages[i].Priority < q.messages[j].Priority
}

func (q *pqueue) Swap(i, j int) {
	q.messages[i], q.messages[j] = q.messages[j], q.messages[i]
	q.messages[i].index = i
	q.messages[j].index = j
}

func (q *pqueue) Push(x any) {
	m := x.(*message)
	m.index = len(q.messages)
	q.messages = append(q.messages, m)
}

func (q *pqueue) Pop() any {
	n := len(q.messages) - 1
	m := q.messages[n]

	q.messages[n] = nil // avoid memory leak
	q.messages = q.messages[:n]

	return m
}

func (q *pqueue) Peek() *envelopespec.Envelope {
	return q.messages[0].Envelope
}
