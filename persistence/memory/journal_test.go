package memory_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/sync/errgroup"
)

var _ = Describe("type Journal", func() {
	var (
		ctx     context.Context
		cancel  context.CancelFunc
		journal *Journal
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
		journal = &Journal{}
	})

	AfterEach(func() {
		cancel()
	})

	Describe("func Open()", func() {
		It("returns a reader that can read historical records", func() {
			lastID := ""
			for i := byte(0); i < 100; i++ {
				var err error
				lastID, err = journal.Append(ctx, lastID, []byte{i})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(lastID).To(Equal(fmt.Sprintf("%d", i)))
			}

			r, err := journal.Open(ctx, "")
			Expect(err).ShouldNot(HaveOccurred())
			defer r.Close()

			for i := byte(0); i < 100; i++ {
				id, rec, more, err := r.Next(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(id).To(Equal(fmt.Sprintf("%d", i)))
				Expect(rec).To(Equal([]byte{i}))
				Expect(more).To(Equal(i < 99))
			}
		})

		It("returns a reader that starts reading from the given record ID", func() {
			lastID := ""
			for i := byte(0); i < 100; i++ {
				var err error
				lastID, err = journal.Append(ctx, lastID, []byte{i})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(lastID).To(Equal(fmt.Sprintf("%d", i)))
			}

			r, err := journal.Open(ctx, "50")
			Expect(err).ShouldNot(HaveOccurred())
			defer r.Close()

			for i := byte(50); i < 100; i++ {
				id, rec, more, err := r.Next(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(id).To(Equal(fmt.Sprintf("%d", i)))
				Expect(rec).To(Equal([]byte{i}))
				Expect(more).To(Equal(i < 99))
			}
		})
	})

	Describe("func Append()", func() {
		It("returns the ID of the record", func() {
			id, err := journal.Append(ctx, "", []byte("<record>"))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(id).To(Equal("0"))

			id, err = journal.Append(ctx, id, []byte("<record>"))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(id).To(Equal("1"))
		})

		It("does not block if there is a stalled reader", func() {
			By("opening a reader")

			r, err := journal.Open(ctx, "")
			Expect(err).ShouldNot(HaveOccurred())
			defer r.Close()

			By("aborting a read, leaving the 'real-time' channel registered with the journal but no read operation")

			canceledCtx, cancel := context.WithCancel(ctx)
			cancel() // cancel immediately
			_, _, _, err = r.Next(canceledCtx)
			Expect(err).To(Equal(context.Canceled))

			By("appending enough new records to fill the reader's buffer")

			lastID := ""
			for i := byte(0); i < 100; i++ {
				var err error
				lastID, err = journal.Append(ctx, lastID, []byte{i})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(lastID).To(Equal(fmt.Sprintf("%d", i)))
			}

			By("resuming reading")

			for i := byte(0); i < 100; i++ {
				id, rec, _, err := r.Next(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(id).To(Equal(fmt.Sprintf("%d", i)))
				Expect(rec).To(Equal([]byte{i}))
			}
		})

		It("wakes readers that are tailing the journal", func() {
			g, ctx := errgroup.WithContext(ctx)

			read := func() error {
				defer GinkgoRecover()

				r, err := journal.Open(ctx, "")
				Expect(err).ShouldNot(HaveOccurred())
				defer r.Close()

				id, rec, _, err := r.Next(ctx)
				if err != nil {
					return err
				}

				Expect(id).To(Equal("0"))
				Expect(rec).To(Equal([]byte("<record>")))
				return nil
			}

			g.Go(read)
			g.Go(read)

			time.Sleep(100 * time.Millisecond)

			_, err := journal.Append(ctx, "", []byte("<record>"))
			Expect(err).ShouldNot(HaveOccurred())

			err = g.Wait()
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
