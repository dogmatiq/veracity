package fsm

import "sync"

// Topic is a source of messages of type T.
type Topic[T any] struct {
	once       sync.Once
	sub, unsub chan chan<- T
	pub        chan T
	done       chan struct{}
}

// Publish returns a channel that, when written, publishes a message to this
// topic.
func (t *Topic[T]) Publish() chan<- T {
	t.init()
	return t.pub
}

// Close closes the topic, preventing further messages from being published.
func (t *Topic[T]) Close() {
	t.init()
	close(t.done)
}

// Subscribe registers ch as a subscriber to messages from this topic.
func (t *Topic[T]) Subscribe(ch chan<- T) {
	t.init()
	select {
	case <-t.done:
	case t.sub <- ch:
	}
}

// Unsubscribe deregisters ch as a subscriber to messages from this topic.
func (t *Topic[T]) Unsubscribe(ch chan<- T) {
	t.init()
	select {
	case <-t.done:
	case t.unsub <- ch:
	}
}

func (t *Topic[T]) run() {
	subscribers := map[chan<- T]struct{}{}

	for {
		select {
		case ch := <-t.sub:
			subscribers[ch] = struct{}{}
		case ch := <-t.unsub:
			delete(subscribers, ch)
		case m := <-t.pub:
			for ch := range subscribers {
				select {
				case ch <- m:
				case <-t.done:
					return
				}
			}
		case <-t.done:
			return
		}
	}
}

func (t *Topic[T]) init() {
	t.once.Do(func() {
		t.pub = make(chan T)
		t.sub = make(chan chan<- T)
		t.unsub = make(chan chan<- T)
		t.done = make(chan struct{})
		go t.run()
	})
}
