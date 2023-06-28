package signal_test

import (
	"testing"

	. "github.com/dogmatiq/veracity/internal/signal"
	"golang.org/x/exp/maps"
	"pgregory.net/rapid"
)

func TestLossy(t *testing.T) {
	type watcher struct {
		ID      int
		Channel <-chan struct{}
		Cancel  CancelFunc
	}

	rapid.Check(t, func(t *rapid.T) {
		var (
			signal    Lossy
			watcherID = 0
			pending   = map[int]*watcher{}
			notified  = map[int]*watcher{}
			canceled  = map[int]*watcher{}
		)

		t.Repeat(map[string]func(*rapid.T){
			"add a watcher": func(t *rapid.T) {
				watcherID++

				ch := make(chan struct{}, 1)
				pending[watcherID] = &watcher{
					ID:      watcherID,
					Channel: ch,
					Cancel:  signal.Watch(ch),
				}

				t.Logf("added watcher %d", watcherID)
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
			"cancel a watcher that has already been notified": func(t *rapid.T) {
				if len(notified) == 0 {
					t.Skip("no notified watchers")
				}

				id := rapid.SampledFrom(maps.Keys(notified)).Draw(t, "id")
				w := notified[id]

				select {
				case <-w.Channel:
					t.Log("read notification from watcher")
				default:
					t.Fatal("expected a pending notification")
				}

				w.Cancel()

				delete(notified, id)
				canceled[id] = w
			},
			"notify the watchers": func(t *rapid.T) {
				signal.Notify()

				for id, w := range pending {
					if rapid.Bool().Draw(t, "read") {
						select {
						case <-w.Channel:
							t.Log("read notification from watcher")
						default:
							t.Fatal("expected pending watcher to be closed")
						}
					} else {
						delete(pending, id)
						notified[id] = w
						t.Log("left watcher in notified state")
					}
				}

				for _, w := range canceled {
					select {
					case <-w.Channel:
						t.Fatal("did not expect canceled watcher to be notified")
					default:
					}
				}
			},
		})
	})
}
