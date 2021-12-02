package memoryjournal_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/dogmatiq/veracity/journal/memoryjournal"
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
			var lastID []byte
			for i := byte(0); i < 100; i++ {
				var err error
				lastID, err = journal.Append(ctx, lastID, []byte{i})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(lastID).To(Equal([]byte(fmt.Sprintf("%d", i))))
			}

			r, err := journal.Open(ctx, nil)
			Expect(err).ShouldNot(HaveOccurred())
			defer r.Close()

			for i := byte(0); i < 100; i++ {
				id, rec, err := r.Next(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(id).To(Equal([]byte(fmt.Sprintf("%d", i))))
				Expect(rec).To(Equal([]byte{i}))
			}
		})

		It("returns a reader that starts reading from the given record ID", func() {
			var lastID []byte
			for i := byte(0); i < 100; i++ {
				var err error
				lastID, err = journal.Append(ctx, lastID, []byte{i})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(lastID).To(Equal([]byte(fmt.Sprintf("%d", i))))
			}

			r, err := journal.Open(ctx, []byte("49"))
			Expect(err).ShouldNot(HaveOccurred())
			defer r.Close()

			for i := byte(50); i < 100; i++ {
				id, rec, err := r.Next(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(id).To(Equal([]byte(fmt.Sprintf("%d", i))))
				Expect(rec).To(Equal([]byte{i}))
			}
		})
	})

	Describe("func LastID()", func() {
		It("returns an empty string when the journal is empty", func() {
			id, err := journal.LastID(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(id).To(BeEmpty())
		})

		It("returns the ID of the last record", func() {
			var lastID []byte
			for i := byte(0); i < 100; i++ {
				var err error
				lastID, err = journal.Append(ctx, lastID, []byte{i})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(lastID).To(Equal([]byte(fmt.Sprintf("%d", i))))
			}

			id, err := journal.LastID(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(id).To(Equal(lastID))
		})
	})

	Describe("func Append()", func() {
		It("returns the ID of the record", func() {
			id, err := journal.Append(ctx, nil, []byte("<record>"))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(id).To(Equal([]byte("0")))

			id, err = journal.Append(ctx, id, []byte("<record>"))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(id).To(Equal([]byte("1")))
		})

		It("does not append the event if the supplied last ID is incorrect", func() {
			_, err := journal.Append(ctx, []byte("<incorrect>"), []byte("<record>"))
			Expect(err).To(MatchError(`last ID does not match (expected "", got "<incorrect>")`))

			lastID, err := journal.LastID(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lastID).To(BeEmpty())
		})

		It("does not block if there is a stalled reader", func() {
			By("opening a reader")

			r, err := journal.Open(ctx, nil)
			Expect(err).ShouldNot(HaveOccurred())
			defer r.Close()

			By("aborting a read, leaving the 'real-time' channel registered with the journal but no read operation")

			canceledCtx, cancel := context.WithCancel(ctx)
			cancel() // cancel immediately
			_, _, err = r.Next(canceledCtx)
			Expect(err).To(Equal(context.Canceled))

			By("appending enough new records to fill the reader's buffer")

			var lastID []byte
			for i := byte(0); i < 100; i++ {
				var err error
				lastID, err = journal.Append(ctx, lastID, []byte{i})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(lastID).To(Equal([]byte(fmt.Sprintf("%d", i))))
			}

			By("resuming reading")

			for i := byte(0); i < 100; i++ {
				id, rec, err := r.Next(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(id).To(Equal([]byte(fmt.Sprintf("%d", i))))
				Expect(rec).To(Equal([]byte{i}))
			}
		})

		It("wakes readers that are tailing the journal", func() {
			g, ctx := errgroup.WithContext(ctx)

			read := func() error {
				defer GinkgoRecover()

				r, err := journal.Open(ctx, nil)
				Expect(err).ShouldNot(HaveOccurred())
				defer r.Close()

				id, rec, err := r.Next(ctx)
				if err != nil {
					return err
				}

				Expect(id).To(Equal([]byte("0")))
				Expect(rec).To(Equal([]byte("<record>")))
				return nil
			}

			g.Go(read)
			g.Go(read)

			time.Sleep(100 * time.Millisecond)

			_, err := journal.Append(ctx, nil, []byte("<record>"))
			Expect(err).ShouldNot(HaveOccurred())

			err = g.Wait()
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
