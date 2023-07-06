package messaging

import "sync"

// Topic is a source of messages of type T.
type Topic[Message any] struct {
	once       sync.Once
	sub, unsub chan chan<- Message
	pub        chan Message
	done       chan struct{}
}

// Subscribe registers ch as a subscriber to messages from this topic.
func (t *Topic[Message]) Subscribe(ch chan<- Message) {
	t.init()
	select {
	case <-t.done:
	case t.sub <- ch:
	}
}

// Unsubscribe deregisters ch as a subscriber to messages from this topic.
func (t *Topic[Message]) Unsubscribe(ch chan<- Message) {
	t.init()
	select {
	case <-t.done:
	case t.unsub <- ch:
	}
}

// Publish returns a channel that, when written, publishes a message to this
// topic.
func (t *Topic[Message]) Publish() chan<- Message {
	t.init()
	return t.pub
}

// Close closes the topic, preventing further messages from being published.
func (t *Topic[Message]) Close() {
	t.init()
	close(t.done)
}

func (t *Topic[Message]) run() {
	subscribers := map[chan<- Message]struct{}{}

	for {
		select {
		case ch := <-t.sub:
			subscribers[ch] = struct{}{}
		case ch := <-t.unsub:
			delete(subscribers, ch)
		case <-t.done:
			return
		case m := <-t.pub:
			for sub := range subscribers {
				select {
				case <-t.done:
					return
				case sub <- m:
				}
			}
		}
	}
}

func (t *Topic[Message]) init() {
	t.once.Do(func() {
		t.pub = make(chan Message)
		t.sub = make(chan chan<- Message)
		t.unsub = make(chan chan<- Message)
		t.done = make(chan struct{})
		go t.run()
	})
}
