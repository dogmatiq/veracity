package eventstream_test

import (
	"context"
	"runtime"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/envelope"
	. "github.com/dogmatiq/veracity/internal/eventstream"
	"github.com/dogmatiq/veracity/internal/zapx"
	"github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var _ = Describe("type EventStream (parallelism)", func() {
	It("appends each event exactly once", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		packer := envelope.NewTestPacker()
		journals := &memory.JournalStore[*JournalRecord]{}

		var (
			parallelism = runtime.NumCPU()
			events      = parallelism * 50
		)

		expect := map[string]*envelopespec.Envelope{}

		for i := 0; i < events; i++ {
			env := packer.Pack(MessageE1)
			expect[env.MessageId] = env
		}

		tick := func(ctx context.Context) error {
			j, err := journals.Open(ctx, "<eventstream>")
			if err != nil {
				return err
			}
			defer j.Close()

			s := &EventStream{
				Journal: j,
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

		j, err := journals.Open(ctx, "<eventstream>")
		Expect(err).ShouldNot(HaveOccurred())
		defer j.Close()

		stream := &EventStream{
			Journal: j,
			Logger:  zapx.NewTesting("eventstream"),
		}

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
