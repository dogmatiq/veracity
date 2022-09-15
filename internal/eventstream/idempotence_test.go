package eventstream_test

import (
	"context"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/envelope"
	. "github.com/dogmatiq/veracity/internal/eventstream"
	"github.com/dogmatiq/veracity/internal/journaltest"
	"github.com/dogmatiq/veracity/internal/zapx"
	"github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/jmalloc/gomegax"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type EventStream (idempotence)", func() {
	var (
		ctx      context.Context
		packer   *envelope.Packer
		journals *memory.JournalStore
	)

	BeforeEach(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		packer = envelope.NewTestPacker()
		journals = &memory.JournalStore{}
	})

	DescribeTable(
		"it acknowledges the message exactly once",
		func(
			expectErr string,
			setup func(stub *journaltest.JournalStub),
		) {
			expect := packer.Pack(MessageE1)
			appended := false

			tick := func(
				ctx context.Context,
				setup func(*journaltest.JournalStub),
			) error {
				j, err := journals.Open(ctx, "<eventstream>")
				if err != nil {
					return err
				}
				defer j.Close()

				stub := &journaltest.JournalStub{
					Journal: j,
				}

				setup(stub)

				stream := &EventStream{
					Journal: stub,
					Logger:  zapx.NewTesting("eventstream-append"),
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
				err := tick(ctx, setup)
				if err == nil {
					break
				}

				Expect(err).To(MatchError(expectErr))
				needError = false
				setup = func(stub *journaltest.JournalStub) {}
			}

			Expect(needError).To(BeFalse(), "process should fail with the expected error at least once")

			j, err := journals.Open(ctx, "<eventstream>")
			Expect(err).ShouldNot(HaveOccurred())
			defer j.Close()

			stream := &EventStream{
				Journal: j,
				Logger:  zapx.NewTesting("eventstream-read"),
			}

			var envelopes []*envelopespec.Envelope
			err = stream.Range(
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
			func(stub *journaltest.JournalStub) {},
		),
		Entry(
			"append fails before journal record is written",
			"unable to append event(s): <error>",
			func(stub *journaltest.JournalStub) {
				journaltest.FailBeforeAppend(
					stub,
					func(r *JournalRecord) bool {
						return r.GetAppend() != nil
					},
				)
			},
		),
		Entry(
			"append fails after journal record is written",
			"unable to append event(s): <error>",
			func(stub *journaltest.JournalStub) {
				journaltest.FailAfterAppend(
					stub,
					func(r *JournalRecord) bool {
						return r.GetAppend() != nil
					},
				)
			},
		),
	)
})
