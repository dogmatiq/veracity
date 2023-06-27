package signal

import "context"

// CancelFunc is a function that cancels a watcher from receiving signals.
type CancelFunc func()

// Signal is an interface for a signal that can notify "watcher channels".
type Signal interface {
	Watch(chan<- struct{}) CancelFunc
	Notify()
}

// Wait adds a watcher to sig and waits for it to be notified, or for ctx to be
// canceled, whichever is first.
func Wait(
	ctx context.Context,
	sig Signal,
) error {
	ch := make(chan struct{}, 1)

	cancel := sig.Watch(ch)
	defer cancel()

	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
