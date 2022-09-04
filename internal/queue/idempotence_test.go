package queue_test

import (
	"context"
	"errors"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/internal/logging"
	"github.com/dogmatiq/veracity/internal/persistence/journal"
	. "github.com/dogmatiq/veracity/internal/queue"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

			envelopes := []*envelopespec.Envelope{
				NewEnvelope("<message-1>", MessageM1),
				NewEnvelope("<message-2>", MessageM2),
			}
			enqueued := false

			tick := func(ctx context.Context) error {
				queue := &Queue{
					Journal: journ,
					Logger:  logging.NewTesting(),
				}

				if !enqueued {
					if err := queue.Enqueue(ctx, envelopes...); err != nil {
						return err
					}
					enqueued = true
				}

				env, ok, err := queue.Acquire(ctx)
				if !ok || err != nil {
					return err
				}

				err = queue.Reject(ctx, env.GetMessageId())
				if err != nil {
					return err
				}

				env, ok, err = queue.Acquire(ctx)
				if !ok || err != nil {
					return err
				}

				return queue.Ack(ctx, env.GetMessageId())
			}

			acks := 0
			for acks < len(envelopes) {
				err := tick(ctx)
				if err == nil {
					acks++
					continue
				}

				Expect(err).To(MatchError(ContainSubstring("<error>")))
				expectErr = false
			}

			Expect(expectErr).To(BeFalse(), "process should fail at least once")

			queue := &Queue{
				Journal: journ,
				Logger:  logging.NewTesting(),
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
					v uint32,
					r *JournalRecord,
				) (bool, error) {
					if r.GetEnqueue() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}
					return journ.Journal.Write(ctx, v, r)
				}
			},
		),
		Entry(
			"enqueue fails after journal record is written",
			func() {
				journ.WriteFunc = func(
					ctx context.Context,
					v uint32,
					r *JournalRecord,
				) (bool, error) {
					ok, err := journ.Journal.Write(ctx, v, r)
					if !ok || err != nil {
						return false, err
					}

					if r.GetEnqueue() != nil {
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
					v uint32,
					r *JournalRecord,
				) (bool, error) {
					if r.GetAcquire() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}
					return journ.Journal.Write(ctx, v, r)
				}
			},
		),
		Entry(
			"acquire fails after journal record is written",
			func() {
				journ.WriteFunc = func(
					ctx context.Context,
					v uint32,
					r *JournalRecord,
				) (bool, error) {
					ok, err := journ.Journal.Write(ctx, v, r)
					if !ok || err != nil {
						return false, err
					}

					if r.GetAcquire() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}

					return true, nil
				}
			},
		),
		Entry(
			"acknowledge fails before journal record is written",
			func() {
				journ.WriteFunc = func(
					ctx context.Context,
					v uint32,
					r *JournalRecord,
				) (bool, error) {
					if r.GetAck() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}
					return journ.Journal.Write(ctx, v, r)
				}
			},
		),
		Entry(
			"acknowledge fails after journal record is written",
			func() {
				journ.WriteFunc = func(
					ctx context.Context,
					v uint32,
					r *JournalRecord,
				) (bool, error) {
					ok, err := journ.Journal.Write(ctx, v, r)
					if !ok || err != nil {
						return false, err
					}

					if r.GetAck() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}

					return true, nil
				}
			},
		),
		Entry(
			"reject fails before journal record is written",
			func() {
				journ.WriteFunc = func(
					ctx context.Context,
					v uint32,
					r *JournalRecord,
				) (bool, error) {
					if r.GetReject() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}
					return journ.Journal.Write(ctx, v, r)
				}
			},
		),
		Entry(
			"reject fails after journal record is written",
			func() {
				journ.WriteFunc = func(
					ctx context.Context,
					v uint32,
					r *JournalRecord,
				) (bool, error) {
					ok, err := journ.Journal.Write(ctx, v, r)
					if !ok || err != nil {
						return false, err
					}

					if r.GetReject() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}

					return true, nil
				}
			},
		),
	)
})
