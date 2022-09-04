package eventstream_test

import (
	"context"
	"errors"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	. "github.com/dogmatiq/veracity/internal/eventstream"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/internal/persistence/journal"
	"github.com/dogmatiq/veracity/internal/zapx"
	. "github.com/jmalloc/gomegax"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type EventStream (idempotence)", func() {
	var (
		ctx   context.Context
		journ *journal.Stub[*JournalRecord]
	)

	BeforeEach(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		journ = &journal.Stub[*JournalRecord]{
			Journal: &journal.InMemory[*JournalRecord]{},
		}
	})

	DescribeTable(
		"it acknowledges the message exactly once",
		func(setup func()) {
			setup()
			expectErr := journ.WriteFunc != nil

			expect := NewEnvelope("<event>", MessageE1)
			appended := false

			tick := func(ctx context.Context) error {
				stream := &EventStream{
					Journal: journ,
					Logger:  zapx.NewTesting(),
				}

				if !appended {
					if err := stream.Append(ctx, expect); err != nil {
						return err
					}
					appended = true
				}

				return nil
			}

			for {
				err := tick(ctx)
				if err == nil {
					break
				}

				Expect(err).To(MatchError(ContainSubstring("<error>")))
				expectErr = false
			}

			Expect(expectErr).To(BeFalse(), "process should fail at least once")

			stream := &EventStream{
				Journal: journ,
				Logger:  zapx.NewTesting(),
			}

			var envelopes []*envelopespec.Envelope
			err := stream.Range(
				ctx,
				0,
				func(
					ctx context.Context,
					env *envelopespec.Envelope,
				) (bool, error) {
					envelopes = append(envelopes, env)
					return true, nil
				},
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(envelopes).To(
				ConsistOf(EqualX(expect)),
				"event should only be written to the event stream once",
			)
		},
		Entry(
			"no faults",
			func() {},
		),
		Entry(
			"append fails before journal record is written",
			func() {
				journ.WriteFunc = func(
					ctx context.Context,
					v uint32,
					r *JournalRecord,
				) (bool, error) {
					if r.GetAppend() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}
					return journ.Journal.Write(ctx, v, r)
				}
			},
		),
		Entry(
			"append fails after journal record is written",
			func() {
				journ.WriteFunc = func(
					ctx context.Context,
					v uint32,
					r *JournalRecord,
				) (bool, error) {
					ok, err := journ.Journal.Write(ctx, v, r)
					if !ok || err != nil {
						return false, err
					}

					if r.GetAppend() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}

					return true, nil
				}
			},
		),
	)
})
