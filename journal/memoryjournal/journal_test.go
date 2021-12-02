package memoryjournal_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/dogmatiq/veracity/journal/memoryjournal"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
			_, err := journal.Append(ctx, []byte("0"), []byte("<record>"))
			Expect(err).To(MatchError(`optimistic lock failure, the last record ID is "" but the caller provided "0"`))

			r, err := journal.OpenReader(ctx, nil)
			Expect(err).ShouldNot(HaveOccurred())
			defer r.Close()

			_, _, ok, err := r.Next(ctx)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ok).To(BeFalse())
		})

		It("does not block if there is a stalled reader", func() {
			barrier := make(chan struct{})

			By("appending a record")

			id, err := journal.Append(ctx, nil, []byte("<record>"))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(id).To(Equal([]byte("0")))

			By("stalling a reader on a separate goroutine")

			go func() {
				defer GinkgoRecover()
				defer close(barrier)

				r, err := journal.OpenReader(ctx, nil)
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

			_, err = journal.Append(ctx, id, []byte("<record>"))
			Expect(err).ShouldNot(HaveOccurred()) // would timeout if reader blocked appending

			<-barrier // allow the reader to finish
			<-barrier // wait until the reader goroutine actually exits
		})
	})

	Describe("func OpenReader()", func() {
		BeforeEach(func() {
			var (
				lastID []byte
				err    error
			)

			for i := byte(0); i < 100; i++ {
				lastID, err = journal.Append(ctx, lastID, []byte{i})
				Expect(err).ShouldNot(HaveOccurred())
			}
		})

		It("can read from the start of the journal", func() {
			r, err := journal.OpenReader(ctx, nil)
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
			r, err := journal.OpenReader(ctx, []byte("49"))
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
