package cluster

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/dogmatiq/enginekit/collections/maps"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/persistencekit/kv"
	"github.com/dogmatiq/veracity/internal/cluster/internal/registrypb"
	"github.com/dogmatiq/veracity/internal/fsm"
	"github.com/dogmatiq/veracity/internal/signaling"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// DefaultRenewInterval is the default interval at which a [Registrar]
	// renews a node's registration.
	//
	// The registration period is always set to 2 * DefaultRenewInterval.
	DefaultRenewInterval = 10 * time.Second

	// DefaultRegistryPollInterval is the default interval at which the registry
	// polls the underlying key-value store for changes.
	DefaultRegistryPollInterval = 3 * time.Second
)

// Registrar registers and periodically renews a node's registration with the
// registry.
type Registrar struct {
	Keyspaces     kv.BinaryStore
	Node          Node
	RenewInterval time.Duration
	Shutdown      signaling.Latch
	Logger        *slog.Logger

	keyspace kv.Keyspace[*uuidpb.UUID, *registrypb.Registration]
	interval time.Duration
}

// Run starts the registrar.
func (r *Registrar) Run(ctx context.Context) error {
	var err error
	r.keyspace, err = newKVStore(r.Keyspaces).Open(ctx, registryKeyspace)
	if err != nil {
		return err
	}
	defer r.keyspace.Close()

	r.interval = r.RenewInterval
	if r.interval <= 0 {
		r.interval = DefaultRenewInterval
	}

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	err = r.register(ctx)

	for err == nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-r.Shutdown.Signaled():
			return r.deregister(ctx)
		case <-ticker.C:
			err = r.renew(ctx)
		}
	}

	return err
}

// register adds a node to the registry.
func (r *Registrar) register(ctx context.Context) error {
	expiresAt, err := r.saveRegistration(ctx)
	if err != nil {
		return err
	}

	r.Logger.DebugContext(
		ctx,
		"cluster node registered",
		slog.String("node_id", r.Node.ID.AsString()),
		slog.String("addresses", strings.Join(r.Node.Addresses, ", ")),
		slog.Time("expires_at", expiresAt),
		slog.Duration("renew_interval", r.interval),
	)

	return nil
}

// deregister removes a node from the registry.
func (r *Registrar) deregister(ctx context.Context) error {
	if err := r.deleteRegistration(ctx); err != nil {
		return err
	}

	r.Logger.DebugContext(
		ctx,
		"cluster node deregistered",
		slog.String("node_id", r.Node.ID.AsString()),
	)

	return nil
}

// renew updates a node's registration expiry time.
func (r *Registrar) renew(ctx context.Context) error {
	reg, err := r.keyspace.Get(ctx, r.Node.ID)
	if err != nil {
		return err
	}
	if reg == nil {
		return errors.New("cluster node not registered")
	}

	if reg.ExpiresAt.AsTime().Before(time.Now()) {
		if err := r.deleteRegistration(ctx); err != nil {
			return err
		}
		return errors.New("cluster node registration expired")
	}

	expiresAt, err := r.saveRegistration(ctx)
	if err != nil {
		return err
	}

	r.Logger.DebugContext(
		ctx,
		"cluster node registration renewed",
		slog.String("node_id", r.Node.ID.AsString()),
		slog.Time("expires_at", expiresAt),
		slog.Duration("renew_interval", r.interval),
	)

	return nil
}

// saveRegistration saves the node's registration information to the registry.
func (r *Registrar) saveRegistration(
	ctx context.Context,
) (time.Time, error) {
	expiresAt := time.Now().Add(r.interval * 2)

	return expiresAt, r.keyspace.Set(
		ctx,
		r.Node.ID,
		&registrypb.Registration{
			Node: &registrypb.Node{
				Id:        r.Node.ID,
				Addresses: r.Node.Addresses,
			},
			ExpiresAt: timestamppb.New(expiresAt),
		},
	)
}

// deleteRegistration removes a registration from the registry.
func (r *Registrar) deleteRegistration(
	ctx context.Context,
) error {
	return r.keyspace.Set(ctx, r.Node.ID, nil)
}

// RegistryObserver emits events about changes to the nodes in the registry.
type RegistryObserver struct {
	Keyspaces         kv.BinaryStore
	MembershipChanged chan<- MembershipChanged
	Shutdown          signaling.Latch
	PollInterval      time.Duration

	keyspace     kv.Keyspace[*uuidpb.UUID, *registrypb.Registration]
	nodes        maps.Proto[*uuidpb.UUID, Node]
	readyForPoll *time.Ticker
}

// Run starts the observer.
func (o *RegistryObserver) Run(ctx context.Context) error {
	var err error
	o.keyspace, err = newKVStore(o.Keyspaces).Open(ctx, registryKeyspace)
	if err != nil {
		return err
	}
	defer o.keyspace.Close()

	interval := o.PollInterval
	if interval <= 0 {
		interval = DefaultRegistryPollInterval
	}

	o.readyForPoll = time.NewTicker(interval)
	defer o.readyForPoll.Stop()

	return fsm.Start(ctx, o.pollState)
}

// idleState waits for until it's time to poll the registry.
func (o *RegistryObserver) idleState(ctx context.Context) fsm.Action {
	select {
	case <-ctx.Done():
		return fsm.Stop()
	case <-o.Shutdown.Signaled():
		return fsm.Stop()
	case <-o.readyForPoll.C:
		return fsm.EnterState(o.pollState)
	}
}

// pollState loads the current set of nodes from the registry to produce a
// [MembershipChange] event describing the changes since the last poll.
func (o *RegistryObserver) pollState(ctx context.Context) fsm.Action {
	nodes, err := o.loadNodes(ctx)
	if err != nil {
		return fsm.Fail(err)
	}

	ev := MembershipChanged{}

	for id, node := range o.nodes.All() {
		if !nodes.Has(id) {
			ev.Deregistered = append(ev.Deregistered, node)
		}
	}

	for id, node := range nodes.All() {
		if !o.nodes.Has(id) {
			ev.Registered = append(ev.Registered, node)
		}
	}

	if len(ev.Registered) == 0 && len(ev.Deregistered) == 0 {
		return fsm.EnterState(o.idleState)
	}

	return fsm.With(ev).EnterState(o.publishState)
}

// publishState publishes a [MembershipChange] event.
func (o *RegistryObserver) publishState(ctx context.Context, ev MembershipChanged) fsm.Action {
	select {
	case <-ctx.Done():
		return fsm.Stop()
	case <-o.readyForPoll.C:
		return fsm.EnterState(o.pollState)
	case o.MembershipChanged <- ev:
		for _, node := range ev.Registered {
			o.nodes.Set(node.ID, node)
		}

		for _, node := range ev.Deregistered {
			o.nodes.Remove(node.ID)
		}

		return fsm.EnterState(o.idleState)
	}
}

// loadNodes loads the current set of nodes from the registry.
func (o *RegistryObserver) loadNodes(ctx context.Context) (*maps.Proto[*uuidpb.UUID, Node], error) {
	nodes := maps.NewProto[*uuidpb.UUID, Node]()

	return nodes, o.keyspace.Range(
		ctx,
		func(
			ctx context.Context,
			id *uuidpb.UUID,
			reg *registrypb.Registration,
		) (bool, error) {
			if reg.ExpiresAt.AsTime().After(time.Now()) {
				nodes.Set(
					reg.Node.Id,
					Node{
						ID:        reg.Node.Id,
						Addresses: reg.Node.Addresses,
					},
				)
			} else if err := o.keyspace.Set(ctx, id, nil); err != nil {
				return false, err
			}

			return true, nil
		},
	)
}
