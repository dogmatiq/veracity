package memory_test

import (
	"context"
	"time"

	. "github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("type KeyValueStore", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		store  *KeyValueStore
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
		store = &KeyValueStore{}
	})

	AfterEach(func() {
		cancel()
	})

	It("associates values with keys", func() {
		err := store.Set(ctx, []byte("<key>"), []byte("<value>"))
		Expect(err).ShouldNot(HaveOccurred())

		x, err := store.Get(ctx, []byte("<key>"))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(x).To(Equal([]byte("<value>")))
	})

	It("deletes the key when set to an empty value", func() {
		err := store.Set(ctx, []byte("<key>"), nil)
		Expect(err).ShouldNot(HaveOccurred())

		v, err := store.Get(ctx, []byte("<key>"))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(v).To(BeEmpty())
	})

	It("allows getting a key that has never been set", func() {
		v, err := store.Get(ctx, []byte("<key>"))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(v).To(BeEmpty())
	})
})
