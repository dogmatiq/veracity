package aggregate_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	. "github.com/dogmatiq/marshalkit/fixtures"
	. "github.com/dogmatiq/veracity/internal/aggregate"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/eventstream"
	"github.com/dogmatiq/veracity/internal/zapx"
	"github.com/dogmatiq/veracity/journal"
	"github.com/dogmatiq/veracity/journal/journaltest"
	"github.com/dogmatiq/veracity/journal/memory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/sync/errgroup"
)

var _ = Describe("type CommandExecutor (idempotence)", func() {
	var (
		ctx             context.Context
		packer          *envelope.Packer
		instanceJournal *journaltest.JournalStub[*JournalRecord]
		eventsJournal   *journaltest.JournalStub[*eventstream.JournalRecord]
		journalOpener   journal.Opener[*JournalRecord]
	)

	BeforeEach(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		packer = envelope.NewTestPacker()

		instanceJournal = &journaltest.JournalStub[*JournalRecord]{
			Journal: &memory.Journal[*JournalRecord]{},
		}

		eventsJournal = &journaltest.JournalStub[*eventstream.JournalRecord]{
			Journal: &memory.Journal[*eventstream.JournalRecord]{},
		}

		journalOpener = &journaltest.OpenerStub[*JournalRecord]{
			OpenFunc: func(
				ctx context.Context,
				key string,
			) (journal.Journal[*JournalRecord], error) {
				if key != "aggregate/<handler-key>/<instance-id>" {
					return nil, fmt.Errorf("unexpected journal key: %s", key)
				}

				return journaltest.NopCloser[*JournalRecord](instanceJournal), nil
			},
		}
	})

	DescribeTable(
		"it handles a command exactly once",
		func(expectErr string, setup func()) {
			setup()

			env := packer.Pack(MessageC1)

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
					Packer:        packer,
					JournalOpener: journalOpener,
					EventAppender: &eventstream.EventStream{
						Journal: eventsJournal,
						Logger:  zapx.NewTesting("eventstream-write"),
					},
					Logger: zapx.NewTesting("<handler-name>"),
				}

				ctx, cancel := context.WithCancel(ctx)
				defer cancel()

				g, ctx := errgroup.WithContext(ctx)

				g.Go(func() error {
					err := exec.Run(ctx)

					// The context is canceled after the command is executed
					// successfully, therefore it is not an error condition.
					if errors.Is(err, context.Canceled) {
						return nil
					}

					return err
				})

				g.Go(func() error {
					err := exec.ExecuteCommand(
						ctx,
						"<instance-id>",
						env,
					)

					// If the command was executed successfully we want to stop
					// the executor.
					if err == nil {
						cancel()
					}

					// But we never report the error, because we expect the
					// errors to be returned by Run().
					return nil
				})

				return g.Wait()
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

			events := &eventstream.EventStream{
				Journal: eventsJournal,
				Logger:  zapx.NewTesting("eventstream-read"),
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
			"", // no error expected
			func() {},
		),
		Entry(
			"revision fails before journal record is written",
			"unable to record revision: <error>",
			func() {
				journaltest.FailOnceBeforeWrite(
					instanceJournal,
					func(r *JournalRecord) bool {
						return r.GetRevision() != nil
					},
				)
			},
		),
		Entry(
			"revision fails after journal record is written",
			"unable to record revision: <error>",
			func() {
				journaltest.FailOnceAfterWrite(
					instanceJournal,
					func(r *JournalRecord) bool {
						return r.GetRevision() != nil
					},
				)
			},
		),
		Entry(
			"event stream append fails before journal record is written",
			"unable to append event(s): <error>",
			func() {
				journaltest.FailOnceBeforeWrite(
					eventsJournal,
					func(r *eventstream.JournalRecord) bool {
						return r.GetAppend() != nil
					},
				)
			},
		),
		Entry(
			"event stream append fails after journal record is written",
			"unable to append event(s): <error>",
			func() {
				journaltest.FailOnceAfterWrite(
					eventsJournal,
					func(r *eventstream.JournalRecord) bool {
						return r.GetAppend() != nil
					},
				)
			},
		),
	)
})
