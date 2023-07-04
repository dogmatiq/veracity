package cluster_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	. "github.com/dogmatiq/veracity/internal/cluster"
	"github.com/dogmatiq/veracity/internal/testutil"
	"github.com/dogmatiq/veracity/persistence/driver/memory"
	"github.com/google/go-cmp/cmp"
)

func TestRegistry(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (
		context.Context,
		context.CancelFunc,
		*Registry,
	) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

		store := &memory.KeyValueStore{}
		keyspace, err := store.Open(context.Background(), "registry")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			keyspace.Close()
		})

		return ctx, cancel, &Registry{
			Keyspace: keyspace,

			// Note that we use a heartbeat interval that's larger than the poll
			// interval so that we don't need to setup heartbeats in every test.
			PollInterval:      10 * time.Millisecond,
			HeartbeatInterval: 50 * time.Millisecond,

			Logger: testutil.NewLogger(t),
		}
	}

	watch := func(
		ctx context.Context,
		reg *Registry,
		notify chan<- MembershipChange,
	) func() {
		ctx, cancel := context.WithCancel(ctx)

		result := make(chan error, 1)
		go func() {
			result <- reg.Watch(ctx, notify)
		}()

		return func() {
			cancel()
			err := <-result

			if !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		}
	}

	node := Node{
		ID: uuidpb.Generate(),
		Addresses: []string{
			"10.0.0.1:50555",
			"129.168.0.1:50555",
		},
	}

	t.Run("it notifies about nodes that are registered before watching starts", func(t *testing.T) {
		t.Parallel()

		ctx, cancel, reg := setup(t)
		defer cancel()

		delay, err := reg.Register(ctx, node)
		if err != nil {
			t.Fatal(err)
		}

		expectDelay := 50 * time.Millisecond
		if delay != expectDelay {
			t.Fatalf("got %s, want %s", delay, expectDelay)
		}

		notify := make(chan MembershipChange)
		stop := watch(ctx, reg, notify)
		defer stop()

		expect := MembershipChange{
			Joins: []Node{node},
		}

		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case actual := <-notify:
			cancel()
			if diff := cmp.Diff(actual, expect); diff != "" {
				t.Fatal(diff)
			}
		}
	})

	t.Run("it does not notify about nodes that are deregistered before watching starts", func(t *testing.T) {
		t.Parallel()

		ctx, cancel, reg := setup(t)
		defer cancel()

		delay, err := reg.Register(ctx, node)
		if err != nil {
			t.Fatal(err)
		}

		expectDelay := 50 * time.Millisecond
		if delay != expectDelay {
			t.Fatalf("got %s, want %s", delay, expectDelay)
		}

		if err := reg.Deregister(ctx, node.ID); err != nil {
			t.Fatal(err)
		}

		notify := make(chan MembershipChange)
		stop := watch(ctx, reg, notify)
		defer stop()

		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(reg.HeartbeatInterval):
			cancel()
		case <-notify:
			t.Fatal("unexpected notification")
		}
	})

	t.Run("it notifies about nodes that are registered after watching starts", func(t *testing.T) {
		t.Parallel()

		ctx, cancel, reg := setup(t)
		defer cancel()

		go func() {
			// Let the first poll happen before we register the node.
			time.Sleep(reg.PollInterval)

			if _, err := reg.Register(ctx, node); err != nil {
				panic(err)
			}
		}()

		notify := make(chan MembershipChange)
		stop := watch(ctx, reg, notify)
		defer stop()

		expect := MembershipChange{
			Joins: []Node{node},
		}

		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case actual := <-notify:
			cancel()
			if diff := cmp.Diff(actual, expect); diff != "" {
				t.Fatal(diff)
			}
		}
	})

	t.Run("it notifies when a node is deregistered", func(t *testing.T) {
		t.Parallel()

		ctx, cancel, reg := setup(t)
		defer cancel()

		if _, err := reg.Register(ctx, node); err != nil {
			t.Fatal(err)
		}

		notify := make(chan MembershipChange)
		stop := watch(ctx, reg, notify)
		defer stop()

		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-notify:
			if err := reg.Deregister(ctx, node.ID); err != nil {
				t.Fatal(err)
			}
		}

		expect := MembershipChange{
			Leaves: []Node{node},
		}

		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case actual := <-notify:
			cancel()
			if diff := cmp.Diff(actual, expect); diff != "" {
				t.Fatal(diff)
			}
		}
	})

	t.Run("it notifies when a node expires", func(t *testing.T) {
		t.Parallel()

		ctx, cancel, reg := setup(t)
		defer cancel()

		if _, err := reg.Register(ctx, node); err != nil {
			t.Fatal(err)
		}

		notify := make(chan MembershipChange)
		stop := watch(ctx, reg, notify)
		defer stop()

		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-notify:
			// join
		}

		expect := MembershipChange{
			Leaves: []Node{node},
		}

		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case actual := <-notify:
			cancel()
			if diff := cmp.Diff(actual, expect); diff != "" {
				t.Fatal(diff)
			}
		}
	})

	t.Run("it does not notify when a node maintains its heartbeat correctly", func(t *testing.T) {
		t.Parallel()

		ctx, cancel, reg := setup(t)
		defer cancel()

		delay, err := reg.Register(ctx, node)
		if err != nil {
			t.Fatal(err)
		}

		result := make(chan error, 1)
		go func() {
			delay := delay
			var err error

			for {
				time.Sleep(delay)

				delay, err = reg.Heartbeat(ctx, node.ID)
				if err != nil {
					result <- err
					return
				}
			}
		}()

		notify := make(chan MembershipChange)
		stop := watch(ctx, reg, notify)
		defer stop()

		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-notify:
			// join
		}

		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(reg.HeartbeatInterval * 5):
			cancel()
			err := <-result
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("got %s, want %s", err, context.Canceled)
			}
		case <-notify:
			t.Fatal("unexpected notification")
		}
	})
}
