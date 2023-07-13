package cluster

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/cluster/internal/registrypb"
	"github.com/dogmatiq/veracity/internal/fsm"
	"github.com/dogmatiq/veracity/internal/protobuf/protokv"
	"github.com/dogmatiq/veracity/internal/signaling"
	"github.com/dogmatiq/veracity/persistence/kv"
	"golang.org/x/exp/slog"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// DefaultRenewInterval is the default interval at which a [Registrar]
	// renews a node's registration.
	//
	// The registration period is always set to 2 * DefaultRenewInterval.
	DefaultRenewInterval = 10 * time.Second

	// RegistryKeyspace is the name of the keyspace that contains registry data.
	RegistryKeyspace = "cluster.registry"

	// DefaultRegistryPollInterval is the default interval at which the registry
	// polls the underlying key-value store for changes.
	DefaultRegistryPollInterval = 3 * time.Second
)

// Registrar registers and periodically renews a node's registration with the
// registry.
type Registrar struct {
	Keyspaces     kv.Store
	Node          Node
	RenewInterval time.Duration
	Shutdown      signaling.Latch
	Logger        *slog.Logger

	keyspace kv.Keyspace
	interval time.Duration
}

// Run starts the registrar.
func (r *Registrar) Run(ctx context.Context) error {
	var err error
	r.keyspace, err = r.Keyspaces.Open(ctx, RegistryKeyspace)
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

	r.Logger.DebugCtx(
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

	r.Logger.DebugCtx(
		ctx,
		"cluster node deregistered",
		slog.String("node_id", r.Node.ID.AsString()),
	)

	return nil
}

// renew updates a node's registration expiry time.
func (r *Registrar) renew(ctx context.Context) error {
	reg, ok, err := protokv.Get[*registrypb.Registration](
		ctx,
		r.keyspace,
		r.Node.ID.AsBytes(),
	)
	if err != nil {
		return err
	}
	if !ok {
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

	r.Logger.DebugCtx(
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
	err := protokv.Set(
		ctx,
		r.keyspace,
		r.Node.ID.AsBytes(),
		&registrypb.Registration{
			Node: &registrypb.Node{
				Id:        r.Node.ID,
				Addresses: r.Node.Addresses,
			},
			ExpiresAt: timestamppb.New(expiresAt),
		},
	)

	return expiresAt, err
}

// deleteRegistration removes a registration from the registry.
func (r *Registrar) deleteRegistration(
	ctx context.Context,
) error {
	return r.keyspace.Set(
		ctx,
		r.Node.ID.AsBytes(),
		nil,
	)
}

// MembershipChanged is an event that indicates a change in membership to the
// cluster.
type MembershipChanged struct {
	Registered, Deregistered []Node
}

// RegistryObserver emits events about changes to the nodes in the registry.
type RegistryObserver struct {
	Keyspaces         kv.Store
	MembershipChanged chan<- MembershipChanged
	Shutdown          signaling.Latch
	PollInterval      time.Duration

	keyspace     kv.Keyspace
	nodes        uuidpb.Map[Node]
	readyForPoll *time.Ticker
}

// Run starts the observer.
func (o *RegistryObserver) Run(ctx context.Context) error {
	var err error
	o.keyspace, err = o.Keyspaces.Open(ctx, RegistryKeyspace)
	if err != nil {
		return err
	}
	defer o.keyspace.Close()

	o.nodes = uuidpb.Map[Node]{}

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

	for id, node := range o.nodes {
		if _, ok := nodes[id]; !ok {
			ev.Deregistered = append(ev.Deregistered, node)
		}
	}

	for id, node := range nodes {
		if _, ok := o.nodes[id]; !ok {
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
			o.nodes.Delete(node.ID)
		}

		return fsm.EnterState(o.idleState)
	}
}

// loadNodes loads the current set of nodes from the registry.
func (o *RegistryObserver) loadNodes(ctx context.Context) (uuidpb.Map[Node], error) {
	nodes := uuidpb.Map[Node]{}

	return nodes, protokv.RangeAll(
		ctx,
		o.keyspace,
		func(
			ctx context.Context,
			k []byte,
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
			} else if err := o.keyspace.Set(ctx, k, nil); err != nil {
				return false, err
			}

			return true, nil
		},
	)
}
