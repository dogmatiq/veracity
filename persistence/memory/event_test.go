package memory_test

import (
	"github.com/dogmatiq/veracity/persistence/internal/persistencetest"
	. "github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("type AggregateEventStore", func() {
	persistencetest.DeclareAggregateEventTests(
		func() persistencetest.AggregateEventContext {
			store := &AggregateEventStore{}

			return persistencetest.AggregateEventContext{
				Reader: store,
				Writer: store,
			}
		},
	)
})
