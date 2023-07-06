package cluster_test

import (
	"context"
	"testing"
	"time"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	. "github.com/dogmatiq/veracity/internal/cluster"
	"github.com/dogmatiq/veracity/internal/test"
	"github.com/dogmatiq/veracity/persistence/driver/memory"
)

func TestRegistry(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (x struct {
		Context           context.Context
		Cancel            context.CancelFunc
		Node              Node
		MembershipChanges chan MembershipChange
		Registrar         *Registrar
		Observer          *RegistryObserver
	}) {
		t.Parallel()

		keyspaces := &memory.KeyValueStore{}

		x.Context, x.Cancel = test.ContextWithTimeout(t, 1*time.Second)

		x.Node = Node{
			ID: uuidpb.Generate(),
			Addresses: []string{
				"10.0.0.1:50555",
				"129.168.0.1:50555",
			},
		}

		x.MembershipChanges = make(chan MembershipChange)

		x.Registrar = &Registrar{
			Keyspaces:     keyspaces,
			Node:          x.Node,
			RenewInterval: 10 * time.Millisecond,
			Logger:        test.NewLogger(t),
		}

		x.Observer = &RegistryObserver{
			Keyspaces:    keyspaces,
			PollInterval: 50 * time.Millisecond,
		}

		x.Observer.MembershipChanged.Subscribe(x.MembershipChanges)

		return x
	}

	t.Run("it observes registration and deregistration", func(t *testing.T) {
		x := setup(t)

		test.CompleteBeforeTestEnds(t, x.Registrar.Run)
		test.RunUntilTestEnds(t, x.Observer.Run)

		test.ExpectToReceive(
			x.Context,
			t,
			x.MembershipChanges,
			MembershipChange{
				Registered: []Node{
					x.Node,
				},
			},
		)

		select {
		case <-x.Context.Done():
			t.Fatal(x.Context.Err())
		case <-x.MembershipChanges:
			t.Fatal("unexpected membership change")
		case <-time.After(2 * (x.Registrar.RenewInterval + x.Observer.PollInterval)):
			// We've waited long enough for a couple of renewals and a couple of
			// polls to occur. We don't expect any membership change during this
			// period as the [Registrar] renews the node's registration.
		}

		x.Registrar.Shutdown.Latch()

		test.ExpectToReceive(
			x.Context,
			t,
			x.MembershipChanges,
			MembershipChange{
				Deregistered: []Node{
					x.Node,
				},
			},
		)
	})

	t.Run("it observes nodes that are registered before the observer starts", func(t *testing.T) {
		x := setup(t)

		test.RunUntilTestEnds(t, x.Registrar.Run)

		select {
		case <-x.Context.Done():
			t.Fatal(x.Context.Err())
		case <-x.MembershipChanges:
			t.Fatal("unexpected membership change")
		case <-time.After(2 * x.Observer.PollInterval):
		}

		test.RunUntilTestEnds(t, x.Observer.Run)

		test.ExpectToReceive(
			x.Context,
			t,
			x.MembershipChanges,
			MembershipChange{
				Registered: []Node{
					x.Node,
				},
			},
		)
	})

	t.Run("it does observe nodes that are deregistered before the observer starts", func(t *testing.T) {
		x := setup(t)

		test.CompleteBeforeTestEnds(t, x.Registrar.Run)
		x.Registrar.Shutdown.Latch()

		test.RunUntilTestEnds(t, x.Observer.Run)

		select {
		case <-x.Context.Done():
			t.Fatal(x.Context.Err())
		case <-x.MembershipChanges:
			t.Fatal("unexpected membership change")
		case <-time.After(2 * x.Observer.PollInterval):
			// We've allowed enough time for polls to occur and have not seen a
			// membership change.
		}
	})

	t.Run("it observes nodes that leave the cluster due to registration expiry", func(t *testing.T) {
		x := setup(t)

		_, stopRegistrar := test.Run(t, x.Registrar.Run)
		test.RunUntilTestEnds(t, x.Observer.Run)

		test.ExpectToReceive(
			x.Context,
			t,
			x.MembershipChanges,
			MembershipChange{
				Registered: []Node{
					x.Node,
				},
			},
		)

		stopRegistrar()

		test.ExpectToReceive(
			x.Context,
			t,
			x.MembershipChanges,
			MembershipChange{
				Deregistered: []Node{
					x.Node,
				},
			},
		)
	})
}
