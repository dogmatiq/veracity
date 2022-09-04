package eventstream_test

import (
	"context"
	"fmt"
	"runtime"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	. "github.com/dogmatiq/veracity/internal/eventstream"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/internal/persistence/journal"
	"github.com/dogmatiq/veracity/internal/zapx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var _ = Describe("type EventStream (parallelism)", func() {
	It("appends each event exactly once", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		stream := &EventStream{
			Journal: &journal.InMemory[*JournalRecord]{},
			Logger:  zapx.NewTesting(),
		}

		var (
			parallelism = runtime.NumCPU()
			events      = parallelism * 50
		)

		expect := map[string]*envelopespec.Envelope{}

		for i := 0; i < events; i++ {
			id := fmt.Sprintf("<id-%d>", i)
			expect[id] = NewEnvelope(id, MessageE1)
		}

		tick := func(ctx context.Context) error {
			s := &EventStream{
				Journal: stream.Journal,
				Logger:  zap.NewNop(),
			}

			for _, env := range expect {
				if err := s.Append(ctx, env); err != nil {
					return err
				}
			}

			return nil
		}

		var g errgroup.Group

		for i := 0; i < parallelism; i++ {
			g.Go(func() error {
				for {
					err := tick(ctx)
					if err == nil {
						return nil
					}
				}
			})
		}

		err := g.Wait()
		Expect(err).ShouldNot(HaveOccurred())

		actual := map[string]*envelopespec.Envelope{}
		err = stream.Range(
			ctx,
			0,
			func(
				ctx context.Context,
				env *envelopespec.Envelope,
			) (bool, error) {
				actual[env.GetMessageId()] = env
				return true, nil
			},
		)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(actual).To(Equal(expect))
	})
})
