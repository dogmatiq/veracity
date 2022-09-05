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
	"github.com/dogmatiq/veracity/internal/persistence/journal"
	"github.com/dogmatiq/veracity/internal/zapx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type CommandExecutor (idempotence)", func() {
	var (
		ctx    context.Context
		journ  *journal.Stub[*JournalRecord]
		events *eventstream.EventStream
		packer *envelope.Packer
	)

	BeforeEach(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		journ = &journal.Stub[*JournalRecord]{
			Journal: &journal.InMemory[*JournalRecord]{},
		}

		events = &eventstream.EventStream{
			Journal: &journal.InMemory[*eventstream.JournalRecord]{},
			Logger:  zapx.NewTesting(),
		}

		packer = envelope.NewTestPacker()
	})

	DescribeTable(
		"it handles a command exactly once",
		func(setup func()) {
			setup()
			expectErr := journ.WriteFunc != nil

			env := NewEnvelope("<command>", MessageC1)

			tick := func(ctx context.Context) error {
				exec := &CommandExecutor{
					Supervisors: map[string]*HandlerSupervisor{
						"<handler-key>": {
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
							JournalOpener: &journal.OpenerStub[*JournalRecord]{
								Opener: &journal.InMemoryOpener[*JournalRecord]{},
								OpenJournalFunc: func(
									ctx context.Context,
									id string,
								) (journal.Journal[*JournalRecord], error) {
									Expect(id).To(Equal("<instance-id>"))
									return journ, nil
								},
							},
							EventAppender: events,
							Logger:        zapx.NewTesting(),
						},
					},
				}

				return exec.ExecuteCommand(
					ctx,
					"<handler-key>",
					"<instance-id>",
					env,
				)
			}

			for {
				err := tick(ctx)
				if err == nil {
					break
				}

				Expect(err).To(MatchError("<error>"))
				expectErr = false
			}

			Expect(expectErr).To(BeFalse(), "process should fail at least once")

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
				journ.WriteFunc = func(
					ctx context.Context,
					v uint32,
					r *JournalRecord,
				) (bool, error) {
					if r.GetRevision() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}
					return journ.Journal.Write(ctx, v, r)
				}
			},
		),
		FEntry(
			"revision fails after journal record is written",
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

					if r.GetRevision() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}

					return true, nil
				}
			},
		),
	)
})
