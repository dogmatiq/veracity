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
			expect := []*envelopespec.Envelope{
				NewEnvelope("<id-1>", MessageM1),
				NewEnvelope("<id-2>", MessageM2),
			}

			err := queue.Enqueue(ctx, expect...)
			Expect(err).ShouldNot(HaveOccurred())

			var actual []*envelopespec.Envelope
			for {
				env, ok, err := queue.Acquire(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				if !ok {
					break
				}
				actual = append(actual, env)
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
			expect := NewEnvelope("<id>", MessageM1)
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
			err := queue.Enqueue(ctx, NewEnvelope("<id>", MessageM1))
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
				env1 := NewEnvelope("<id-1>", MessageM1, now.Add(1*time.Second))
				env2 := NewEnvelope("<id-2>", MessageM2, now.Add(2*time.Second))
				env3 := NewEnvelope("<id-3>", MessageM3, now.Add(3*time.Second))

				err := queue.Enqueue(ctx, env2, env3, env1)
				Expect(err).ShouldNot(HaveOccurred())
				expect = append(expect, "<id-1>", "<id-2>", "<id-3>")
			})

			var actual []string
			By("acquiring the messages", func() {
				for {
					env, ok, err := queue.Acquire(ctx)
					Expect(err).ShouldNot(HaveOccurred())
					if !ok {
						break
					}

					actual = append(actual, env.GetMessageId())
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
			err := queue.Enqueue(ctx, NewEnvelope("<id>", MessageM1))
			Expect(err).ShouldNot(HaveOccurred())

			env, ok, err := queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue(), "queue should not be empty")

			err = queue.Ack(ctx, env.GetMessageId())
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
			err := queue.Enqueue(ctx, NewEnvelope("<id>", MessageM1))
			Expect(err).ShouldNot(HaveOccurred())

			expect, ok, err := queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue(), "queue should not be empty")

			err = queue.Reject(ctx, expect.GetMessageId())
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

				for _, m := range envelopes {
					err := queue.Enqueue(ctx, m)
					Expect(err).ShouldNot(HaveOccurred())
					expect = append(expect, m.GetMessageId())
				}
			})

			var acquired []string
			By("acquiring the messages", func() {
				for {
					env, ok, err := queue.Acquire(ctx)
					Expect(err).ShouldNot(HaveOccurred())
					if !ok {
						break
					}
					acquired = append(acquired, env.GetMessageId())
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
					env, ok, err := queue.Acquire(ctx)
					Expect(err).ShouldNot(HaveOccurred())
					if !ok {
						break
					}

					actual = append(actual, env.GetMessageId())
				}
			})

			Expect(actual).To(
				EqualX(expect),
				"rejected messages should still be prioritized by creation time",
			)
		})
	})
})
