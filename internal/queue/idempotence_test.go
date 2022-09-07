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
		"it acknowledges the message exactly once",
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
					Logger:  zapx.NewTesting(),
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

				err = queue.Reject(ctx, m)
				if err != nil {
					return err
				}

				m, ok, err = queue.Acquire(ctx)
				if !ok || err != nil {
					return err
				}

				return queue.Ack(ctx, m)
			}

			needAcks := len(messages)
			needError := expectErr != ""

			for needAcks > 0 {
				err := tick(ctx)
				if err == nil {
					needAcks--
					continue
				}

				Expect(err).To(MatchError(expectErr))
				needError = false
			}

			Expect(needError).To(BeFalse(), "process should fail with the expected error")

			queue := &Queue{
				Journal: journ,
				Logger:  zapx.NewTesting(),
			}
			_, ok, err := queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse(), "message should be acknowledged")
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
			"acknowledge fails before journal record is written",
			"unable to acknowledge message: <error>",
			func() {
				journaltest.FailOnceBeforeWrite(
					journ,
					func(r *JournalRecord) bool {
						return r.GetAck() != nil
					},
				)
			},
		),
		Entry(
			"acknowledge fails after journal record is written",
			"unable to acknowledge message: <error>",
			func() {
				journaltest.FailOnceAfterWrite(
					journ,
					func(r *JournalRecord) bool {
						return r.GetAck() != nil
					},
				)
			},
		),
		Entry(
			"reject fails before journal record is written",
			"unable to reject message: <error>",
			func() {
				journaltest.FailOnceBeforeWrite(
					journ,
					func(r *JournalRecord) bool {
						return r.GetReject() != nil
					},
				)
			},
		),
		Entry(
			"reject fails after journal record is written",
			"unable to reject message: <error>",
			func() {
				journaltest.FailOnceAfterWrite(
					journ,
					func(r *JournalRecord) bool {
						return r.GetReject() != nil
					},
				)
			},
		),
	)
})
