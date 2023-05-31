package veracity

import (
	"context"
	"fmt"
	"reflect"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/veracity/internal/integration"
)

// Engine hosts a Dogma application.
type Engine struct {
	executors map[reflect.Type]dogma.CommandExecutor
}

// New returns an engine that hosts the given application.
func New(app dogma.Application, options ...EngineOption) *Engine {
	if app == nil {
		panic("application must not be nil")
	}

	e := &Engine{
		executors: map[reflect.Type]dogma.CommandExecutor{},
	}

	for _, opt := range options {
		opt.applyEngineOption(e)
	}

	cfg := configkit.FromApplication(app)
	b := &engineBuilder{engine: e}
	if err := cfg.AcceptRichVisitor(context.Background(), b); err != nil {
		panic(err)
	}

	return e
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

type engineBuilder struct {
	engine *Engine
}

func (b *engineBuilder) VisitRichApplication(ctx context.Context, cfg configkit.RichApplication) error {
	return cfg.RichHandlers().AcceptRichVisitor(ctx, b)
}

func (b *engineBuilder) VisitRichAggregate(ctx context.Context, cfg configkit.RichAggregate) error {
	return nil
}

func (b *engineBuilder) VisitRichProcess(ctx context.Context, cfg configkit.RichProcess) error {
	return nil
}

func (b *engineBuilder) VisitRichIntegration(ctx context.Context, cfg configkit.RichIntegration) error {
	for t := range cfg.MessageTypes().Consumed {
		b.engine.executors[t.ReflectType()] = &integration.CommandExecutor{
			Handler: cfg.Handler(),
		}
	}

	return nil
}

func (b *engineBuilder) VisitRichProjection(ctx context.Context, cfg configkit.RichProjection) error {
	return nil
}
