package filter_test

import (
	. "github.com/dogmatiq/veracity/internal/filter"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("func Slice()", func() {
	It("removes the elements that do not match the predicate", func() {
		even := Slice(
			[]int{0, 1, 2, 3, 4, 5, 6},
			func(v int) bool {
				return v%2 == 0
			},
		)

		Expect(even).To(Equal([]int{0, 2, 4, 6}))
	})
})
