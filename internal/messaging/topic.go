package messaging

import (
	"slices"
	"sync"
)

// Topic is a source of messages of type T.
type Topic[T any] struct {
	init     sync.Once
	messages chan T

	m           sync.Mutex
	subscribers []subscriber[T]
}

type subscriber[T any] struct {
	messages     chan<- T
	unsubscribed chan struct{}
}

// Subscribe registers ch as a subscriber to messages from this topic.
func (t *Topic[T]) Subscribe(ch chan<- T) {
	t.m.Lock()
	defer t.m.Unlock()

	t.subscribers = append(
		t.subscribers,
		subscriber[T]{
			messages:     ch,
			unsubscribed: make(chan struct{}),
		},
	)
}

// Unsubscribe removes ch as a subscriber to messages from this topic.
func (t *Topic[T]) Unsubscribe(ch chan<- T) {
	t.m.Lock()
	defer t.m.Unlock()

	for i, sub := range t.subscribers {
		if sub.messages == ch {
			n := len(t.subscribers) - 1
			t.subscribers[i] = t.subscribers[n]
			t.subscribers[n] = subscriber[T]{}
			t.subscribers = t.subscribers[:n]
			close(sub.unsubscribed)
		}
	}
}

// Publish returns a channel that, when written, publishes a message to this
// topic.
func (t *Topic[T]) Publish() chan<- T {
	t.init.Do(func() {
		t.messages = make(chan T)
		go t.run()
	})
	return t.messages
}

// Close closes the topic, preventing further messages from being published.
func (t *Topic[T]) Close() {
	t.init.Do(func() {
		t.messages = make(chan T)
	})
	close(t.messages)
}

func (t *Topic[T]) run() {
	for m := range t.messages {
		t.m.Lock()
		subscribers := slices.Clone(t.subscribers)
		t.m.Unlock()

		for _, sub := range subscribers {
			select {
			case sub.messages <- m:
			case <-sub.unsubscribed:
			}
		}
	}
}
