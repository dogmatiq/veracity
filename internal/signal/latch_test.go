package signal_test

import (
	"testing"

	. "github.com/dogmatiq/veracity/internal/signal"
	"golang.org/x/exp/maps"
	"pgregory.net/rapid"
)

func TestLatch(t *testing.T) {
	type watcher struct {
		ID      int
		Channel <-chan struct{}
		Cancel  CancelFunc
	}

	rapid.Check(t, func(t *rapid.T) {
		var (
			signal     Latch
			watcherID  = 0
			isNotified = false
			pending    = map[int]*watcher{}
			canceled   = map[int]*watcher{}
		)

		t.Repeat(map[string]func(*rapid.T){
			"add a watcher": func(t *rapid.T) {
				watcherID++

				ch := make(chan struct{})

				pending[watcherID] = &watcher{
					ID:      watcherID,
					Channel: ch,
					Cancel:  signal.Watch(ch),
				}

				t.Logf("added watcher %d", watcherID)

				if isNotified {
					select {
					case <-ch:
						t.Log("read notification from watcher")
						delete(pending, watcherID)
					default:
						t.Fatal("expected watcher to be notified immediately")
					}
				}

			},
			"cancel a watcher that has not yet been notified": func(t *rapid.T) {
				if len(pending) == 0 {
					t.Skip("no pending watchers")
				}

				id := rapid.SampledFrom(maps.Keys(pending)).Draw(t, "watcher")
				w := pending[id]

				w.Cancel()

				delete(pending, id)
				canceled[id] = w
			},
			"notify the watchers": func(t *rapid.T) {
				signal.Notify()
				isNotified = true

				for id, w := range pending {
					select {
					case <-w.Channel:
						t.Log("read notification from watcher")
						delete(pending, id)
					default:
						t.Fatal("expected pending watcher to be closed")
					}
				}

				for _, w := range canceled {
					select {
					case <-w.Channel:
						t.Fatal("did not expect canceled watcher to be closed")
					default:
					}
				}
			},
		})
	})
}
