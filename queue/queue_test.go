package queue_test

import (
	"context"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/parcel"
	. "github.com/dogmatiq/veracity/queue"
	. "github.com/jmalloc/gomegax"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Queue", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc

		queue *Queue
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		queue = &Queue{
			Journal: &MemoryJournal{},
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
		messages := []parcel.Parcel{
			NewParcel("<id-1>", MessageM1),
			NewParcel("<id-2>", MessageM2),
			NewParcel("<id-3>", MessageM3),
		}

		var expect, actual []string

		for _, m := range messages {
			err := queue.Enqueue(ctx, m)
			Expect(err).ShouldNot(HaveOccurred())
			expect = append(expect, m.ID())
		}

		for {
			m, ok, err := queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			if !ok {
				break
			}

			actual = append(actual, m.ID())
		}

		Expect(actual).To(EqualX(expect))
	})
})
