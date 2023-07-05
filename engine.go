package veracity

import (
	"context"
	"fmt"
	"net"
	"reflect"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/veracity/internal/cluster"
	"github.com/dogmatiq/veracity/internal/engineconfig"
	"golang.org/x/sync/errgroup"
)

// Engine hosts a Dogma application.
type Engine struct {
	config engineconfig.Config
}

// New returns an engine that hosts the given application.
func New(app dogma.Application, options ...EngineOption) *Engine {
	if app == nil {
		panic("application must not be nil")
	}

	cfg := engineconfig.New(app, options)
	return &Engine{cfg}
}

// ExecuteCommand enqueues a command for execution.
func (e *Engine) ExecuteCommand(ctx context.Context, c dogma.Command) error {
	if c == nil {
		panic("command must not be nil")
	}

	if err := c.Validate(); err != nil {
		panic(fmt.Sprintf("command is invalid: %s", err))
	}

	x, ok := e.config.Application.Executors[reflect.TypeOf(c)]
	if !ok {
		panic(fmt.Sprintf("command is unrecognized: %T", c))
	}

	return x.ExecuteCommand(ctx, c)
}

// Run joins the cluster as a worker that handles the application's messages.
//
// [Engine.ExecuteCommand] may be called without calling [Engine.Run]. In this
// mode of operation, the engine acts solely as a router that forwards messages
// to worker nodes.
//
// It blocks until ctx is canceled or an error occurs.
func (e *Engine) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	lis, err := net.Listen("tcp", e.config.GRPC.ListenAddress)
	if err != nil {
		return err
	}
	defer lis.Close()

	addresses := e.config.GRPC.AdvertiseAddresses
	if len(addresses) == 0 {
		// TODO: cross-product of this port with non-loopback IP addresses
		addresses = []string{lis.Addr().String()}
	}

	g.Go(func() error {
		registrar := &cluster.Registrar{
			Keyspaces: e.config.Persistence.Keyspaces,
			Node: cluster.Node{
				ID:        e.config.NodeID,
				Addresses: addresses,
			},
			Logger: e.config.Telemetry.Logger,
		}
		return registrar.Run(ctx)
	})

	for _, task := range e.config.Tasks {
		task := task // capture loop variable
		g.Go(func() error {
			return task(ctx)
		})
	}

	return g.Wait()
}

// Close stops the engine immediately.
func (e *Engine) Close() error {
	return nil
}
