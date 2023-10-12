package engineevent

type subscriber interface {
	Send(e Event, abort <-chan struct{})
}

type subscriberOf[E Event] chan<- E

func (s subscriberOf[E]) Send(event Event, abort <-chan struct{}) {
	if event, ok := event.(E); ok {
		select {
		case s <- event:
		case <-abort:
		}
	}
}
