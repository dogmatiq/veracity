package memory_test

import (
	"github.com/dogmatiq/marshalkit"
	"github.com/dogmatiq/veracity/persistence/internal/persistencetest"
	. "github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("type AggregateSnapshotStore", func() {
	persistencetest.DeclareAggregateSnapshotTests(
		func(m marshalkit.ValueMarshaler) persistencetest.AggregateSnapshotContext {
			store := &AggregateSnapshotStore{
				Marshaler: m,
			}

			return persistencetest.AggregateSnapshotContext{
				Reader: store,
				Writer: store,
			}
		},
	)
})
