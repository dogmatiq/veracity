package eventstream_test

import (
	"context"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/envelope"
	. "github.com/dogmatiq/veracity/internal/eventstream"
	"github.com/dogmatiq/veracity/internal/zapx"
	"github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/jmalloc/gomegax"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type EventStream", func() {
	var (
		ctx    context.Context
		packer *envelope.Packer
		stream *EventStream
	)

	BeforeEach(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		packer = envelope.NewTestPacker()

		stream = &EventStream{
			Journal: memory.NewJournal[*JournalRecord](),
			Logger:  zapx.NewTesting("eventstream"),
		}
	})

	Describe("func Append()", func() {
		It("allows enqueuing appending messages", func() {
			expect := []*envelopespec.Envelope{
				packer.Pack(MessageE1),
				packer.Pack(MessageE2),
			}

			err := stream.Append(ctx, expect...)
			Expect(err).ShouldNot(HaveOccurred())

			var actual []*envelopespec.Envelope
			err = stream.Range(
				ctx,
				0,
				func(
					ctx context.Context,
					env *envelopespec.Envelope,
				) (bool, error) {
					actual = append(actual, env)
					return true, nil
				},
			)
			Expect(err).ShouldNot(HaveOccurred())

			var matchers []any
			for _, env := range expect {
				matchers = append(matchers, EqualX(env))
			}
			Expect(actual).To(ConsistOf(matchers...))
		})
	})

	Describe("func Range()", func() {
		It("seeks to the correct offset", func() {
			var expect []string

			for i := 0; i < 5; i++ {
				var envelopes []*envelopespec.Envelope

				for j := 0; j <= i; j++ {
					env := packer.Pack(MessageE1)
					envelopes = append(envelopes, env)
					expect = append(expect, env.GetMessageId())
				}

				err := stream.Append(ctx, envelopes...)
				Expect(err).ShouldNot(HaveOccurred())
			}

			for i := 0; i < len(expect); i++ {
				var actual []string

				err := stream.Range(
					ctx,
					uint64(i),
					func(
						ctx context.Context,
						env *envelopespec.Envelope,
					) (bool, error) {
						actual = append(actual, env.GetMessageId())
						return true, nil
					},
				)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(actual).To(EqualX(expect[i:]))
			}
		})
	})
})
