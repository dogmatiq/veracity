package queue_test

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/internal/persistence/journal"
	. "github.com/dogmatiq/veracity/internal/queue"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var _ = Describe("type Queue (parallelism)", func() {
	It("acknowledges each message exactly once", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		queue := &Queue{
			Journal: &journal.InMemory[*JournalRecord]{},
			Logger:  zap.NewExample(),
		}

		var (
			parallelism = runtime.NumCPU()
			messages    = parallelism * 50
		)

		var mutex sync.Mutex
		actual := map[string]int{}
		expect := map[string]int{}
		var envelopes []*envelopespec.Envelope

		for i := 0; i < messages; i++ {
			id := fmt.Sprintf("<id-%d>", i)
			expect[id] = 1
			envelopes = append(envelopes, NewEnvelope(id, MessageM1))
		}

		tick := func(ctx context.Context) (bool, error) {
			q := &Queue{
				Journal: queue.Journal,
				Logger:  zap.NewNop(),
			}

			for _, env := range envelopes {
				if err := q.Enqueue(ctx, env); err != nil {
					return false, err
				}
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
			actual[env.GetMessageId()]++
			mutex.Unlock()

			return false, nil
		}

		var g errgroup.Group

		for i := 0; i < parallelism; i++ {
			g.Go(func() error {
				for {
					done, err := tick(ctx)
					if err != nil {
						continue
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
