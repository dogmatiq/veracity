package aggregate_test

import (
	"context"
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
	. "github.com/jmalloc/gomegax"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type CommandExecutor (idempotence)", func() {
	var (
		ctx        context.Context
		events     *eventstream.EventStream
		handler    *AggregateMessageHandler
		packer     *envelope.Packer
		supervisor *HandlerSupervisor
	)

	BeforeEach(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		events = &eventstream.EventStream{
			Journal: &journal.InMemory[*eventstream.JournalRecord]{},
			Logger:  zapx.NewTesting(),
		}

		handler = &AggregateMessageHandler{
			HandleCommandFunc: func(
				r dogma.AggregateRoot,
				s dogma.AggregateCommandScope,
				m dogma.Message,
			) {
				s.RecordEvent(MessageE1)
			},
		}

		packer = envelope.NewTestPacker()

		supervisor = &HandlerSupervisor{
			HandlerIdentity: &envelopespec.Identity{
				Name: "<handler-name>",
				Key:  "<handler-key>",
			},
			Handler:       handler,
			Packer:        packer,
			EventAppender: events,
			Logger:        zapx.NewTesting(),
		}
	})

	DescribeTable(
		"it handles a command exactly once",
		func(setup func()) {
			setup()
			expectErr := false //journ.WriteFunc != nil

			env := NewEnvelope("<command>", MessageC1)

			tick := func(ctx context.Context) error {
				exec := &CommandExecutor{
					Supervisors: map[string]*HandlerSupervisor{
						"<handler-key>": supervisor,
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
			Expect(actual).To(EqualX(
				[]*envelopespec.Envelope{
					{
						MessageId:     "0",
						CausationId:   "<command>",
						CorrelationId: "<correlation>",
						SourceSite: &envelopespec.Identity{
							Name: "<site-name>",
							Key:  "<site-key>",
						},
						SourceApplication: &envelopespec.Identity{
							Name: "<app-name>",
							Key:  "<app-key>",
						},
						SourceHandler: &envelopespec.Identity{
							Name: "<handler-name>",
							Key:  "<handler-key>",
						},
						SourceInstanceId: "<instance-id>",
						CreatedAt:        "2000-01-01T00:00:00Z",
						Description:      "{E1}",
						PortableName:     MessageEPortableName,
						MediaType:        MessageE1Packet.MediaType,
						Data:             MessageE1Packet.Data,
					},
				},
			))
		},
		Entry(
			"no faults",
			func() {},
		),
	// 		// Entry(
	// 		// 	"revision fails before journal record is written",
	// 		// 	func() {
	// 		// 		journ.WriteFunc = func(
	// 		// 			ctx context.Context,
	// 		// 			ver uint64,
	// 		// 			rec *JournalRecord,
	// 		// 		) (bool, error) {
	// 		// 			if rec.GetRevision() != nil {
	// 		// 				journ.WriteFunc = nil
	// 		// 				return false, errors.New("<error>")
	// 		// 			}
	// 		// 			return journ.Journal.Write(ctx, ver, rec)
	// 		// 		}
	// 		// 	},
	// 		// ),
	// 		// Entry(
	// 		// 	"revision fails after journal record is written",
	// 		// 	func() {
	// 		// 		journ.WriteFunc = func(
	// 		// 			ctx context.Context,
	// 		// 			ver uint64,
	// 		// 			rec *JournalRecord,
	// 		// 		) (bool, error) {
	// 		// 			ok, err := journ.Journal.Write(ctx, ver, rec)
	// 		// 			if !ok || err != nil {
	// 		// 				return false, err
	// 		// 			}

	// 		// 			if rec.GetRevision() != nil {
	// 		// 				journ.WriteFunc = nil
	// 		// 				return false, errors.New("<error>")
	// 		// 			}

	//	// 			return true, nil
	//	// 		}
	//	// 	},
	//	// ),
	//
	)
})
