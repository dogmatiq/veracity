package queue_test

import (
	"context"
	"math/rand"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/internal/persistence/journal"
	. "github.com/dogmatiq/veracity/internal/queue"
	"github.com/dogmatiq/veracity/internal/zapx"
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
			Journal: &journal.InMemory[*JournalRecord]{},
			Logger:  zapx.NewTesting(),
		}
	})

	Describe("func Enqueue()", func() {
		It("allows enqueuing multiple messages", func() {
			expect := []Message{
				{
					Envelope: NewEnvelope("<id-1>", MessageM1),
				},
				{
					Envelope: NewEnvelope("<id-2>", MessageM2),
				},
			}

			err := queue.Enqueue(ctx, expect...)
			Expect(err).ShouldNot(HaveOccurred())

			var actual []Message
			for {
				m, ok, err := queue.Acquire(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				if !ok {
					break
				}
				actual = append(actual, m)
			}

			var matchers []any
			for _, env := range expect {
				matchers = append(matchers, EqualX(env))
			}
			Expect(actual).To(ConsistOf(matchers...))
		})
	})

	Describe("func Acquire()", func() {
		It("returns a message from the queue", func() {
			expect := Message{
				Envelope: NewEnvelope("<id>", MessageM1),
			}
			err := queue.Enqueue(ctx, expect)
			Expect(err).ShouldNot(HaveOccurred())

			actual, ok, err := queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue(), "queue should not be empty")
			Expect(actual).To(EqualX(expect))
		})

		It("returns false if the queue is empty", func() {
			_, ok, err := queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse(), "queue should be empty")
		})

		It("does not return a message that has already been acquired", func() {
			err := queue.Enqueue(
				ctx,
				Message{
					Envelope: NewEnvelope("<id>", MessageM1),
				},
			)
			Expect(err).ShouldNot(HaveOccurred())

			_, ok, err := queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue(), "queue should not be empty")

			_, ok, err = queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse(), "queue should be empty")
		})

		It("prioritizes messages by their creation time", func() {
			var expect []string
			By("enqueueing several messages", func() {
				now := time.Now()

				m1 := Message{
					Envelope: NewEnvelope("<id-1>", MessageM1, now.Add(1*time.Second)),
				}

				m2 := Message{
					Envelope: NewEnvelope("<id-2>", MessageM2, now.Add(2*time.Second)),
				}

				m3 := Message{
					Envelope: NewEnvelope("<id-3>", MessageM3, now.Add(3*time.Second)),
				}

				err := queue.Enqueue(ctx, m2, m3, m1)
				Expect(err).ShouldNot(HaveOccurred())
				expect = append(expect, "<id-1>", "<id-2>", "<id-3>")
			})

			var actual []string
			By("acquiring the messages", func() {
				for {
					m, ok, err := queue.Acquire(ctx)
					Expect(err).ShouldNot(HaveOccurred())
					if !ok {
						break
					}

					actual = append(actual, m.Envelope.GetMessageId())
				}
			})

			Expect(actual).To(
				EqualX(expect),
				"messages should be prioritized by their creation time",
			)
		})
	})

	Describe("func Ack()", func() {
		It("removes the message from the queue", func() {
			err := queue.Enqueue(
				ctx,
				Message{
					Envelope: NewEnvelope("<id>", MessageM1),
				},
			)
			Expect(err).ShouldNot(HaveOccurred())

			m, ok, err := queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue(), "queue should not be empty")

			err = queue.Ack(ctx, m.Envelope.GetMessageId())
			Expect(err).ShouldNot(HaveOccurred())

			By("re-reading the queue state from the journal", func() {
				queue = &Queue{
					Journal: &journal.InMemory[*JournalRecord]{},
					Logger:  zapx.NewTesting(),
				}
			})

			_, ok, err = queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse(), "queue should be empty")
		})
	})

	Describe("func Reject()", func() {
		It("allows the message to be re-acquired", func() {
			err := queue.Enqueue(
				ctx,
				Message{
					Envelope: NewEnvelope("<id>", MessageM1),
				},
			)
			Expect(err).ShouldNot(HaveOccurred())

			expect, ok, err := queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue(), "queue should not be empty")

			err = queue.Reject(ctx, expect.Envelope.GetMessageId())
			Expect(err).ShouldNot(HaveOccurred())

			actual, ok, err := queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue(), "queue should not be empty")
			Expect(actual).To(EqualX(expect))
		})

		It("does not affect the queue priority", func() {
			var expect []string
			By("enqueueing several messages", func() {
				now := time.Now()
				envelopes := []*envelopespec.Envelope{
					NewEnvelope("<id-1>", MessageM1, now.Add(1*time.Second)),
					NewEnvelope("<id-2>", MessageM2, now.Add(2*time.Second)),
					NewEnvelope("<id-3>", MessageM3, now.Add(3*time.Second)),
				}

				for _, env := range envelopes {
					err := queue.Enqueue(
						ctx,
						Message{
							Envelope: env,
						},
					)
					Expect(err).ShouldNot(HaveOccurred())
					expect = append(expect, env.GetMessageId())
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
					acquired = append(acquired, m.Envelope.GetMessageId())
				}
			})

			By("rejecting the messages in a random order", func() {
				rand.Shuffle(
					len(acquired),
					func(i, j int) {
						acquired[i], acquired[j] = acquired[j], acquired[i]
					},
				)

				for _, id := range acquired {
					err := queue.Reject(ctx, id)
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

					actual = append(actual, m.Envelope.GetMessageId())
				}
			})

			Expect(actual).To(
				EqualX(expect),
				"rejected messages should still be prioritized by creation time",
			)
		})
	})
})
