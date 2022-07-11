package aggregate_test

import (
	"context"
	"time"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"
	. "github.com/dogmatiq/marshalkit/fixtures"
	. "github.com/dogmatiq/veracity/aggregate"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Worker", func() {
	var (
		ctx    context.Context
		cancel context.Cancel

		eventStore  *memory.AggregateEventStore
		eventReader *eventReaderStub
		eventWriter *eventWriterStub

		snapshotStore  *memory.AggregateSnapshotStore
		snapshotReader *snapshotReaderStub
		snapshotWriter *snapshotWriterStub

		loader   *Loader
		commands chan Command
		handler  *AggregateMessageHandler
		worker   *Worker
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

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

		commands = make(chan Command)

		handler = &AggregateMessageHandler{
			ConfigureFunc: func(c dogma.AggregateConfigurer) {
				c.Identity("<handler-name>", "<handler-key>")
				c.ConsumesCommandType(MessageC{})
				c.ProducesEventType(MessageE{})
			},
			HandleCommandFunc: func(
				r dogma.AggregateRoot,
				s dogma.AggregateCommandScope,
				m dogma.Message,
			) {
				s.RecordEvent(MessageE{})
			},
		}

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
			InstanceID: "<instance>",
			Commands:   commands,
		}
	})

	AfterEach(func() {
		cancel()
	})

	Describe("func Run()", func() {
		When("a command is received", func() {
			It("writes events recorded by the handler", func() {
				// cmd := Command{
				// 	Context: context.Background(),
				// 	Parcel:  NewParcel("<command-1>", MessageC1),
				// 	Result:  make(chan<- error),
				// }

				// select {
				// case <-ctx.Done():
				// 	Expect(ctx.Err()).ShouldNot(HaveOccurred())
				// case commands <- cmd:
				// }
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
