package aggregate_test

import (
	"context"
	"time"

	"github.com/dogmatiq/configkit"
	. "github.com/dogmatiq/configkit/fixtures"
	"github.com/dogmatiq/configkit/message"
	"github.com/dogmatiq/dogma"
	. "github.com/dogmatiq/dogma/fixtures"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/marshalkit"
	. "github.com/dogmatiq/marshalkit/fixtures"
	. "github.com/dogmatiq/veracity/aggregate"
	. "github.com/dogmatiq/veracity/internal/fixtures"
	"github.com/dogmatiq/veracity/parcel"
	"github.com/dogmatiq/veracity/persistence/memory"
	. "github.com/jmalloc/gomegax"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("type Worker", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc

		eventStore  *memory.AggregateEventStore
		eventReader *eventReaderStub
		eventWriter *eventWriterStub

		snapshotStore  *memory.AggregateSnapshotStore
		snapshotReader *snapshotReaderStub
		snapshotWriter *snapshotWriterStub

		packer   *parcel.Packer
		loader   *Loader
		commands chan Command
		handler  *AggregateMessageHandler
		worker   *Worker
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		DeferCleanup(cancel)

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
				s.RecordEvent(MessageE1)
			},
		}

		packer = NewPacker(
			message.TypeRoles{
				MessageCType: message.CommandRole,
				MessageEType: message.EventRole,
			},
		)

		worker = &Worker{
			WorkerConfig: WorkerConfig{
				Handler:              handler,
				HandlerIdentity:      configkit.MustNewIdentity("<handler-name>", "<handler-key>"),
				Packer:               packer,
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

	Describe("func Run()", func() {
		When("a command is received", func() {
			It("writes events recorded by the handler", func() {
				go func() {
					defer GinkgoRecover()

					executeCommand(
						ctx,
						commands,
						NewParcel("<command>", MessageC1),
					)

					cancel()
				}()

				err := worker.Run(ctx)
				Expect(err).To(Equal(context.Canceled))

				expectEvents(
					eventStore,
					"<handler-key>",
					"<instance>",
					0,
					[]*envelopespec.Envelope{
						{
							MessageId:         "0",
							CausationId:       "<command>",
							CorrelationId:     "<correlation>",
							SourceApplication: packer.Application,
							SourceHandler:     marshalkit.MustMarshalEnvelopeIdentity(worker.HandlerIdentity),
							SourceInstanceId:  "<instance>",
							CreatedAt:         "2000-01-01T00:00:00Z",
							Description:       "{E1}",
							PortableName:      MessageEPortableName,
							MediaType:         MessageE1Packet.MediaType,
							Data:              MessageE1Packet.Data,
						},
					},
				)
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
				err := worker.Run(ctx)
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

// expectEvents reads all events from store starting from offset and asserts
// that they are equal to expectedEvents.
func expectEvents(
	reader EventReader,
	hk, id string,
	offset uint64,
	expectedEvents []*envelopespec.Envelope,
) {
	var producedEvents []*envelopespec.Envelope

	for {
		events, more, err := reader.ReadEvents(
			context.Background(),
			hk,
			id,
			offset,
		)
		Expect(err).ShouldNot(HaveOccurred())

		producedEvents = append(producedEvents, events...)

		if !more {
			break
		}

		offset += uint64(len(events))
	}

	Expect(producedEvents).To(EqualX(expectedEvents))
}

// executeCommand executes a command by sending it to a command channel and
// waiting for it to complete.
func executeCommand(
	ctx context.Context,
	commands chan<- Command,
	command parcel.Parcel,
) {
	result := make(chan error)

	cmd := Command{
		Context: ctx,
		Parcel:  command,
		Result:  result,
	}

	select {
	case <-ctx.Done():
		Expect(ctx.Err()).ShouldNot(HaveOccurred())
	case commands <- cmd:
	}

	select {
	case <-ctx.Done():
		Expect(ctx.Err()).ShouldNot(HaveOccurred())
	case err := <-result:
		Expect(err).ShouldNot(HaveOccurred())
	}
}
