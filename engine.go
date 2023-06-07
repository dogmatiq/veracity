package veracity

import (
	"context"
	"fmt"
	"reflect"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/veracity/internal/engineconfig"
	"golang.org/x/sync/errgroup"
)

// Engine hosts a Dogma application.
type Engine struct {
	executors   map[reflect.Type]dogma.CommandExecutor
	heartbeater *heartbeater
}

// New returns an engine that hosts the given application.
func New(app dogma.Application, options ...EngineOption) *Engine {
	if app == nil {
		panic("application must not be nil")
	}

	return newEngine(
		engineconfig.New(app, options),
	)
}

func newEngine(cfg engineconfig.Config) *Engine {
	return &Engine{
		executors:   cfg.Application.Executors,
		heartbeater: newHeartbeater(cfg),
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

	x, ok := e.executors[reflect.TypeOf(c)]
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

	g.Go(func() error {
		return e.heartbeater.Run(ctx)
	})

	return g.Wait()
}

// Close stops the engine immediately.
func (e *Engine) Close() error {
	return nil
}
