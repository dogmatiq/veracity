package queue_test

import (
	"context"
	"math/rand"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/parcel"
	"github.com/dogmatiq/veracity/persistence/occjournal"
	. "github.com/dogmatiq/veracity/queue"
	. "github.com/jmalloc/gomegax"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Queue", func() {
	var (
		ctx   context.Context
		queue *Queue
	)

	BeforeEach(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		queue = &Queue{
			Journal: &occjournal.InMemory[JournalEntry]{},
		}
	})

	It("allows acquiring messages from the queue", func() {
		expect := NewParcel("<id>", MessageM1)
		err := queue.Enqueue(ctx, expect)
		Expect(err).ShouldNot(HaveOccurred())

		actual, ok, err := queue.Acquire(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ok).To(BeTrue(), "queue should not be empty")
		Expect(actual).To(EqualX(expect))
	})

	It("does not allow acquiring a message that has already been acquired", func() {
		m := NewParcel("<id>", MessageM1)
		err := queue.Enqueue(ctx, m)
		Expect(err).ShouldNot(HaveOccurred())

		_, ok, err := queue.Acquire(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ok).To(BeTrue(), "queue should not be empty")

		_, ok, err = queue.Acquire(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ok).To(BeFalse(), "queue should be empty")
	})

	It("removes a message from the queue when it is ack'd", func() {
		m := NewParcel("<id>", MessageM1)
		err := queue.Enqueue(ctx, m)
		Expect(err).ShouldNot(HaveOccurred())

		expect, ok, err := queue.Acquire(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ok).To(BeTrue(), "queue should not be empty")

		err = queue.Ack(ctx, expect.ID())
		Expect(err).ShouldNot(HaveOccurred())

		_, ok, err = queue.Acquire(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ok).To(BeFalse(), "queue should be empty")
	})

	It("allows re-acquiring a message that has been nack'd", func() {
		m := NewParcel("<id>", MessageM1)
		err := queue.Enqueue(ctx, m)
		Expect(err).ShouldNot(HaveOccurred())

		expect, ok, err := queue.Acquire(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ok).To(BeTrue(), "queue should not be empty")

		err = queue.Nack(ctx, expect.ID())
		Expect(err).ShouldNot(HaveOccurred())

		actual, ok, err := queue.Acquire(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ok).To(BeTrue(), "queue should not be empty")
		Expect(actual).To(EqualX(expect))
	})

	It("acquires messages in the order they are enqueued", func() {
		var expect []string
		By("enqueueing several messages", func() {
			messages := []parcel.Parcel{
				NewParcel("<id-1>", MessageM1),
				NewParcel("<id-2>", MessageM2),
				NewParcel("<id-3>", MessageM3),
			}

			for _, m := range messages {
				err := queue.Enqueue(ctx, m)
				Expect(err).ShouldNot(HaveOccurred())
				expect = append(expect, m.ID())
			}
		})

		var actual []string
		By("re-acquiring the messages", func() {
			for {
				m, ok, err := queue.Acquire(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				if !ok {
					break
				}

				actual = append(actual, m.ID())
			}
		})

		Expect(actual).To(
			EqualX(expect),
			"acquired messages should be in the same order as they enqueued",
		)
	})

	It("acquires nack'd messages in the order they were enqueued", func() {
		var expect []string
		By("enqueueing several messages", func() {
			messages := []parcel.Parcel{
				NewParcel("<id-1>", MessageM1),
				NewParcel("<id-2>", MessageM2),
				NewParcel("<id-3>", MessageM3),
			}

			for _, m := range messages {
				err := queue.Enqueue(ctx, m)
				Expect(err).ShouldNot(HaveOccurred())
				expect = append(expect, m.ID())
			}
		})

		var acquired []string
		By("acquiring the messages", func() {
			for {
				m, ok, err := queue.Acquire(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				if !ok {
					break
				}
				acquired = append(acquired, m.ID())
			}
		})

		By("nack'ing the messages in a random order", func() {
			rand.Shuffle(
				len(acquired),
				func(i, j int) {
					acquired[i], acquired[j] = acquired[j], acquired[i]
				},
			)

			for _, id := range acquired {
				err := queue.Nack(ctx, id)
				Expect(err).ShouldNot(HaveOccurred())
			}
		})

		var actual []string
		By("re-acquiring the messages", func() {
			for {
				m, ok, err := queue.Acquire(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				if !ok {
					break
				}

				actual = append(actual, m.ID())
			}
		})

		Expect(actual).To(
			EqualX(expect),
			"acquired messages should be in the same order as they enqueued",
		)
	})
})
