package queue_test

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	. "github.com/dogmatiq/dogma/fixtures"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	. "github.com/dogmatiq/veracity/queue"
	"github.com/dogmatiq/veracity/queue/memory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/sync/errgroup"
)

var _ = Describe("type Queue (parallelism)", func() {
	It("acknowledges each message exactly once", func() {
		journal := &memory.Journal{}

		queue := &Queue{
			Journal: journal,
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

			m := NewParcel(id, MessageM1)
			err := queue.Enqueue(context.Background(), m)
			Expect(err).ShouldNot(HaveOccurred())
		}

		tick := func(ctx context.Context) (bool, error) {
			queue := &Queue{
				Journal: journal,
			}

			m, ok, err := queue.Acquire(ctx)
			if err != nil {
				return false, err
			}
			if !ok {
				return true, nil
			}

			if err = queue.Nack(ctx, m.ID()); err != nil {
				return false, err
			}

			m, ok, err = queue.Acquire(ctx)
			if err != nil {
				return false, err
			}
			if !ok {
				return true, nil
			}

			if err = queue.Ack(ctx, m.ID()); err != nil {
				return false, err
			}

			mutex.Lock()
			actual[m.ID()] = struct{}{}
			mutex.Unlock()

			return false, nil
		}

		var g errgroup.Group

		for i := 0; i < parallelism; i++ {
			g.Go(func() error {
				for {
					done, err := tick(context.Background())
					if err == ErrConflict {
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
