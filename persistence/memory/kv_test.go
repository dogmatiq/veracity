package memory_test

import (
	"context"
	"time"

	"github.com/dogmatiq/veracity/persistence"
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
		err := store.Set(ctx, persistence.Key("<key>"), []byte("<value>"))
		Expect(err).ShouldNot(HaveOccurred())

		x, err := store.Get(ctx, persistence.Key("<key>"))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(x).To(Equal([]byte("<value>")))

		exists, err := store.Exists(ctx, persistence.Key("<key>"))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(exists).To(BeTrue())
	})

	It("deletes the key when set to an empty value", func() {
		err := store.Set(ctx, persistence.Key("<key>"), nil)
		Expect(err).ShouldNot(HaveOccurred())

		exists, err := store.Exists(ctx, persistence.Key("<key>"))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(exists).To(BeFalse())
	})

	It("allows getting a key that has never been set", func() {
		v, err := store.Get(ctx, persistence.Key("<key>"))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(v).To(BeEmpty())
	})
})
