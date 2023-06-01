package veracity

import (
	"context"
	"reflect"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/veracity/internal/integration"
	"github.com/dogmatiq/veracity/internal/telemetry"
	"github.com/dogmatiq/veracity/persistence/journal"
	"github.com/dogmatiq/veracity/persistence/kv"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// An EngineOption configures the behavior of an [Engine].
type EngineOption func(*engineConfig)

// engineConfig encapsulates the configuration of an [Engine], built by applying
// [EngineOption] functions, and visiting the [configkit.RichApplication] that
// represents the Dogma application that the engine hosts.
type engineConfig struct {
	NodeID    uuid.UUID
	Telemetry *telemetry.Provider

	Persistence struct {
		Journals  journal.Store
		Keyspaces kv.Store
	}

	GRPC struct {
		DialOptions        []grpc.DialOption
		ServerOptions      []grpc.ServerOption
		ListenAddress      string
		AdvertiseAddresses []string
	}

	Application struct {
		Executors map[reflect.Type]dogma.CommandExecutor
	}
}

func newEngineConfig(app dogma.Application, options []EngineOption) engineConfig {
	cfg := engineConfig{}

	for _, opt := range options {
		opt(&cfg)
	}

	err := configkit.
		FromApplication(app).
		AcceptRichVisitor(context.Background(), &cfg)

	if err != nil {
		panic(err)
	}

	return cfg
}

func (c *engineConfig) VisitRichApplication(ctx context.Context, cfg configkit.RichApplication) error {
	return cfg.RichHandlers().AcceptRichVisitor(ctx, c)
}

func (c *engineConfig) VisitRichAggregate(ctx context.Context, cfg configkit.RichAggregate) error {
	return nil
}

func (c *engineConfig) VisitRichProcess(ctx context.Context, cfg configkit.RichProcess) error {
	return nil
}

func (c *engineConfig) VisitRichIntegration(ctx context.Context, cfg configkit.RichIntegration) error {
	for t := range cfg.MessageTypes().Consumed {
		if c.Application.Executors == nil {
			c.Application.Executors = map[reflect.Type]dogma.CommandExecutor{}
		}

		c.Application.Executors[t.ReflectType()] = &integration.CommandExecutor{
			Handler: cfg.Handler(),
		}
	}

	return nil
}

func (c *engineConfig) VisitRichProjection(ctx context.Context, cfg configkit.RichProjection) error {
	return nil
}
