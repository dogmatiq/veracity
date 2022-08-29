package queue_test

import (
	"context"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	. "github.com/dogmatiq/veracity/queue"
	. "github.com/jmalloc/gomegax"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = XDescribe("type Queue", func() {
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
		cmd := NewParcel("<id>", MessageM1)
		err := queue.Enqueue(ctx, cmd)
		Expect(err).ShouldNot(HaveOccurred())

		_, ok, err := queue.Acquire(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ok).To(BeTrue(), "queue should not be empty")

		_, ok, err = queue.Acquire(ctx)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ok).To(BeFalse(), "queue should be empty")
	})

	It("removes a message from the queue when it is ack'd", func() {
		cmd := NewParcel("<id>", MessageM1)
		err := queue.Enqueue(ctx, cmd)
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
		cmd := NewParcel("<id>", MessageM1)
		err := queue.Enqueue(ctx, cmd)
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
})
