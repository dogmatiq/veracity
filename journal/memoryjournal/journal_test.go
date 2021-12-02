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

			lastID, err := journal.Read(
				ctx,
				nil,
				func(ctx context.Context, id, data []byte) error {
					Fail("unexpected record")
					return nil
				},
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lastID).To(BeEmpty())
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

				_, err := journal.Read(
					ctx,
					nil,
					func(ctx context.Context, id, data []byte) error {
						barrier <- struct{}{} // notify reader has entered function
						barrier <- struct{}{} // stall
						return nil
					},
				)
				Expect(err).ShouldNot(HaveOccurred())
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

	Describe("func Read()", func() {
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

		It("calls the function for each record in the journal", func() {
			index := byte(0)
			lastID, err := journal.Read(
				ctx,
				nil,
				func(ctx context.Context, id, data []byte) error {
					Expect(id).To(Equal([]byte(fmt.Sprintf("%d", index))))
					Expect(data).To(Equal([]byte{index}))
					index++
					return nil
				},
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lastID).To(Equal([]byte("99")))
		})

		It("can start reading midway through the journal", func() {
			index := byte(50)
			lastID, err := journal.Read(
				ctx,
				[]byte("49"),
				func(ctx context.Context, id, data []byte) error {
					Expect(id).To(Equal([]byte(fmt.Sprintf("%d", index))))
					Expect(data).To(Equal([]byte{index}))
					index++
					return nil
				},
			)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lastID).To(Equal([]byte("99")))
		})
	})
})
