package queue_test

import (
	"context"
	"errors"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	. "github.com/dogmatiq/veracity/internal/queue"
	"github.com/dogmatiq/veracity/persistence/occjournal"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Queue (idempotence)", func() {
	DescribeTable(
		"it acknowledges the message exactly once",
		func(before, after func(*JournalRecord) error) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			impl := &occjournal.InMemory[*JournalRecord]{}
			journal := &occjournal.Stub[*JournalRecord]{
				Journal: impl,
				WriteFunc: func(
					ctx context.Context,
					offset uint64,
					rec *JournalRecord,
				) error {
					if before != nil {
						if err := before(rec); err != nil {
							before = nil
							return err
						}
					}

					if err := impl.Write(ctx, offset, rec); err != nil {
						return err
					}

					if after != nil {
						if err := after(rec); err != nil {
							after = nil
							return err
						}
					}

					return nil
				},
			}

			env := NewEnvelope("<message>", MessageM1)
			enqueued := false

			tick := func(ctx context.Context) error {
				queue := &Queue{
					Journal: journal,
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

			expectErr := before != nil || after != nil

		retry:
			if err := tick(ctx); err != nil {
				Expect(err).To(MatchError("<error>"))
				expectErr = false
				goto retry
			}

			Expect(expectErr).To(BeFalse(), "process should fail at least once")

			queue := &Queue{
				Journal: journal,
			}
			_, ok, err := queue.Acquire(context.Background())
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse(), "message should be acknowledged")
		},
		Entry(
			"no faults",
			nil,
			nil,
		),
		Entry(
			"enqueue fails before journal record is written",
			func(rec *JournalRecord) error {
				if rec.GetEnqueue() != nil {
					return errors.New("<error>")
				}
				return nil
			},
			nil,
		),
		Entry(
			"enqueue fails after journal record is written",
			nil,
			func(rec *JournalRecord) error {
				if rec.GetEnqueue() != nil {
					return errors.New("<error>")
				}
				return nil
			},
		),
		Entry(
			"acquire fails before journal record is written",
			func(rec *JournalRecord) error {
				if rec.GetAcquire() != "" {
					return errors.New("<error>")
				}
				return nil
			},
			nil,
		),
		Entry(
			"acquire fails after journal record is written",
			nil,
			func(rec *JournalRecord) error {
				if rec.GetAcquire() != "" {
					return errors.New("<error>")
				}
				return nil
			},
		),
		Entry(
			"ack fails before journal record is written",
			func(rec *JournalRecord) error {
				if rec.GetAck() != "" {
					return errors.New("<error>")
				}
				return nil
			},
			nil,
		),
		Entry(
			"ack fails after journal record is written",
			nil,
			func(rec *JournalRecord) error {
				if rec.GetAck() != "" {
					return errors.New("<error>")
				}
				return nil
			},
		),
		Entry(
			"nack fails before journal record is written",
			func(rec *JournalRecord) error {
				if rec.GetNack() != "" {
					return errors.New("<error>")
				}
				return nil
			},
			nil,
		),
		Entry(
			"nack fails after journal record is written",
			nil,
			func(rec *JournalRecord) error {
				if rec.GetNack() != "" {
					return errors.New("<error>")
				}
				return nil
			},
		),
	)
})
