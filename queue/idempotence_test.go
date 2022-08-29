package queue_test

import (
	"context"
	"errors"

	. "github.com/dogmatiq/dogma/fixtures"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	. "github.com/dogmatiq/veracity/queue"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Queue (idempotence)", func() {
	DescribeTable(
		"it acknowledges the message exactly once",
		func(before, after func(JournalEntry) error) {
			impl := &MemoryJournal{}
			journal := &journalStub{
				Journal: impl,
				WriteFunc: func(
					ctx context.Context,
					offset uint64,
					e JournalEntry,
				) error {
					if before != nil {
						if err := before(e); err != nil {
							before = nil
							return err
						}
					}

					if err := impl.Write(ctx, offset, e); err != nil {
						return err
					}

					if after != nil {
						if err := after(e); err != nil {
							after = nil
							return err
						}
					}

					return nil
				},
			}

			message := NewParcel("<message>", MessageM1)
			enqueued := false

			tick := func(ctx context.Context) error {
				queue := &Queue{
					Journal: journal,
				}

				if !enqueued {
					if err := queue.Enqueue(ctx, message); err != nil {
						return err
					}
					enqueued = true
				}

				m, ok, err := queue.Acquire(ctx)
				if !ok || err != nil {
					return err
				}

				err = queue.Nack(ctx, m.ID())
				if err != nil {
					return err
				}

				m, ok, err = queue.Acquire(ctx)
				if !ok || err != nil {
					return err
				}

				return queue.Ack(ctx, m.ID())
			}

			expectErr := before != nil || after != nil

		retry:
			if err := tick(context.Background()); err != nil {
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
			"enqueue fails before journal entry is written",
			func(e JournalEntry) error {
				if _, ok := e.(EnqueueEntry); ok {
					return errors.New("<error>")
				}
				return nil
			},
			nil,
		),
		Entry(
			"enqueue fails after journal entry is written",
			nil,
			func(e JournalEntry) error {
				if _, ok := e.(EnqueueEntry); ok {
					return errors.New("<error>")
				}
				return nil
			},
		),
		Entry(
			"acquire fails before journal entry is written",
			func(e JournalEntry) error {
				if _, ok := e.(AcquireEntry); ok {
					return errors.New("<error>")
				}
				return nil
			},
			nil,
		),
		Entry(
			"acquire fails after journal entry is written",
			nil,
			func(e JournalEntry) error {
				if _, ok := e.(AcquireEntry); ok {
					return errors.New("<error>")
				}
				return nil
			},
		),
		Entry(
			"ack fails before journal entry is written",
			func(e JournalEntry) error {
				if _, ok := e.(AckEntry); ok {
					return errors.New("<error>")
				}
				return nil
			},
			nil,
		),
		Entry(
			"ack fails after journal entry is written",
			nil,
			func(e JournalEntry) error {
				if _, ok := e.(AckEntry); ok {
					return errors.New("<error>")
				}
				return nil
			},
		),
		Entry(
			"nack fails before journal entry is written",
			func(e JournalEntry) error {
				if _, ok := e.(NackEntry); ok {
					return errors.New("<error>")
				}
				return nil
			},
			nil,
		),
		Entry(
			"nack fails after journal entry is written",
			nil,
			func(e JournalEntry) error {
				if _, ok := e.(NackEntry); ok {
					return errors.New("<error>")
				}
				return nil
			},
		),
	)
})
