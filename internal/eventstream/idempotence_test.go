package eventstream_test

import (
	"context"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	. "github.com/dogmatiq/veracity/internal/eventstream"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/internal/zapx"
	"github.com/dogmatiq/veracity/journal/journaltest"
	"github.com/dogmatiq/veracity/journal/memory"
	. "github.com/jmalloc/gomegax"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type EventStream (idempotence)", func() {
	var (
		ctx     context.Context
		journal *journaltest.JournalStub[*JournalRecord]
	)

	BeforeEach(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		journal = &journaltest.JournalStub[*JournalRecord]{
			Journal: &memory.Journal[*JournalRecord]{},
		}
	})

	DescribeTable(
		"it acknowledges the message exactly once",
		func(expectErr string, setup func()) {
			setup()

			expect := NewEnvelope("<event>", MessageE1)
			appended := false

			tick := func(ctx context.Context) error {
				stream := &EventStream{
					Journal: journal,
					Logger:  zapx.NewTesting("eventstream-write"),
				}

				if !appended {
					if err := stream.Append(ctx, expect); err != nil {
						return err
					}
					appended = true
				}

				return nil
			}

			needError := expectErr != ""

			for {
				err := tick(ctx)
				if err == nil {
					break
				}

				Expect(err).To(MatchError(expectErr))
				needError = false
			}

			Expect(needError).To(BeFalse(), "process should fail with the expected error at least once")

			stream := &EventStream{
				Journal: journal,
				Logger:  zapx.NewTesting("eventstream-read"),
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
			"", // no error expected
			func() {},
		),
		Entry(
			"append fails before journal record is written",
			"unable to append event(s): <error>",
			func() {
				journaltest.FailOnceBeforeWrite(
					journal,
					func(r *JournalRecord) bool {
						return r.GetAppend() != nil
					},
				)
			},
		),
		Entry(
			"append fails after journal record is written",
			"unable to append event(s): <error>",
			func() {
				journaltest.FailOnceAfterWrite(
					journal,
					func(r *JournalRecord) bool {
						return r.GetAppend() != nil
					},
				)
			},
		),
	)
})
