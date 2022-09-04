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
	"github.com/dogmatiq/veracity/internal/zapx"
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
			Logger:  zapx.NewTesting(),
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

		tick := func(ctx context.Context) error {
			q := &Queue{
				Journal: queue.Journal,
				Logger:  zap.NewNop(),
			}

			for _, env := range envelopes {
				if err := q.Enqueue(
					ctx,
					Message{
						Envelope: env,
					},
				); err != nil {
					return err
				}
			}

			m, ok, err := q.Acquire(ctx)
			if !ok || err != nil {
				return err
			}

			if err = q.Reject(ctx, m); err != nil {
				return err
			}

			m, ok, err = q.Acquire(ctx)
			if !ok || err != nil {
				return err
			}

			if err = q.Ack(ctx, m); err != nil {
				return err
			}

			mutex.Lock()
			actual[m.Envelope.GetMessageId()]++
			mutex.Unlock()

			return nil
		}

		var g errgroup.Group

		for i := 0; i < parallelism; i++ {
			g.Go(func() error {
				for {
					err := tick(ctx)
					if err != nil {
						continue
					}

					mutex.Lock()
					count := len(actual)
					mutex.Unlock()

					if count == len(expect) {
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
