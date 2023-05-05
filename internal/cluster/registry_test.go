package cluster_test

import (
	"context"
	"errors"
	"time"

	. "github.com/dogmatiq/veracity/internal/cluster"
	"github.com/dogmatiq/veracity/persistence/driver/memory"
	"github.com/dogmatiq/veracity/persistence/kv"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/exp/slog"
)

var _ = Describe("type Registry", func() {
	var (
		ctx      context.Context
		cancel   context.CancelFunc
		keyspace kv.Keyspace
		registry *Registry
	)

	watch := func(notify chan<- MembershipChange) func() {
		ctx, cancel := context.WithCancel(ctx)
		result := make(chan error, 1)

		go func() {
			err := registry.Watch(ctx, notify)
			result <- err
		}()

		return func() {
			cancel()
			err := <-result
			Expect(errors.Is(err, context.Canceled)).To(BeTrue())
		}
	}

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
		DeferCleanup(cancel)

		store := &memory.KeyValueStore{}

		var err error
		keyspace, err = store.Open(context.Background(), "registry")
		Expect(err).ShouldNot(HaveOccurred())
		DeferCleanup(keyspace.Close)

		registry = &Registry{
			Keyspace: keyspace,

			// Note that we use a heartbeat interval that's larger than the poll
			// interval so that we don't need to setup heartbeats in every test.
			PollInterval:      10 * time.Millisecond,
			HeartbeatInterval: 50 * time.Millisecond,

			Logger: slog.New(
				slog.HandlerOptions{
					Level: slog.LevelDebug,
				}.NewTextHandler(GinkgoWriter),
			),
		}
	})

	Describe("func Watch()", func() {
		It("notifies about nodes that are registered before watching starts", func() {
			node := Node{
				ID: uuid.New(),
			}

			err := registry.Register(ctx, node)
			Expect(err).ShouldNot(HaveOccurred())

			notify := make(chan MembershipChange)
			stop := watch(notify)
			DeferCleanup(stop)

			select {
			case <-ctx.Done():
				Expect(ctx.Err()).ShouldNot(HaveOccurred())
			case change := <-notify:
				cancel()
				Expect(change).To(Equal(
					MembershipChange{
						Joins: []Node{node},
					},
				))
			}
		})

		It("does not notify about nodes that are deregistered before watching starts", func() {
			node := Node{
				ID: uuid.New(),
			}

			err := registry.Register(ctx, node)
			Expect(err).ShouldNot(HaveOccurred())

			err = registry.Deregister(ctx, node.ID)
			Expect(err).ShouldNot(HaveOccurred())

			notify := make(chan MembershipChange)
			stop := watch(notify)
			DeferCleanup(stop)

			select {
			case <-ctx.Done():
				Expect(ctx.Err()).ShouldNot(HaveOccurred())
			case <-time.After(registry.HeartbeatInterval):
				cancel()
			case <-notify:
				Fail("unexpected notification")
			}
		})

		It("notifies about nodes that are registered after watching starts", func() {
			node := Node{
				ID: uuid.New(),
			}

			go func() {
				// Let the first poll happen before we register the node.
				time.Sleep(registry.PollInterval)

				defer GinkgoRecover()
				err := registry.Register(ctx, node)
				Expect(err).ShouldNot(HaveOccurred())
			}()

			notify := make(chan MembershipChange)
			stop := watch(notify)
			DeferCleanup(stop)

			select {
			case <-ctx.Done():
				Expect(ctx.Err()).ShouldNot(HaveOccurred())
			case change := <-notify:
				cancel()
				Expect(change).To(Equal(
					MembershipChange{
						Joins: []Node{node},
					},
				))
			}
		})

		It("notifies when a node is deregistered", func() {
			node := Node{
				ID: uuid.New(),
			}

			err := registry.Register(ctx, node)
			Expect(err).ShouldNot(HaveOccurred())

			notify := make(chan MembershipChange)
			stop := watch(notify)
			DeferCleanup(stop)

			select {
			case <-ctx.Done():
				Expect(ctx.Err()).ShouldNot(HaveOccurred())
			case <-notify:
				err := registry.Deregister(ctx, node.ID)
				Expect(err).ShouldNot(HaveOccurred())
			}

			select {
			case <-ctx.Done():
				Expect(ctx.Err()).ShouldNot(HaveOccurred())
			case change := <-notify:
				cancel()
				Expect(change).To(Equal(
					MembershipChange{
						Leaves: []Node{node},
					},
				))
			}
		})

		It("notifies when a node expires", func() {
			node := Node{
				ID: uuid.New(),
			}

			err := registry.Register(ctx, node)
			Expect(err).ShouldNot(HaveOccurred())

			notify := make(chan MembershipChange)
			stop := watch(notify)
			DeferCleanup(stop)

			select {
			case <-ctx.Done():
				Expect(ctx.Err()).ShouldNot(HaveOccurred())
			case <-notify:
				// join
			}

			select {
			case <-ctx.Done():
				Expect(ctx.Err()).ShouldNot(HaveOccurred())
			case change := <-notify:
				cancel()
				Expect(change).To(Equal(
					MembershipChange{
						Leaves: []Node{node},
					},
				))
			}
		})

		It("does not notify when a node maintains its heartbeat correctly", func() {
			node := Node{
				ID: uuid.New(),
			}

			err := registry.Register(ctx, node)
			Expect(err).ShouldNot(HaveOccurred())

			result := make(chan error, 1)
			go func() {
				for {
					time.Sleep(registry.HeartbeatInterval)

					err := registry.Heartbeat(ctx, node.ID)
					if err != nil {
						result <- err
						return
					}
				}
			}()

			notify := make(chan MembershipChange)
			stop := watch(notify)
			DeferCleanup(stop)

			select {
			case <-ctx.Done():
				Expect(ctx.Err()).ShouldNot(HaveOccurred())
			case <-notify:
				// join
			}

			select {
			case <-ctx.Done():
				Expect(ctx.Err()).ShouldNot(HaveOccurred())
			case <-time.After(registry.HeartbeatInterval * 5):
				cancel()
				err := <-result
				Expect(errors.Is(err, context.Canceled)).To(BeTrue())
			case <-notify:
				Fail("unexpected notification")
			}
		})
	})
})
