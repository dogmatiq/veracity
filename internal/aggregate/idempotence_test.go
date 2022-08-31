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
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/internal/journal"
	"github.com/dogmatiq/veracity/internal/queue"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Instance (idempotence)", func() {
	var (
		ctx      context.Context
		handler  *AggregateMessageHandler
		packer   *envelope.Packer
		commands *queue.Queue
		journ    *journal.Stub[*JournalRecord]
	)

	BeforeEach(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		handler = &AggregateMessageHandler{
			HandleCommandFunc: func(
				r dogma.AggregateRoot,
				s dogma.AggregateCommandScope,
				m dogma.Message,
			) {
				s.RecordEvent(MessageE1)
			},
		}

		packer = &envelope.Packer{
			Application: &envelopespec.Identity{
				Name: "<app-name>",
				Key:  "<app-key>",
			},
			Marshaler: Marshaler,
		}

		commands = &queue.Queue{
			Journal: &journal.InMemory[*queue.JournalRecord]{},
		}

		journ = &journal.Stub[*JournalRecord]{
			Journal: &journal.InMemory[*JournalRecord]{},
		}
	})

	DescribeTable(
		"it handles a command exactly once",
		func(setup func()) {
			setup()
			expectErr := journ.WriteFunc != nil

			err := commands.Enqueue(ctx, NewEnvelope("<command>", MessageC1))
			Expect(err).ShouldNot(HaveOccurred())

			tick := func(ctx context.Context) error {
				inst := &Instance{
					HandlerIdentity: &envelopespec.Identity{
						Name: "<handler-name>",
						Key:  "<handler-key>",
					},
					InstanceID:   "<instance>",
					Handler:      handler,
					Packer:       packer,
					Journal:      journ,
					Acknowledger: commands,
					Root:         handler.New(),
				}

				env, ok, err := commands.Acquire(ctx)
				if !ok || err != nil {
					return err
				}

				return inst.ExecuteCommand(ctx, env)
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

			_, ok, err := commands.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse(), "command should be acknowledged")
		},
		Entry(
			"no faults",
			func() {},
		),
		// Entry(
		// 	"revision fails before journal record is written",
		// 	func() {
		// 		journ.WriteFunc = func(
		// 			ctx context.Context,
		// 			ver uint64,
		// 			rec *JournalRecord,
		// 		) (bool, error) {
		// 			if rec.GetRevision() != nil {
		// 				journ.WriteFunc = nil
		// 				return false, errors.New("<error>")
		// 			}
		// 			return journ.Journal.Write(ctx, ver, rec)
		// 		}
		// 	},
		// ),
		// Entry(
		// 	"revision fails after journal record is written",
		// 	func() {
		// 		journ.WriteFunc = func(
		// 			ctx context.Context,
		// 			ver uint64,
		// 			rec *JournalRecord,
		// 		) (bool, error) {
		// 			ok, err := journ.Journal.Write(ctx, ver, rec)
		// 			if !ok || err != nil {
		// 				return false, err
		// 			}

		// 			if rec.GetRevision() != nil {
		// 				journ.WriteFunc = nil
		// 				return false, errors.New("<error>")
		// 			}

		// 			return true, nil
		// 		}
		// 	},
		// ),
	)
})
