package aggregate_test

import (
	"context"
	"time"

	"github.com/dogmatiq/configkit"
	. "github.com/dogmatiq/dogma/fixtures"
	. "github.com/dogmatiq/marshalkit/fixtures"
	. "github.com/dogmatiq/veracity/aggregate"
	"github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Worker", func() {
	var (
		eventStore  *memory.AggregateEventStore
		eventReader *eventReaderStub
		eventWriter *eventWriterStub

		snapshotStore  *memory.AggregateSnapshotStore
		snapshotReader *snapshotReaderStub
		snapshotWriter *snapshotWriterStub

		loader  *Loader
		handler *AggregateMessageHandler
		worker  *Worker
	)

	BeforeEach(func() {
		eventStore = &memory.AggregateEventStore{}

		eventReader = &eventReaderStub{
			EventReader: eventStore,
		}

		eventWriter = &eventWriterStub{
			EventWriter: eventStore,
		}

		snapshotStore = &memory.AggregateSnapshotStore{}

		snapshotReader = &snapshotReaderStub{
			SnapshotReader: snapshotStore,
		}

		snapshotWriter = &snapshotWriterStub{
			SnapshotWriter: snapshotStore,
		}

		loader = &Loader{
			EventReader:    eventReader,
			SnapshotReader: snapshotReader,
			Marshaler:      Marshaler,
		}

		handler = &AggregateMessageHandler{}

		worker = &Worker{
			WorkerConfig: WorkerConfig{
				Handler:              handler,
				HandlerIdentity:      configkit.MustNewIdentity("<handler-name>", "<handler-key>"),
				Loader:               loader,
				EventWriter:          eventWriter,
				SnapshotWriter:       snapshotWriter,
				IdleTimeout:          5 * time.Millisecond,
				SnapshotInterval:     5,
				SnapshotWriteTimeout: 5 * time.Millisecond,
				Logger:               nil,
			},
		}
	})

	Describe("func Run()", func() {
		When("a command is received", func() {
			XIt("writes events recorded by the handler", func() {
			})

			XIt("writes a nil error to the command's result channel", func() {

			})

			When("a second command is received", func() {
				XIt("does not produce an OCC error", func() {
				})
			})
		})

		When("the instance is destroyed", func() {
			XIt("archives historical events", func() {
			})

			XIt("archives snapshots", func() {

			})

			XIt("returns nil", func() {
			})
		})

		When("the idle timeout is reached", func() {
			BeforeEach(func() {
				worker.IdleTimeout = 1 * time.Millisecond
			})

			XIt("takes a snapshot if the existing snapshot is out-of-date", func() {
			})

			XIt("does not take a snapshot if the existing snapshot is up-to-date", func() {
			})

			It("returns nil", func() {
				err := worker.Run(context.Background())
				Expect(err).ShouldNot(HaveOccurred())
			})
		})

		XIt("applies the snapshot write timeout", func() {

		})

		XIt("does not return an error if writing a snapshot fails", func() {

		})

		XIt("returns an error if the root cannot be loaded", func() {
		})

		XIt("returns an error if events cannot be written", func() {
		})

		XIt("returns an error if the context is canceled while waiting for a command", func() {
		})

		XIt("returns an error if the context is canceled while persisting a snapshot", func() {
		})
	})
})
