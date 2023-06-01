package veracity

import (
	"context"
	"fmt"
	"reflect"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/veracity/internal/telemetry/instrumentedpersistence"
	"github.com/dogmatiq/veracity/persistence/journal"
	"github.com/dogmatiq/veracity/persistence/kv"
)

// Engine hosts a Dogma application.
type Engine struct {
	journals  journal.Store
	keyspaces kv.Store
	executors map[reflect.Type]dogma.CommandExecutor
}

// New returns an engine that hosts the given application.
func New(app dogma.Application, options ...EngineOption) *Engine {
	if app == nil {
		panic("application must not be nil")
	}

	cfg := newEngineConfig(app, options)

	return &Engine{
		journals: &instrumentedpersistence.JournalStore{
			Next:      cfg.Persistence.Journals,
			Telemetry: cfg.Telemetry,
		},
		keyspaces: &instrumentedpersistence.KeyValueStore{
			Next:      cfg.Persistence.Keyspaces,
			Telemetry: cfg.Telemetry,
		},
		executors: cfg.Application.Executors,
	}
}

// ExecuteCommand enqueues a command for execution.
func (e *Engine) ExecuteCommand(ctx context.Context, c dogma.Command) error {
	if c == nil {
		panic("command must not be nil")
	}

	if err := c.Validate(); err != nil {
		panic(fmt.Sprintf("command is invalid: %s", err))
	}

	if x, ok := e.executors[reflect.TypeOf(c)]; ok {
		return x.ExecuteCommand(ctx, c)
	}

	panic(fmt.Sprintf("command is unrecognized: %T", c))
}

// Run joins the cluster as a worker that handles the application's messages.
//
// [Engine.ExecuteCommand] may be called without calling [Engine.Run]. In this
// mode of operation, the engine acts solely as a router that forwards messages
// to worker nodes.
//
// It blocks until ctx is canceled or an error occurs.
func (e *Engine) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

// Close stops the engine.
func (e *Engine) Close() error {
	return nil
}
