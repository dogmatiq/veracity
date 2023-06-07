package engineconfig

import (
	"context"
	"reflect"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/ferrite"
	"github.com/dogmatiq/veracity/internal/telemetry"
	"github.com/dogmatiq/veracity/persistence/journal"
	"github.com/dogmatiq/veracity/persistence/kv"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// FerriteRegistry is a registry of the environment variables used by Veracity.
var FerriteRegistry = ferrite.NewRegistry("dogmatiq/veracity")

// Config encapsulates the configuration of a [veracity.Engine], built by
// applying [veracity.EngineOption] functions, and visiting the
// [configkit.RichApplication] that represents the Dogma application that the
// engine hosts.
type Config struct {
	UseEnv    bool
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

// New returns a new configuration for a [veracity.Engine] that hosts the given
// Dogma application.
func New[Option ~func(*Config)](
	app dogma.Application,
	options []Option,
) Config {
	c := Config{
		NodeID:    uuid.New(),
		Telemetry: &telemetry.Provider{},
	}

	for _, opt := range options {
		opt(&c)
	}

	err := configkit.
		FromApplication(app).
		AcceptRichVisitor(context.Background(), applicationVisitor{&c})

	if err != nil {
		panic(err)
	}

	c.finalize()

	return c
}

func (c *Config) finalize() {
	c.finalizeNodeID()
	c.finalizeTelemetry()
	c.finalizePersistence()
	c.finalizeGRPC()
}
