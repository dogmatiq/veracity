package uuidpb_test

import (
	"fmt"

	. "github.com/dogmatiq/veracity/internal/protobuf/uuidpb"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type UUID", func() {
	Describe("func FromNative() / ToNative()", func() {
		It("converts a native UUID to a protobuf UUID and back", func() {
			expect := uuid.New()
			pb := FromNative(expect)
			actual := pb.ToNative()
			Expect(actual).To(Equal(expect))
		})
	})

	Describe("func Format()", func() {
		It("provides a useful string representation", func() {
			pb := New()
			expect := pb.ToNative().String()
			actual := fmt.Sprintf("%s", pb)
			Expect(actual).To(Equal(expect))
		})
	})
})
