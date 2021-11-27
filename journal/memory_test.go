package journal_test

import (
	"context"
	"time"

	. "github.com/dogmatiq/veracity/journal"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/sync/errgroup"
)

var _ = Describe("type MemoryJournal", func() {
	var (
		ctx     context.Context
		cancel  context.CancelFunc
		journal *MemoryJournal
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
		journal = &MemoryJournal{}
	})

	AfterEach(func() {
		cancel()
	})

	Describe("func Open()", func() {
		It("returns a reader that can read historical records", func() {
			for i := byte(0); i < 100; i++ {
				offset, err := journal.Append(ctx, []byte{i})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(offset).To(BeNumerically("==", i))
			}

			r, err := journal.Open(ctx, 0)
			Expect(err).ShouldNot(HaveOccurred())
			defer r.Close()

			for i := byte(0); i < 100; i++ {
				offset, rec, err := r.Next(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(offset).To(BeNumerically("==", i))
				Expect(rec).To(Equal([]byte{i}))
			}
		})

		It("returns a reader that starts reading at the given offset", func() {
			for i := byte(0); i < 100; i++ {
				offset, err := journal.Append(ctx, []byte{i})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(offset).To(BeNumerically("==", i))
			}

			r, err := journal.Open(ctx, 50)
			Expect(err).ShouldNot(HaveOccurred())
			defer r.Close()

			for i := byte(50); i < 100; i++ {
				offset, rec, err := r.Next(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(offset).To(BeNumerically("==", i))
				Expect(rec).To(Equal([]byte{i}))
			}
		})
	})

	Describe("func Append()", func() {
		It("returns the offset of the record", func() {
			offset, err := journal.Append(ctx, []byte("<record>"))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(offset).To(BeNumerically("==", 0))

			offset, err = journal.Append(ctx, []byte("<record>"))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(offset).To(BeNumerically("==", 1))
		})

		It("does not block if there is a stalled reader", func() {
			By("opening a reader")

			r, err := journal.Open(ctx, 0)
			Expect(err).ShouldNot(HaveOccurred())
			defer r.Close()

			By("aborting a read, leaving the 'real-time' channel registered with the journal but no read operation")

			canceledCtx, cancel := context.WithCancel(ctx)
			cancel() // cancel immediately
			_, _, err = r.Next(canceledCtx)
			Expect(err).To(Equal(context.Canceled))

			By("appending enough new records to fill the reader's buffer")

			for i := byte(0); i < 100; i++ {
				offset, err := journal.Append(ctx, []byte{i})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(offset).To(BeNumerically("==", i))
			}

			By("resuming reading")

			for i := byte(0); i < 100; i++ {
				offset, rec, err := r.Next(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(offset).To(BeNumerically("==", i))
				Expect(rec).To(Equal([]byte{i}))
			}
		})

		It("wakes readers that are tailing the journal", func() {
			g, ctx := errgroup.WithContext(ctx)

			read := func() error {
				defer GinkgoRecover()

				r, err := journal.Open(ctx, 0)
				Expect(err).ShouldNot(HaveOccurred())
				defer r.Close()

				offset, rec, err := r.Next(ctx)
				if err != nil {
					return err
				}

				Expect(offset).To(BeNumerically("==", 0))
				Expect(rec).To(Equal([]byte("<record>")))
				return nil
			}

			g.Go(read)
			g.Go(read)

			time.Sleep(100 * time.Millisecond)

			_, err := journal.Append(ctx, []byte("<record>"))
			Expect(err).ShouldNot(HaveOccurred())

			err = g.Wait()
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
