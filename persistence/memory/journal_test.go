package memory_test

import (
	"context"
	"fmt"
	"time"

	"github.com/dogmatiq/veracity/persistence"
	. "github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Journal", func() {
	var (
		ctx     context.Context
		cancel  context.CancelFunc
		journal persistence.Journal
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
		journal = &Journal{}
	})

	AfterEach(func() {
		cancel()
	})

	Describe("func Append()", func() {
		It("returns the ID of the record", func() {
			id, ok, err := journal.Append(ctx, nil, []byte("<record>"))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(id).To(Equal([]byte("0")))
			Expect(ok).To(BeTrue())

			id, ok, err = journal.Append(ctx, id, []byte("<record>"))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(id).To(Equal([]byte("1")))
			Expect(ok).To(BeTrue())
		})

		It("does not append the record if the supplied 'previous ID' is incorrect", func() {
			_, ok, err := journal.Append(ctx, []byte("0"), []byte("<record>"))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse())

			r, err := journal.NewReader(ctx, nil, persistence.JournalReaderOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			defer r.Close()

			_, _, ok, err = r.Next(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse())
		})

		It("does not block if there is a stalled reader", func() {
			barrier := make(chan struct{})

			By("appending a record")

			id, ok, err := journal.Append(ctx, nil, []byte("<record>"))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeTrue())

			By("stalling a reader on a separate goroutine")

			go func() {
				defer GinkgoRecover()
				defer close(barrier)

				r, err := journal.NewReader(ctx, nil, persistence.JournalReaderOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				defer r.Close()

				_, _, ok, err := r.Next(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(ok).To(BeTrue())

				barrier <- struct{}{} // signal that reader obtained a record
				barrier <- struct{}{} // stall until unblocked
			}()

			By("waiting for the reader to begin")

			<-barrier

			By("appending another record")

			_, ok, err = journal.Append(ctx, id, []byte("<record>"))
			Expect(err).ShouldNot(HaveOccurred()) // would timeout if reader blocked appending
			Expect(ok).To(BeTrue())

			<-barrier // allow the reader to finish
			<-barrier // wait until the reader goroutine actually exits
		})
	})

	Describe("func Truncate()", func() {
		BeforeEach(func() {
			var (
				lastID []byte
				ok     bool
				err    error
			)

			for index := byte(0); index < 100; index++ {
				lastID, ok, err = journal.Append(ctx, lastID, []byte{index})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			}

			err = journal.Truncate(ctx, []byte("50"))
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("removes records before the given ID", func() {
			r, err := journal.NewReader(ctx, []byte("48"), persistence.JournalReaderOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			defer r.Close()

			_, _, _, err = r.Next(ctx)
			Expect(err).To(MatchError(`record "49" has been truncated`))
		})

		It("does not remove records after the given ID", func() {
			r, err := journal.NewReader(ctx, []byte("49"), persistence.JournalReaderOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			defer r.Close()

			for index := byte(50); index < 100; index++ {
				id, data, ok, err := r.Next(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(id).To(Equal([]byte(fmt.Sprintf("%d", index))))
				Expect(data).To(Equal([]byte{index}))
				Expect(ok).To(BeTrue())
			}

			_, _, ok, err := r.Next(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse())
		})

		It("allows the reader to skip truncated records with the appropriate option", func() {
			r, err := journal.NewReader(ctx, nil, persistence.JournalReaderOptions{
				SkipTruncated: true,
			})
			Expect(err).ShouldNot(HaveOccurred())
			defer r.Close()

			for index := byte(50); index < 100; index++ {
				id, data, ok, err := r.Next(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(id).To(Equal([]byte(fmt.Sprintf("%d", index))))
				Expect(data).To(Equal([]byte{index}))
				Expect(ok).To(BeTrue())
			}

			_, _, ok, err := r.Next(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})

	Describe("func NewReader()", func() {
		BeforeEach(func() {
			var (
				lastID []byte
				ok     bool
				err    error
			)

			for index := byte(0); index < 100; index++ {
				lastID, ok, err = journal.Append(ctx, lastID, []byte{index})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			}
		})

		It("can read from the start of the journal", func() {
			r, err := journal.NewReader(ctx, nil, persistence.JournalReaderOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			defer r.Close()

			for index := byte(0); index < 100; index++ {
				id, data, ok, err := r.Next(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(id).To(Equal([]byte(fmt.Sprintf("%d", index))))
				Expect(data).To(Equal([]byte{index}))
				Expect(ok).To(BeTrue())
			}

			_, _, ok, err := r.Next(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse())
		})

		It("can start reading midway through the journal", func() {
			r, err := journal.NewReader(ctx, []byte("49"), persistence.JournalReaderOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			defer r.Close()

			for index := byte(50); index < 100; index++ {
				id, data, ok, err := r.Next(ctx)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(id).To(Equal([]byte(fmt.Sprintf("%d", index))))
				Expect(data).To(Equal([]byte{index}))
				Expect(ok).To(BeTrue())
			}

			_, _, ok, err := r.Next(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})
})
