package veracity

import (
	"context"
	"net"
	"time"

	"github.com/dogmatiq/veracity/internal/cluster"
	"github.com/dogmatiq/veracity/internal/engineconfig"
	"github.com/dogmatiq/veracity/persistence/kv"
	"github.com/google/uuid"
	"golang.org/x/exp/slog"
)

type heartbeater struct {
	NodeID         uuid.UUID
	Keyspaces      kv.Store
	ListenAddr     string
	AdvertiseAddrs []string
	Logger         *slog.Logger
}

func newHeartbeater(cfg engineconfig.Config) *heartbeater {
	return &heartbeater{
		NodeID:         cfg.NodeID,
		Keyspaces:      cfg.Persistence.Keyspaces,
		ListenAddr:     cfg.GRPC.ListenAddress,
		AdvertiseAddrs: cfg.GRPC.AdvertiseAddresses,
		Logger:         cfg.Telemetry.Logger,
	}
}

func (r *heartbeater) Run(ctx context.Context) error {
	lis, err := net.Listen("tcp", r.ListenAddr)
	if err != nil {
		return err
	}
	defer lis.Close()

	ks, err := r.Keyspaces.Open(ctx, cluster.RegistryKeyspace)
	if err != nil {
		return err
	}
	defer ks.Close()

	reg := &cluster.Registry{
		Keyspace:          ks,
		PollInterval:      0, // TODO: make configurable via an engine option
		HeartbeatInterval: 0, // TODO: make configurable via an engine option
		Logger:            r.Logger,
	}

	addresses := r.AdvertiseAddrs
	if len(addresses) == 0 {
		addresses = []string{
			lis.Addr().String(),
		}
	}

	node := cluster.Node{
		ID:        r.NodeID,
		Addresses: addresses,
	}

	delay, err := reg.Register(ctx, node)

	for err == nil {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := shutdownContextFor(ctx)
			defer cancel()

			if err := reg.Deregister(shutdownCtx, node.ID); err != nil {
				return err
			}

			return ctx.Err()

		case <-time.After(delay):
			delay, err = reg.Heartbeat(ctx, node.ID)
		}
	}

	return err
}
