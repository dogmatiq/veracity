package engineevent

import (
	"reflect"
	"sync"

	"github.com/dogmatiq/veracity/internal/signaling"
)

// Bus is an in-memory message bus for delivering "engine events" between
// subsystems of the engine.
type Bus struct {
	topics sync.Map
	once   sync.Once
	closed signaling.Latch
}

func NewPublisher[T any](b *Bus) (chan<- T, func()) {
	return topicFrom[T](b).publish, func() {}
}

func NewSubscriber[T any](b *Bus, ch chan<- T) func() {
	t := topicFrom[T](b)

	select {
	case t.sub <- ch:
	case <-t.closed.Signaled():
		panic("bus closed")
	}
}

type topic[T any] struct {
	publish    chan T
	closed     *signaling.Latch
	sub, unsub chan chan<- T
}

func topicFrom[T any](b *Bus) *topic[T] {
	var zero T
	key := reflect.TypeOf(zero)

	if x, ok := b.topics.Load(key); ok {
		return x.(*topic[T])
	}

	t := &topic[T]{
		publish: make(chan T),
		closed:  &b.closed,
		sub:     make(chan chan<- T),
		unsub:   make(chan chan<- T),
	}

	if x, ok := b.topics.LoadOrStore(key, t); ok {
		return x.(*topic[T])
	}

	// go t.run()

	return t
}

// func GetTopic[T any](b *Bus) Topic[T] {
// }

// func (b *Bus) Topic(name string) Topic {
// }

// func (t *Bus) Close() {
// 	t.closed.Signal()
// }

// type Topic[T any] struct {
// 	sub, unsub chan subscriber[T]
// 	publish    chan T
// 	closed     *signaling.Latch
// }

// // Publish returns a channel to which events can be published.
// func (t *Topic) Publish() chan<- Event {
// 	return b.publish
// }

// // Subscribe registers a channel to receive events of type E.
// func Subscribe[E Event](b *Bus, events chan<- E) {
// 	b.init()

// 	select {
// 	case b.sub <- subscriberOf[E](events):
// 	case <-b.closed:
// 	}
// }

// // Unsubscribe stops a channel from receiving events of type E.
// func Unsubscribe[E Event](b *Bus, events chan<- E) {
// 	b.init()

// 	select {
// 	case b.unsub <- subscriberOf[E](events):
// 	case <-b.closed:
// 	}
// }

// // Close stops the bus, preventing any further events from being published.
// func (b *Bus) Close() {
// 	b.init()
// 	close(b.closed)
// }

// func (b *Bus) init() {
// 	b.once.Do(func() {
// 		b.sub = make(chan subscriber)
// 		b.unsub = make(chan subscriber)
// 		b.publish = make(chan Event)
// 		b.closed = make(chan struct{})
// 		go b.run()
// 	})
// }

// func (b *Bus) run() {
// 	var subscribers []subscriber

// 	for {
// 		select {
// 		case c := <-b.sub:
// 			subscribers = append(subscribers, c)
// 		case c := <-b.unsub:
// 			subscribers = remove(subscribers, c)
// 		case e := <-b.publish:
// 			b.dispatch(subscribers, e)
// 		case <-b.closed:
// 			return
// 		}
// 	}
// }

// func (b *Bus) dispatch(subscribers []subscriber, e Event) {
// 	var g sync.WaitGroup
// 	g.Add(len(subscribers))

// 	for _, c := range subscribers {
// 		c := c // capture loop variable
// 		go func() {
// 			c.Send(e, b.closed)
// 			g.Done()
// 		}()
// 	}

// 	g.Wait()
// }

// func remove[E comparable, S ~[]E](slice S, elem E) S {
// 	for i, x := range slice {
// 		if x == elem {
// 			n := len(slice) - 1
// 			slice[i] = slice[n]

// 			var zero E
// 			slice[n] = zero
// 			return slice[:n]
// 		}
// 	}

// 	return slice
// }
