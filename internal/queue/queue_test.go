package queue_test

import (
	"context"
	"math/rand"
	"time"

	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/envelope"
	. "github.com/dogmatiq/veracity/internal/queue"
	"github.com/dogmatiq/veracity/internal/zapx"
	"github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/jmalloc/gomegax"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var _ = Describe("type Queue", func() {
	var (
		ctx    context.Context
		packer *envelope.Packer
		queue  *Queue
	)

	BeforeEach(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
		DeferCleanup(cancel)

		packer = envelope.NewTestPacker()

		queue = &Queue{
			Journal: &memory.Journal[*JournalRecord]{},
			Logger:  zapx.NewTesting("queue"),
		}
	})

	Describe("func Enqueue()", func() {
		It("allows enqueuing multiple messages", func() {
			expect := []*envelopespec.Envelope{
				packer.Pack(MessageM1),
				packer.Pack(MessageM2),
			}

			var (
				messages []Message
				matchers []any
			)
			for _, env := range expect {
				messages = append(messages, Message{Envelope: env})
				matchers = append(matchers, EqualX(env))
			}

			err := queue.Enqueue(ctx, messages...)
			Expect(err).ShouldNot(HaveOccurred())

			var actual []*envelopespec.Envelope
			for {
				m, ok, err := queue.Acquire(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				if !ok {
					break
				}

				actual = append(actual, m.Envelope)
			}
			Expect(actual).To(ConsistOf(matchers...))
		})

		It("allows associating meta-data with each message", func() {
			expect := Message{
				Envelope: packer.Pack(MessageM1),
				MetaData: wrapperspb.Int32(123),
			}

			err := queue.Enqueue(ctx, expect)
			Expect(err).ShouldNot(HaveOccurred())

			By("re-reading the queue state from the journal", func() {
				queue = &Queue{
					Journal: queue.Journal,
					Logger:  zapx.NewTesting("queue-reread"),
				}
			})

			actual, ok, err := queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue(), "queue should not be empty")
			Expect(actual.MetaData).To(EqualX(expect.MetaData))
		})
	})

	Describe("func Acquire()", func() {
		It("returns the next message on the queue", func() {
			expect := Message{
				Envelope: packer.Pack(MessageM1),
			}
			err := queue.Enqueue(ctx, expect)
			Expect(err).ShouldNot(HaveOccurred())

			actual, ok, err := queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue(), "queue should not be empty")
			Expect(actual.Envelope).To(EqualX(expect.Envelope))
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
					Envelope: packer.Pack(MessageM1),
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
					Envelope: packer.Pack(
						MessageM1,
						envelope.WithCreatedAt(now.Add(1*time.Second)),
					),
				}

				m2 := Message{
					Envelope: packer.Pack(
						MessageM2,
						envelope.WithCreatedAt(now.Add(2*time.Second)),
					),
				}

				m3 := Message{
					Envelope: packer.Pack(
						MessageM3,
						envelope.WithCreatedAt(now.Add(3*time.Second)),
					),
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

	Describe("func Release()", func() {
		It("allows the message to be re-acquired", func() {
			err := queue.Enqueue(
				ctx,
				Message{
					Envelope: packer.Pack(MessageM1),
				},
			)
			Expect(err).ShouldNot(HaveOccurred())

			expect, ok, err := queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue(), "queue should not be empty")

			err = queue.Release(ctx, expect)
			Expect(err).ShouldNot(HaveOccurred())

			actual, ok, err := queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue(), "queue should not be empty")
			Expect(actual.Envelope).To(EqualX(expect.Envelope))
		})

		It("does not affect the queue priority", func() {
			var expect []string
			By("enqueueing several messages", func() {
				now := time.Now()
				envelopes := []*envelopespec.Envelope{
					packer.Pack(
						MessageM1,
						envelope.WithCreatedAt(now.Add(1*time.Second)),
					),
					packer.Pack(
						MessageM2,
						envelope.WithCreatedAt(now.Add(2*time.Second)),
					),
					packer.Pack(
						MessageM3,
						envelope.WithCreatedAt(now.Add(3*time.Second)),
					),
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

			var acquired []AcquiredMessage
			By("acquiring the messages", func() {
				for {
					m, ok, err := queue.Acquire(ctx)
					Expect(err).ShouldNot(HaveOccurred())
					if !ok {
						break
					}
					acquired = append(acquired, m)
				}
			})

			By("releasing the messages in a random order", func() {
				rand.Shuffle(
					len(acquired),
					func(i, j int) {
						acquired[i], acquired[j] = acquired[j], acquired[i]
					},
				)

				for _, m := range acquired {
					err := queue.Release(ctx, m)
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
				"released messages should still be prioritized by creation time",
			)
		})
	})

	Describe("func Remove()", func() {
		It("removes the message from the queue", func() {
			err := queue.Enqueue(
				ctx,
				Message{
					Envelope: packer.Pack(MessageM1),
				},
			)
			Expect(err).ShouldNot(HaveOccurred())

			m, ok, err := queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue(), "queue should not be empty")

			err = queue.Remove(ctx, m)
			Expect(err).ShouldNot(HaveOccurred())

			By("re-reading the queue state from the journal", func() {
				queue = &Queue{
					Journal: queue.Journal,
					Logger:  zapx.NewTesting("queue-reread"),
				}
			})

			_, ok, err = queue.Acquire(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse(), "queue should be empty")
		})
	})
})
