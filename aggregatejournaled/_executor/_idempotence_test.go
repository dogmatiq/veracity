package executor_test

import (
	"context"

	. "github.com/dogmatiq/dogma/fixtures"
	. "github.com/dogmatiq/veracity/aggregatejournaled/executor"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/journal"
	"github.com/dogmatiq/veracity/queue"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Executor (idempotence)", func() {
	DescribeTable(
		"it acknowledges the message exactly once",
		func(before, after func(JournalEntry) error) {
			queue := &queue.Queue{
				Journal: &journal.InMemory[queue.JournalEntry]{},
			}

			journalStub := &journal.Stub[JournalEntry]{
				Journal: &journal.InMemory[JournalEntry]{},
			}
			journalStub.WriteFunc = func(
				ctx context.Context,
				offset uint64,
				entry JournalEntry,
			) error {
				if before != nil {
					if err := before(entry); err != nil {
						before = nil
						return err
					}
				}

				if err := journalStub.Journal.Write(ctx, offset, entry); err != nil {
					return err
				}

				if after != nil {
					if err := after(entry); err != nil {
						after = nil
						return err
					}
				}

				return nil
			}

			tick := func(ctx context.Context) error {
				executor := &Executor{
					Journal:      journalStub,
					Acknowledger: queue,
				}

				return nil
			}

			expectErr := before != nil || after != nil

		retry:
			if err := tick(context.Background()); err != nil {
				Expect(err).To(MatchError("<error>"))
				expectErr = false
				goto retry
			}

			Expect(expectErr).To(BeFalse(), "process should fail at least once")

			// queue := &Queue{
			// 	Journal: stub,
			// }
			// _, ok, err := queue.Acquire(context.Background())
			// Expect(err).ShouldNot(HaveOccurred())
			// Expect(ok).To(BeFalse(), "message should be acknowledged")
		},
		Entry(
			"no faults",
			nil,
			nil,
		),
		// Entry(
		// 	"enqueue fails before journal entry is written",
		// 	func(e JournalEntry) error {
		// 		if _, ok := e.(Enqueue); ok {
		// 			return errors.New("<error>")
		// 		}
		// 		return nil
		// 	},
		// 	nil,
		// ),
		// Entry(
		// 	"enqueue fails after journal entry is written",
		// 	nil,
		// 	func(e JournalEntry) error {
		// 		if _, ok := e.(Enqueue); ok {
		// 			return errors.New("<error>")
		// 		}
		// 		return nil
		// 	},
		// ),
	)
})
