package memoryindex_test

import (
	"context"
	"time"

	. "github.com/dogmatiq/veracity/index/memoryindex"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Index", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		index  *Index
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
		index = &Index{}
	})

	AfterEach(func() {
		cancel()
	})

	It("associates values with keys", func() {
		err := index.Set(ctx, []byte("<key>"), []byte("<value>"))
		Expect(err).ShouldNot(HaveOccurred())

		x, err := index.Get(ctx, []byte("<key>"))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(x).To(Equal([]byte("<value>")))
	})

	It("effectively deletes the key when set to an empty value", func() {
		err := index.Set(ctx, []byte("<key>"), nil)
		Expect(err).ShouldNot(HaveOccurred())

		v, err := index.Get(ctx, []byte("<key>"))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(v).To(BeEmpty())
	})

	It("allows getting a key that has never been set", func() {
		v, err := index.Get(ctx, []byte("<key>"))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(v).To(BeEmpty())
	})
})
