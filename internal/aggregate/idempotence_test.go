package aggregate_test

import (
	"context"
	"errors"
	"time"

	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	. "github.com/dogmatiq/marshalkit/fixtures"
	. "github.com/dogmatiq/veracity/internal/aggregate"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/eventstream"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/internal/zapx"
	"github.com/dogmatiq/veracity/journal"
	"github.com/dogmatiq/veracity/journal/journaltest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type CommandExecutor (idempotence)", func() {
	var (
		ctx             context.Context
		instanceJournal *journaltest.JournalStub[*JournalRecord]
		eventsJournal   *journaltest.JournalStub[*eventstream.JournalRecord]
		packer          *envelope.Packer
	)

	BeforeEach(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		instanceJournal = &journaltest.JournalStub[*JournalRecord]{
			Journal: &journal.InMemory[*JournalRecord]{},
		}

		eventsJournal = &journaltest.JournalStub[*eventstream.JournalRecord]{
			Journal: &journal.InMemory[*eventstream.JournalRecord]{},
		}

		packer = envelope.NewTestPacker()
	})

	DescribeTable(
		"it handles a command exactly once",
		func(setup func()) {
			setup()
			expectErr := instanceJournal.WriteFunc != nil || eventsJournal.WriteFunc != nil

			env := NewEnvelope("<command>", MessageC1)

			tick := func(ctx context.Context) error {
				exec := &CommandExecutor{
					HandlerIdentity: &envelopespec.Identity{
						Name: "<handler-name>",
						Key:  "<handler-key>",
					},
					Handler: &AggregateMessageHandler{
						HandleCommandFunc: func(
							r dogma.AggregateRoot,
							s dogma.AggregateCommandScope,
							m dogma.Message,
						) {
							s.RecordEvent(MessageE1)
						},
					},
					Packer: packer,
					JournalOpener: &journaltest.OpenerStub[*JournalRecord]{
						Opener: &journal.InMemoryOpener[*JournalRecord]{},
						OpenJournalFunc: func(
							ctx context.Context,
							id string,
						) (journal.Journal[*JournalRecord], error) {
							Expect(id).To(Equal("<instance-id>"))
							return instanceJournal, nil
						},
					},
					EventAppender: &eventstream.EventStream{
						Journal: eventsJournal,
						Logger:  zapx.NewTesting(),
					},
					Logger: zapx.NewTesting(),
				}

				return exec.ExecuteCommand(
					ctx,
					"<instance-id>",
					env,
				)
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

			events := &eventstream.EventStream{
				Journal: eventsJournal,
				Logger:  zapx.NewTesting(),
			}

			var actual []*envelopespec.Envelope
			err := events.Range(
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
			Expect(actual).To(HaveLen(1))
			Expect(actual[0].MediaType).To(Equal(MessageE1Packet.MediaType))
			Expect(actual[0].Data).To(Equal(MessageE1Packet.Data))
		},
		Entry(
			"no faults",
			func() {},
		),
		Entry(
			"revision fails before journal record is written",
			func() {
				instanceJournal.WriteFunc = func(
					ctx context.Context,
					v uint32,
					r *JournalRecord,
				) (bool, error) {
					if r.GetRevision() != nil {
						instanceJournal.WriteFunc = nil
						return false, errors.New("<error>")
					}
					return instanceJournal.Journal.Write(ctx, v, r)
				}
			},
		),
		Entry(
			"revision fails after journal record is written",
			func() {
				instanceJournal.WriteFunc = func(
					ctx context.Context,
					v uint32,
					r *JournalRecord,
				) (bool, error) {
					ok, err := instanceJournal.Journal.Write(ctx, v, r)
					if !ok || err != nil {
						return false, err
					}

					if r.GetRevision() != nil {
						instanceJournal.WriteFunc = nil
						return false, errors.New("<error>")
					}

					return true, nil
				}
			},
		),
		Entry(
			"event stream append fails before journal record is written",
			func() {
				eventsJournal.WriteFunc = func(
					ctx context.Context,
					v uint32,
					r *eventstream.JournalRecord,
				) (bool, error) {
					if r.GetAppend() != nil {
						eventsJournal.WriteFunc = nil
						return false, errors.New("<error>")
					}
					return eventsJournal.Journal.Write(ctx, v, r)
				}
			},
		),
		Entry(
			"event stream append fails after journal record is written",
			func() {
				eventsJournal.WriteFunc = func(
					ctx context.Context,
					v uint32,
					r *eventstream.JournalRecord,
				) (bool, error) {
					ok, err := eventsJournal.Journal.Write(ctx, v, r)
					if !ok || err != nil {
						return false, err
					}

					if r.GetAppend() != nil {
						eventsJournal.WriteFunc = nil
						return false, errors.New("<error>")
					}

					return true, nil
				}
			},
		),
	)
})
