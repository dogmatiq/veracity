package cluster_test

import (
	"testing"
	"time"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	. "github.com/dogmatiq/veracity/internal/cluster"
	"github.com/dogmatiq/veracity/internal/test"
	"github.com/dogmatiq/veracity/persistence/driver/memory"
)

func TestRegistry(t *testing.T) {
	t.Parallel()

	setup := func(t test.TestingT) (
		deps struct {
			Node              Node
			Registrar         *Registrar
			MembershipChanged chan MembershipChanged
			Observer          *RegistryObserver
		},
	) {
		keyspaces := &memory.KeyValueStore{}

		deps.Node = Node{
			ID: uuidpb.Generate(),
			Addresses: []string{
				"10.0.0.1:50555",
				"129.168.0.1:50555",
			},
		}

		deps.Registrar = &Registrar{
			Keyspaces:     keyspaces,
			Node:          deps.Node,
			RenewInterval: 10 * time.Millisecond,
			Logger:        test.NewLogger(t),
		}

		deps.MembershipChanged = make(chan MembershipChanged)

		deps.Observer = &RegistryObserver{
			Keyspaces:         keyspaces,
			MembershipChanged: deps.MembershipChanged,
			PollInterval:      50 * time.Millisecond,
		}

		return deps
	}

	t.Run("it observes registration and deregistration", func(t *testing.T) {
		t.Parallel()

		tctx := test.WithContext(t)
		deps := setup(tctx)

		t.Log("start the observer before the registrar")

		test.
			RunInBackground(t, "observer", deps.Observer.Run).
			UntilTestEnds()

		t.Log("wait several poll intervals to ensure that the observer is running")

		test.ExpectChannelToBlockForDuration(
			t,
			3*deps.Observer.PollInterval,
			deps.MembershipChanged,
		)

		t.Log("start the registrar and await notification of registration")

		test.
			RunInBackground(t, "registrar", deps.Registrar.Run).
			BeforeTestEnds()

		test.
			ExpectChannelToReceive(
				tctx,
				deps.MembershipChanged,
				MembershipChanged{
					Registered: []Node{
						deps.Node,
					},
				},
			)

		t.Log("wait several renew/poll intervals to ensure that the node's registration is renewed properly")

		test.
			ExpectChannelToBlockForDuration(
				tctx,
				3*(deps.Registrar.RenewInterval+deps.Observer.PollInterval),
				deps.MembershipChanged,
			)

		t.Log("shutdown the registrar and await notification of deregistration")

		deps.Registrar.Shutdown.Signal()

		test.
			ExpectChannelToReceive(
				tctx,
				deps.MembershipChanged,
				MembershipChanged{
					Deregistered: []Node{
						deps.Node,
					},
				},
			)
	})

	t.Run("it observes nodes that are registered before the observer starts", func(t *testing.T) {
		t.Parallel()

		tctx := test.WithContext(t)
		deps := setup(tctx)

		t.Log("start the registrar before the observer")

		test.
			RunInBackground(t, "registrar", deps.Registrar.Run).
			UntilTestEnds()

		t.Log("wait several renew intervals to ensure that the node is registered")

		test.
			ExpectChannelToBlockForDuration(
				tctx,
				3*deps.Registrar.RenewInterval,
				deps.MembershipChanged,
			)

		t.Log("start an observer and await notification of registration")

		test.
			RunInBackground(t, "observer", deps.Observer.Run).
			UntilTestEnds()

		test.
			ExpectChannelToReceive(
				tctx,
				deps.MembershipChanged,
				MembershipChanged{
					Registered: []Node{
						deps.Node,
					},
				},
			)
	})

	t.Run("it does not observe nodes that are deregistered before the observer starts", func(t *testing.T) {
		t.Parallel()

		tctx := test.WithContext(t)
		deps := setup(tctx)

		t.Log("start the registrar and shut it down immediately")

		task := test.
			RunInBackground(t, "registrar", deps.Registrar.Run).
			BeforeTestEnds()

		deps.Registrar.Shutdown.Signal()
		test.
			ExpectChannelToClose(
				tctx,
				task.Done(),
			)

		t.Log("start an observer and ensure it is never notified")

		test.
			RunInBackground(t, "observer", deps.Observer.Run).
			UntilTestEnds()

		test.
			ExpectChannelToBlockForDuration(
				tctx,
				3*deps.Observer.PollInterval,
				deps.MembershipChanged,
			)
	})

	t.Run("it observes nodes that leave the cluster due to registration expiry", func(t *testing.T) {
		t.Parallel()

		tctx := test.WithContext(t)
		deps := setup(tctx)

		t.Log("start the registrar and observer and await notification of registration")

		registrar := test.
			RunInBackground(t, "registrar", deps.Registrar.Run).
			UntilStopped()

		test.
			RunInBackground(t, "observer", deps.Observer.Run).
			UntilTestEnds()

		test.ExpectChannelToReceive(
			tctx,
			deps.MembershipChanged,
			MembershipChanged{
				Registered: []Node{
					deps.Node,
				},
			},
		)

		t.Log("stop the registrary forcefully and await notification of deregistration")

		registrar.Stop()

		test.ExpectChannelToReceive(
			tctx,
			deps.MembershipChanged,
			MembershipChanged{
				Deregistered: []Node{
					deps.Node,
				},
			},
		)
	})
}
