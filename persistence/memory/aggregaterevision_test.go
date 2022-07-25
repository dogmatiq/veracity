package memory_test

import (
	"github.com/dogmatiq/veracity/persistence/internal/persistencetest"
	. "github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("type AggregateRevisionStore", func() {
	persistencetest.DeclareAggregateRevisionTests(
		func() persistencetest.AggregateRevisionContext {
			store := &AggregateRevisionStore{}

			return persistencetest.AggregateRevisionContext{
				Reader: store,
				Writer: store,
			}
		},
	)
})
