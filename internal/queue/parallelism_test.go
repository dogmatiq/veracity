package queue_test

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/envelope"
	. "github.com/dogmatiq/veracity/internal/queue"
	"github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/jmalloc/gomegax"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var _ = Describe("type Queue (parallelism)", func() {
	It("removes each message exactly once", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		packer := envelope.NewTestPacker()
		journals := &memory.JournalStore[*JournalRecord]{}

		var (
			parallelism = runtime.NumCPU()
			messages    = parallelism * 10
		)

		var mutex sync.Mutex
		actual := map[string]int{}
		expect := map[string]int{}
		var envelopes []*envelopespec.Envelope

		for i := 0; i < messages; i++ {
			env := packer.Pack(MessageM1)
			expect[env.MessageId] = 1
			envelopes = append(envelopes, env)
		}

		tick := func(ctx context.Context) error {
			j, err := journals.Open(ctx, "<queue>")
			if err != nil {
				return err
			}
			defer j.Close()

			q := &Queue{
				Journal: j,
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

			if err = q.Release(ctx, m); err != nil {
				return err
			}

			m, ok, err = q.Acquire(ctx)
			if !ok || err != nil {
				return err
			}

			if err = q.Remove(ctx, m); err != nil {
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
						if !strings.Contains(err.Error(), "optimistic concurrency conflict") {
							return err
						}
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
		Expect(actual).To(EqualX(expect))
	})
})
