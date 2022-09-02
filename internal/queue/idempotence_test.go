package queue_test

import (
	"context"
	"errors"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/internal/persistence/journal"
	. "github.com/dogmatiq/veracity/internal/queue"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("type Queue (idempotence)", func() {
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

			env := NewEnvelope("<message>", MessageM1)
			enqueued := false

			tick := func(ctx context.Context) error {
				queue := &Queue{
					Journal: journ,
					Logger:  zap.NewExample(),
				}

				if !enqueued {
					if err := queue.Enqueue(ctx, env); err != nil {
						return err
					}
					enqueued = true
				}

				env, ok, err := queue.Acquire(ctx)
				if !ok || err != nil {
					return err
				}

				err = queue.Nack(ctx, env.GetMessageId())
				if err != nil {
					return err
				}

				env, ok, err = queue.Acquire(ctx)
				if !ok || err != nil {
					return err
				}

				return queue.Ack(ctx, env.GetMessageId())
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

			queue := &Queue{
				Journal: journ,
				Logger:  zap.NewExample(),
			}
			_, ok, err := queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse(), "message should be acknowledged")
		},
		Entry(
			"no faults",
			func() {},
		),
		Entry(
			"enqueue fails before journal record is written",
			func() {
				journ.WriteFunc = func(
					ctx context.Context,
					ver uint64,
					rec *JournalRecord,
				) (bool, error) {
					if rec.GetEnqueue() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}
					return journ.Journal.Write(ctx, ver, rec)
				}
			},
		),
		Entry(
			"enqueue fails after journal record is written",
			func() {
				journ.WriteFunc = func(
					ctx context.Context,
					ver uint64,
					rec *JournalRecord,
				) (bool, error) {
					ok, err := journ.Journal.Write(ctx, ver, rec)
					if !ok || err != nil {
						return false, err
					}

					if rec.GetEnqueue() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}

					return true, nil
				}
			},
		),
		Entry(
			"acquire fails before journal record is written",
			func() {
				journ.WriteFunc = func(
					ctx context.Context,
					ver uint64,
					rec *JournalRecord,
				) (bool, error) {
					if rec.GetAcquire() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}
					return journ.Journal.Write(ctx, ver, rec)
				}
			},
		),
		Entry(
			"acquire fails after journal record is written",
			func() {
				journ.WriteFunc = func(
					ctx context.Context,
					ver uint64,
					rec *JournalRecord,
				) (bool, error) {
					ok, err := journ.Journal.Write(ctx, ver, rec)
					if !ok || err != nil {
						return false, err
					}

					if rec.GetAcquire() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}

					return true, nil
				}
			},
		),
		Entry(
			"ack fails before journal record is written",
			func() {
				journ.WriteFunc = func(
					ctx context.Context,
					ver uint64,
					rec *JournalRecord,
				) (bool, error) {
					if rec.GetAck() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}
					return journ.Journal.Write(ctx, ver, rec)
				}
			},
		),
		Entry(
			"ack fails after journal record is written",
			func() {
				journ.WriteFunc = func(
					ctx context.Context,
					ver uint64,
					rec *JournalRecord,
				) (bool, error) {
					ok, err := journ.Journal.Write(ctx, ver, rec)
					if !ok || err != nil {
						return false, err
					}

					if rec.GetAck() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}

					return true, nil
				}
			},
		),
		Entry(
			"nack fails before journal record is written",
			func() {
				journ.WriteFunc = func(
					ctx context.Context,
					ver uint64,
					rec *JournalRecord,
				) (bool, error) {
					if rec.GetNack() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}
					return journ.Journal.Write(ctx, ver, rec)
				}
			},
		),
		Entry(
			"nack fails after journal record is written",
			func() {
				journ.WriteFunc = func(
					ctx context.Context,
					ver uint64,
					rec *JournalRecord,
				) (bool, error) {
					ok, err := journ.Journal.Write(ctx, ver, rec)
					if !ok || err != nil {
						return false, err
					}

					if rec.GetNack() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}

					return true, nil
				}
			},
		),
	)
})
