package queue_test

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/internal/occjournal"
	. "github.com/dogmatiq/veracity/internal/queue"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/sync/errgroup"
)

var _ = Describe("type Queue (parallelism)", func() {
	It("acknowledges each message exactly once", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		queue := &Queue{
			Journal: &occjournal.InMemory[*JournalRecord]{},
		}

		var (
			parallelism = runtime.NumCPU()
			messages    = parallelism * 50
		)

		var mutex sync.Mutex
		actual := map[string]struct{}{}
		expect := map[string]struct{}{}

		for i := 0; i < messages; i++ {
			id := fmt.Sprintf("<id-%d>", i)
			expect[id] = struct{}{}

			env := NewEnvelope(id, MessageM1)
			err := queue.Enqueue(ctx, env)
			Expect(err).ShouldNot(HaveOccurred())
		}

		tick := func(ctx context.Context) (bool, error) {
			q := &Queue{
				Journal: queue.Journal,
			}

			env, ok, err := q.Acquire(ctx)
			if err != nil {
				return false, err
			}
			if !ok {
				return true, nil
			}

			if err = q.Nack(ctx, env.GetMessageId()); err != nil {
				return false, err
			}

			env, ok, err = q.Acquire(ctx)
			if err != nil {
				return false, err
			}
			if !ok {
				return true, nil
			}

			if err = q.Ack(ctx, env.GetMessageId()); err != nil {
				return false, err
			}

			mutex.Lock()
			actual[env.GetMessageId()] = struct{}{}
			mutex.Unlock()

			return false, nil
		}

		var g errgroup.Group

		for i := 0; i < parallelism; i++ {
			g.Go(func() error {
				for {
					done, err := tick(ctx)
					if err == occjournal.ErrConflict {
						continue
					} else if err != nil {
						return err
					} else if done {
						return nil
					}
				}
			})
		}

		err := g.Wait()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(actual).To(Equal(expect))
	})
})
