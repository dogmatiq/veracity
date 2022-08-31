package eventstore_test

import (
	"context"
	"errors"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	. "github.com/dogmatiq/veracity/internal/eventstore"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/internal/persistence/journal"
	. "github.com/jmalloc/gomegax"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type EventStore (idempotence)", func() {
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

			expect := NewEnvelope("<event>", MessageE1)
			appended := false

			tick := func(ctx context.Context) error {
				store := &EventStore{
					Journal: journ,
				}

				if !appended {
					if err := store.Append(ctx, expect); err != nil {
						return err
					}
					appended = true
				}

				return nil
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

			store := &EventStore{
				Journal: journ,
			}
			env, ok, err := store.GetByOffset(ctx, 0)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue(), "event should be written to event store")
			Expect(env).To(EqualX(expect))

			_, ok, err = store.GetByOffset(ctx, 1)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse(), "event should only be written to the event store once")
		},
		Entry(
			"no faults",
			func() {},
		),
		Entry(
			"append fails before journal record is written",
			func() {
				journ.WriteFunc = func(
					ctx context.Context,
					ver uint64,
					rec *JournalRecord,
				) (bool, error) {
					if rec.GetAppend() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}
					return journ.Journal.Write(ctx, ver, rec)
				}
			},
		),
		Entry(
			"append fails after journal record is written",
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

					if rec.GetAppend() != nil {
						journ.WriteFunc = nil
						return false, errors.New("<error>")
					}

					return true, nil
				}
			},
		),
	)
})
