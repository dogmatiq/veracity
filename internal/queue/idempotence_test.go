package queue_test

import (
	"context"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	. "github.com/dogmatiq/veracity/internal/queue"
	"github.com/dogmatiq/veracity/internal/zapx"
	"github.com/dogmatiq/veracity/journal"
	"github.com/dogmatiq/veracity/journal/journaltest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Queue (idempotence)", func() {
	var (
		ctx   context.Context
		journ *journaltest.JournalStub[*JournalRecord]
	)

	BeforeEach(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		journ = &journaltest.JournalStub[*JournalRecord]{
			Journal: &journal.InMemory[*JournalRecord]{},
		}
	})

	DescribeTable(
		"it eventually removes each message",
		func(expectErr string, setup func()) {
			setup()

			messages := []Message{
				{
					Envelope: NewEnvelope("<message-1>", MessageM1),
				},
				{
					Envelope: NewEnvelope("<message-2>", MessageM2),
				},
			}
			enqueued := false

			tick := func(ctx context.Context) error {
				queue := &Queue{
					Journal: journ,
					Logger:  zapx.NewTesting("queue-write"),
				}

				if !enqueued {
					if err := queue.Enqueue(ctx, messages...); err != nil {
						return err
					}
					enqueued = true
				}

				m, ok, err := queue.Acquire(ctx)
				if !ok || err != nil {
					return err
				}

				err = queue.Release(ctx, m)
				if err != nil {
					return err
				}

				m, ok, err = queue.Acquire(ctx)
				if !ok || err != nil {
					return err
				}

				return queue.Remove(ctx, m)
			}

			remaining := len(messages)
			needError := expectErr != ""

			for remaining > 0 {
				err := tick(ctx)
				if err == nil {
					remaining--
					continue
				}

				Expect(err).To(MatchError(expectErr))
				needError = false
			}

			Expect(needError).To(BeFalse(), "process should fail with the expected error at least once")

			queue := &Queue{
				Journal: journ,
				Logger:  zapx.NewTesting("queue-read"),
			}
			_, ok, err := queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse(), "message should be removed from the queue")
		},
		Entry(
			"no faults",
			"", // no error expected
			func() {},
		),
		Entry(
			"enqueue fails before journal record is written",
			"unable to enqueue message(s): <error>",
			func() {
				journaltest.FailOnceBeforeWrite(
					journ,
					func(r *JournalRecord) bool {
						return r.GetEnqueue() != nil
					},
				)
			},
		),
		Entry(
			"enqueue fails after journal record is written",
			"unable to enqueue message(s): <error>",
			func() {
				journaltest.FailOnceAfterWrite(
					journ,
					func(r *JournalRecord) bool {
						return r.GetEnqueue() != nil
					},
				)
			},
		),
		Entry(
			"acquire fails before journal record is written",
			"unable to acquire message: <error>",
			func() {
				journaltest.FailOnceBeforeWrite(
					journ,
					func(r *JournalRecord) bool {
						return r.GetAcquire() != nil
					},
				)
			},
		),
		Entry(
			"acquire fails after journal record is written",
			"unable to acquire message: <error>",
			func() {
				journaltest.FailOnceAfterWrite(
					journ,
					func(r *JournalRecord) bool {
						return r.GetAcquire() != nil
					},
				)
			},
		),
		Entry(
			"release fails before journal record is written",
			"unable to release message: <error>",
			func() {
				journaltest.FailOnceBeforeWrite(
					journ,
					func(r *JournalRecord) bool {
						return r.GetRelease() != nil
					},
				)
			},
		),
		Entry(
			"release fails after journal record is written",
			"unable to release message: <error>",
			func() {
				journaltest.FailOnceAfterWrite(
					journ,
					func(r *JournalRecord) bool {
						return r.GetRelease() != nil
					},
				)
			},
		),
		Entry(
			"remove fails before journal record is written",
			"unable to remove message: <error>",
			func() {
				journaltest.FailOnceBeforeWrite(
					journ,
					func(r *JournalRecord) bool {
						return r.GetRemove() != nil
					},
				)
			},
		),
		Entry(
			"remove fails after journal record is written",
			"unable to remove message: <error>",
			func() {
				journaltest.FailOnceAfterWrite(
					journ,
					func(r *JournalRecord) bool {
						return r.GetRemove() != nil
					},
				)
			},
		),
	)
})
