package engineconfig

import (
	"context"
	"reflect"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/enginekit/protobuf/identitypb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/ferrite"
	"github.com/dogmatiq/veracity/internal/telemetry"
	"github.com/dogmatiq/veracity/persistence/journal"
	"github.com/dogmatiq/veracity/persistence/kv"
	"google.golang.org/grpc"
)

// FerriteRegistry is a registry of the environment variables used by Veracity.
var FerriteRegistry = ferrite.NewRegistry(
	"dogmatiq.veracity",
	"Veracity",
	ferrite.WithDocumentationURL("https://github.com/dogmatiq/veracity#readme"),
)

// Config encapsulates the configuration of a [veracity.Engine], built by
// applying [veracity.EngineOption] functions, and visiting the
// [configkit.RichApplication] that represents the Dogma application that the
// engine hosts.
type Config struct {
	UseEnv    bool
	SiteID    *identitypb.Identity
	NodeID    *uuidpb.UUID
	Telemetry *telemetry.Provider
	Tasks     []func(context.Context) error

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
		Telemetry: &telemetry.Provider{},
	}

	for _, opt := range options {
		opt(&c)
	}

	c.finalize()

	err := configkit.
		FromApplication(app).
		AcceptRichVisitor(
			context.Background(),
			applicationVisitor{
				Config: &c,
			},
		)

	if err != nil {
		panic(err)
	}

	return c
}

func (c *Config) finalize() {
	c.finalizeNodeID()
	c.finalizeTelemetry()
	c.finalizePersistence()
	c.finalizeGRPC()
}
