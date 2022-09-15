package queue_test

import (
	"context"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/journaltest"
	. "github.com/dogmatiq/veracity/internal/queue"
	"github.com/dogmatiq/veracity/internal/zapx"
	"github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Queue (idempotence)", func() {
	var (
		ctx      context.Context
		packer   *envelope.Packer
		journals *memory.JournalStore
	)

	BeforeEach(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		packer = envelope.NewTestPacker()
		journals = &memory.JournalStore{}
	})

	DescribeTable(
		"it eventually removes each message",
		func(
			expectErr string,
			setup func(*journaltest.JournalStub),
		) {
			messages := []Message{
				{
					Envelope: packer.Pack(MessageM1),
				},
				{
					Envelope: packer.Pack(MessageM2),
				},
			}
			enqueued := false

			tick := func(
				ctx context.Context,
				setup func(*journaltest.JournalStub),
			) error {
				j, err := journals.Open(ctx, "<queue>")
				if err != nil {
					return err
				}
				defer j.Close()

				stub := &journaltest.JournalStub{
					Journal: j,
				}

				setup(stub)

				queue := &Queue{
					Journal: stub,
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
				err := tick(ctx, setup)
				if err == nil {
					remaining--
					continue
				}

				Expect(err).To(MatchError(expectErr))
				needError = false
				setup = func(j *journaltest.JournalStub) {}
			}

			Expect(needError).To(BeFalse(), "process should fail with the expected error at least once")

			j, err := journals.Open(ctx, "<queue>")
			Expect(err).ShouldNot(HaveOccurred())
			defer j.Close()

			queue := &Queue{
				Journal: j,
				Logger:  zapx.NewTesting("queue-read"),
			}
			_, ok, err := queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse(), "message should be removed from the queue")
		},
		Entry(
			"no faults",
			"", // no error expected
			func(stub *journaltest.JournalStub) {},
		),
		Entry(
			"enqueue fails before journal record is written",
			"unable to enqueue message(s): <error>",
			func(stub *journaltest.JournalStub) {
				journaltest.FailBeforeWrite(
					stub,
					func(r *JournalRecord) bool {
						return r.GetEnqueue() != nil
					},
				)
			},
		),
		Entry(
			"enqueue fails after journal record is written",
			"unable to enqueue message(s): <error>",
			func(stub *journaltest.JournalStub) {
				journaltest.FailAfterWrite(
					stub,
					func(r *JournalRecord) bool {
						return r.GetEnqueue() != nil
					},
				)
			},
		),
		Entry(
			"acquire fails before journal record is written",
			"unable to acquire message: <error>",
			func(stub *journaltest.JournalStub) {
				journaltest.FailBeforeWrite(
					stub,
					func(r *JournalRecord) bool {
						return r.GetAcquire() != nil
					},
				)
			},
		),
		Entry(
			"acquire fails after journal record is written",
			"unable to acquire message: <error>",
			func(stub *journaltest.JournalStub) {
				journaltest.FailAfterWrite(
					stub,
					func(r *JournalRecord) bool {
						return r.GetAcquire() != nil
					},
				)
			},
		),
		Entry(
			"release fails before journal record is written",
			"unable to release message: <error>",
			func(stub *journaltest.JournalStub) {
				journaltest.FailBeforeWrite(
					stub,
					func(r *JournalRecord) bool {
						return r.GetRelease() != nil
					},
				)
			},
		),
		Entry(
			"release fails after journal record is written",
			"unable to release message: <error>",
			func(stub *journaltest.JournalStub) {
				journaltest.FailAfterWrite(
					stub,
					func(r *JournalRecord) bool {
						return r.GetRelease() != nil
					},
				)
			},
		),
		Entry(
			"remove fails before journal record is written",
			"unable to remove message: <error>",
			func(stub *journaltest.JournalStub) {
				journaltest.FailBeforeWrite(
					stub,
					func(r *JournalRecord) bool {
						return r.GetRemove() != nil
					},
				)
			},
		),
		Entry(
			"remove fails after journal record is written",
			"unable to remove message: <error>",
			func(stub *journaltest.JournalStub) {
				journaltest.FailAfterWrite(
					stub,
					func(r *JournalRecord) bool {
						return r.GetRemove() != nil
					},
				)
			},
		),
	)
})
