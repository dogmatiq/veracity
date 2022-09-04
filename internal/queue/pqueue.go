package queue

import (
	"container/heap"
	"time"

	"github.com/dogmatiq/interopspec/envelopespec"
)

// message represents a message on a queue.
type message struct {
	Envelope  *envelopespec.Envelope
	CreatedAt time.Time

	index int
}

// pqueue is an in-memory priority queue. It implements heap.Interface.
type pqueue struct {
	heap []*message
}

func (q *pqueue) PushMessage(m *message) {
	heap.Push(q, m)
}

func (q *pqueue) RemoveMessage(m *message) {
	heap.Remove(q, m.index)
}

func (q *pqueue) PeekMessage() *message {
	return q.heap[0]
}

func (q *pqueue) Len() int {
	return len(q.heap)
}

func (q *pqueue) Less(i, j int) bool {
	return q.heap[i].CreatedAt.Before(q.heap[j].CreatedAt)
}

func (q *pqueue) Swap(i, j int) {
	q.heap[i], q.heap[j] = q.heap[j], q.heap[i]
	q.heap[i].index = i
	q.heap[j].index = j
}

func (q *pqueue) Push(x any) {
	m := x.(*message)
	m.index = len(q.heap)
	q.heap = append(q.heap, m)
}

func (q *pqueue) Pop() any {
	n := len(q.heap) - 1
	m := q.heap[n]

	q.heap[n] = nil // avoid memory leak
	q.heap = q.heap[:n]

	return m
}
