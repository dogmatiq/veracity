package queue

// elem is an element within the queue.
//
// It contains the queued message and additional internal meta-data.
type elem struct {
	*JournalMessage
	index int
}

// pqueue is an in-memory priority queue. It implements heap.Interface.
type pqueue struct {
	elements []*elem
}

func (q *pqueue) Len() int {
	return len(q.elements)
}

func (q *pqueue) Less(i, j int) bool {
	return q.elements[i].Priority < q.elements[j].Priority
}

func (q *pqueue) Swap(i, j int) {
	q.elements[i], q.elements[j] = q.elements[j], q.elements[i]
	q.elements[i].index = i
	q.elements[j].index = j
}

func (q *pqueue) Push(x any) {
	m := x.(*elem)
	m.index = len(q.elements)
	q.elements = append(q.elements, m)
}

func (q *pqueue) Pop() any {
	n := len(q.elements) - 1
	m := q.elements[n]

	q.elements[n] = nil // avoid memory leak
	q.elements = q.elements[:n]

	return m
}
