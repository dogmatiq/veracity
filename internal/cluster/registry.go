package cluster

import (
	"context"
	"fmt"
	"time"

	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/cluster/internal/registrypb"
	"github.com/dogmatiq/veracity/internal/protobuf/protokv"
	"github.com/dogmatiq/veracity/persistence/kv"
	"github.com/google/uuid"
	"golang.org/x/exp/slog"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// DefaultRegistryPollInterval is the default interval at which the registry polls
	// the underlying key-value store for changes.
	DefaultRegistryPollInterval = 3 * time.Second

	// DefaultHeartbeatInterval is the default interval at which nodes send
	// heartbeats to the registry.
	DefaultHeartbeatInterval = 10 * time.Second

	// RegistryKeyspace is the name of the keyspace that contains registry data.
	RegistryKeyspace = "cluster.registry"
)

// MembershipChange is an event that indicates a change in membership to the
// cluster.
type MembershipChange struct {
	Joins  []Node
	Leaves []Node
}

// A Registry tracks the nodes that are members of the cluster.
type Registry struct {
	Keyspace          kv.Keyspace
	PollInterval      time.Duration
	HeartbeatInterval time.Duration
	Logger            *slog.Logger
}

// Register adds a node to the registry.
//
// It returns the delay before the node should send a heartbeat.
func (r *Registry) Register(ctx context.Context, n Node) (time.Duration, error) {
	interval := r.HeartbeatInterval
	if interval <= 0 {
		interval = DefaultHeartbeatInterval
	}

	expiresAt := time.Now().Add(interval * 2)

	if err := protokv.Set(
		ctx,
		r.Keyspace,
		n.ID[:],
		marshalNode(n, expiresAt),
	); err != nil {
		return 0, fmt.Errorf("unable to register node: %w", err)
	}

	r.Logger.DebugCtx(
		ctx,
		"member node registered",
		slog.String("node_id", n.ID.String()),
		slog.Duration("heartbeat_interval", interval),
		slog.Time("expires_at", expiresAt),
	)

	return interval, nil
}

// Deregister removes the node with the given ID from the registry.
func (r *Registry) Deregister(ctx context.Context, id uuid.UUID) error {
	if err := r.Keyspace.Set(ctx, id[:], nil); err != nil {
		return fmt.Errorf("unable to deregister node: %w", err)
	}

	r.Logger.DebugCtx(
		ctx,
		"member node deregistered",
		slog.String("node_id", id.String()),
	)

	return nil
}

// Heartbeat updates the expiry time of the node with the given ID.
//
// It returns the delay before the node should send its next heartbeat.
func (r *Registry) Heartbeat(ctx context.Context, id uuid.UUID) (time.Duration, error) {
	n, ok, err := protokv.Get[*registrypb.Node](
		ctx,
		r.Keyspace,
		id[:],
	)
	if err != nil {
		return 0, fmt.Errorf("unable to update heartbeat: %w", err)
	}
	if !ok {
		return 0, fmt.Errorf("node not registered")
	}
	if expired, err := r.deleteIfExpired(ctx, n); expired {
		if err != nil {
			return 0, fmt.Errorf("unable to update heartbeat: %w", err)
		}
		return 0, fmt.Errorf("node has expired")
	}

	interval := r.HeartbeatInterval
	if interval <= 0 {
		interval = DefaultHeartbeatInterval
	}

	expiresAt := time.Now().Add(interval * 2)
	n.ExpiresAt = timestamppb.New(expiresAt)

	if err := protokv.Set(
		ctx,
		r.Keyspace,
		id[:],
		n,
	); err != nil {
		return 0, fmt.Errorf("unable to register node: %w", err)
	}

	r.Logger.DebugCtx(
		ctx,
		"member node expiry extended",
		slog.String("node_id", n.Id.AsString()),
		slog.Duration("heartbeat_interval", interval),
		slog.Time("expires_at", expiresAt),
	)

	return interval, nil
}

// Watch sends notifications about changes to the cluster's membership to the
// given channel.
func (r *Registry) Watch(
	ctx context.Context,
	ch chan<- MembershipChange,
) error {
	interval := r.PollInterval
	if interval <= 0 {
		interval = DefaultRegistryPollInterval
	}

	r.Logger.DebugCtx(
		ctx,
		"watching registry for membership changes",
		slog.Duration("poll_interval", interval),
	)

	poll := time.NewTicker(interval)
	defer poll.Stop()

	var before map[uuid.UUID]Node

	for {
		after, err := r.members(ctx)
		if err != nil {
			return fmt.Errorf("unable to query cluster membership: %w", err)
		}

		change := membershipDiff(before, after)

		// Setup a channel to notify, but leave it nil if there are no changes
		// (write to a nil channel blocks forever).
		var notify chan<- MembershipChange
		if len(change.Joins) > 0 || len(change.Leaves) > 0 {
			notify = ch

			r.Logger.DebugCtx(
				ctx,
				"cluster membership has changed",
				slog.Int("nodes_before", len(before)),
				slog.Int("nodes_after", len(after)),
			)
		}

		select {
		case <-poll.C:
			continue
		case <-ctx.Done():
			return ctx.Err()
		case notify <- change:
			r.Logger.DebugCtx(
				ctx,
				"membership notification delivered",
				slog.Int("nodes_before", len(before)),
				slog.Int("nodes_after", len(after)),
			)
			before = after
		}
	}
}

// members returns a map of the nodes that are currently members of the cluster.
func (r *Registry) members(ctx context.Context) (map[uuid.UUID]Node, error) {
	// Don't allow the query to consume the entire poll interval.
	timeout := r.PollInterval / 2
	if timeout <= 0 {
		timeout = DefaultRegistryPollInterval / 2
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	members := map[uuid.UUID]Node{}

	return members, protokv.RangeAll(
		ctx,
		r.Keyspace,
		func(
			ctx context.Context,
			k []byte,
			n *registrypb.Node,
		) (bool, error) {
			if expired, err := r.deleteIfExpired(ctx, n); expired {
				return true, err
			}

			nn := unmarshalNode(n)
			members[nn.ID] = nn

			return true, nil
		},
	)
}

// deleteIfExpired deletes the node with the given ID if it has expired.
func (r *Registry) deleteIfExpired(
	ctx context.Context,
	n *registrypb.Node,
) (bool, error) {
	if n.ExpiresAt.AsTime().After(time.Now()) {
		return false, nil
	}

	return true, r.Keyspace.Set(
		ctx,
		n.Id.AsBytes(),
		nil,
	)
}

// membershipDiff returns the changes to membership since the last notification
// was sent.
func membershipDiff(before, after map[uuid.UUID]Node) MembershipChange {
	var change MembershipChange

	for id, n := range after {
		if _, ok := before[id]; !ok {
			change.Joins = append(change.Joins, n)
		}
	}

	for id, n := range before {
		if _, ok := after[id]; !ok {
			change.Leaves = append(change.Leaves, n)
		}
	}

	return change
}

// marshalNode converts a Node to its protocol buffer representation.
func marshalNode(n Node, expiresAt time.Time) *registrypb.Node {
	return &registrypb.Node{
		Id:        uuidpb.FromByteArray(n.ID),
		ExpiresAt: timestamppb.New(expiresAt),
		Addresses: n.Addresses,
	}
}

// unmarshalNode converts a Node from its protocol buffer representation.
func unmarshalNode(n *registrypb.Node) Node {
	return Node{
		ID:        uuidpb.AsByteArray[uuid.UUID](n.Id),
		Addresses: n.Addresses,
	}
}
